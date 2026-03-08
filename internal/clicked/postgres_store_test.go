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

func TestPostgresStoreConvertToInterestedAndIdempotent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	updateQuery := `UPDATE enquiries SET status = 'interested', buyer_name = $1, buyer_phone = $2, buyer_note = $3, converted_by = $4, converted_at = NOW() WHERE id = $5 AND status = 'clicked' RETURNING id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, buyer_name, buyer_phone, buyer_note, converted_by, converted_at, created_at`
	getQuery := `SELECT id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, buyer_name, buyer_phone, buyer_note, converted_by, converted_at, created_at FROM enquiries WHERE id = $1`
	now := time.Date(2026, time.March, 8, 8, 0, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta(updateQuery)).
		WithArgs("Rajesh", "+919999999999", "Call after 6", "system-admin", 9).
		WillReturnRows(sqlmock.NewRows([]string{"id", "item_id", "item_type", "item_title", "source_page", "source_rail_id", "source_rail_title", "status", "buyer_name", "buyer_phone", "buyer_note", "converted_by", "converted_at", "created_at"}).
			AddRow(9, 7, "BOOK", "Book One", "catalog", 2, "Fresh Books", "interested", "Rajesh", "+919999999999", "Call after 6", "system-admin", now, now))

	mock.ExpectQuery(regexp.QuoteMeta(updateQuery)).
		WithArgs("Rajesh", "+919999999999", "Call after 6", "system-admin", 9).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(regexp.QuoteMeta(getQuery)).
		WithArgs(9).
		WillReturnRows(sqlmock.NewRows([]string{"id", "item_id", "item_type", "item_title", "source_page", "source_rail_id", "source_rail_title", "status", "buyer_name", "buyer_phone", "buyer_note", "converted_by", "converted_at", "created_at"}).
			AddRow(9, 7, "BOOK", "Book One", "catalog", 2, "Fresh Books", "interested", "Rajesh", "+919999999999", "Call after 6", "system-admin", now, now))

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
		t.Fatalf("expected already-converted=true on second conversion")
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
		getQuery := `SELECT id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, buyer_name, buyer_phone, buyer_note, converted_by, converted_at, created_at FROM enquiries WHERE id = $1`

		mock.ExpectQuery(regexp.QuoteMeta(updateQuery)).
			WithArgs("Rajesh", "+919999999999", "", "system-admin", 42).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(regexp.QuoteMeta(getQuery)).
			WithArgs(42).
			WillReturnError(sql.ErrNoRows)

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
		getQuery := `SELECT id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, buyer_name, buyer_phone, buyer_note, converted_by, converted_at, created_at FROM enquiries WHERE id = $1`
		now := time.Date(2026, time.March, 8, 9, 0, 0, 0, time.UTC)

		mock.ExpectQuery(regexp.QuoteMeta(updateQuery)).
			WithArgs("Rajesh", "+919999999999", "", "system-admin", 77).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(regexp.QuoteMeta(getQuery)).
			WithArgs(77).
			WillReturnRows(sqlmock.NewRows([]string{"id", "item_id", "item_type", "item_title", "source_page", "source_rail_id", "source_rail_title", "status", "buyer_name", "buyer_phone", "buyer_note", "converted_by", "converted_at", "created_at"}).
				AddRow(77, 9, "BOOK", "Book Nine", "catalog", 1, "Fresh Books", "clicked", "", "", "", nil, nil, now))

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
