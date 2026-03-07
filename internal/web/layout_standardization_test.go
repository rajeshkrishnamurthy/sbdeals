package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rajeshkrishnamurthy/sbdeals/internal/books"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/suppliers"
)

func TestAdminListPagesUseSameContainerWidthAndFullTableWidth(t *testing.T) {
	supplierStore := suppliers.NewMemoryStore()
	supplier, err := supplierStore.Create(suppliers.Input{
		Name:     "A1",
		WhatsApp: "+91-1",
		Location: "Bengaluru",
	})
	if err != nil {
		t.Fatalf("create supplier: %v", err)
	}

	bookStore := books.NewMemoryStore()
	_, err = bookStore.Create(books.CreateInput{
		Title:      "Book",
		Cover:      books.Cover{Data: []byte("img"), MimeType: "image/png"},
		SupplierID: supplier.ID,
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

	suppliersBody := fetchBody(t, s, "/admin/suppliers")
	booksBody := fetchBody(t, s, "/admin/books")
	bundlesBody := fetchBody(t, s, "/admin/bundles")
	railsBody := fetchBody(t, s, "/admin/rails")

	const layoutToken = `.shell { width:min(1100px, 94vw); margin:0 auto; padding:16px; }`
	if !strings.Contains(suppliersBody, layoutToken) {
		t.Fatalf("expected suppliers page to include standardized shell width token")
	}
	if !strings.Contains(booksBody, layoutToken) {
		t.Fatalf("expected books page to include standardized shell width token")
	}
	if !strings.Contains(bundlesBody, layoutToken) {
		t.Fatalf("expected bundles page to include standardized shell width token")
	}
	if !strings.Contains(railsBody, layoutToken) {
		t.Fatalf("expected rails page to include standardized shell width token")
	}

	const tableToken = `table { width:100%;`
	if !strings.Contains(suppliersBody, tableToken) {
		t.Fatalf("expected suppliers page table width to be 100%%")
	}
	if !strings.Contains(booksBody, tableToken) {
		t.Fatalf("expected books page table width to be 100%%")
	}
	if !strings.Contains(bundlesBody, tableToken) {
		t.Fatalf("expected bundles page table width to be 100%%")
	}
	if !strings.Contains(railsBody, tableToken) {
		t.Fatalf("expected rails page table width to be 100%%")
	}
}

func fetchBody(t *testing.T, s *Server, path string) string {
	t.Helper()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for %s, got %d", path, rr.Code)
	}
	return rr.Body.String()
}
