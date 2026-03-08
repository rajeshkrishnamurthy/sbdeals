package clicked

import (
	"database/sql"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestPostgresStoreCreateClickedAndListByStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	createQuery := `INSERT INTO enquiries (item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status) VALUES ($1, $2, $3, $4, $5, $6, 'clicked') RETURNING id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, buyer_name, buyer_phone, buyer_note, converted_by, converted_at, created_at`
	now := time.Date(2026, time.March, 8, 7, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta(createQuery)).
		WithArgs(7, "BOOK", "Book One", "catalog", 2, "Fresh Books").
		WillReturnRows(sqlmock.NewRows([]string{"id", "item_id", "item_type", "item_title", "source_page", "source_rail_id", "source_rail_title", "status", "buyer_name", "buyer_phone", "buyer_note", "converted_by", "converted_at", "created_at"}).
			AddRow(1, 7, "BOOK", "Book One", "catalog", 2, "Fresh Books", "clicked", "", "", "", nil, nil, now))

	listQuery := `SELECT id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, buyer_name, buyer_phone, buyer_note, converted_by, converted_at, created_at FROM enquiries WHERE status = $1 ORDER BY created_at DESC, id DESC`
	mock.ExpectQuery(regexp.QuoteMeta(listQuery)).
		WithArgs("clicked").
		WillReturnRows(sqlmock.NewRows([]string{"id", "item_id", "item_type", "item_title", "source_page", "source_rail_id", "source_rail_title", "status", "buyer_name", "buyer_phone", "buyer_note", "converted_by", "converted_at", "created_at"}).
			AddRow(1, 7, "BOOK", "Book One", "catalog", 2, "Fresh Books", "clicked", "", "", "", nil, nil, now))

	store := NewPostgresStore(db)
	created, err := store.CreateClicked(CreateInput{
		ItemID:       7,
		ItemType:     ItemTypeBook,
		ItemTitle:    "Book One",
		SourcePage:   "catalog",
		SourceRailID: 2,
		SourceRail:   "Fresh Books",
	})
	if err != nil {
		t.Fatalf("create clicked returned error: %v", err)
	}
	if created.Status != StatusClicked {
		t.Fatalf("unexpected created status: %q", created.Status)
	}

	items, err := store.ListByStatus(StatusClicked)
	if err != nil {
		t.Fatalf("list by status returned error: %v", err)
	}
	if len(items) != 1 || items[0].ID != 1 {
		t.Fatalf("unexpected list rows: %+v", items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresStoreConvertToInterestedBookSideEffectsAndIdempotent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	updateQuery := `UPDATE enquiries SET status = 'interested', buyer_name = $1, buyer_phone = $2, buyer_note = $3, converted_by = $4, converted_at = NOW() WHERE id = $5 AND status = 'clicked' RETURNING id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, buyer_name, buyer_phone, buyer_note, converted_by, converted_at, created_at`
	getByIDQuery := `SELECT id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, buyer_name, buyer_phone, buyer_note, converted_by, converted_at, created_at FROM enquiries WHERE id = $1`
	bookFlagQuery := `SELECT out_of_stock_on_interested FROM books WHERE id = $1`
	updateBookQuery := `UPDATE books SET in_stock = FALSE, is_published = FALSE, unpublished_at = CASE WHEN is_published THEN NOW() ELSE unpublished_at END, unpublished_reason = 'out_of_stock', updated_at = NOW() WHERE id = $1 AND (in_stock = TRUE OR is_published = TRUE)`
	unpublishBundlesQuery := `UPDATE bundles b SET is_published = FALSE, unpublished_at = NOW(), unpublished_reason = 'out_of_stock', updated_at = NOW() WHERE b.is_published = TRUE AND EXISTS (SELECT 1 FROM bundle_books bb WHERE bb.bundle_id = b.id AND bb.book_id = $1)`
	now := time.Date(2026, time.March, 8, 8, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(updateQuery)).
		WithArgs("Rajesh", "+919999999999", "Call after 6", "system-admin", 9).
		WillReturnRows(sqlmock.NewRows([]string{"id", "item_id", "item_type", "item_title", "source_page", "source_rail_id", "source_rail_title", "status", "buyer_name", "buyer_phone", "buyer_note", "converted_by", "converted_at", "created_at"}).
			AddRow(9, 7, "BOOK", "Book One", "catalog", 2, "Fresh Books", "interested", "Rajesh", "+919999999999", "Call after 6", "system-admin", now, now))
	mock.ExpectQuery(regexp.QuoteMeta(bookFlagQuery)).
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"out_of_stock_on_interested"}).AddRow(true))
	mock.ExpectExec(regexp.QuoteMeta(updateBookQuery)).
		WithArgs(7).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(unpublishBundlesQuery)).
		WithArgs(7).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(updateQuery)).
		WithArgs("Rajesh", "+919999999999", "Call after 6", "system-admin", 9).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(regexp.QuoteMeta(getByIDQuery)).
		WithArgs(9).
		WillReturnRows(sqlmock.NewRows([]string{"id", "item_id", "item_type", "item_title", "source_page", "source_rail_id", "source_rail_title", "status", "buyer_name", "buyer_phone", "buyer_note", "converted_by", "converted_at", "created_at"}).
			AddRow(9, 7, "BOOK", "Book One", "catalog", 2, "Fresh Books", "interested", "Rajesh", "+919999999999", "Call after 6", "system-admin", now, now))
	mock.ExpectCommit()

	store := NewPostgresStore(db)
	updated, alreadyConverted, err := store.ConvertToInterested(9, ConvertInput{
		BuyerName:   "Rajesh",
		BuyerPhone:  "+919999999999",
		BuyerNote:   "Call after 6",
		ConvertedBy: "system-admin",
	})
	if err != nil {
		t.Fatalf("convert returned error: %v", err)
	}
	if alreadyConverted || updated.Status != StatusInterested {
		t.Fatalf("unexpected convert output: already=%v, enquiry=%+v", alreadyConverted, updated)
	}

	_, alreadyConverted, err = store.ConvertToInterested(9, ConvertInput{
		BuyerName:   "Rajesh",
		BuyerPhone:  "+919999999999",
		BuyerNote:   "Call after 6",
		ConvertedBy: "system-admin",
	})
	if err != nil {
		t.Fatalf("second convert returned error: %v", err)
	}
	if !alreadyConverted {
		t.Fatalf("expected alreadyConverted=true on second conversion")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresStoreConvertToInterestedBundleSideEffects(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	updateQuery := `UPDATE enquiries SET status = 'interested', buyer_name = $1, buyer_phone = $2, buyer_note = $3, converted_by = $4, converted_at = NOW() WHERE id = $5 AND status = 'clicked' RETURNING id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, buyer_name, buyer_phone, buyer_note, converted_by, converted_at, created_at`
	bundleFlagQuery := `SELECT out_of_stock_on_interested FROM bundles WHERE id = $1`
	updateBundleQuery := `UPDATE bundles SET in_stock = FALSE, is_published = FALSE, unpublished_at = CASE WHEN is_published THEN NOW() ELSE unpublished_at END, unpublished_reason = 'out_of_stock', updated_at = NOW() WHERE id = $1 AND (in_stock = TRUE OR is_published = TRUE)`
	now := time.Date(2026, time.March, 8, 8, 30, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(updateQuery)).
		WithArgs("Rajesh", "+919999999999", "Bundle", "system-admin", 10).
		WillReturnRows(sqlmock.NewRows([]string{"id", "item_id", "item_type", "item_title", "source_page", "source_rail_id", "source_rail_title", "status", "buyer_name", "buyer_phone", "buyer_note", "converted_by", "converted_at", "created_at"}).
			AddRow(10, 23, "BUNDLE", "Bundle One", "catalog", 5, "Premium bundles", "interested", "Rajesh", "+919999999999", "Bundle", "system-admin", now, now))
	mock.ExpectQuery(regexp.QuoteMeta(bundleFlagQuery)).
		WithArgs(23).
		WillReturnRows(sqlmock.NewRows([]string{"out_of_stock_on_interested"}).AddRow(true))
	mock.ExpectExec(regexp.QuoteMeta(updateBundleQuery)).
		WithArgs(23).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	store := NewPostgresStore(db)
	updated, alreadyConverted, err := store.ConvertToInterested(10, ConvertInput{
		BuyerName:   "Rajesh",
		BuyerPhone:  "+919999999999",
		BuyerNote:   "Bundle",
		ConvertedBy: "system-admin",
	})
	if err != nil {
		t.Fatalf("convert returned error: %v", err)
	}
	if alreadyConverted || updated.Status != StatusInterested {
		t.Fatalf("unexpected convert output: already=%v, enquiry=%+v", alreadyConverted, updated)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresStoreConvertToInterestedSkipsSideEffectsWhenDisabled(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	updateQuery := `UPDATE enquiries SET status = 'interested', buyer_name = $1, buyer_phone = $2, buyer_note = $3, converted_by = $4, converted_at = NOW() WHERE id = $5 AND status = 'clicked' RETURNING id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, buyer_name, buyer_phone, buyer_note, converted_by, converted_at, created_at`
	bookFlagQuery := `SELECT out_of_stock_on_interested FROM books WHERE id = $1`
	now := time.Date(2026, time.March, 8, 8, 45, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(updateQuery)).
		WithArgs("Rajesh", "+919999999999", "", "system-admin", 11).
		WillReturnRows(sqlmock.NewRows([]string{"id", "item_id", "item_type", "item_title", "source_page", "source_rail_id", "source_rail_title", "status", "buyer_name", "buyer_phone", "buyer_note", "converted_by", "converted_at", "created_at"}).
			AddRow(11, 44, "BOOK", "Book Disabled", "catalog", 3, "Deals", "interested", "Rajesh", "+919999999999", "", "system-admin", now, now))
	mock.ExpectQuery(regexp.QuoteMeta(bookFlagQuery)).
		WithArgs(44).
		WillReturnRows(sqlmock.NewRows([]string{"out_of_stock_on_interested"}).AddRow(false))
	mock.ExpectCommit()

	store := NewPostgresStore(db)
	updated, alreadyConverted, err := store.ConvertToInterested(11, ConvertInput{
		BuyerName:   "Rajesh",
		BuyerPhone:  "+919999999999",
		BuyerNote:   "",
		ConvertedBy: "system-admin",
	})
	if err != nil {
		t.Fatalf("convert returned error: %v", err)
	}
	if alreadyConverted || updated.Status != StatusInterested {
		t.Fatalf("unexpected convert output: already=%v, enquiry=%+v", alreadyConverted, updated)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresStoreConvertToInterestedNotFoundPaths(t *testing.T) {
	t.Run("missing enquiry returns ErrNotFound", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("failed to create sqlmock: %v", err)
		}
		defer db.Close()

		updateQuery := `UPDATE enquiries SET status = 'interested', buyer_name = $1, buyer_phone = $2, buyer_note = $3, converted_by = $4, converted_at = NOW() WHERE id = $5 AND status = 'clicked' RETURNING id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, buyer_name, buyer_phone, buyer_note, converted_by, converted_at, created_at`
		getByIDQuery := `SELECT id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, buyer_name, buyer_phone, buyer_note, converted_by, converted_at, created_at FROM enquiries WHERE id = $1`

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(updateQuery)).
			WithArgs("Rajesh", "+919999999999", "", "system-admin", 42).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(regexp.QuoteMeta(getByIDQuery)).
			WithArgs(42).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectRollback()

		store := NewPostgresStore(db)
		_, alreadyConverted, err := store.ConvertToInterested(42, ConvertInput{
			BuyerName:   "Rajesh",
			BuyerPhone:  "+919999999999",
			BuyerNote:   "",
			ConvertedBy: "system-admin",
		})
		if err != ErrNotFound {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
		if alreadyConverted {
			t.Fatalf("expected alreadyConverted=false")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet SQL expectations: %v", err)
		}
	})

	t.Run("still-clicked enquiry returns ErrNotFound", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("failed to create sqlmock: %v", err)
		}
		defer db.Close()

		updateQuery := `UPDATE enquiries SET status = 'interested', buyer_name = $1, buyer_phone = $2, buyer_note = $3, converted_by = $4, converted_at = NOW() WHERE id = $5 AND status = 'clicked' RETURNING id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, buyer_name, buyer_phone, buyer_note, converted_by, converted_at, created_at`
		getByIDQuery := `SELECT id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, buyer_name, buyer_phone, buyer_note, converted_by, converted_at, created_at FROM enquiries WHERE id = $1`
		now := time.Date(2026, time.March, 8, 9, 0, 0, 0, time.UTC)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(updateQuery)).
			WithArgs("Rajesh", "+919999999999", "", "system-admin", 77).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(regexp.QuoteMeta(getByIDQuery)).
			WithArgs(77).
			WillReturnRows(sqlmock.NewRows([]string{"id", "item_id", "item_type", "item_title", "source_page", "source_rail_id", "source_rail_title", "status", "buyer_name", "buyer_phone", "buyer_note", "converted_by", "converted_at", "created_at"}).
				AddRow(77, 9, "BOOK", "Book Nine", "catalog", 1, "Fresh Books", "clicked", "", "", "", nil, nil, now))
		mock.ExpectRollback()

		store := NewPostgresStore(db)
		_, alreadyConverted, err := store.ConvertToInterested(77, ConvertInput{
			BuyerName:   "Rajesh",
			BuyerPhone:  "+919999999999",
			BuyerNote:   "",
			ConvertedBy: "system-admin",
		})
		if err != ErrNotFound {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
		if alreadyConverted {
			t.Fatalf("expected alreadyConverted=false")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet SQL expectations: %v", err)
		}
	})
}
