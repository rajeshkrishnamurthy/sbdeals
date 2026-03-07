package web

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/rajeshkrishnamurthy/sbdeals/internal/books"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/suppliers"
)

func TestBooksListRendersColumnsAndRows(t *testing.T) {
	supplierStore := suppliers.NewMemoryStore()
	supplier, err := supplierStore.Create(suppliers.Input{Name: "A1", WhatsApp: "+91-9", Location: "Bengaluru"})
	if err != nil {
		t.Fatalf("create supplier: %v", err)
	}

	bookStore := books.NewMemoryStore()
	_, err = bookStore.Create(books.CreateInput{
		Title:      "Book One",
		Cover:      books.Cover{Data: []byte("img"), MimeType: "image/png"},
		SupplierID: supplier.ID,
		Category:   "Fiction",
		Format:     "Paperback",
		Condition:  "Very good",
		MRP:        300,
		MyPrice:    220,
		Author:     "Author",
	})
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	s := NewServer(supplierStore, bookStore)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/books", nil)
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	body := rr.Body.String()
	checks := []string{"Books", "Add Book", "Cover", "Title", "Author", "Category", "My price", "In-stock", "View/Edit", "Book One"}
	for _, check := range checks {
		if !strings.Contains(body, check) {
			t.Fatalf("expected body to contain %q", check)
		}
	}
	assertAdminNav(t, body, "/admin/books")
}

func TestBooksListTrailingSlashRedirectsToCanonicalPath(t *testing.T) {
	s := NewServer(suppliers.NewMemoryStore(), books.NewMemoryStore())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/books/", nil)
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusMovedPermanently {
		t.Fatalf("expected 301, got %d", rr.Code)
	}
	if rr.Header().Get("Location") != "/admin/books" {
		t.Fatalf("expected redirect to /admin/books, got %q", rr.Header().Get("Location"))
	}
}

func TestCreateBookDefaultsInStockAndRedirects(t *testing.T) {
	supplierStore := suppliers.NewMemoryStore()
	supplier, err := supplierStore.Create(suppliers.Input{Name: "A1", WhatsApp: "+91-9", Location: "Bengaluru"})
	if err != nil {
		t.Fatalf("create supplier: %v", err)
	}
	bookStore := books.NewMemoryStore()

	s := NewServer(supplierStore, bookStore)
	contentType, body := multipartBookForm(t, bookFormParts{
		Fields: map[string]string{
			"title":       "Deep Work",
			"supplier_id": "1",
			"category":    "Non-Fiction",
			"format":      "Paperback",
			"condition":   "Good as new",
			"mrp":         "500",
			"my_price":    "350",
			"author":      "Cal Newport",
			"notes":       "Productivity",
		},
		FileField:   "cover",
		FileName:    "cover.jpg",
		FileContent: []byte("jpeg-bytes"),
	})

	req := httptest.NewRequest(http.MethodPost, "/admin/books", body)
	req.Header.Set("Content-Type", contentType)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", rr.Code)
	}
	if !strings.HasPrefix(rr.Header().Get("Location"), "/admin/books?flash=") {
		t.Fatalf("unexpected redirect: %s", rr.Header().Get("Location"))
	}

	book, err := bookStore.Get(1)
	if err != nil {
		t.Fatalf("expected created book: %v", err)
	}
	if book.Title != "Deep Work" || book.SupplierID != supplier.ID {
		t.Fatalf("unexpected created book: %+v", book)
	}
	if !book.InStock {
		t.Fatalf("expected created book InStock=true")
	}
}

func TestCreateBookRequiresCover(t *testing.T) {
	supplierStore := suppliers.NewMemoryStore()
	_, err := supplierStore.Create(suppliers.Input{Name: "A1", WhatsApp: "+91-9", Location: "Bengaluru"})
	if err != nil {
		t.Fatalf("create supplier: %v", err)
	}
	bookStore := books.NewMemoryStore()

	s := NewServer(supplierStore, bookStore)
	contentType, body := multipartBookForm(t, bookFormParts{
		Fields: map[string]string{
			"title":       "Deep Work",
			"supplier_id": "1",
			"category":    "Non-Fiction",
			"format":      "Paperback",
			"condition":   "Good as new",
			"mrp":         "500",
			"my_price":    "350",
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/admin/books", body)
	req.Header.Set("Content-Type", contentType)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Cover image is required") {
		t.Fatalf("expected cover validation error")
	}
	if !strings.Contains(rr.Body.String(), `class="toast-error"`) {
		t.Fatalf("expected toast error for validation failure")
	}
	if !strings.Contains(rr.Body.String(), "Please fix: Cover image is required.") {
		t.Fatalf("expected toast summary text for validation failure")
	}
}

func TestInlineStockInvalidValueRedirectsWithToastError(t *testing.T) {
	supplierStore := suppliers.NewMemoryStore()
	_, err := supplierStore.Create(suppliers.Input{Name: "A1", WhatsApp: "+91-9", Location: "Bengaluru"})
	if err != nil {
		t.Fatalf("create supplier: %v", err)
	}
	bookStore := books.NewMemoryStore()
	_, err = bookStore.Create(books.CreateInput{
		Title:      "Book",
		Cover:      books.Cover{Data: []byte("img"), MimeType: "image/png"},
		SupplierID: 1,
		Category:   "Fiction",
		Format:     "Paperback",
		Condition:  "Very good",
		MRP:        100,
		MyPrice:    90,
	})
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	s := NewServer(supplierStore, bookStore)
	form := url.Values{}
	form.Set("in_stock", "maybe")
	req := httptest.NewRequest(http.MethodPost, "/admin/books/1/stock", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", rr.Code)
	}
	if rr.Header().Get("Location") != "/admin/books?error=Please+choose+a+valid+in-stock+value." {
		t.Fatalf("unexpected redirect: %s", rr.Header().Get("Location"))
	}

	listReq := httptest.NewRequest(http.MethodGet, rr.Header().Get("Location"), nil)
	listRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d", listRR.Code)
	}
	body := listRR.Body.String()
	if !strings.Contains(body, `class="toast-error"`) {
		t.Fatalf("expected toast error on books list")
	}
	if !strings.Contains(body, "Please choose a valid in-stock value.") {
		t.Fatalf("expected validation message in toast")
	}
}

func TestBookEditAndInlineStockToggle(t *testing.T) {
	supplierStore := suppliers.NewMemoryStore()
	_, err := supplierStore.Create(suppliers.Input{Name: "A1", WhatsApp: "+91-9", Location: "Bengaluru"})
	if err != nil {
		t.Fatalf("create supplier: %v", err)
	}

	bookStore := books.NewMemoryStore()
	_, err = bookStore.Create(books.CreateInput{
		Title:      "Old",
		Cover:      books.Cover{Data: []byte("cover-1"), MimeType: "image/png"},
		SupplierID: 1,
		Category:   "Fiction",
		Format:     "Paperback",
		Condition:  "Very good",
		MRP:        400,
		MyPrice:    300,
	})
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	s := NewServer(supplierStore, bookStore)

	editType, editBody := multipartBookForm(t, bookFormParts{
		Fields: map[string]string{
			"title":       "Updated",
			"supplier_id": "1",
			"category":    "Fiction",
			"format":      "Hardcover",
			"condition":   "Used",
			"mrp":         "450",
			"my_price":    "250",
			"author":      "A",
			"notes":       "N",
			"in_stock":    "no",
		},
	})

	updateReq := httptest.NewRequest(http.MethodPost, "/admin/books/1", editBody)
	updateReq.Header.Set("Content-Type", editType)
	updateRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(updateRR, updateReq)

	if updateRR.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 on edit, got %d", updateRR.Code)
	}
	if updateRR.Header().Get("Location") != "/admin/books/1?flash=Book+updated+successfully." {
		t.Fatalf("unexpected edit redirect: %s", updateRR.Header().Get("Location"))
	}

	updated, err := bookStore.Get(1)
	if err != nil {
		t.Fatalf("get updated: %v", err)
	}
	if updated.Title != "Updated" || updated.Format != "Hardcover" || updated.InStock {
		t.Fatalf("unexpected updated state: %+v", updated)
	}

	stockForm := url.Values{}
	stockForm.Set("in_stock", "yes")
	stockReq := httptest.NewRequest(http.MethodPost, "/admin/books/1/stock", strings.NewReader(stockForm.Encode()))
	stockReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	stockRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(stockRR, stockReq)

	if stockRR.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 on stock toggle, got %d", stockRR.Code)
	}
	if !strings.HasPrefix(stockRR.Header().Get("Location"), "/admin/books?flash=") {
		t.Fatalf("unexpected stock redirect: %s", stockRR.Header().Get("Location"))
	}

	toggled, err := bookStore.Get(1)
	if err != nil {
		t.Fatalf("get toggled: %v", err)
	}
	if !toggled.InStock {
		t.Fatalf("expected in-stock true after inline toggle")
	}
}

func TestBookCoverRouteReturnsImage(t *testing.T) {
	supplierStore := suppliers.NewMemoryStore()
	_, err := supplierStore.Create(suppliers.Input{Name: "A1", WhatsApp: "+91-9", Location: "Bengaluru"})
	if err != nil {
		t.Fatalf("create supplier: %v", err)
	}
	bookStore := books.NewMemoryStore()
	_, err = bookStore.Create(books.CreateInput{
		Title:      "Cover",
		Cover:      books.Cover{Data: []byte("img-body"), MimeType: "image/webp"},
		SupplierID: 1,
		Category:   "Fiction",
		Format:     "Paperback",
		Condition:  "Used",
		MRP:        100,
		MyPrice:    80,
	})
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	s := NewServer(supplierStore, bookStore)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/books/1/cover", nil)
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if rr.Header().Get("Content-Type") != "image/webp" {
		t.Fatalf("unexpected content-type: %s", rr.Header().Get("Content-Type"))
	}
	if rr.Body.String() != "img-body" {
		t.Fatalf("unexpected body: %q", rr.Body.String())
	}
}

func TestBookNewIncludesEnhancementScript(t *testing.T) {
	supplierStore := suppliers.NewMemoryStore()
	_, err := supplierStore.Create(suppliers.Input{Name: "A1", WhatsApp: "+91-9", Location: "Bengaluru"})
	if err != nil {
		t.Fatalf("create supplier: %v", err)
	}

	s := NewServer(supplierStore, books.NewMemoryStore())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/books/new", nil)
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "/assets/books-form.js") {
		t.Fatalf("expected books form enhancement script tag")
	}
}

func TestBuildBookFormViewRoundsDiscountToWholePercent(t *testing.T) {
	view := buildBookFormView(bookFormViewOptions{
		PageTitle:       "Add Book",
		Action:          "/admin/books",
		SubmitLabel:     "Save",
		Input:           bookFormInput{MRP: "500", MyPrice: "349"},
		SupplierOptions: []suppliers.Supplier{},
		Errors:          map[string]string{},
	})

	if view.DiscountReadOnly != "30%" {
		t.Fatalf("expected rounded discount 30%%, got %q", view.DiscountReadOnly)
	}
}

func TestBooksFormAssetServesClientEnhancements(t *testing.T) {
	s := NewServer(suppliers.NewMemoryStore(), books.NewMemoryStore())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/assets/books-form.js", nil)
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "updateDiscount") {
		t.Fatalf("expected books form asset content")
	}
}

func TestBookFormsUseUnifiedAdminNav(t *testing.T) {
	supplierStore := suppliers.NewMemoryStore()
	supplier, err := supplierStore.Create(suppliers.Input{Name: "A1", WhatsApp: "+91-9", Location: "Bengaluru"})
	if err != nil {
		t.Fatalf("create supplier: %v", err)
	}
	bookStore := books.NewMemoryStore()
	created, err := bookStore.Create(books.CreateInput{
		Title:      "Book",
		Cover:      books.Cover{Data: []byte("img"), MimeType: "image/png"},
		SupplierID: supplier.ID,
		Category:   "Fiction",
		Format:     "Paperback",
		Condition:  "Very good",
		MRP:        100,
		MyPrice:    80,
	})
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	s := NewServer(supplierStore, bookStore)

	newReq := httptest.NewRequest(http.MethodGet, "/admin/books/new", nil)
	newRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(newRR, newReq)
	if newRR.Code != http.StatusOK {
		t.Fatalf("expected 200 for new book page, got %d", newRR.Code)
	}
	assertAdminNav(t, newRR.Body.String(), "/admin/books")

	detailReq := httptest.NewRequest(http.MethodGet, "/admin/books/"+strconv.Itoa(created.ID), nil)
	detailRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(detailRR, detailReq)
	if detailRR.Code != http.StatusOK {
		t.Fatalf("expected 200 for book detail page, got %d", detailRR.Code)
	}
	assertAdminNav(t, detailRR.Body.String(), "/admin/books")
}

type bookFormParts struct {
	Fields      map[string]string
	FileField   string
	FileName    string
	FileContent []byte
}

func multipartBookForm(t *testing.T, parts bookFormParts) (string, io.Reader) {
	t.Helper()
	buf := &bytes.Buffer{}
	writer := multipart.NewWriter(buf)
	for key, value := range parts.Fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("write field %s: %v", key, err)
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
