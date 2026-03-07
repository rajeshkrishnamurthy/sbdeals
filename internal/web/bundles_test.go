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
	"github.com/rajeshkrishnamurthy/sbdeals/internal/suppliers"
)

func TestBundlesListRendersColumnsAndNav(t *testing.T) {
	s := newBundleTestServer(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/bundles", nil)
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	body := rr.Body.String()
	checks := []string{"Bundles", "Add Bundle", "Supplier", "Category", "Allowed condition(s)", "# of books", "Bundle price", "View/Edit"}
	for _, check := range checks {
		if !strings.Contains(body, check) {
			t.Fatalf("expected body to contain %q", check)
		}
	}
	assertAdminNav(t, body, "/admin/bundles")
}

func TestCreateBundleSuccess(t *testing.T) {
	supplierStore, bundleStore := newBundleStores(t)
	bookStore := books.NewMemoryStore()
	s := NewServerWithStores(supplierStore, bookStore, bundleStore)

	form := url.Values{}
	form.Set("name", "Starter")
	form.Set("supplier_id", "1")
	form.Set("category", "Fiction")
	form.Add("allowed_conditions", "Very good")
	form.Add("allowed_conditions", "Good as new")
	form.Add("book_ids", "10")
	form.Add("book_ids", "11")
	form.Set("bundle_price", "499")
	form.Set("notes", "Weekend deal")

	req := httptest.NewRequest(http.MethodPost, "/admin/bundles", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", rr.Code)
	}
	if !strings.HasPrefix(rr.Header().Get("Location"), "/admin/bundles?flash=") {
		t.Fatalf("unexpected redirect: %s", rr.Header().Get("Location"))
	}

	created, err := bundleStore.Get(1)
	if err != nil {
		t.Fatalf("expected created bundle: %v", err)
	}
	if created.SupplierID != 1 || created.Category != "Fiction" || len(created.BookIDs) != 2 {
		t.Fatalf("unexpected created bundle: %+v", created)
	}
}

func TestCreateBundleRequiresAtLeastTwoBooks(t *testing.T) {
	supplierStore, bundleStore := newBundleStores(t)
	s := NewServerWithStores(supplierStore, books.NewMemoryStore(), bundleStore)

	form := url.Values{}
	form.Set("supplier_id", "1")
	form.Set("category", "Fiction")
	form.Add("allowed_conditions", "Very good")
	form.Add("book_ids", "10")
	form.Set("bundle_price", "300")

	req := httptest.NewRequest(http.MethodPost, "/admin/bundles", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Bundle must include at least 2 books") {
		t.Fatalf("expected min books validation error")
	}
}

func TestEditBundleRevalidatesWhenSupplierChanges(t *testing.T) {
	supplierStore, bundleStore := newBundleStores(t)
	_, err := bundleStore.Create(bundles.CreateInput{
		Name:              "Starter",
		SupplierID:        1,
		Category:          "Fiction",
		AllowedConditions: []string{"Very good", "Good as new"},
		BookIDs:           []int{10, 11},
		BundlePrice:       499,
		Notes:             "x",
	})
	if err != nil {
		t.Fatalf("seed create failed: %v", err)
	}

	s := NewServerWithStores(supplierStore, books.NewMemoryStore(), bundleStore)

	form := url.Values{}
	form.Set("name", "Starter")
	form.Set("supplier_id", "2")
	form.Set("category", "Fiction")
	form.Add("allowed_conditions", "Very good")
	form.Add("book_ids", "10")
	form.Add("book_ids", "11")
	form.Set("bundle_price", "499")

	req := httptest.NewRequest(http.MethodPost, "/admin/bundles/1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Selected books must match the chosen supplier, category, and allowed conditions") {
		t.Fatalf("expected revalidation error")
	}
}

func TestBundlesTrailingSlashRedirects(t *testing.T) {
	s := newBundleTestServer(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/bundles/", nil)
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusMovedPermanently {
		t.Fatalf("expected 301, got %d", rr.Code)
	}
	if rr.Header().Get("Location") != "/admin/bundles" {
		t.Fatalf("expected redirect to /admin/bundles, got %q", rr.Header().Get("Location"))
	}
}

func TestBundleNewIncludesEnhancementScript(t *testing.T) {
	s := newBundleTestServer(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/bundles/new", nil)
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "/assets/bundles-form.js") {
		t.Fatalf("expected bundles form enhancement script tag")
	}
}

func TestBundleNewUsesInternalScrollForEligibleBooks(t *testing.T) {
	s := newBundleTestServer(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/bundles/new", nil)
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `class="eligible-scroll"`) {
		t.Fatalf("expected eligible books internal scroll container")
	}
	if !strings.Contains(body, `.eligible-scroll { max-height:420px; overflow-y:auto;`) {
		t.Fatalf("expected fixed-height scroll styling for eligible books")
	}
}

func newBundleTestServer(t *testing.T) *Server {
	t.Helper()
	supplierStore, bundleStore := newBundleStores(t)
	_, err := bundleStore.Create(bundles.CreateInput{
		Name:              "Starter",
		SupplierID:        1,
		Category:          "Fiction",
		AllowedConditions: []string{"Very good", "Good as new"},
		BookIDs:           []int{10, 11},
		BundlePrice:       499,
	})
	if err != nil {
		t.Fatalf("seed create failed: %v", err)
	}
	return NewServerWithStores(supplierStore, books.NewMemoryStore(), bundleStore)
}

func newBundleStores(t *testing.T) (*suppliers.MemoryStore, *bundles.MemoryStore) {
	t.Helper()
	supplierStore := suppliers.NewMemoryStore()
	first, err := supplierStore.Create(suppliers.Input{Name: "Supplier A", WhatsApp: "+91-1", Location: "Bengaluru"})
	if err != nil {
		t.Fatalf("create supplier 1 failed: %v", err)
	}
	second, err := supplierStore.Create(suppliers.Input{Name: "Supplier B", WhatsApp: "+91-2", Location: "Chennai"})
	if err != nil {
		t.Fatalf("create supplier 2 failed: %v", err)
	}

	supplierNames := map[int]string{first.ID: first.Name, second.ID: second.Name}
	pickerBooks := []bundles.PickerBook{
		{BookID: 10, Title: "Book A", Author: "Author A", SupplierID: first.ID, Category: "Fiction", Condition: "Very good", MRP: 400, MyPrice: 250},
		{BookID: 11, Title: "Book B", Author: "Author B", SupplierID: first.ID, Category: "Fiction", Condition: "Good as new", MRP: 500, MyPrice: 300},
		{BookID: 12, Title: "Book C", Author: "Author C", SupplierID: first.ID, Category: "Fiction", Condition: "Used", MRP: 350, MyPrice: 220},
		{BookID: 20, Title: "Book Z", Author: "Author Z", SupplierID: second.ID, Category: "Non-Fiction", Condition: "Very good", MRP: 280, MyPrice: 180},
	}
	return supplierStore, bundles.NewMemoryStore(supplierNames, pickerBooks)
}

func TestBundlePathRouteRoundTrip(t *testing.T) {
	id, ok := parseBundlePath("/admin/bundles/123")
	if !ok || id != 123 {
		t.Fatalf("unexpected parse result: %v %v", id, ok)
	}

	_, ok = parseBundlePath("/admin/bundles/123/books")
	if ok {
		t.Fatalf("expected parse failure for nested path")
	}

	_, ok = parseBundlePath("/admin/bundles/" + strconv.Itoa(0))
	if ok {
		t.Fatalf("expected invalid parse for id 0")
	}
}

func TestBundlesFormAssetRouteServesJavaScript(t *testing.T) {
	s := newBundleTestServer(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/assets/bundles-form.js", nil)
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/javascript; charset=utf-8" {
		t.Fatalf("unexpected content type: %q", got)
	}
	if !strings.Contains(rr.Body.String(), "bundle-book-search") {
		t.Fatalf("expected bundles-form.js content")
	}
}
