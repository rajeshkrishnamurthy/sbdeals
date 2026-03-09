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
	form.Set("customer_search", "raje")
	form.Set("note", "call later")
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
	if !strings.Contains(location, "open_convert_modal=1") {
		t.Fatalf("expected modal to reopen on validation, got %q", location)
	}
	if !strings.Contains(location, "modal_enquiry_id="+strconv.Itoa(created.ID)) {
		t.Fatalf("expected enquiry id in modal redirect, got %q", location)
	}
	if !strings.Contains(location, "customer_id="+strconv.Itoa(customer.ID)) {
		t.Fatalf("expected customer id to be preserved, got %q", location)
	}
	if !strings.Contains(location, "quick_customer_name=Asha") {
		t.Fatalf("expected quick customer name to be preserved, got %q", location)
	}
	if !strings.Contains(location, "quick_customer_mobile=9123456789") {
		t.Fatalf("expected quick customer mobile to be preserved, got %q", location)
	}
	if !strings.Contains(location, "customer_search=raje") {
		t.Fatalf("expected customer search to be preserved, got %q", location)
	}
	if !strings.Contains(location, "note=call+later") {
		t.Fatalf("expected note to be preserved, got %q", location)
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

func TestEnquiriesListValidationToastShownInModalWhenReopenRequested(t *testing.T) {
	clickedStore := clicked.NewMemoryStore()
	created, _ := clickedStore.CreateClicked(clicked.CreateInput{
		ItemID:     9,
		ItemType:   clicked.ItemTypeBook,
		ItemTitle:  "Book Nine",
		SourcePage: "catalog",
	})
	server, _ := newEnquiriesTestServer(clickedStore)

	req := httptest.NewRequest(
		http.MethodGet,
		"/admin/enquiries?status=clicked&error=Choose+either+existing+customer+or+quick-create+details.&open_convert_modal=1&modal_enquiry_id="+strconv.Itoa(created.ID)+"&customer_id=7&quick_customer_name=Asha&quick_customer_mobile=9123456789&note=call+later&customer_search=ra",
		nil,
	)
	rr := httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `data-open-on-load="1"`) {
		t.Fatalf("expected modal open-on-load marker, got body: %s", body)
	}
	if !strings.Contains(body, `role="alert">Choose either existing customer or quick-create details.`) {
		t.Fatalf("expected in-modal validation toast, got body: %s", body)
	}
	if !strings.Contains(body, `data-enquiry-id="`+strconv.Itoa(created.ID)+`"`) {
		t.Fatalf("expected modal enquiry id in markup, got body: %s", body)
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

	orderRR := httptest.NewRecorder()
	orderReq := httptest.NewRequest(http.MethodGet, "/admin/enquiries/1/order", nil)
	server.Handler().ServeHTTP(orderRR, orderReq)
	if orderRR.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for order route, got %d", orderRR.Code)
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

func TestConvertEnquiryToOrderedSuccess(t *testing.T) {
	clickedStore := clicked.NewMemoryStore()
	created, _ := clickedStore.CreateClicked(clicked.CreateInput{
		ItemID:     12,
		ItemType:   clicked.ItemTypeBundle,
		ItemTitle:  "Bundle Twelve",
		SourcePage: "catalog",
	})
	server, customerStore := newEnquiriesTestServer(clickedStore)
	address := "Block B, Indiranagar"
	city := "Bengaluru"
	apartment := "Skyline Residency"
	customer, _ := customerStore.Create(customers.CreateInput{
		Name:          "Asha",
		Mobile:        "9999999999",
		Address:       &address,
		CityName:      &city,
		ApartmentName: &apartment,
	})
	_, _, _ = clickedStore.ConvertToInterested(created.ID, clicked.ConvertInput{
		CustomerID: customer.ID,
		ModifiedBy: "system-admin",
	})

	form := url.Values{}
	form.Set("order_amount", "499")
	form.Set("note", "Confirmed")
	form.Set("city_name", "Chennai")
	form.Set("apartment_name", "Marina Heights")
	form.Set("address", "Flat 401, Marina")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/enquiries/"+strconv.Itoa(created.ID)+"/order", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", rr.Code)
	}
	location := rr.Header().Get("Location")
	if !strings.Contains(location, "flash=Enquiry+converted+to+Ordered.") {
		t.Fatalf("expected ordered success redirect, got %q", location)
	}

	interestedRows, _ := clickedStore.ListByStatus(clicked.StatusInterested)
	if len(interestedRows) != 0 {
		t.Fatalf("expected interested queue empty, got %+v", interestedRows)
	}
	orderedRows, _ := clickedStore.ListByStatus(clicked.StatusOrdered)
	if len(orderedRows) != 1 {
		t.Fatalf("expected one ordered row, got %+v", orderedRows)
	}
	if orderedRows[0].OrderAmount == nil || *orderedRows[0].OrderAmount != 499 {
		t.Fatalf("expected order amount 499, got %+v", orderedRows[0].OrderAmount)
	}
	updatedCustomer, err := customerStore.Get(customer.ID)
	if err != nil {
		t.Fatalf("get customer: %v", err)
	}
	if updatedCustomer.CityName != "Chennai" || updatedCustomer.ApartmentName != "Marina Heights" {
		t.Fatalf("expected updated city/apartment, got city=%q apartment=%q", updatedCustomer.CityName, updatedCustomer.ApartmentName)
	}
	if updatedCustomer.Address == nil || *updatedCustomer.Address != "Flat 401, Marina" {
		t.Fatalf("expected updated address, got %+v", updatedCustomer.Address)
	}
}

func TestConvertEnquiryToOrderedRequiresAddressIfMissing(t *testing.T) {
	clickedStore := clicked.NewMemoryStore()
	created, _ := clickedStore.CreateClicked(clicked.CreateInput{
		ItemID:     13,
		ItemType:   clicked.ItemTypeBook,
		ItemTitle:  "Book Thirteen",
		SourcePage: "catalog",
	})
	server, customerStore := newEnquiriesTestServer(clickedStore)
	customer, _ := customerStore.Create(customers.CreateInput{Name: "Rohan", Mobile: "8888888888"})
	_, _, _ = clickedStore.ConvertToInterested(created.ID, clicked.ConvertInput{
		CustomerID: customer.ID,
		ModifiedBy: "system-admin",
	})

	form := url.Values{}
	form.Set("order_amount", "350")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/enquiries/"+strconv.Itoa(created.ID)+"/order", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", rr.Code)
	}
	location := rr.Header().Get("Location")
	if !strings.Contains(location, "Address+is+required+to+convert+to+Ordered.") {
		t.Fatalf("expected address validation message, got %q", location)
	}
	if !strings.Contains(location, "open_order_modal=1") {
		t.Fatalf("expected order modal reopen marker, got %q", location)
	}
}

func TestConvertEnquiryToOrderedRejectsClickedStatus(t *testing.T) {
	clickedStore := clicked.NewMemoryStore()
	created, _ := clickedStore.CreateClicked(clicked.CreateInput{
		ItemID:     14,
		ItemType:   clicked.ItemTypeBundle,
		ItemTitle:  "Bundle Fourteen",
		SourcePage: "catalog",
	})
	server, _ := newEnquiriesTestServer(clickedStore)

	form := url.Values{}
	form.Set("order_amount", "250")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/enquiries/"+strconv.Itoa(created.ID)+"/order", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", rr.Code)
	}
	if !strings.Contains(rr.Header().Get("Location"), "Only+interested+enquiries+can+be+ordered.") {
		t.Fatalf("expected invalid transition message, got %q", rr.Header().Get("Location"))
	}
}

func TestEnquiriesInterestedTabShowsConvertToOrderedAction(t *testing.T) {
	clickedStore := clicked.NewMemoryStore()
	created, _ := clickedStore.CreateClicked(clicked.CreateInput{
		ItemID:     15,
		ItemType:   clicked.ItemTypeBook,
		ItemTitle:  "Book Fifteen",
		SourcePage: "catalog",
	})
	server, customerStore := newEnquiriesTestServer(clickedStore)
	address := "HSR Layout"
	city := "Bengaluru"
	apartment := "Palm Meadows"
	customer, _ := customerStore.Create(customers.CreateInput{
		Name:          "Vikram",
		Mobile:        "7777777777",
		Address:       &address,
		CityName:      &city,
		ApartmentName: &apartment,
	})
	_, _, _ = clickedStore.ConvertToInterested(created.ID, clicked.ConvertInput{
		CustomerID: customer.ID,
		ModifiedBy: "system-admin",
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/enquiries?status=interested", nil)
	server.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Convert to Ordered") {
		t.Fatalf("expected convert-to-ordered action, got body: %s", body)
	}
	if strings.Contains(body, "Convert to Interested") {
		t.Fatalf("did not expect clicked conversion action on interested tab")
	}
	if !strings.Contains(body, `/assets/enquiries-form.js`) {
		t.Fatalf("expected enquiries form asset on interested tab")
	}
	if !strings.Contains(body, `name="city_name"`) || !strings.Contains(body, `name="apartment_name"`) || !strings.Contains(body, `name="address"`) {
		t.Fatalf("expected city/apartment/address fields in order modal")
	}
	if !strings.Contains(body, `data-customer-city="Bengaluru"`) || !strings.Contains(body, `data-customer-apartment="Palm Meadows"`) || !strings.Contains(body, `data-customer-address="HSR Layout"`) {
		t.Fatalf("expected existing city/apartment/address values on order action")
	}
}

func TestEnquiriesFormAssetServesModalWiring(t *testing.T) {
	server, _ := newEnquiriesTestServer(clicked.NewMemoryStore())

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/assets/enquiries-form.js", nil)
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "initOrderModal") {
		t.Fatalf("expected enquiries form asset content")
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
