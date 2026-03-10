package web

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"regexp"
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
	checks := []string{"Bundles", "Add Bundle", "Image", "Supplier", "Category", "# of books", "Bundle price", "Discount %", "Publish", "View/Edit"}
	for _, check := range checks {
		if !strings.Contains(body, check) {
			t.Fatalf("expected body to contain %q", check)
		}
	}
	if strings.Contains(body, "Allowed condition(s)") {
		t.Fatalf("did not expect allowed conditions list column")
	}
	if !strings.Contains(body, "No image") {
		t.Fatalf("expected no-image placeholder in list")
	}
	assertAdminNav(t, body, "/admin/bundles")
	if !strings.Contains(body, `action="/admin/bundles/1/publish"`) {
		t.Fatalf("expected publish toggle action in list row")
	}
	if !strings.Contains(body, "45%") {
		t.Fatalf("expected discount column to show rounded percent")
	}
	if !regexp.MustCompile(`\(\d+d\)`).MatchString(body) {
		t.Fatalf("expected recency indicator like (Xd)")
	}
}

func TestCreateBundleSuccess(t *testing.T) {
	supplierStore, bundleStore := newBundleStores(t)
	bookStore := books.NewMemoryStore()
	s := NewServerWithStores(supplierStore, bookStore, bundleStore)

	contentType, body := multipartBundleForm(t, bundleFormParts{
		Fields: map[string][]string{
			"name":               {"Starter"},
			"supplier_id":        {"1"},
			"category":           {"Fiction"},
			"allowed_conditions": {"Very good", "Good as new"},
			"book_ids":           {"10", "11"},
			"bundle_price":       {"499"},
			"notes":              {"Weekend deal"},
		},
		FileField:   "image",
		FileName:    "bundle.jpg",
		FileContent: []byte("bundle-image"),
	})
	req := httptest.NewRequest(http.MethodPost, "/admin/bundles", body)
	req.Header.Set("Content-Type", contentType)
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

	contentType, body := multipartBundleForm(t, bundleFormParts{
		Fields: map[string][]string{
			"supplier_id":        {"1"},
			"category":           {"Fiction"},
			"allowed_conditions": {"Very good"},
			"book_ids":           {"10"},
			"bundle_price":       {"300"},
		},
		FileField:   "image",
		FileName:    "bundle.jpg",
		FileContent: []byte("bundle-image"),
	})
	req := httptest.NewRequest(http.MethodPost, "/admin/bundles", body)
	req.Header.Set("Content-Type", contentType)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Minimum 2 items required unless one selected item is marked Box Set.") {
		t.Fatalf("expected min books validation error")
	}
	if !strings.Contains(rr.Body.String(), `class="toast-error"`) {
		t.Fatalf("expected toast error for validation failure")
	}
	if !strings.Contains(rr.Body.String(), "Please fix: Minimum 2 items required unless one selected item is marked Box Set.") {
		t.Fatalf("expected toast summary text for validation failure")
	}
}

func TestCreateBundleAllowsSingleBoxSetBook(t *testing.T) {
	supplierStore, bundleStore := newBundleStores(t)
	s := NewServerWithStores(supplierStore, books.NewMemoryStore(), bundleStore)

	contentType, body := multipartBundleForm(t, bundleFormParts{
		Fields: map[string][]string{
			"name":               {"Box Set Single"},
			"supplier_id":        {"1"},
			"category":           {"Fiction"},
			"allowed_conditions": {"Very good"},
			"book_ids":           {"30"},
			"bundle_price":       {"300"},
		},
		FileField:   "image",
		FileName:    "bundle.jpg",
		FileContent: []byte("bundle-image"),
	})
	req := httptest.NewRequest(http.MethodPost, "/admin/bundles", body)
	req.Header.Set("Content-Type", contentType)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", rr.Code)
	}
	if !strings.HasPrefix(rr.Header().Get("Location"), "/admin/bundles?flash=") {
		t.Fatalf("unexpected redirect: %s", rr.Header().Get("Location"))
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

	contentType, body := multipartBundleForm(t, bundleFormParts{
		Fields: map[string][]string{
			"name":               {"Starter"},
			"supplier_id":        {"2"},
			"category":           {"Fiction"},
			"allowed_conditions": {"Very good"},
			"book_ids":           {"10", "11"},
			"bundle_price":       {"499"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/admin/bundles/1", body)
	req.Header.Set("Content-Type", contentType)
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
	if !strings.Contains(rr.Body.String(), "Bundle image") {
		t.Fatalf("expected bundle image field")
	}
	if !strings.Contains(rr.Body.String(), "Minimum 2 items required unless one selected item is marked Box Set.") {
		t.Fatalf("expected conditional minimum-items helper text")
	}
}

func TestCreateBundleRequiresImage(t *testing.T) {
	supplierStore, bundleStore := newBundleStores(t)
	s := NewServerWithStores(supplierStore, books.NewMemoryStore(), bundleStore)

	contentType, body := multipartBundleForm(t, bundleFormParts{
		Fields: map[string][]string{
			"supplier_id":        {"1"},
			"category":           {"Fiction"},
			"allowed_conditions": {"Very good"},
			"book_ids":           {"10", "11"},
			"bundle_price":       {"499"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/admin/bundles", body)
	req.Header.Set("Content-Type", contentType)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	bodyText := rr.Body.String()
	if !strings.Contains(bodyText, "Bundle image is required.") {
		t.Fatalf("expected missing-image validation message")
	}
	if !strings.Contains(bodyText, `class="toast-error"`) {
		t.Fatalf("expected toast validation error")
	}
}

func TestBundleImageRouteServesImage(t *testing.T) {
	supplierStore, bundleStore := newBundleStores(t)
	created, err := bundleStore.Create(bundles.CreateInput{
		Name:              "Starter",
		SupplierID:        1,
		Category:          "Fiction",
		AllowedConditions: []string{"Very good", "Good as new"},
		BookIDs:           []int{10, 11},
		BundlePrice:       499,
		Image:             bundles.Image{Data: []byte("bundle-image"), MimeType: "image/png"},
	})
	if err != nil {
		t.Fatalf("seed create failed: %v", err)
	}

	s := NewServerWithStores(supplierStore, books.NewMemoryStore(), bundleStore)
	req := httptest.NewRequest(http.MethodGet, "/admin/bundles/"+strconv.Itoa(created.ID)+"/image", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if rr.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("expected image/png content type, got %q", rr.Header().Get("Content-Type"))
	}
	if rr.Body.String() != "bundle-image" {
		t.Fatalf("unexpected image bytes")
	}
}

func TestBundleEditIncludesPublishToggleAndRecency(t *testing.T) {
	supplierStore, bundleStore := newBundleStores(t)
	created, err := bundleStore.Create(bundles.CreateInput{
		Name:              "Starter",
		SupplierID:        1,
		Category:          "Fiction",
		AllowedConditions: []string{"Very good", "Good as new"},
		BookIDs:           []int{10, 11},
		BundlePrice:       499,
		Image:             bundles.Image{Data: []byte("bundle-image"), MimeType: "image/png"},
	})
	if err != nil {
		t.Fatalf("seed create failed: %v", err)
	}
	s := NewServerWithStores(supplierStore, books.NewMemoryStore(), bundleStore)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/bundles/"+strconv.Itoa(created.ID), nil)
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `action="/admin/bundles/1/publish?from=edit"`) {
		t.Fatalf("expected edit publish toggle action")
	}
	if !strings.Contains(body, `/admin/bundles/1/image`) {
		t.Fatalf("expected existing bundle image preview on edit")
	}
	if !regexp.MustCompile(`\(\d+d\)`).MatchString(body) {
		t.Fatalf("expected recency indicator on edit page")
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
		{BookID: 10, Title: "Book A", Author: "Author A", SupplierID: first.ID, Category: "Fiction", Condition: "Very good", MRP: 400, MyPrice: 250, InStock: true},
		{BookID: 11, Title: "Book B", Author: "Author B", SupplierID: first.ID, Category: "Fiction", Condition: "Good as new", MRP: 500, MyPrice: 300, InStock: true},
		{BookID: 12, Title: "Book C", Author: "Author C", SupplierID: first.ID, Category: "Fiction", Condition: "Used", MRP: 350, MyPrice: 220, InStock: true},
		{BookID: 30, Title: "Box Set A", Author: "Author Box", SupplierID: first.ID, IsBoxSet: true, Category: "Fiction", Condition: "Very good", MRP: 600, MyPrice: 350, InStock: true},
		{BookID: 20, Title: "Book Z", Author: "Author Z", SupplierID: second.ID, Category: "Non-Fiction", Condition: "Very good", MRP: 280, MyPrice: 180, InStock: true},
	}
	return supplierStore, bundles.NewMemoryStore(supplierNames, pickerBooks)
}

func TestBundlePathRouteRoundTrip(t *testing.T) {
	id, action, ok := parseBundlePath("/admin/bundles/123")
	if !ok || id != 123 || action != "" {
		t.Fatalf("unexpected parse result: %v %v %v", id, action, ok)
	}

	id, action, ok = parseBundlePath("/admin/bundles/123/publish")
	if !ok || id != 123 || action != "publish" {
		t.Fatalf("unexpected publish parse result: %v %v %v", id, action, ok)
	}

	id, action, ok = parseBundlePath("/admin/bundles/123/unpublish")
	if !ok || id != 123 || action != "unpublish" {
		t.Fatalf("unexpected unpublish parse result: %v %v %v", id, action, ok)
	}
	id, action, ok = parseBundlePath("/admin/bundles/123/image")
	if !ok || id != 123 || action != "image" {
		t.Fatalf("unexpected image parse result: %v %v %v", id, action, ok)
	}

	_, _, ok = parseBundlePath("/admin/bundles/123/books")
	if ok {
		t.Fatalf("expected parse failure for nested path")
	}

	_, _, ok = parseBundlePath("/admin/bundles/" + strconv.Itoa(0))
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

func TestBundlePublishAndUnpublishActions(t *testing.T) {
	supplierStore, bundleStore := newBundleStores(t)
	created, err := bundleStore.Create(bundles.CreateInput{
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

	s := NewServerWithStores(supplierStore, books.NewMemoryStore(), bundleStore)

	publishReq := httptest.NewRequest(http.MethodPost, "/admin/bundles/"+strconv.Itoa(created.ID)+"/publish", nil)
	publishRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(publishRR, publishReq)
	if publishRR.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 on publish, got %d", publishRR.Code)
	}
	if publishRR.Header().Get("Location") != "/admin/bundles?flash=Bundle+published+successfully." {
		t.Fatalf("unexpected publish redirect: %s", publishRR.Header().Get("Location"))
	}
	afterPublish, err := bundleStore.Get(created.ID)
	if err != nil {
		t.Fatalf("get after publish: %v", err)
	}
	if !afterPublish.IsPublished {
		t.Fatalf("expected published=true")
	}

	unpublishReq := httptest.NewRequest(http.MethodPost, "/admin/bundles/"+strconv.Itoa(created.ID)+"/unpublish", nil)
	unpublishRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(unpublishRR, unpublishReq)
	if unpublishRR.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 on unpublish, got %d", unpublishRR.Code)
	}
	if unpublishRR.Header().Get("Location") != "/admin/bundles?flash=Bundle+unpublished+successfully." {
		t.Fatalf("unexpected unpublish redirect: %s", unpublishRR.Header().Get("Location"))
	}
	afterUnpublish, err := bundleStore.Get(created.ID)
	if err != nil {
		t.Fatalf("get after unpublish: %v", err)
	}
	if afterUnpublish.IsPublished {
		t.Fatalf("expected published=false")
	}
}

func TestBundlePublishFailsWithOutOfStockTitlesShowsToast(t *testing.T) {
	supplierStore := suppliers.NewMemoryStore()
	first, err := supplierStore.Create(suppliers.Input{Name: "Supplier A", WhatsApp: "+91-1", Location: "Bengaluru"})
	if err != nil {
		t.Fatalf("create supplier 1 failed: %v", err)
	}
	supplierNames := map[int]string{first.ID: first.Name}
	bundleStore := bundles.NewMemoryStore(supplierNames, []bundles.PickerBook{
		{BookID: 10, Title: "Book A", Author: "Author A", SupplierID: first.ID, Category: "Fiction", Condition: "Very good", MRP: 400, MyPrice: 250, InStock: true},
		{BookID: 11, Title: "Book B", Author: "Author B", SupplierID: first.ID, Category: "Fiction", Condition: "Good as new", MRP: 500, MyPrice: 300, InStock: false},
	})
	created, err := bundleStore.Create(bundles.CreateInput{
		Name:              "Starter",
		SupplierID:        first.ID,
		Category:          "Fiction",
		AllowedConditions: []string{"Very good", "Good as new"},
		BookIDs:           []int{10, 11},
		BundlePrice:       499,
	})
	if err != nil {
		t.Fatalf("seed create failed: %v", err)
	}

	s := NewServerWithStores(supplierStore, books.NewMemoryStore(), bundleStore)
	req := httptest.NewRequest(http.MethodPost, "/admin/bundles/"+strconv.Itoa(created.ID)+"/publish", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", rr.Code)
	}
	if !strings.HasPrefix(rr.Header().Get("Location"), "/admin/bundles?error=") {
		t.Fatalf("unexpected redirect: %s", rr.Header().Get("Location"))
	}
	listReq := httptest.NewRequest(http.MethodGet, rr.Header().Get("Location"), nil)
	listRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(listRR, listReq)
	body := listRR.Body.String()
	if !strings.Contains(body, "Cannot publish bundle because these books are out of stock: Book B.") {
		t.Fatalf("expected bundle out-of-stock toast message")
	}
}

func TestBundlePublishFailsWhenBundleOutOfStockShowsToast(t *testing.T) {
	supplierStore := suppliers.NewMemoryStore()
	_, err := supplierStore.Create(suppliers.Input{Name: "Supplier A", WhatsApp: "+91-1", Location: "Bengaluru"})
	if err != nil {
		t.Fatalf("create supplier 1 failed: %v", err)
	}

	s := NewServerWithStores(supplierStore, books.NewMemoryStore(), bundleOutOfStockPublishStore{})
	req := httptest.NewRequest(http.MethodPost, "/admin/bundles/4/publish", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", rr.Code)
	}
	location := rr.Header().Get("Location")
	if !strings.HasPrefix(location, "/admin/bundles?error=") {
		t.Fatalf("unexpected redirect: %s", location)
	}
	if !strings.Contains(location, "Cannot+publish+bundle+because+it+is+out+of+stock.") {
		t.Fatalf("expected out-of-stock toast redirect, got %s", location)
	}
}

type bundleFormParts struct {
	Fields      map[string][]string
	FileField   string
	FileName    string
	FileContent []byte
}

func multipartBundleForm(t *testing.T, parts bundleFormParts) (string, io.Reader) {
	t.Helper()
	if parts.Fields == nil {
		parts.Fields = map[string][]string{}
	}
	if _, ok := parts.Fields["out_of_stock_on_interested"]; !ok {
		parts.Fields["out_of_stock_on_interested"] = []string{"yes"}
	}

	buf := &bytes.Buffer{}
	writer := multipart.NewWriter(buf)
	for key, values := range parts.Fields {
		for _, value := range values {
			if err := writer.WriteField(key, value); err != nil {
				t.Fatalf("write field %s: %v", key, err)
			}
		}
	}
	if parts.FileField != "" {
		fw, err := writer.CreateFormFile(parts.FileField, parts.FileName)
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		if _, err := fw.Write(parts.FileContent); err != nil {
			t.Fatalf("write form file: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	return writer.FormDataContentType(), buf
}

type bundleOutOfStockPublishStore struct{}

func (bundleOutOfStockPublishStore) List() ([]bundles.ListItem, error) { return nil, nil }
func (bundleOutOfStockPublishStore) Create(input bundles.CreateInput) (bundles.Bundle, error) {
	return bundles.Bundle{}, nil
}
func (bundleOutOfStockPublishStore) Get(id int) (bundles.Bundle, error) { return bundles.Bundle{}, nil }
func (bundleOutOfStockPublishStore) Update(id int, input bundles.UpdateInput) (bundles.Bundle, error) {
	return bundles.Bundle{}, nil
}
func (bundleOutOfStockPublishStore) Publish(id int) (bundles.Bundle, error) {
	return bundles.Bundle{}, bundles.ErrCannotPublishOutOfStock
}
func (bundleOutOfStockPublishStore) Unpublish(id int) (bundles.Bundle, error) { return bundles.Bundle{}, nil }
func (bundleOutOfStockPublishStore) ListBooksForPicker() ([]bundles.PickerBook, error) {
	return nil, nil
}
func (bundleOutOfStockPublishStore) GetImage(id int) (bundles.Image, error) { return bundles.Image{}, nil }
