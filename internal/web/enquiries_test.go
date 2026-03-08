package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/rajeshkrishnamurthy/sbdeals/internal/books"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/bundles"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/clicked"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/rails"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/suppliers"
)

func TestEnquiriesListDefaultsToClickedTab(t *testing.T) {
	clickedStore := clicked.NewMemoryStore()
	_, _ = clickedStore.CreateClicked(clicked.CreateInput{
		ItemID:     1,
		ItemType:   clicked.ItemTypeBundle,
		ItemTitle:  "Bundle One",
		SourcePage: "catalog",
	})
	server := newEnquiriesTestServer(clickedStore)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/enquiries", nil)
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Clicked") || !strings.Contains(body, "Bundle One") {
		t.Fatalf("expected clicked tab/body content, got: %s", body)
	}
}

func TestConvertEnquiryToInterestedSuccessAndIdempotent(t *testing.T) {
	clickedStore := clicked.NewMemoryStore()
	created, _ := clickedStore.CreateClicked(clicked.CreateInput{
		ItemID:     5,
		ItemType:   clicked.ItemTypeBook,
		ItemTitle:  "Book Five",
		SourcePage: "catalog",
	})
	server := newEnquiriesTestServer(clickedStore)

	form := url.Values{}
	form.Set("buyer_name", "Rajesh")
	form.Set("buyer_phone", "9876543210")
	form.Set("buyer_note", "Evening call")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/enquiries/"+strconvI(created.ID)+"/convert", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", rr.Code)
	}
	location := rr.Header().Get("Location")
	if !strings.Contains(location, "flash=Enquiry+converted+to+Interested.") {
		t.Fatalf("expected success flash redirect, got %q", location)
	}

	clickedRows, _ := clickedStore.ListByStatus(clicked.StatusClicked)
	if len(clickedRows) != 0 {
		t.Fatalf("expected clicked tab to be empty after conversion, got %+v", clickedRows)
	}
	interestedRows, _ := clickedStore.ListByStatus(clicked.StatusInterested)
	if len(interestedRows) != 1 || interestedRows[0].BuyerPhone != "+919876543210" {
		t.Fatalf("unexpected interested rows: %+v", interestedRows)
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/admin/enquiries/"+strconvI(created.ID)+"/convert", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	server.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 on second conversion, got %d", rr.Code)
	}
	if !strings.Contains(rr.Header().Get("Location"), "Already+converted.") {
		t.Fatalf("expected already converted feedback, got %q", rr.Header().Get("Location"))
	}
}

func TestConvertEnquiryValidation(t *testing.T) {
	clickedStore := clicked.NewMemoryStore()
	created, _ := clickedStore.CreateClicked(clicked.CreateInput{
		ItemID:     5,
		ItemType:   clicked.ItemTypeBook,
		ItemTitle:  "Book Five",
		SourcePage: "catalog",
	})
	server := newEnquiriesTestServer(clickedStore)

	form := url.Values{}
	form.Set("buyer_name", "Rajesh")
	form.Set("buyer_phone", "123")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/enquiries/"+strconvI(created.ID)+"/convert", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", rr.Code)
	}
	if !strings.Contains(rr.Header().Get("Location"), "valid+10-digit+India+mobile+number") {
		t.Fatalf("expected validation redirect, got %q", rr.Header().Get("Location"))
	}
}

func TestConvertEnquiryNotFoundReturns404(t *testing.T) {
	clickedStore := clicked.NewMemoryStore()
	server := newEnquiriesTestServer(clickedStore)

	form := url.Values{}
	form.Set("buyer_name", "Rajesh")
	form.Set("buyer_phone", "9876543210")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/enquiries/999/convert", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestEnquiryItemMethodGuard(t *testing.T) {
	clickedStore := clicked.NewMemoryStore()
	server := newEnquiriesTestServer(clickedStore)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/enquiries/1/convert", nil)
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

func TestEnquiriesInterestedTabRendersConvertedRows(t *testing.T) {
	clickedStore := clicked.NewMemoryStore()
	created, _ := clickedStore.CreateClicked(clicked.CreateInput{
		ItemID:     8,
		ItemType:   clicked.ItemTypeBundle,
		ItemTitle:  "Bundle Eight",
		SourcePage: "catalog",
	})
	_, _, _ = clickedStore.ConvertToInterested(created.ID, clicked.ConvertInput{
		BuyerName:   "Rajesh",
		BuyerPhone:  "+919876543210",
		BuyerNote:   "note",
		ConvertedBy: "system-admin",
	})

	server := newEnquiriesTestServer(clickedStore)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/enquiries?status=interested", nil)
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Bundle Eight") || !strings.Contains(body, "919876543210") {
		t.Fatalf("expected interested row details in body, got: %s", body)
	}
}

func newEnquiriesTestServer(clickedStore clicked.Store) *Server {
	return NewServerWithStoresAndClicked(
		suppliers.NewMemoryStore(),
		books.NewMemoryStore(),
		bundles.NewMemoryStore(nil, nil),
		rails.NewMemoryStore(),
		clickedStore,
	)
}

func strconvI(v int) string { return strconv.Itoa(v) }
