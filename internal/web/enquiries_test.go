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
	"github.com/rajeshkrishnamurthy/sbdeals/internal/customers"
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
	server, _ := newEnquiriesTestServer(clickedStore)

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
	server, customerStore := newEnquiriesTestServer(clickedStore)
	customer, _ := customerStore.Create(customers.CreateInput{Name: "Rajesh", Mobile: "9876543210"})

	form := url.Values{}
	form.Set("customer_id", strconv.Itoa(customer.ID))
	form.Set("note", "Evening call")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/enquiries/"+strconv.Itoa(created.ID)+"/convert", strings.NewReader(form.Encode()))
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
	if len(interestedRows) != 1 || interestedRows[0].CustomerID != customer.ID {
		t.Fatalf("unexpected interested rows: %+v", interestedRows)
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/admin/enquiries/"+strconv.Itoa(created.ID)+"/convert", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	server.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 on second conversion, got %d", rr.Code)
	}
	if !strings.Contains(rr.Header().Get("Location"), "Already+converted.") {
		t.Fatalf("expected already converted feedback, got %q", rr.Header().Get("Location"))
	}
}

func TestConvertEnquiryQuickCreateCustomerWhenNotSelected(t *testing.T) {
	clickedStore := clicked.NewMemoryStore()
	created, _ := clickedStore.CreateClicked(clicked.CreateInput{
		ItemID:     5,
		ItemType:   clicked.ItemTypeBook,
		ItemTitle:  "Book Five",
		SourcePage: "catalog",
	})
	server, _ := newEnquiriesTestServer(clickedStore)

	form := url.Values{}
	form.Set("quick_customer_name", "Asha")
	form.Set("quick_customer_mobile", "9123456789")
	form.Set("note", "quick create")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/enquiries/"+strconv.Itoa(created.ID)+"/convert", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", rr.Code)
	}
	if !strings.Contains(rr.Header().Get("Location"), "flash=Enquiry+converted+to+Interested.") {
		t.Fatalf("unexpected redirect: %q", rr.Header().Get("Location"))
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
	server, _ := newEnquiriesTestServer(clickedStore)

	form := url.Values{}
	form.Set("note", "x")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/enquiries/"+strconv.Itoa(created.ID)+"/convert", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", rr.Code)
	}
	if !strings.Contains(rr.Header().Get("Location"), "Customer+selection+is+required.") {
		t.Fatalf("expected validation redirect, got %q", rr.Header().Get("Location"))
	}
}

func TestConvertEnquiryRejectsMixedExistingAndQuickCreate(t *testing.T) {
	clickedStore := clicked.NewMemoryStore()
	created, _ := clickedStore.CreateClicked(clicked.CreateInput{
		ItemID:     6,
		ItemType:   clicked.ItemTypeBook,
		ItemTitle:  "Book Six",
		SourcePage: "catalog",
	})
	server, customerStore := newEnquiriesTestServer(clickedStore)
	customer, _ := customerStore.Create(customers.CreateInput{Name: "Rajesh", Mobile: "9876543210"})

	form := url.Values{}
	form.Set("customer_id", strconv.Itoa(customer.ID))
	form.Set("quick_customer_name", "Asha")
	form.Set("quick_customer_mobile", "9123456789")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/enquiries/"+strconv.Itoa(created.ID)+"/convert", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", rr.Code)
	}
	location := rr.Header().Get("Location")
	if !strings.Contains(location, "Choose+either+existing+customer+or+quick-create+details.") {
		t.Fatalf("expected mixed input validation error, got %q", location)
	}

	clickedRows, _ := clickedStore.ListByStatus(clicked.StatusClicked)
	if len(clickedRows) != 1 {
		t.Fatalf("expected enquiry to remain clicked, got %+v", clickedRows)
	}
	interestedRows, _ := clickedStore.ListByStatus(clicked.StatusInterested)
	if len(interestedRows) != 0 {
		t.Fatalf("expected no interested rows, got %+v", interestedRows)
	}
}

func TestConvertEnquiryNotFoundReturns404(t *testing.T) {
	clickedStore := clicked.NewMemoryStore()
	server, customerStore := newEnquiriesTestServer(clickedStore)
	customer, _ := customerStore.Create(customers.CreateInput{Name: "Rajesh", Mobile: "9876543210"})

	form := url.Values{}
	form.Set("customer_id", strconv.Itoa(customer.ID))
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
	server, _ := newEnquiriesTestServer(clickedStore)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/enquiries/1/convert", nil)
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

func TestEnquiriesInterestedTabRendersCustomerDetails(t *testing.T) {
	clickedStore := clicked.NewMemoryStore()
	created, _ := clickedStore.CreateClicked(clicked.CreateInput{
		ItemID:     8,
		ItemType:   clicked.ItemTypeBundle,
		ItemTitle:  "Bundle Eight",
		SourcePage: "catalog",
	})
	server, customerStore := newEnquiriesTestServer(clickedStore)
	customer, _ := customerStore.Create(customers.CreateInput{Name: "Rajesh", Mobile: "9876543210"})
	_, _, _ = clickedStore.ConvertToInterested(created.ID, clicked.ConvertInput{
		CustomerID: customer.ID,
		Note:       "note",
		ModifiedBy: "system-admin",
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/enquiries?status=interested", nil)
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Bundle Eight") || !strings.Contains(body, customer.Name) || !strings.Contains(body, customer.Mobile) {
		t.Fatalf("expected interested row customer details in body, got: %s", body)
	}
}

func newEnquiriesTestServer(clickedStore clicked.Store) (*Server, *customers.MemoryStore) {
	customerStore := customers.NewMemoryStore()
	return NewServerWithAllStores(
		suppliers.NewMemoryStore(),
		books.NewMemoryStore(),
		bundles.NewMemoryStore(nil, nil),
		rails.NewMemoryStore(),
		clickedStore,
		customerStore,
	), customerStore
}
