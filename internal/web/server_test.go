package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/rajeshkrishnamurthy/sbdeals/internal/suppliers"
)

func TestSuppliersListIncludesColumnsAndAction(t *testing.T) {
	s := NewServer(suppliers.NewMemoryStore())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/suppliers", nil)

	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	body := rr.Body.String()
	checks := []string{"Suppliers", "Add Supplier", "Name", "WhatsApp number", "Location"}
	for _, check := range checks {
		if !strings.Contains(body, check) {
			t.Fatalf("expected body to contain %q", check)
		}
	}
	assertAdminNav(t, body, "/admin/suppliers")
}

func TestCreateSupplierRedirectsAndListShowsCreatedSupplier(t *testing.T) {
	store := suppliers.NewMemoryStore()
	s := NewServer(store)

	form := url.Values{}
	form.Set("name", "City Books")
	form.Set("whatsapp", "+91-9000011111")
	form.Set("location", "Bengaluru")
	form.Set("notes", "Near metro station")

	createReq := httptest.NewRequest(http.MethodPost, "/admin/suppliers", strings.NewReader(form.Encode()))
	createReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	createRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(createRR, createReq)

	if createRR.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", createRR.Code)
	}
	redirect := createRR.Header().Get("Location")
	if !strings.HasPrefix(redirect, "/admin/suppliers?flash=") {
		t.Fatalf("unexpected redirect target: %s", redirect)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/suppliers", nil)
	listRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(listRR, listReq)

	if listRR.Code != http.StatusOK {
		t.Fatalf("expected 200 on list, got %d", listRR.Code)
	}
	body := listRR.Body.String()
	if !strings.Contains(body, "City Books") {
		t.Fatalf("expected list to include created supplier")
	}
	if !strings.Contains(body, "View/Edit") {
		t.Fatalf("expected list to include row action View/Edit")
	}
}

func TestCreateSupplierMissingLocationReturnsValidationError(t *testing.T) {
	s := NewServer(suppliers.NewMemoryStore())
	form := url.Values{}
	form.Set("name", "City Books")
	form.Set("whatsapp", "+91-9000011111")
	form.Set("location", "")

	req := httptest.NewRequest(http.MethodPost, "/admin/suppliers", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Location is required") {
		t.Fatalf("expected location validation error in response body")
	}
	if !strings.Contains(rr.Body.String(), `class="toast-error"`) {
		t.Fatalf("expected toast error for validation failure")
	}
	if !strings.Contains(rr.Body.String(), "Please fix: Location is required.") {
		t.Fatalf("expected toast summary text for validation failure")
	}
}

func TestSupplierDetailAndUpdateFlow(t *testing.T) {
	store := suppliers.NewMemoryStore()
	created, err := store.Create(suppliers.Input{
		Name:     "Old Name",
		WhatsApp: "+91-9000099999",
		Location: "Chennai",
		Notes:    "Old note",
	})
	if err != nil {
		t.Fatalf("create returned error: %v", err)
	}
	s := NewServer(store)

	viewReq := httptest.NewRequest(http.MethodGet, "/admin/suppliers/1", nil)
	viewRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(viewRR, viewReq)

	if viewRR.Code != http.StatusOK {
		t.Fatalf("expected 200 for detail page, got %d", viewRR.Code)
	}
	viewBody := viewRR.Body.String()
	if !strings.Contains(viewBody, "View/Edit Supplier") || !strings.Contains(viewBody, "Old Name") {
		t.Fatalf("detail page missing expected content")
	}

	form := url.Values{}
	form.Set("name", "Updated Name")
	form.Set("whatsapp", "+91-9888877777")
	form.Set("location", "Hyderabad")
	form.Set("notes", "Updated note")

	updateReq := httptest.NewRequest(http.MethodPost, "/admin/suppliers/1", strings.NewReader(form.Encode()))
	updateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	updateRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(updateRR, updateReq)

	if updateRR.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 after update, got %d", updateRR.Code)
	}
	if updateRR.Header().Get("Location") != "/admin/suppliers/1?flash=Supplier+updated+successfully." {
		t.Fatalf("unexpected update redirect: %s", updateRR.Header().Get("Location"))
	}

	updated, err := store.Get(created.ID)
	if err != nil {
		t.Fatalf("expected supplier to exist after update: %v", err)
	}
	if updated.Name != "Updated Name" || updated.Location != "Hyderabad" {
		t.Fatalf("supplier was not updated correctly: %+v", updated)
	}
}

func TestMethodNotAllowedOnSuppliersRoutes(t *testing.T) {
	s := NewServer(suppliers.NewMemoryStore())

	req := httptest.NewRequest(http.MethodDelete, "/admin/suppliers", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
	if allow := rr.Header().Get("Allow"); allow != "GET, POST" {
		t.Fatalf("expected Allow header GET, POST, got %q", allow)
	}

	body, err := io.ReadAll(rr.Body)
	if err != nil {
		t.Fatalf("failed reading response: %v", err)
	}
	if !strings.Contains(string(body), "method not allowed") {
		t.Fatalf("expected method not allowed response body")
	}
}

func TestSupplierFormsUseUnifiedAdminNav(t *testing.T) {
	store := suppliers.NewMemoryStore()
	created, err := store.Create(suppliers.Input{
		Name:     "A1",
		WhatsApp: "+91-1",
		Location: "Bengaluru",
	})
	if err != nil {
		t.Fatalf("create supplier: %v", err)
	}

	s := NewServer(store)

	newReq := httptest.NewRequest(http.MethodGet, "/admin/suppliers/new", nil)
	newRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(newRR, newReq)
	if newRR.Code != http.StatusOK {
		t.Fatalf("expected 200 for new supplier page, got %d", newRR.Code)
	}
	assertAdminNav(t, newRR.Body.String(), "/admin/suppliers")

	detailReq := httptest.NewRequest(http.MethodGet, "/admin/suppliers/"+strconv.Itoa(created.ID), nil)
	detailRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(detailRR, detailReq)
	if detailRR.Code != http.StatusOK {
		t.Fatalf("expected 200 for supplier detail page, got %d", detailRR.Code)
	}
	assertAdminNav(t, detailRR.Body.String(), "/admin/suppliers")
}

func assertAdminNav(t *testing.T, body string, activePath string) {
	t.Helper()
	if !strings.Contains(body, `href="/admin/suppliers"`) {
		t.Fatalf("expected suppliers link in admin nav")
	}
	if !strings.Contains(body, `href="/admin/books"`) {
		t.Fatalf("expected books link in admin nav")
	}
	if !strings.Contains(body, `href="/admin/bundles"`) {
		t.Fatalf("expected bundles link in admin nav")
	}
	if !strings.Contains(body, `href="/admin/rails"`) {
		t.Fatalf("expected rails link in admin nav")
	}
	activeMarkup := `href="` + activePath + `" class="admin-nav-link active"`
	if !strings.Contains(body, activeMarkup) {
		t.Fatalf("expected active nav link %q", activePath)
	}
}
