package books

import (
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestPostgresStoreList(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "supplier_id", "title", "author", "category", "mrp", "my_price", "in_stock", "has_cover", "is_published", "published_at", "unpublished_at"}).
		AddRow(1, 11, "The Hobbit", "Tolkien", "Fiction", 499.0, 299.0, true, true, false, nil, time.Date(2026, time.March, 7, 0, 0, 0, 0, time.UTC)).
		AddRow(2, 12, "Legacy Row", "Anon", "Fiction", 299.0, 199.0, true, false, false, nil, nil)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, supplier_id, title, author, category, mrp, my_price, in_stock, COALESCE(OCTET_LENGTH(cover_image), 0) > 0 AS has_cover, is_published, published_at, unpublished_at FROM books ORDER BY id ASC`)).
		WillReturnRows(rows)

	store := NewPostgresStore(db)
	items, err := store.List()
	if err != nil {
		t.Fatalf("list returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(items))
	}
	if !items[0].HasCover || items[0].Title != "The Hobbit" {
		t.Fatalf("unexpected list row: %+v", items[0])
	}
	if items[0].SupplierID != 11 {
		t.Fatalf("expected first row supplier_id=11, got %+v", items[0])
	}
	if items[0].MRP != 499 {
		t.Fatalf("expected first row MRP=499, got %+v", items[0])
	}
	if items[1].HasCover {
		t.Fatalf("expected second row has_cover=false, got %+v", items[1])
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
		Title:                  "Dune",
		Cover:                  Cover{Data: []byte("img"), MimeType: "image/jpeg"},
		SupplierID:             5,
		IsBoxSet:               true,
		Category:               "Fiction",
		Format:                 "Paperback",
		Condition:              "Very good",
		MRP:                    499,
		MyPrice:                349,
		BundlePrice:            &bundlePrice,
		Author:                 "Frank Herbert",
		Notes:                  "Sci-fi",
		OutOfStockOnInterested: true,
	}

	createQuery := `INSERT INTO books (title, cover_image, cover_mime_type, supplier_id, is_box_set, category, format, condition, mrp, my_price, bundle_price, author, notes, out_of_stock_on_interested) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14) RETURNING id, title, supplier_id, cover_mime_type, is_box_set, category, format, condition, mrp, my_price, bundle_price, author, notes, in_stock, out_of_stock_on_interested, is_published, published_at, unpublished_at`
	mock.ExpectQuery(regexp.QuoteMeta(createQuery)).
		WithArgs(input.Title, input.Cover.Data, input.Cover.MimeType, input.SupplierID, input.IsBoxSet, input.Category, input.Format, input.Condition, input.MRP, input.MyPrice, input.BundlePrice, input.Author, input.Notes, input.OutOfStockOnInterested).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "supplier_id", "cover_mime_type", "is_box_set", "category", "format", "condition", "mrp", "my_price", "bundle_price", "author", "notes", "in_stock", "out_of_stock_on_interested", "is_published", "published_at", "unpublished_at"}).
			AddRow(7, input.Title, input.SupplierID, input.Cover.MimeType, input.IsBoxSet, input.Category, input.Format, input.Condition, input.MRP, input.MyPrice, bundlePrice, input.Author, input.Notes, true, true, false, nil, time.Date(2026, time.March, 7, 0, 0, 0, 0, time.UTC)))

	getQuery := `SELECT id, title, supplier_id, cover_mime_type, is_box_set, category, format, condition, mrp, my_price, bundle_price, author, notes, in_stock, out_of_stock_on_interested, is_published, published_at, unpublished_at FROM books WHERE id = $1`
	mock.ExpectQuery(regexp.QuoteMeta(getQuery)).
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "supplier_id", "cover_mime_type", "is_box_set", "category", "format", "condition", "mrp", "my_price", "bundle_price", "author", "notes", "in_stock", "out_of_stock_on_interested", "is_published", "published_at", "unpublished_at"}).
			AddRow(7, input.Title, input.SupplierID, input.Cover.MimeType, input.IsBoxSet, input.Category, input.Format, input.Condition, input.MRP, input.MyPrice, bundlePrice, input.Author, input.Notes, true, true, false, nil, time.Date(2026, time.March, 7, 0, 0, 0, 0, time.UTC)))

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
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT cover_image, cover_mime_type FROM books WHERE id = $1`)).
		WithArgs(11).
		WillReturnRows(sqlmock.NewRows([]string{"cover_image", "cover_mime_type"}).AddRow(nil, "image/png"))

	setStockQuery := `UPDATE books SET in_stock = $1, is_published = CASE WHEN $1 THEN is_published ELSE FALSE END, unpublished_at = CASE WHEN $1 THEN unpublished_at ELSE CASE WHEN is_published THEN NOW() ELSE unpublished_at END END, unpublished_reason = CASE WHEN $1 THEN unpublished_reason ELSE CASE WHEN is_published THEN 'out_of_stock' ELSE unpublished_reason END END, updated_at = NOW() WHERE id = $2 RETURNING id, title, supplier_id, cover_mime_type, is_box_set, category, format, condition, mrp, my_price, bundle_price, author, notes, in_stock, out_of_stock_on_interested, is_published, published_at, unpublished_at`
	mock.ExpectQuery(regexp.QuoteMeta(setStockQuery)).
		WithArgs(false, 10).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "supplier_id", "cover_mime_type", "is_box_set", "category", "format", "condition", "mrp", "my_price", "bundle_price", "author", "notes", "in_stock", "out_of_stock_on_interested", "is_published", "published_at", "unpublished_at"}).
			AddRow(10, "Book", 1, "image/png", false, "Fiction", "Paperback", "Used", 100.0, 90.0, nil, "", "", false, true, false, nil, time.Date(2026, time.March, 7, 0, 0, 0, 0, time.UTC)))

	store := NewPostgresStore(db)
	cover, err := store.GetCover(10)
	if err != nil {
		t.Fatalf("get cover returned error: %v", err)
	}
	if cover.MimeType != "image/png" {
		t.Fatalf("unexpected mime type: %q", cover.MimeType)
	}
	_, err = store.GetCover(11)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for null cover bytes, got %v", err)
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

func TestPostgresStoreSetInStockFalseAutoUnpublishes(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	setStockQuery := `UPDATE books SET in_stock = $1, is_published = CASE WHEN $1 THEN is_published ELSE FALSE END, unpublished_at = CASE WHEN $1 THEN unpublished_at ELSE CASE WHEN is_published THEN NOW() ELSE unpublished_at END END, unpublished_reason = CASE WHEN $1 THEN unpublished_reason ELSE CASE WHEN is_published THEN 'out_of_stock' ELSE unpublished_reason END END, updated_at = NOW() WHERE id = $2 RETURNING id, title, supplier_id, cover_mime_type, is_box_set, category, format, condition, mrp, my_price, bundle_price, author, notes, in_stock, out_of_stock_on_interested, is_published, published_at, unpublished_at`
	now := time.Date(2026, time.March, 9, 15, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta(setStockQuery)).
		WithArgs(false, 12).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "supplier_id", "cover_mime_type", "is_box_set", "category", "format", "condition", "mrp", "my_price", "bundle_price", "author", "notes", "in_stock", "out_of_stock_on_interested", "is_published", "published_at", "unpublished_at"}).
			AddRow(12, "Book", 1, "image/png", false, "Fiction", "Paperback", "Used", 100.0, 90.0, nil, "", "", false, true, false, now, now))

	store := NewPostgresStore(db)
	book, err := store.SetInStock(12, false)
	if err != nil {
		t.Fatalf("set in stock returned error: %v", err)
	}
	if book.InStock {
		t.Fatalf("expected in-stock false")
	}
	if book.IsPublished {
		t.Fatalf("expected published false after out-of-stock")
	}
	if book.UnpublishedAt == nil {
		t.Fatalf("expected unpublished timestamp")
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
		Title:                  "Updated",
		SupplierID:             3,
		IsBoxSet:               false,
		Category:               "Non-Fiction",
		Format:                 "Hardcover",
		Condition:              "Good as new",
		MRP:                    400,
		MyPrice:                300,
		BundlePrice:            &bundlePrice,
		Author:                 "Author",
		Notes:                  "Notes",
		InStock:                true,
		OutOfStockOnInterested: true,
	}

	updateQuery := `UPDATE books SET title = $1, supplier_id = $2, is_box_set = $3, category = $4, format = $5, condition = $6, mrp = $7, my_price = $8, bundle_price = $9, author = $10, notes = $11, in_stock = $12, is_published = CASE WHEN $12 THEN is_published ELSE FALSE END, unpublished_at = CASE WHEN $12 THEN unpublished_at ELSE CASE WHEN is_published THEN NOW() ELSE unpublished_at END END, unpublished_reason = CASE WHEN $12 THEN unpublished_reason ELSE CASE WHEN is_published THEN 'out_of_stock' ELSE unpublished_reason END END, out_of_stock_on_interested = $13, updated_at = NOW() WHERE id = $14 RETURNING id, title, supplier_id, cover_mime_type, is_box_set, category, format, condition, mrp, my_price, bundle_price, author, notes, in_stock, out_of_stock_on_interested, is_published, published_at, unpublished_at`
	mock.ExpectQuery(regexp.QuoteMeta(updateQuery)).
		WithArgs(input.Title, input.SupplierID, input.IsBoxSet, input.Category, input.Format, input.Condition, input.MRP, input.MyPrice, input.BundlePrice, input.Author, input.Notes, input.InStock, input.OutOfStockOnInterested, 404).
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
		Title:                  "Updated With Cover",
		Cover:                  cover,
		SupplierID:             3,
		IsBoxSet:               true,
		Category:               "Fiction",
		Format:                 "Paperback",
		Condition:              "Gently used",
		MRP:                    500,
		MyPrice:                325,
		BundlePrice:            &bundlePrice,
		Author:                 "Author Name",
		Notes:                  "Updated note",
		InStock:                false,
		OutOfStockOnInterested: false,
	}

	updateWithCoverQuery := `UPDATE books SET title = $1, cover_image = $2, cover_mime_type = $3, supplier_id = $4, is_box_set = $5, category = $6, format = $7, condition = $8, mrp = $9, my_price = $10, bundle_price = $11, author = $12, notes = $13, in_stock = $14, is_published = CASE WHEN $14 THEN is_published ELSE FALSE END, unpublished_at = CASE WHEN $14 THEN unpublished_at ELSE CASE WHEN is_published THEN NOW() ELSE unpublished_at END END, unpublished_reason = CASE WHEN $14 THEN unpublished_reason ELSE CASE WHEN is_published THEN 'out_of_stock' ELSE unpublished_reason END END, out_of_stock_on_interested = $15, updated_at = NOW() WHERE id = $16 RETURNING id, title, supplier_id, cover_mime_type, is_box_set, category, format, condition, mrp, my_price, bundle_price, author, notes, in_stock, out_of_stock_on_interested, is_published, published_at, unpublished_at`
	mock.ExpectQuery(regexp.QuoteMeta(updateWithCoverQuery)).
		WithArgs(
			input.Title,
			input.Cover.Data,
			input.Cover.MimeType,
			input.SupplierID,
			input.IsBoxSet,
			input.Category,
			input.Format,
			input.Condition,
			input.MRP,
			input.MyPrice,
			input.BundlePrice,
			input.Author,
			input.Notes,
			input.InStock,
			input.OutOfStockOnInterested,
			12,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "supplier_id", "cover_mime_type", "is_box_set", "category", "format", "condition", "mrp", "my_price", "bundle_price", "author", "notes", "in_stock", "out_of_stock_on_interested", "is_published", "published_at", "unpublished_at"}).
			AddRow(12, input.Title, input.SupplierID, input.Cover.MimeType, input.IsBoxSet, input.Category, input.Format, input.Condition, input.MRP, input.MyPrice, bundlePrice, input.Author, input.Notes, input.InStock, input.OutOfStockOnInterested, false, nil, time.Date(2026, time.March, 7, 0, 0, 0, 0, time.UTC)))

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

func TestPostgresStorePublishUnpublishAndOutOfStockRule(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	publishQuery := `UPDATE books SET is_published = TRUE, published_at = NOW(), unpublished_reason = '', updated_at = NOW() WHERE id = $1 AND in_stock = TRUE RETURNING id, title, supplier_id, cover_mime_type, is_box_set, category, format, condition, mrp, my_price, bundle_price, author, notes, in_stock, out_of_stock_on_interested, is_published, published_at, unpublished_at`
	unpublishQuery := `UPDATE books SET is_published = FALSE, unpublished_at = NOW(), unpublished_reason = '', updated_at = NOW() WHERE id = $1 RETURNING id, title, supplier_id, cover_mime_type, is_box_set, category, format, condition, mrp, my_price, bundle_price, author, notes, in_stock, out_of_stock_on_interested, is_published, published_at, unpublished_at`
	getQuery := `SELECT id, title, supplier_id, cover_mime_type, is_box_set, category, format, condition, mrp, my_price, bundle_price, author, notes, in_stock, out_of_stock_on_interested, is_published, published_at, unpublished_at FROM books WHERE id = $1`

	now := time.Date(2026, time.March, 7, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta(publishQuery)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "supplier_id", "cover_mime_type", "is_box_set", "category", "format", "condition", "mrp", "my_price", "bundle_price", "author", "notes", "in_stock", "out_of_stock_on_interested", "is_published", "published_at", "unpublished_at"}).
			AddRow(1, "Book", 1, "image/png", false, "Fiction", "Paperback", "Used", 100.0, 90.0, nil, "", "", true, true, true, now, now))

	mock.ExpectQuery(regexp.QuoteMeta(unpublishQuery)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "supplier_id", "cover_mime_type", "is_box_set", "category", "format", "condition", "mrp", "my_price", "bundle_price", "author", "notes", "in_stock", "out_of_stock_on_interested", "is_published", "published_at", "unpublished_at"}).
			AddRow(1, "Book", 1, "image/png", false, "Fiction", "Paperback", "Used", 100.0, 90.0, nil, "", "", true, true, false, now, now))

	mock.ExpectQuery(regexp.QuoteMeta(publishQuery)).
		WithArgs(2).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(regexp.QuoteMeta(getQuery)).
		WithArgs(2).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "supplier_id", "cover_mime_type", "is_box_set", "category", "format", "condition", "mrp", "my_price", "bundle_price", "author", "notes", "in_stock", "out_of_stock_on_interested", "is_published", "published_at", "unpublished_at"}).
			AddRow(2, "Book2", 1, "image/png", false, "Fiction", "Paperback", "Used", 100.0, 90.0, nil, "", "", false, true, false, nil, now))

	store := NewPostgresStore(db)
	published, err := store.Publish(1)
	if err != nil {
		t.Fatalf("publish returned error: %v", err)
	}
	if !published.IsPublished {
		t.Fatalf("expected published state")
	}

	unpublished, err := store.Unpublish(1)
	if err != nil {
		t.Fatalf("unpublish returned error: %v", err)
	}
	if unpublished.IsPublished {
		t.Fatalf("expected unpublished state")
	}

	_, err = store.Publish(2)
	if !errors.Is(err, ErrCannotPublishOutOfStock) {
		t.Fatalf("expected ErrCannotPublishOutOfStock, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}
