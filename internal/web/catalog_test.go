package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rajeshkrishnamurthy/sbdeals/internal/books"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/bundles"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/rails"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/suppliers"
)

func TestCatalogRootRendersShellAndAsset(t *testing.T) {
	s := newCatalogTestServer(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	body := rr.Body.String()
	checks := []string{
		"Srikar Book Deals",
		`id="catalog-root"`,
		`data-endpoint="/api/catalog"`,
		`/assets/catalog.js`,
	}
	for _, check := range checks {
		if !strings.Contains(body, check) {
			t.Fatalf("expected root page to contain %q", check)
		}
	}
}

func TestCatalogDataReturnsPublishedRailsInBackendOrder(t *testing.T) {
	s := newCatalogTestServer(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)

	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("unexpected content type: %q", got)
	}

	var payload catalogResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if len(payload.Rails) != 3 {
		t.Fatalf("expected 3 published rails, got %d", len(payload.Rails))
	}
	if payload.Rails[0].Title != "Weekend Bundles" || payload.Rails[1].Title != "Fresh Books" || payload.Rails[2].Title != "Quiet Rail" {
		t.Fatalf("unexpected rail order: %+v", payload.Rails)
	}

	bundleRail := payload.Rails[0]
	if len(bundleRail.Items) != 1 {
		t.Fatalf("expected one visible bundle item, got %+v", bundleRail.Items)
	}
	if bundleRail.Items[0].Type != "BUNDLE" {
		t.Fatalf("expected bundle item type, got %+v", bundleRail.Items[0])
	}
	if bundleRail.Items[0].ImageURL != "/admin/bundles/1/image" {
		t.Fatalf("unexpected bundle image url: %+v", bundleRail.Items[0])
	}
	if bundleRail.Items[0].ReserveButtonLabel != "Reserve on WhatsApp" {
		t.Fatalf("unexpected bundle CTA: %+v", bundleRail.Items[0])
	}

	bookRail := payload.Rails[1]
	if len(bookRail.Items) != 1 {
		t.Fatalf("expected one visible book item, got %+v", bookRail.Items)
	}
	if bookRail.Items[0].Title != "Published Book" {
		t.Fatalf("unexpected book item title: %+v", bookRail.Items[0])
	}
	if bookRail.Items[0].ImageURL != "/admin/books/1/cover" {
		t.Fatalf("unexpected book image url: %+v", bookRail.Items[0])
	}
	if bookRail.Items[0].CurrentPriceText != "Rs. 250.00" || bookRail.Items[0].OriginalPriceText != "Rs. 400.00" || bookRail.Items[0].DiscountText != "38%" {
		t.Fatalf("unexpected book pricing block: %+v", bookRail.Items[0])
	}

	emptyRail := payload.Rails[2]
	if len(emptyRail.Items) != 0 {
		t.Fatalf("expected empty rail shell, got %+v", emptyRail.Items)
	}
}

func TestCatalogDataReturnsInlineFailureContract(t *testing.T) {
	supplierStore := suppliers.NewMemoryStore()
	bookStore := books.NewMemoryStore()
	bundleStore := bundles.NewMemoryStore(nil, nil)
	s := NewServerWithStores(supplierStore, bookStore, bundleStore, failingRailStore{})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("unexpected content type: %q", got)
	}
	if !strings.Contains(rr.Body.String(), "failed to load catalog") {
		t.Fatalf("expected catalog error payload")
	}
}

func TestCatalogAssetServesRetryAndComingSoonBehavior(t *testing.T) {
	s := newCatalogTestServer(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/assets/catalog.js", nil)

	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/javascript; charset=utf-8" {
		t.Fatalf("unexpected content type: %q", got)
	}

	body := rr.Body.String()
	checks := []string{"Coming soon", "Retry", "loadCatalog"}
	for _, check := range checks {
		if !strings.Contains(body, check) {
			t.Fatalf("expected asset body to contain %q", check)
		}
	}
}

func newCatalogTestServer(t *testing.T) *Server {
	t.Helper()

	supplierStore := suppliers.NewMemoryStore()
	supplier, err := supplierStore.Create(suppliers.Input{Name: "Supplier A", WhatsApp: "+91-1", Location: "Bengaluru"})
	if err != nil {
		t.Fatalf("create supplier: %v", err)
	}

	bookStore := books.NewMemoryStore()
	publishedBook, err := bookStore.Create(books.CreateInput{
		Title:      "Published Book",
		Cover:      books.Cover{Data: []byte("book-image"), MimeType: "image/png"},
		SupplierID: supplier.ID,
		Category:   "Fiction",
		Format:     "Paperback",
		Condition:  "Very good",
		MRP:        400,
		MyPrice:    250,
		Author:     "Author One",
	})
	if err != nil {
		t.Fatalf("create published book: %v", err)
	}
	if _, err := bookStore.Publish(publishedBook.ID); err != nil {
		t.Fatalf("publish published book: %v", err)
	}

	unpublishedBook, err := bookStore.Create(books.CreateInput{
		Title:      "Hidden Book",
		Cover:      books.Cover{Data: []byte("hidden-image"), MimeType: "image/png"},
		SupplierID: supplier.ID,
		Category:   "Fiction",
		Format:     "Paperback",
		Condition:  "Very good",
		MRP:        300,
		MyPrice:    200,
		Author:     "Author Two",
	})
	if err != nil {
		t.Fatalf("create hidden book: %v", err)
	}

	supplierNames := map[int]string{supplier.ID: supplier.Name}
	pickerBooks := []bundles.PickerBook{
		{BookID: 10, Title: "Bundle Book One", Author: "Bundle Author One", SupplierID: supplier.ID, Category: "Fiction", Condition: "Very good", MRP: 500, MyPrice: 280, InStock: true},
		{BookID: 11, Title: "Bundle Book Two", Author: "Bundle Author Two", SupplierID: supplier.ID, Category: "Fiction", Condition: "Good as new", MRP: 400, MyPrice: 220, InStock: true},
	}
	bundleStore := bundles.NewMemoryStore(supplierNames, pickerBooks)
	publishedBundle, err := bundleStore.Create(bundles.CreateInput{
		Name:              "Weekend Starter",
		SupplierID:        supplier.ID,
		Category:          "Fiction",
		AllowedConditions: []string{"Very good", "Good as new"},
		BookIDs:           []int{10, 11},
		BundlePrice:       499,
		Image:             bundles.Image{Data: []byte("bundle-image"), MimeType: "image/png"},
	})
	if err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	if _, err := bundleStore.Publish(publishedBundle.ID); err != nil {
		t.Fatalf("publish bundle: %v", err)
	}

	railStore := rails.NewMemoryStore()
	bookRail, err := railStore.Create(rails.CreateInput{Title: "Fresh Books", Type: rails.RailTypeBook})
	if err != nil {
		t.Fatalf("create book rail: %v", err)
	}
	if _, err := railStore.AddItem(bookRail.ID, publishedBook.ID); err != nil {
		t.Fatalf("add published book to rail: %v", err)
	}
	if _, err := railStore.Publish(bookRail.ID); err != nil {
		t.Fatalf("publish book rail: %v", err)
	}

	bundleRail, err := railStore.Create(rails.CreateInput{Title: "Weekend Bundles", Type: rails.RailTypeBundle})
	if err != nil {
		t.Fatalf("create bundle rail: %v", err)
	}
	if _, err := railStore.AddItem(bundleRail.ID, publishedBundle.ID); err != nil {
		t.Fatalf("add bundle to rail: %v", err)
	}
	if _, err := railStore.Publish(bundleRail.ID); err != nil {
		t.Fatalf("publish bundle rail: %v", err)
	}
	if err := railStore.MoveUp(bundleRail.ID); err != nil {
		t.Fatalf("move bundle rail up: %v", err)
	}

	emptyRail, err := railStore.Create(rails.CreateInput{Title: "Quiet Rail", Type: rails.RailTypeBook})
	if err != nil {
		t.Fatalf("create empty rail: %v", err)
	}
	if _, err := railStore.AddItem(emptyRail.ID, unpublishedBook.ID); err != nil {
		t.Fatalf("add hidden book to empty rail: %v", err)
	}
	if _, err := railStore.Publish(emptyRail.ID); err != nil {
		t.Fatalf("publish empty rail: %v", err)
	}

	unpublishedRail, err := railStore.Create(rails.CreateInput{Title: "Do Not Show", Type: rails.RailTypeBook})
	if err != nil {
		t.Fatalf("create unpublished rail: %v", err)
	}
	if _, err := railStore.AddItem(unpublishedRail.ID, publishedBook.ID); err != nil {
		t.Fatalf("add book to unpublished rail: %v", err)
	}

	return NewServerWithStores(supplierStore, bookStore, bundleStore, railStore)
}

type failingRailStore struct{}

func (failingRailStore) List() ([]rails.ListItem, error) { return nil, errCatalogTestFailure }
func (failingRailStore) Create(input rails.CreateInput) (rails.Rail, error) {
	return rails.Rail{}, errCatalogTestFailure
}
func (failingRailStore) Get(id int) (rails.Rail, error) { return rails.Rail{}, errCatalogTestFailure }
func (failingRailStore) Update(id int, input rails.UpdateInput) (rails.Rail, error) {
	return rails.Rail{}, errCatalogTestFailure
}
func (failingRailStore) AddItem(id int, itemID int) (rails.Rail, error) {
	return rails.Rail{}, errCatalogTestFailure
}
func (failingRailStore) RemoveItem(id int, itemID int) (rails.Rail, error) {
	return rails.Rail{}, errCatalogTestFailure
}
func (failingRailStore) Publish(id int) (rails.Rail, error) {
	return rails.Rail{}, errCatalogTestFailure
}
func (failingRailStore) Unpublish(id int) (rails.Rail, error) {
	return rails.Rail{}, errCatalogTestFailure
}
func (failingRailStore) MoveUp(id int) error   { return errCatalogTestFailure }
func (failingRailStore) MoveDown(id int) error { return errCatalogTestFailure }

var errCatalogTestFailure = http.ErrAbortHandler
