package books

import (
	"database/sql"
	"errors"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestPostgresStoreList(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "title", "author", "category", "my_price", "in_stock", "has_cover"}).
		AddRow(1, "The Hobbit", "Tolkien", "Fiction", 299.0, true, true)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, title, author, category, my_price, in_stock, OCTET_LENGTH(cover_image) > 0 AS has_cover FROM books ORDER BY id ASC`)).
		WillReturnRows(rows)

	store := NewPostgresStore(db)
	items, err := store.List()
	if err != nil {
		t.Fatalf("list returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 row, got %d", len(items))
	}
	if !items[0].HasCover || items[0].Title != "The Hobbit" {
		t.Fatalf("unexpected list row: %+v", items[0])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresStoreCreateAndGet(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	bundlePrice := 199.0
	input := CreateInput{
		Title:       "Dune",
		Cover:       Cover{Data: []byte("img"), MimeType: "image/jpeg"},
		SupplierID:  5,
		Category:    "Fiction",
		Format:      "Paperback",
		Condition:   "Very good",
		MRP:         499,
		MyPrice:     349,
		BundlePrice: &bundlePrice,
		Author:      "Frank Herbert",
		Notes:       "Sci-fi",
	}

	createQuery := `INSERT INTO books (title, cover_image, cover_mime_type, supplier_id, category, format, condition, mrp, my_price, bundle_price, author, notes) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING id, title, supplier_id, cover_mime_type, category, format, condition, mrp, my_price, bundle_price, author, notes, in_stock`
	mock.ExpectQuery(regexp.QuoteMeta(createQuery)).
		WithArgs(input.Title, input.Cover.Data, input.Cover.MimeType, input.SupplierID, input.Category, input.Format, input.Condition, input.MRP, input.MyPrice, input.BundlePrice, input.Author, input.Notes).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "supplier_id", "cover_mime_type", "category", "format", "condition", "mrp", "my_price", "bundle_price", "author", "notes", "in_stock"}).
			AddRow(7, input.Title, input.SupplierID, input.Cover.MimeType, input.Category, input.Format, input.Condition, input.MRP, input.MyPrice, bundlePrice, input.Author, input.Notes, true))

	getQuery := `SELECT id, title, supplier_id, cover_mime_type, category, format, condition, mrp, my_price, bundle_price, author, notes, in_stock FROM books WHERE id = $1`
	mock.ExpectQuery(regexp.QuoteMeta(getQuery)).
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "supplier_id", "cover_mime_type", "category", "format", "condition", "mrp", "my_price", "bundle_price", "author", "notes", "in_stock"}).
			AddRow(7, input.Title, input.SupplierID, input.Cover.MimeType, input.Category, input.Format, input.Condition, input.MRP, input.MyPrice, bundlePrice, input.Author, input.Notes, true))

	store := NewPostgresStore(db)
	created, err := store.Create(input)
	if err != nil {
		t.Fatalf("create returned error: %v", err)
	}
	if !created.InStock {
		t.Fatalf("expected create to return in-stock=true")
	}

	book, err := store.Get(7)
	if err != nil {
		t.Fatalf("get returned error: %v", err)
	}
	if book.Title != "Dune" || book.SupplierID != 5 {
		t.Fatalf("unexpected book: %+v", book)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresStoreGetCoverAndSetInStock(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT cover_image, cover_mime_type FROM books WHERE id = $1`)).
		WithArgs(10).
		WillReturnRows(sqlmock.NewRows([]string{"cover_image", "cover_mime_type"}).AddRow([]byte("cover-bytes"), "image/png"))

	setStockQuery := `UPDATE books SET in_stock = $1, updated_at = NOW() WHERE id = $2 RETURNING id, title, supplier_id, cover_mime_type, category, format, condition, mrp, my_price, bundle_price, author, notes, in_stock`
	mock.ExpectQuery(regexp.QuoteMeta(setStockQuery)).
		WithArgs(false, 10).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "supplier_id", "cover_mime_type", "category", "format", "condition", "mrp", "my_price", "bundle_price", "author", "notes", "in_stock"}).
			AddRow(10, "Book", 1, "image/png", "Fiction", "Paperback", "Used", 100.0, 90.0, nil, "", "", false))

	store := NewPostgresStore(db)
	cover, err := store.GetCover(10)
	if err != nil {
		t.Fatalf("get cover returned error: %v", err)
	}
	if cover.MimeType != "image/png" {
		t.Fatalf("unexpected mime type: %q", cover.MimeType)
	}

	book, err := store.SetInStock(10, false)
	if err != nil {
		t.Fatalf("set in stock returned error: %v", err)
	}
	if book.InStock {
		t.Fatalf("expected in-stock false")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresStoreUpdateWithoutCoverAndNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	bundlePrice := 129.0
	input := UpdateInput{
		Title:       "Updated",
		SupplierID:  3,
		Category:    "Non-Fiction",
		Format:      "Hardcover",
		Condition:   "Good as new",
		MRP:         400,
		MyPrice:     300,
		BundlePrice: &bundlePrice,
		Author:      "Author",
		Notes:       "Notes",
		InStock:     true,
	}

	updateQuery := `UPDATE books SET title = $1, supplier_id = $2, category = $3, format = $4, condition = $5, mrp = $6, my_price = $7, bundle_price = $8, author = $9, notes = $10, in_stock = $11, updated_at = NOW() WHERE id = $12 RETURNING id, title, supplier_id, cover_mime_type, category, format, condition, mrp, my_price, bundle_price, author, notes, in_stock`
	mock.ExpectQuery(regexp.QuoteMeta(updateQuery)).
		WithArgs(input.Title, input.SupplierID, input.Category, input.Format, input.Condition, input.MRP, input.MyPrice, input.BundlePrice, input.Author, input.Notes, input.InStock, 404).
		WillReturnError(sql.ErrNoRows)

	store := NewPostgresStore(db)
	_, err = store.Update(404, input)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresStoreUpdateWithCoverPath(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	bundlePrice := 175.0
	cover := &Cover{Data: []byte("new-cover"), MimeType: "image/webp"}
	input := UpdateInput{
		Title:       "Updated With Cover",
		Cover:       cover,
		SupplierID:  3,
		Category:    "Fiction",
		Format:      "Paperback",
		Condition:   "Gently used",
		MRP:         500,
		MyPrice:     325,
		BundlePrice: &bundlePrice,
		Author:      "Author Name",
		Notes:       "Updated note",
		InStock:     false,
	}

	updateWithCoverQuery := `UPDATE books SET title = $1, cover_image = $2, cover_mime_type = $3, supplier_id = $4, category = $5, format = $6, condition = $7, mrp = $8, my_price = $9, bundle_price = $10, author = $11, notes = $12, in_stock = $13, updated_at = NOW() WHERE id = $14 RETURNING id, title, supplier_id, cover_mime_type, category, format, condition, mrp, my_price, bundle_price, author, notes, in_stock`
	mock.ExpectQuery(regexp.QuoteMeta(updateWithCoverQuery)).
		WithArgs(
			input.Title,
			input.Cover.Data,
			input.Cover.MimeType,
			input.SupplierID,
			input.Category,
			input.Format,
			input.Condition,
			input.MRP,
			input.MyPrice,
			input.BundlePrice,
			input.Author,
			input.Notes,
			input.InStock,
			12,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "supplier_id", "cover_mime_type", "category", "format", "condition", "mrp", "my_price", "bundle_price", "author", "notes", "in_stock"}).
			AddRow(12, input.Title, input.SupplierID, input.Cover.MimeType, input.Category, input.Format, input.Condition, input.MRP, input.MyPrice, bundlePrice, input.Author, input.Notes, input.InStock))

	store := NewPostgresStore(db)
	updated, err := store.Update(12, input)
	if err != nil {
		t.Fatalf("update returned error: %v", err)
	}
	if updated.CoverMimeType != "image/webp" {
		t.Fatalf("expected cover mime type image/webp, got %q", updated.CoverMimeType)
	}
	if updated.Title != input.Title || updated.SupplierID != input.SupplierID || updated.InStock != input.InStock {
		t.Fatalf("unexpected updated book: %+v", updated)
	}
	if updated.BundlePrice == nil || *updated.BundlePrice != bundlePrice {
		t.Fatalf("expected bundle price %.2f, got %+v", bundlePrice, updated.BundlePrice)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}
