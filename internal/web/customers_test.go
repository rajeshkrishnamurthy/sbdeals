package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/rajeshkrishnamurthy/sbdeals/internal/books"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/bundles"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/clicked"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/customers"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/rails"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/suppliers"
)

func TestCreateCustomerAndDuplicateFlow(t *testing.T) {
	server, customerStore := newCustomerTestServer()

	form := url.Values{
		"name":   {"Rajesh"},
		"mobile": {"98765-43210"},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/customers", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", rr.Code)
	}
	if !strings.HasPrefix(rr.Header().Get("Location"), "/admin/customers/") {
		t.Fatalf("unexpected redirect: %s", rr.Header().Get("Location"))
	}

	created, err := customerStore.Get(1)
	if err != nil {
		t.Fatalf("expected created customer: %v", err)
	}
	if created.Mobile != "9876543210" {
		t.Fatalf("expected normalized mobile, got %q", created.Mobile)
	}

	dupReq := httptest.NewRequest(http.MethodPost, "/admin/customers", strings.NewReader(form.Encode()))
	dupReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	dupRR := httptest.NewRecorder()
	server.Handler().ServeHTTP(dupRR, dupReq)

	if dupRR.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", dupRR.Code)
	}
	body := dupRR.Body.String()
	if !strings.Contains(body, "Customer already exists.") {
		t.Fatalf("expected duplicate error in body")
	}
	if !strings.Contains(body, "/admin/customers/1") || !strings.Contains(body, "View Existing") || !strings.Contains(body, "Edit Existing") {
		t.Fatalf("expected duplicate action links in body")
	}
}

func TestCustomerRequiresCityBeforeApartment(t *testing.T) {
	server, _ := newCustomerTestServer()
	form := url.Values{
		"name":           {"Asha"},
		"mobile":         {"9876543211"},
		"apartment_name": {"Skyline"},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/customers", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Select city before apartment complex.") {
		t.Fatalf("expected city-before-apartment validation")
	}
}

func TestCustomerListSearchAndCityFilter(t *testing.T) {
	server, customerStore := newCustomerTestServer()
	_, _ = customerStore.Create(customers.CreateInput{Name: "Asha", Mobile: "9999999991", CityName: textPtr("Bengaluru"), ApartmentName: textPtr("Skyline")})
	_, _ = customerStore.Create(customers.CreateInput{Name: "Rohan", Mobile: "9999999992", CityName: textPtr("Chennai"), ApartmentName: textPtr("Marina")})
	cities, _ := customerStore.ListCities()
	var bengaluruID string
	for _, city := range cities {
		if city.Name == "Bengaluru" {
			bengaluruID = strconvI(city.ID)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/customers?q=9992", nil)
	rr := httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "Rohan") || strings.Contains(rr.Body.String(), "Asha") {
		t.Fatalf("unexpected search list output")
	}

	filterReq := httptest.NewRequest(http.MethodGet, "/admin/customers?city_id="+url.QueryEscape(bengaluruID), nil)
	filterRR := httptest.NewRecorder()
	server.Handler().ServeHTTP(filterRR, filterReq)
	if filterRR.Code != http.StatusOK || !strings.Contains(filterRR.Body.String(), "Asha") || strings.Contains(filterRR.Body.String(), "Rohan") {
		t.Fatalf("unexpected city-filter list output")
	}
}

func TestCustomerEditDoesNotChangeMobile(t *testing.T) {
	server, customerStore := newCustomerTestServer()
	created, _ := customerStore.Create(customers.CreateInput{Name: "Asha", Mobile: "9999999991"})

	form := url.Values{
		"name":   {"Asha Updated"},
		"mobile": {"1111111111"},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/customers/"+strconvI(created.ID), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", rr.Code)
	}
	updated, err := customerStore.Get(created.ID)
	if err != nil {
		t.Fatalf("get updated customer: %v", err)
	}
	if updated.Mobile != "9999999991" {
		t.Fatalf("expected immutable mobile, got %s", updated.Mobile)
	}
	if updated.Name != "Asha Updated" {
		t.Fatalf("expected updated name, got %s", updated.Name)
	}
}

func newCustomerTestServer() (*Server, *customers.MemoryStore) {
	supplierStore := suppliers.NewMemoryStore()
	customerStore := customers.NewMemoryStore()
	return NewServerWithAllStores(
		supplierStore,
		books.NewMemoryStore(),
		bundles.NewMemoryStore(nil, nil),
		rails.NewMemoryStore(),
		clicked.NewMemoryStore(),
		customerStore,
	), customerStore
}

func textPtr(value string) *string {
	return &value
}
