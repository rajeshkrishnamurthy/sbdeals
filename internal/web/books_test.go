package web

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/rajeshkrishnamurthy/sbdeals/internal/books"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/bundles"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/rails"
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
	checks := []string{"Books", "Add Book", "Cover", "Title", "Author", "Category", "MRP", "My price", "In-stock", "Publish", "View/Edit", "Book One"}
	for _, check := range checks {
		if !strings.Contains(body, check) {
			t.Fatalf("expected body to contain %q", check)
		}
	}
	assertAdminNav(t, body, "/admin/books")
	if !strings.Contains(body, `action="/admin/books/1/publish"`) {
		t.Fatalf("expected publish toggle action in list row")
	}
	if !regexp.MustCompile(`\(\d+d\)`).MatchString(body) {
		t.Fatalf("expected recency indicator like (Xd)")
	}
}

func TestBooksListApplyFiltersUsesDeterministicAND(t *testing.T) {
	supplierStore := suppliers.NewMemoryStore()
	supplier, err := supplierStore.Create(suppliers.Input{Name: "A1", WhatsApp: "+91-9", Location: "Bengaluru"})
	if err != nil {
		t.Fatalf("create supplier: %v", err)
	}
	otherSupplier, err := supplierStore.Create(suppliers.Input{Name: "A2", WhatsApp: "+91-8", Location: "Bengaluru"})
	if err != nil {
		t.Fatalf("create second supplier: %v", err)
	}
	bookStore := books.NewMemoryStore()

	first, err := bookStore.Create(books.CreateInput{
		Title:      "Diary of a Kid",
		Cover:      books.Cover{Data: []byte("img-1"), MimeType: "image/png"},
		SupplierID: supplier.ID,
		Category:   "Children",
		Format:     "Paperback",
		Condition:  "Very good",
		MRP:        300,
		MyPrice:    200,
		Author:     "Jeff Kinney",
	})
	if err != nil {
		t.Fatalf("create first book: %v", err)
	}
	if _, err := bookStore.Publish(first.ID); err != nil {
		t.Fatalf("publish first book: %v", err)
	}

	_, err = bookStore.Create(books.CreateInput{
		Title:      "Diary of an Adult",
		Cover:      books.Cover{Data: []byte("img-2"), MimeType: "image/png"},
		SupplierID: supplier.ID,
		Category:   "Children",
		Format:     "Paperback",
		Condition:  "Very good",
		MRP:        350,
		MyPrice:    240,
		Author:     "Someone Else",
	})
	if err != nil {
		t.Fatalf("create second book: %v", err)
	}

	third, err := bookStore.Create(books.CreateInput{
		Title:      "Diary of a Kid Deluxe",
		Cover:      books.Cover{Data: []byte("img-3"), MimeType: "image/png"},
		SupplierID: supplier.ID,
		Category:   "Children",
		Format:     "Paperback",
		Condition:  "Very good",
		MRP:        300,
		MyPrice:    200,
		Author:     "Jeff Kinney",
	})
	if err != nil {
		t.Fatalf("create third book: %v", err)
	}
	if _, err := bookStore.SetInStock(third.ID, false); err != nil {
		t.Fatalf("set third book out of stock: %v", err)
	}

	fourth, err := bookStore.Create(books.CreateInput{
		Title:      "Diary of a Kid Supplier 2",
		Cover:      books.Cover{Data: []byte("img-4"), MimeType: "image/png"},
		SupplierID: otherSupplier.ID,
		Category:   "Children",
		Format:     "Paperback",
		Condition:  "Very good",
		MRP:        300,
		MyPrice:    200,
		Author:     "Jeff Kinney",
	})
	if err != nil {
		t.Fatalf("create fourth book: %v", err)
	}
	if _, err := bookStore.Publish(fourth.ID); err != nil {
		t.Fatalf("publish fourth book: %v", err)
	}

	s := NewServer(supplierStore, bookStore)
	req := httptest.NewRequest(http.MethodGet, "/admin/books?apply=1&title=diary&author=jeff&supplierId="+strconv.Itoa(supplier.ID)+"&category=Children&inStock=yes&published=yes&mrpMin=250&mrpMax=350&myPriceMin=190&myPriceMax=210", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Diary of a Kid") {
		t.Fatalf("expected matching book in results")
	}
	if strings.Contains(body, "Diary of an Adult") {
		t.Fatalf("did not expect non-matching author row in filtered results")
	}
	if strings.Contains(body, "Diary of a Kid Deluxe") {
		t.Fatalf("did not expect out-of-stock row in filtered results")
	}
	if strings.Contains(body, "Diary of a Kid Supplier 2") {
		t.Fatalf("did not expect different-supplier row in filtered results")
	}
}

func TestBooksListResetFiltersRestoresDefaultList(t *testing.T) {
	supplierStore := suppliers.NewMemoryStore()
	supplier, err := supplierStore.Create(suppliers.Input{Name: "A1", WhatsApp: "+91-9", Location: "Bengaluru"})
	if err != nil {
		t.Fatalf("create supplier: %v", err)
	}
	bookStore := books.NewMemoryStore()

	_, err = bookStore.Create(books.CreateInput{
		Title:      "Low Price",
		Cover:      books.Cover{Data: []byte("img-1"), MimeType: "image/png"},
		SupplierID: supplier.ID,
		Category:   "Fiction",
		Format:     "Paperback",
		Condition:  "Very good",
		MRP:        200,
		MyPrice:    120,
		Author:     "One",
	})
	if err != nil {
		t.Fatalf("create first book: %v", err)
	}
	_, err = bookStore.Create(books.CreateInput{
		Title:      "High Price",
		Cover:      books.Cover{Data: []byte("img-2"), MimeType: "image/png"},
		SupplierID: supplier.ID,
		Category:   "Fiction",
		Format:     "Paperback",
		Condition:  "Very good",
		MRP:        900,
		MyPrice:    700,
		Author:     "Two",
	})
	if err != nil {
		t.Fatalf("create second book: %v", err)
	}

	s := NewServer(supplierStore, bookStore)
	filteredReq := httptest.NewRequest(http.MethodGet, "/admin/books?apply=1&mrpMax=250", nil)
	filteredRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(filteredRR, filteredReq)
	if filteredRR.Code != http.StatusOK {
		t.Fatalf("expected filtered 200, got %d", filteredRR.Code)
	}
	filteredBody := filteredRR.Body.String()
	if !strings.Contains(filteredBody, "Low Price") || strings.Contains(filteredBody, "High Price") {
		t.Fatalf("expected filtered output to include only low-priced row")
	}

	resetReq := httptest.NewRequest(http.MethodGet, "/admin/books", nil)
	resetRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(resetRR, resetReq)
	if resetRR.Code != http.StatusOK {
		t.Fatalf("expected reset 200, got %d", resetRR.Code)
	}
	resetBody := resetRR.Body.String()
	if !strings.Contains(resetBody, "Low Price") || !strings.Contains(resetBody, "High Price") {
		t.Fatalf("expected reset list to include all rows")
	}
}

func TestBooksListInvalidNumericFiltersShowToastAndBlockApply(t *testing.T) {
	supplierStore := suppliers.NewMemoryStore()
	supplier, err := supplierStore.Create(suppliers.Input{Name: "A1", WhatsApp: "+91-9", Location: "Bengaluru"})
	if err != nil {
		t.Fatalf("create supplier: %v", err)
	}
	bookStore := books.NewMemoryStore()
	_, err = bookStore.Create(books.CreateInput{
		Title:      "One",
		Cover:      books.Cover{Data: []byte("img-1"), MimeType: "image/png"},
		SupplierID: supplier.ID,
		Category:   "Fiction",
		Format:     "Paperback",
		Condition:  "Very good",
		MRP:        200,
		MyPrice:    120,
	})
	if err != nil {
		t.Fatalf("create first book: %v", err)
	}
	_, err = bookStore.Create(books.CreateInput{
		Title:      "Two",
		Cover:      books.Cover{Data: []byte("img-2"), MimeType: "image/png"},
		SupplierID: supplier.ID,
		Category:   "Fiction",
		Format:     "Paperback",
		Condition:  "Very good",
		MRP:        900,
		MyPrice:    700,
	})
	if err != nil {
		t.Fatalf("create second book: %v", err)
	}

	s := NewServer(supplierStore, bookStore)
	req := httptest.NewRequest(http.MethodGet, "/admin/books?apply=1&mrpMin=-1", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `class="toast-error"`) {
		t.Fatalf("expected toast error for invalid filter")
	}
	if !strings.Contains(body, "MRP minimum must be a non-negative number.") {
		t.Fatalf("expected numeric validation toast")
	}
	if !strings.Contains(body, "One") || !strings.Contains(body, "Two") {
		t.Fatalf("expected apply to be blocked and default list shown")
	}
}

func TestBooksListInvalidSupplierFilterShowsToastAndBlocksApply(t *testing.T) {
	supplierStore := suppliers.NewMemoryStore()
	supplier, err := supplierStore.Create(suppliers.Input{Name: "A1", WhatsApp: "+91-9", Location: "Bengaluru"})
	if err != nil {
		t.Fatalf("create supplier: %v", err)
	}
	bookStore := books.NewMemoryStore()
	_, err = bookStore.Create(books.CreateInput{
		Title:      "One",
		Cover:      books.Cover{Data: []byte("img-1"), MimeType: "image/png"},
		SupplierID: supplier.ID,
		Category:   "Fiction",
		Format:     "Paperback",
		Condition:  "Very good",
		MRP:        200,
		MyPrice:    120,
	})
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	s := NewServer(supplierStore, bookStore)
	req := httptest.NewRequest(http.MethodGet, "/admin/books?apply=1&supplierId=999", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `class="toast-error"`) {
		t.Fatalf("expected toast error for invalid supplier filter")
	}
	if !strings.Contains(body, "Please choose a valid supplier.") {
		t.Fatalf("expected supplier validation toast")
	}
	if !strings.Contains(body, "One") {
		t.Fatalf("expected apply to be blocked and default list shown")
	}
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
			"is_box_set":  "yes",
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
	if !book.IsBoxSet {
		t.Fatalf("expected created book IsBoxSet=true")
	}
}

func TestBookEditCanUnsetBoxSet(t *testing.T) {
	supplierStore := suppliers.NewMemoryStore()
	_, err := supplierStore.Create(suppliers.Input{Name: "A1", WhatsApp: "+91-9", Location: "Bengaluru"})
	if err != nil {
		t.Fatalf("create supplier: %v", err)
	}
	bookStore := books.NewMemoryStore()
	_, err = bookStore.Create(books.CreateInput{
		Title:      "Boxed",
		Cover:      books.Cover{Data: []byte("img"), MimeType: "image/png"},
		SupplierID: 1,
		IsBoxSet:   true,
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
	editType, editBody := multipartBookForm(t, bookFormParts{
		Fields: map[string]string{
			"title":       "Boxed",
			"supplier_id": "1",
			"is_box_set":  "no",
			"category":    "Fiction",
			"format":      "Paperback",
			"condition":   "Very good",
			"mrp":         "100",
			"my_price":    "90",
			"in_stock":    "yes",
		},
	})

	updateReq := httptest.NewRequest(http.MethodPost, "/admin/books/1", editBody)
	updateReq.Header.Set("Content-Type", editType)
	updateRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(updateRR, updateReq)

	if updateRR.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 on edit, got %d", updateRR.Code)
	}
	updated, err := bookStore.Get(1)
	if err != nil {
		t.Fatalf("get updated: %v", err)
	}
	if updated.IsBoxSet {
		t.Fatalf("expected IsBoxSet=false after edit")
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

func TestBookPublishAndUnpublishActions(t *testing.T) {
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

	publishReq := httptest.NewRequest(http.MethodPost, "/admin/books/1/publish", nil)
	publishRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(publishRR, publishReq)
	if publishRR.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 on publish, got %d", publishRR.Code)
	}
	if publishRR.Header().Get("Location") != "/admin/books?flash=Book+published+successfully." {
		t.Fatalf("unexpected publish redirect: %s", publishRR.Header().Get("Location"))
	}
	afterPublish, err := bookStore.Get(1)
	if err != nil {
		t.Fatalf("get after publish: %v", err)
	}
	if !afterPublish.IsPublished {
		t.Fatalf("expected published=true after publish")
	}

	unpublishReq := httptest.NewRequest(http.MethodPost, "/admin/books/1/unpublish", nil)
	unpublishRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(unpublishRR, unpublishReq)
	if unpublishRR.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 on unpublish, got %d", unpublishRR.Code)
	}
	if unpublishRR.Header().Get("Location") != "/admin/books?flash=Book+unpublished+successfully." {
		t.Fatalf("unexpected unpublish redirect: %s", unpublishRR.Header().Get("Location"))
	}
	afterUnpublish, err := bookStore.Get(1)
	if err != nil {
		t.Fatalf("get after unpublish: %v", err)
	}
	if afterUnpublish.IsPublished {
		t.Fatalf("expected published=false after unpublish")
	}
}

func TestBookPublishFailsWhenOutOfStockShowsToast(t *testing.T) {
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
	if _, err := bookStore.SetInStock(1, false); err != nil {
		t.Fatalf("set in-stock false: %v", err)
	}

	s := NewServer(supplierStore, bookStore)
	req := httptest.NewRequest(http.MethodPost, "/admin/books/1/publish", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", rr.Code)
	}
	if rr.Header().Get("Location") != "/admin/books?error=Cannot+publish+book+because+it+is+out+of+stock." {
		t.Fatalf("unexpected redirect: %s", rr.Header().Get("Location"))
	}
	listReq := httptest.NewRequest(http.MethodGet, rr.Header().Get("Location"), nil)
	listRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(listRR, listReq)
	if !strings.Contains(listRR.Body.String(), "Cannot publish book because it is out of stock.") {
		t.Fatalf("expected toast error on books list")
	}
}

func TestBookStockFlipCascadesBundleStockAndRepublishStaysManual(t *testing.T) {
	supplierStore := suppliers.NewMemoryStore()
	supplier, err := supplierStore.Create(suppliers.Input{Name: "A1", WhatsApp: "+91-9", Location: "Bengaluru"})
	if err != nil {
		t.Fatalf("create supplier: %v", err)
	}

	bookStore := books.NewMemoryStore()
	firstBook, err := bookStore.Create(books.CreateInput{
		Title:      "Book One",
		Cover:      books.Cover{Data: []byte("img"), MimeType: "image/png"},
		SupplierID: supplier.ID,
		Category:   "Fiction",
		Format:     "Paperback",
		Condition:  "Very good",
		MRP:        300,
		MyPrice:    220,
		Author:     "Author A",
	})
	if err != nil {
		t.Fatalf("create first book: %v", err)
	}
	secondBook, err := bookStore.Create(books.CreateInput{
		Title:      "Book Two",
		Cover:      books.Cover{Data: []byte("img"), MimeType: "image/png"},
		SupplierID: supplier.ID,
		Category:   "Fiction",
		Format:     "Paperback",
		Condition:  "Good as new",
		MRP:        320,
		MyPrice:    240,
		Author:     "Author B",
	})
	if err != nil {
		t.Fatalf("create second book: %v", err)
	}

	supplierNames := map[int]string{supplier.ID: supplier.Name}
	pickerBooks := []bundles.PickerBook{
		{BookID: firstBook.ID, Title: firstBook.Title, Author: firstBook.Author, SupplierID: supplier.ID, Category: firstBook.Category, Condition: firstBook.Condition, MRP: firstBook.MRP, MyPrice: firstBook.MyPrice, InStock: true},
		{BookID: secondBook.ID, Title: secondBook.Title, Author: secondBook.Author, SupplierID: supplier.ID, Category: secondBook.Category, Condition: secondBook.Condition, MRP: secondBook.MRP, MyPrice: secondBook.MyPrice, InStock: true},
	}
	bundleStore := bundles.NewMemoryStore(supplierNames, pickerBooks)
	createdBundle, err := bundleStore.Create(bundles.CreateInput{
		Name:              "Starter Bundle",
		SupplierID:        supplier.ID,
		Category:          "Fiction",
		AllowedConditions: []string{"Very good", "Good as new"},
		BookIDs:           []int{firstBook.ID, secondBook.ID},
		BundlePrice:       399,
	})
	if err != nil {
		t.Fatalf("create bundle: %v", err)
	}

	if _, err := bookStore.Publish(firstBook.ID); err != nil {
		t.Fatalf("publish book: %v", err)
	}
	if _, err := bundleStore.Publish(createdBundle.ID); err != nil {
		t.Fatalf("publish bundle: %v", err)
	}

	s := NewServerWithStores(supplierStore, bookStore, bundleStore, rails.NewMemoryStore())

	stockNoForm := url.Values{}
	stockNoForm.Set("in_stock", "no")
	stockNoReq := httptest.NewRequest(http.MethodPost, "/admin/books/"+strconv.Itoa(firstBook.ID)+"/stock", strings.NewReader(stockNoForm.Encode()))
	stockNoReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	stockNoRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(stockNoRR, stockNoReq)
	if stockNoRR.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 on out-of-stock toggle, got %d", stockNoRR.Code)
	}

	updatedBook, err := bookStore.Get(firstBook.ID)
	if err != nil {
		t.Fatalf("get updated book: %v", err)
	}
	if updatedBook.InStock {
		t.Fatalf("expected in-stock=false after toggle")
	}
	if updatedBook.IsPublished {
		t.Fatalf("expected book unpublished when set out-of-stock")
	}

	updatedBundle, err := bundleStore.Get(createdBundle.ID)
	if err != nil {
		t.Fatalf("get updated bundle: %v", err)
	}
	if updatedBundle.InStock {
		t.Fatalf("expected bundle in-stock=false after child book out-of-stock")
	}
	if updatedBundle.IsPublished {
		t.Fatalf("expected bundle unpublished when it becomes out-of-stock")
	}

	stockYesForm := url.Values{}
	stockYesForm.Set("in_stock", "yes")
	stockYesReq := httptest.NewRequest(http.MethodPost, "/admin/books/"+strconv.Itoa(firstBook.ID)+"/stock", strings.NewReader(stockYesForm.Encode()))
	stockYesReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	stockYesRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(stockYesRR, stockYesReq)
	if stockYesRR.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 on in-stock restore, got %d", stockYesRR.Code)
	}

	restoredBundle, err := bundleStore.Get(createdBundle.ID)
	if err != nil {
		t.Fatalf("get restored bundle: %v", err)
	}
	if !restoredBundle.InStock {
		t.Fatalf("expected bundle in-stock=true after all child books restored")
	}
	if restoredBundle.IsPublished {
		t.Fatalf("expected bundle to remain unpublished until manual publish")
	}
}

func TestBookUnpublishAutoUnpublishesRailWithoutPublishedItems(t *testing.T) {
	supplierStore := suppliers.NewMemoryStore()
	supplier, err := supplierStore.Create(suppliers.Input{Name: "A1", WhatsApp: "+91-9", Location: "Bengaluru"})
	if err != nil {
		t.Fatalf("create supplier: %v", err)
	}

	bookStore := books.NewMemoryStore()
	createdBook, err := bookStore.Create(books.CreateInput{
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
	if _, err := bookStore.Publish(createdBook.ID); err != nil {
		t.Fatalf("publish book: %v", err)
	}

	railStore := rails.NewMemoryStore()
	railData, err := railStore.Create(rails.CreateInput{Title: "Book Rail", Type: rails.RailTypeBook})
	if err != nil {
		t.Fatalf("create rail: %v", err)
	}
	if _, err := railStore.AddItem(railData.ID, createdBook.ID); err != nil {
		t.Fatalf("add rail item: %v", err)
	}
	if _, err := railStore.Publish(railData.ID); err != nil {
		t.Fatalf("publish rail: %v", err)
	}

	s := NewServerWithStores(supplierStore, bookStore, bundles.NewMemoryStore(nil, nil), railStore)
	req := httptest.NewRequest(http.MethodPost, "/admin/books/"+strconv.Itoa(createdBook.ID)+"/unpublish", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", rr.Code)
	}

	updatedRail, err := railStore.Get(railData.ID)
	if err != nil {
		t.Fatalf("get updated rail: %v", err)
	}
	if updatedRail.IsPublished {
		t.Fatalf("expected rail unpublished when it has no published items")
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

func TestBookEditIncludesPublishToggleAndRecency(t *testing.T) {
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
		MyPrice:    80,
	})
	if err != nil {
		t.Fatalf("create book: %v", err)
	}
	s := NewServer(supplierStore, bookStore)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/books/1", nil)
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `action="/admin/books/1/publish?from=edit"`) {
		t.Fatalf("expected edit publish toggle action")
	}
	if !regexp.MustCompile(`\(\d+d\)`).MatchString(body) {
		t.Fatalf("expected recency indicator on edit page")
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
	if parts.Fields == nil {
		parts.Fields = map[string]string{}
	}
	if _, ok := parts.Fields["out_of_stock_on_interested"]; !ok {
		parts.Fields["out_of_stock_on_interested"] = "yes"
	}

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
