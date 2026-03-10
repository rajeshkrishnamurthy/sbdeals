package bundles

import (
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestPostgresStoreListAndListBooksForPicker(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	listQuery := `SELECT b.id, b.name, s.name AS supplier_name, b.category, array_to_string(b.allowed_conditions, '||') AS allowed_conditions, b.bundle_price, COUNT(bb.book_id) AS book_count, COALESCE(SUM(bk.mrp), 0) AS bundle_mrp, COALESCE(OCTET_LENGTH(b.bundle_image), 0) > 0 AS has_image, b.is_published, b.published_at, b.unpublished_at FROM bundles b JOIN suppliers s ON s.id = b.supplier_id LEFT JOIN bundle_books bb ON bb.bundle_id = b.id LEFT JOIN books bk ON bk.id = bb.book_id GROUP BY b.id, b.name, s.name, b.category, b.allowed_conditions, b.bundle_price, b.bundle_image, b.is_published, b.published_at, b.unpublished_at ORDER BY b.id ASC`
	mock.ExpectQuery(regexp.QuoteMeta(listQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "supplier_name", "category", "allowed_conditions", "bundle_price", "book_count", "bundle_mrp", "has_image", "is_published", "published_at", "unpublished_at"}).
			AddRow(1, "Starter", "A1", "Fiction", "Very good||Good as new", 499.0, 2, 900.0, false, false, nil, time.Date(2026, time.March, 7, 0, 0, 0, 0, time.UTC)))

	pickerQuery := `SELECT id, title, author, supplier_id, is_box_set, category, condition, mrp, my_price, bundle_price, in_stock FROM books ORDER BY id ASC`
	mock.ExpectQuery(regexp.QuoteMeta(pickerQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "author", "supplier_id", "is_box_set", "category", "condition", "mrp", "my_price", "bundle_price", "in_stock"}).
			AddRow(10, "Book A", "Author A", 1, false, "Fiction", "Very good", 400.0, 250.0, nil, true))

	store := NewPostgresStore(db)
	items, err := store.List()
	if err != nil {
		t.Fatalf("list returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if len(items[0].AllowedConditions) != 2 {
		t.Fatalf("expected two conditions, got %+v", items[0].AllowedConditions)
	}
	if items[0].HasImage {
		t.Fatalf("expected list item to report no image for NULL bundle_image")
	}

	picker, err := store.ListBooksForPicker()
	if err != nil {
		t.Fatalf("picker returned error: %v", err)
	}
	if len(picker) != 1 || picker[0].BookID != 10 {
		t.Fatalf("unexpected picker rows: %+v", picker)
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

	input := CreateInput{
		Name:                   "Starter",
		SupplierID:             1,
		Category:               "Fiction",
		AllowedConditions:      []string{"Very good", "Good as new"},
		BookIDs:                []int{10, 11},
		BundlePrice:            499,
		Notes:                  "Weekend deal",
		Image:                  Image{Data: []byte("bundle-image"), MimeType: "image/png"},
		OutOfStockOnInterested: true,
	}

	insertBundle := `INSERT INTO bundles (name, supplier_id, category, allowed_conditions, bundle_price, notes, bundle_image, bundle_image_mime_type, out_of_stock_on_interested) VALUES ($1, $2, $3, string_to_array($4, '||'), $5, $6, $7, $8, $9) RETURNING id`
	insertBundleBook := `INSERT INTO bundle_books (bundle_id, book_id, position) VALUES ($1, $2, $3)`
	syncBundleStockQuery := `UPDATE bundles b SET in_stock = derived.all_in_stock, is_published = CASE WHEN derived.all_in_stock THEN b.is_published ELSE FALSE END, unpublished_at = CASE WHEN NOT derived.all_in_stock AND b.is_published THEN NOW() ELSE b.unpublished_at END, unpublished_reason = CASE WHEN NOT derived.all_in_stock AND b.is_published THEN 'out_of_stock' ELSE b.unpublished_reason END, updated_at = NOW() FROM (SELECT $1::bigint AS bundle_id, COALESCE((SELECT bool_and(bk.in_stock) FROM bundle_books bb JOIN books bk ON bk.id = bb.book_id WHERE bb.bundle_id = $1), TRUE) AS all_in_stock) AS derived WHERE b.id = derived.bundle_id`
	getBundleQuery := `SELECT b.id, b.name, b.supplier_id, s.name AS supplier_name, b.category, array_to_string(b.allowed_conditions, '||') AS allowed_conditions, b.bundle_price, b.notes, b.in_stock, b.out_of_stock_on_interested, b.bundle_image_mime_type, b.is_published, b.published_at, b.unpublished_at FROM bundles b JOIN suppliers s ON s.id = b.supplier_id WHERE b.id = $1`
	getBooksQuery := `SELECT bb.book_id, b.title, b.author, b.supplier_id, b.is_box_set, b.category, b.condition, b.mrp, b.my_price, b.bundle_price, b.in_stock FROM bundle_books bb JOIN books b ON b.id = bb.book_id WHERE bb.bundle_id = $1 ORDER BY bb.position ASC`

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(insertBundle)).
		WithArgs(input.Name, input.SupplierID, input.Category, "Very good||Good as new", input.BundlePrice, input.Notes, input.Image.Data, input.Image.MimeType, input.OutOfStockOnInterested).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(22))
	mock.ExpectExec(regexp.QuoteMeta(insertBundleBook)).WithArgs(22, 10, 0).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(insertBundleBook)).WithArgs(22, 11, 1).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(syncBundleStockQuery)).WithArgs(22).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	mock.ExpectQuery(regexp.QuoteMeta(getBundleQuery)).WithArgs(22).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "supplier_id", "supplier_name", "category", "allowed_conditions", "bundle_price", "notes", "in_stock", "out_of_stock_on_interested", "bundle_image_mime_type", "is_published", "published_at", "unpublished_at"}).
			AddRow(22, input.Name, input.SupplierID, "Supplier A", input.Category, "Very good||Good as new", input.BundlePrice, input.Notes, true, true, "image/png", false, nil, time.Date(2026, time.March, 7, 0, 0, 0, 0, time.UTC)))
	mock.ExpectQuery(regexp.QuoteMeta(getBooksQuery)).WithArgs(22).
		WillReturnRows(sqlmock.NewRows([]string{"book_id", "title", "author", "supplier_id", "is_box_set", "category", "condition", "mrp", "my_price", "bundle_price", "in_stock"}).
			AddRow(10, "Book A", "Author A", 1, false, "Fiction", "Very good", 400.0, 250.0, nil, true).
			AddRow(11, "Book B", "Author B", 1, false, "Fiction", "Good as new", 500.0, 300.0, nil, true))

	store := NewPostgresStore(db)
	created, err := store.Create(input)
	if err != nil {
		t.Fatalf("create returned error: %v", err)
	}
	if created.ID != 22 || len(created.Books) != 2 {
		t.Fatalf("unexpected created bundle: %+v", created)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresStoreUpdateAndNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	updateWithImageQuery := `UPDATE bundles SET name = $1, supplier_id = $2, category = $3, allowed_conditions = string_to_array($4, '||'), bundle_price = $5, notes = $6, bundle_image = $7, bundle_image_mime_type = $8, out_of_stock_on_interested = $9, updated_at = NOW() WHERE id = $10`
	deleteBooks := `DELETE FROM bundle_books WHERE bundle_id = $1`
	insertBundleBook := `INSERT INTO bundle_books (bundle_id, book_id, position) VALUES ($1, $2, $3)`
	syncBundleStockQuery := `UPDATE bundles b SET in_stock = derived.all_in_stock, is_published = CASE WHEN derived.all_in_stock THEN b.is_published ELSE FALSE END, unpublished_at = CASE WHEN NOT derived.all_in_stock AND b.is_published THEN NOW() ELSE b.unpublished_at END, unpublished_reason = CASE WHEN NOT derived.all_in_stock AND b.is_published THEN 'out_of_stock' ELSE b.unpublished_reason END, updated_at = NOW() FROM (SELECT $1::bigint AS bundle_id, COALESCE((SELECT bool_and(bk.in_stock) FROM bundle_books bb JOIN books bk ON bk.id = bb.book_id WHERE bb.bundle_id = $1), TRUE) AS all_in_stock) AS derived WHERE b.id = derived.bundle_id`
	getBundleQuery := `SELECT b.id, b.name, b.supplier_id, s.name AS supplier_name, b.category, array_to_string(b.allowed_conditions, '||') AS allowed_conditions, b.bundle_price, b.notes, b.in_stock, b.out_of_stock_on_interested, b.bundle_image_mime_type, b.is_published, b.published_at, b.unpublished_at FROM bundles b JOIN suppliers s ON s.id = b.supplier_id WHERE b.id = $1`
	getBooksQuery := `SELECT bb.book_id, b.title, b.author, b.supplier_id, b.is_box_set, b.category, b.condition, b.mrp, b.my_price, b.bundle_price, b.in_stock FROM bundle_books bb JOIN books b ON b.id = bb.book_id WHERE bb.bundle_id = $1 ORDER BY bb.position ASC`

	updateInput := UpdateInput{
		Name:                   "Updated",
		SupplierID:             2,
		Category:               "Non-Fiction",
		AllowedConditions:      []string{"Used"},
		BookIDs:                []int{40, 41},
		BundlePrice:            199,
		Notes:                  "Updated",
		Image:                  &Image{Data: []byte("updated-image"), MimeType: "image/jpeg"},
		OutOfStockOnInterested: false,
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(updateWithImageQuery)).
		WithArgs(updateInput.Name, updateInput.SupplierID, updateInput.Category, "Used", updateInput.BundlePrice, updateInput.Notes, updateInput.Image.Data, updateInput.Image.MimeType, updateInput.OutOfStockOnInterested, 9).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(deleteBooks)).WithArgs(9).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(regexp.QuoteMeta(insertBundleBook)).WithArgs(9, 40, 0).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(insertBundleBook)).WithArgs(9, 41, 1).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(syncBundleStockQuery)).WithArgs(9).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(regexp.QuoteMeta(getBundleQuery)).WithArgs(9).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "supplier_id", "supplier_name", "category", "allowed_conditions", "bundle_price", "notes", "in_stock", "out_of_stock_on_interested", "bundle_image_mime_type", "is_published", "published_at", "unpublished_at"}).
			AddRow(9, updateInput.Name, updateInput.SupplierID, "Supplier 2", updateInput.Category, "Used", updateInput.BundlePrice, updateInput.Notes, true, false, "image/jpeg", false, nil, time.Date(2026, time.March, 7, 0, 0, 0, 0, time.UTC)))
	mock.ExpectQuery(regexp.QuoteMeta(getBooksQuery)).WithArgs(9).
		WillReturnRows(sqlmock.NewRows([]string{"book_id", "title", "author", "supplier_id", "is_box_set", "category", "condition", "mrp", "my_price", "bundle_price", "in_stock"}).
			AddRow(40, "B1", "A1", 2, false, "Non-Fiction", "Used", 300.0, 200.0, nil, true).
			AddRow(41, "B2", "A2", 2, false, "Non-Fiction", "Used", 250.0, 150.0, nil, true))

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(updateWithImageQuery)).
		WithArgs(updateInput.Name, updateInput.SupplierID, updateInput.Category, "Used", updateInput.BundlePrice, updateInput.Notes, updateInput.Image.Data, updateInput.Image.MimeType, updateInput.OutOfStockOnInterested, 99).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	store := NewPostgresStore(db)
	updated, err := store.Update(9, updateInput)
	if err != nil {
		t.Fatalf("update returned error: %v", err)
	}
	if updated.ID != 9 || len(updated.Books) != 2 {
		t.Fatalf("unexpected updated bundle: %+v", updated)
	}

	_, err = store.Update(99, updateInput)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresStoreGetNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	getBundleQuery := `SELECT b.id, b.name, b.supplier_id, s.name AS supplier_name, b.category, array_to_string(b.allowed_conditions, '||') AS allowed_conditions, b.bundle_price, b.notes, b.in_stock, b.out_of_stock_on_interested, b.bundle_image_mime_type, b.is_published, b.published_at, b.unpublished_at FROM bundles b JOIN suppliers s ON s.id = b.supplier_id WHERE b.id = $1`
	mock.ExpectQuery(regexp.QuoteMeta(getBundleQuery)).WithArgs(33).WillReturnError(sql.ErrNoRows)

	store := NewPostgresStore(db)
	_, err = store.Get(33)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
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

	publishQuery := `UPDATE bundles b SET is_published = TRUE, published_at = NOW(), unpublished_reason = '', updated_at = NOW() WHERE b.id = $1 AND b.in_stock = TRUE AND NOT EXISTS (SELECT 1 FROM bundle_books bb JOIN books bk ON bk.id = bb.book_id WHERE bb.bundle_id = b.id AND bk.in_stock = FALSE) RETURNING b.id`
	unpublishQuery := `UPDATE bundles SET is_published = FALSE, unpublished_at = NOW(), unpublished_reason = '', updated_at = NOW() WHERE id = $1 RETURNING id`
	getBundleQuery := `SELECT b.id, b.name, b.supplier_id, s.name AS supplier_name, b.category, array_to_string(b.allowed_conditions, '||') AS allowed_conditions, b.bundle_price, b.notes, b.in_stock, b.out_of_stock_on_interested, b.bundle_image_mime_type, b.is_published, b.published_at, b.unpublished_at FROM bundles b JOIN suppliers s ON s.id = b.supplier_id WHERE b.id = $1`
	getBooksQuery := `SELECT bb.book_id, b.title, b.author, b.supplier_id, b.is_box_set, b.category, b.condition, b.mrp, b.my_price, b.bundle_price, b.in_stock FROM bundle_books bb JOIN books b ON b.id = bb.book_id WHERE bb.bundle_id = $1 ORDER BY bb.position ASC`
	outOfStockTitlesQuery := `SELECT b.title FROM bundle_books bb JOIN books b ON b.id = bb.book_id WHERE bb.bundle_id = $1 AND b.in_stock = FALSE ORDER BY b.title ASC`

	now := time.Date(2026, time.March, 7, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta(publishQuery)).WithArgs(9).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(9))
	mock.ExpectQuery(regexp.QuoteMeta(getBundleQuery)).WithArgs(9).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "supplier_id", "supplier_name", "category", "allowed_conditions", "bundle_price", "notes", "in_stock", "out_of_stock_on_interested", "bundle_image_mime_type", "is_published", "published_at", "unpublished_at"}).
			AddRow(9, "Bundle", 1, "Supplier A", "Fiction", "Very good", 299.0, "", true, true, "image/png", true, now, now))
	mock.ExpectQuery(regexp.QuoteMeta(getBooksQuery)).WithArgs(9).
		WillReturnRows(sqlmock.NewRows([]string{"book_id", "title", "author", "supplier_id", "is_box_set", "category", "condition", "mrp", "my_price", "bundle_price", "in_stock"}).
			AddRow(10, "Book A", "A", 1, false, "Fiction", "Very good", 200.0, 100.0, nil, true).
			AddRow(11, "Book B", "B", 1, false, "Fiction", "Very good", 250.0, 150.0, nil, true))

	mock.ExpectQuery(regexp.QuoteMeta(unpublishQuery)).WithArgs(9).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(9))
	mock.ExpectQuery(regexp.QuoteMeta(getBundleQuery)).WithArgs(9).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "supplier_id", "supplier_name", "category", "allowed_conditions", "bundle_price", "notes", "in_stock", "out_of_stock_on_interested", "bundle_image_mime_type", "is_published", "published_at", "unpublished_at"}).
			AddRow(9, "Bundle", 1, "Supplier A", "Fiction", "Very good", 299.0, "", true, true, "image/png", false, now, now))
	mock.ExpectQuery(regexp.QuoteMeta(getBooksQuery)).WithArgs(9).
		WillReturnRows(sqlmock.NewRows([]string{"book_id", "title", "author", "supplier_id", "is_box_set", "category", "condition", "mrp", "my_price", "bundle_price", "in_stock"}).
			AddRow(10, "Book A", "A", 1, false, "Fiction", "Very good", 200.0, 100.0, nil, true).
			AddRow(11, "Book B", "B", 1, false, "Fiction", "Very good", 250.0, 150.0, nil, true))

	mock.ExpectQuery(regexp.QuoteMeta(publishQuery)).WithArgs(20).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(regexp.QuoteMeta(getBundleQuery)).WithArgs(20).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "supplier_id", "supplier_name", "category", "allowed_conditions", "bundle_price", "notes", "in_stock", "out_of_stock_on_interested", "bundle_image_mime_type", "is_published", "published_at", "unpublished_at"}).
			AddRow(20, "Bundle 20", 1, "Supplier A", "Fiction", "Very good", 199.0, "", true, true, "", false, nil, now))
	mock.ExpectQuery(regexp.QuoteMeta(getBooksQuery)).WithArgs(20).
		WillReturnRows(sqlmock.NewRows([]string{"book_id", "title", "author", "supplier_id", "is_box_set", "category", "condition", "mrp", "my_price", "bundle_price", "in_stock"}).
			AddRow(40, "Out Book", "A", 1, false, "Fiction", "Very good", 100.0, 90.0, nil, false))
	mock.ExpectQuery(regexp.QuoteMeta(outOfStockTitlesQuery)).WithArgs(20).
		WillReturnRows(sqlmock.NewRows([]string{"title"}).AddRow("Out Book"))

	mock.ExpectQuery(regexp.QuoteMeta(publishQuery)).WithArgs(21).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(regexp.QuoteMeta(getBundleQuery)).WithArgs(21).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "supplier_id", "supplier_name", "category", "allowed_conditions", "bundle_price", "notes", "in_stock", "out_of_stock_on_interested", "bundle_image_mime_type", "is_published", "published_at", "unpublished_at"}).
			AddRow(21, "Bundle 21", 1, "Supplier A", "Fiction", "Very good", 199.0, "", false, true, "", false, nil, now))
	mock.ExpectQuery(regexp.QuoteMeta(getBooksQuery)).WithArgs(21).
		WillReturnRows(sqlmock.NewRows([]string{"book_id", "title", "author", "supplier_id", "is_box_set", "category", "condition", "mrp", "my_price", "bundle_price", "in_stock"}).
			AddRow(41, "In Book", "A", 1, false, "Fiction", "Very good", 100.0, 90.0, nil, true))
	mock.ExpectQuery(regexp.QuoteMeta(outOfStockTitlesQuery)).WithArgs(21).
		WillReturnRows(sqlmock.NewRows([]string{"title"}))

	store := NewPostgresStore(db)
	published, err := store.Publish(9)
	if err != nil {
		t.Fatalf("publish returned error: %v", err)
	}
	if !published.IsPublished {
		t.Fatalf("expected published state")
	}

	unpublished, err := store.Unpublish(9)
	if err != nil {
		t.Fatalf("unpublish returned error: %v", err)
	}
	if unpublished.IsPublished {
		t.Fatalf("expected unpublished state")
	}

	_, err = store.Publish(20)
	var outOfStockErr *ErrCannotPublishWithOutOfStockBooks
	if !errors.As(err, &outOfStockErr) {
		t.Fatalf("expected ErrCannotPublishWithOutOfStockBooks, got %v", err)
	}
	if len(outOfStockErr.BookTitles) != 1 || outOfStockErr.BookTitles[0] != "Out Book" {
		t.Fatalf("unexpected out-of-stock titles: %+v", outOfStockErr.BookTitles)
	}

	_, err = store.Publish(21)
	if !errors.Is(err, ErrCannotPublishOutOfStock) {
		t.Fatalf("expected ErrCannotPublishOutOfStock, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresStoreSyncDerivedStockByBook(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	query := `WITH affected AS (SELECT DISTINCT bb.bundle_id FROM bundle_books bb WHERE bb.book_id = $1), derived AS (SELECT a.bundle_id, COALESCE(bool_and(bk.in_stock), TRUE) AS all_in_stock FROM affected a LEFT JOIN bundle_books bb ON bb.bundle_id = a.bundle_id LEFT JOIN books bk ON bk.id = bb.book_id GROUP BY a.bundle_id) UPDATE bundles b SET in_stock = derived.all_in_stock, is_published = CASE WHEN derived.all_in_stock THEN b.is_published ELSE FALSE END, unpublished_at = CASE WHEN NOT derived.all_in_stock AND b.is_published THEN NOW() ELSE b.unpublished_at END, unpublished_reason = CASE WHEN NOT derived.all_in_stock AND b.is_published THEN 'out_of_stock' ELSE b.unpublished_reason END, updated_at = NOW() FROM derived WHERE b.id = derived.bundle_id`
	mock.ExpectExec(regexp.QuoteMeta(query)).WithArgs(10).WillReturnResult(sqlmock.NewResult(0, 2))

	store := NewPostgresStore(db)
	if err := store.SyncDerivedStockByBook(10, false); err != nil {
		t.Fatalf("sync returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresStoreGetImage(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	query := `SELECT bundle_image, bundle_image_mime_type FROM bundles WHERE id = $1`
	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(3).
		WillReturnRows(sqlmock.NewRows([]string{"bundle_image", "bundle_image_mime_type"}).AddRow([]byte("img"), "image/png"))
	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(4).
		WillReturnRows(sqlmock.NewRows([]string{"bundle_image", "bundle_image_mime_type"}).AddRow([]byte{}, ""))

	store := NewPostgresStore(db)
	image, err := store.GetImage(3)
	if err != nil {
		t.Fatalf("get image returned error: %v", err)
	}
	if string(image.Data) != "img" || image.MimeType != "image/png" {
		t.Fatalf("unexpected image payload: %+v", image)
	}

	_, err = store.GetImage(4)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for empty image bytes, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}
