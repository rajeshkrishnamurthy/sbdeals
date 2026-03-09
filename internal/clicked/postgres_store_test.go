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

	createQuery := `INSERT INTO enquiries (item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, customer_id) VALUES ($1, $2, $3, $4, $5, $6, 'clicked', 1) RETURNING id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, customer_id, buyer_note, order_amount, last_modified_by, l_m_at, created_at`
	listQuery := `SELECT id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, customer_id, buyer_note, order_amount, last_modified_by, l_m_at, created_at FROM enquiries WHERE status = $1 ORDER BY created_at DESC, id DESC`
	now := time.Date(2026, time.March, 9, 7, 0, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta(createQuery)).
		WithArgs(7, "BOOK", "Book One", "catalog", 2, "Fresh Books").
		WillReturnRows(sqlmock.NewRows([]string{"id", "item_id", "item_type", "item_title", "source_page", "source_rail_id", "source_rail_title", "status", "customer_id", "buyer_note", "order_amount", "last_modified_by", "l_m_at", "created_at"}).
			AddRow(1, 7, "BOOK", "Book One", "catalog", 2, "Fresh Books", "clicked", 1, "", nil, "", nil, now))
	mock.ExpectQuery(regexp.QuoteMeta(listQuery)).
		WithArgs("clicked").
		WillReturnRows(sqlmock.NewRows([]string{"id", "item_id", "item_type", "item_title", "source_page", "source_rail_id", "source_rail_title", "status", "customer_id", "buyer_note", "order_amount", "last_modified_by", "l_m_at", "created_at"}).
			AddRow(1, 7, "BOOK", "Book One", "catalog", 2, "Fresh Books", "clicked", 1, "", nil, "", nil, now))

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

	updateQuery := `UPDATE enquiries SET status = 'interested', customer_id = $1, buyer_note = $2, last_modified_by = $3, l_m_at = NOW() WHERE id = $4 AND status = 'clicked' RETURNING id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, customer_id, buyer_note, order_amount, last_modified_by, l_m_at, created_at`
	getByIDQuery := `SELECT id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, customer_id, buyer_note, order_amount, last_modified_by, l_m_at, created_at FROM enquiries WHERE id = $1`
	bookFlagQuery := `SELECT out_of_stock_on_interested FROM books WHERE id = $1`
	updateBookQuery := `UPDATE books SET in_stock = FALSE, is_published = FALSE, unpublished_at = CASE WHEN is_published THEN NOW() ELSE unpublished_at END, unpublished_reason = 'out_of_stock', updated_at = NOW() WHERE id = $1 AND (in_stock = TRUE OR is_published = TRUE)`
	unpublishBundlesQuery := `UPDATE bundles b SET is_published = FALSE, unpublished_at = NOW(), unpublished_reason = 'out_of_stock', updated_at = NOW() WHERE b.is_published = TRUE AND EXISTS (SELECT 1 FROM bundle_books bb WHERE bb.bundle_id = b.id AND bb.book_id = $1)`
	now := time.Date(2026, time.March, 9, 8, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(updateQuery)).
		WithArgs(11, "Call after 6", "system-admin", 9).
		WillReturnRows(sqlmock.NewRows([]string{"id", "item_id", "item_type", "item_title", "source_page", "source_rail_id", "source_rail_title", "status", "customer_id", "buyer_note", "order_amount", "last_modified_by", "l_m_at", "created_at"}).
			AddRow(9, 7, "BOOK", "Book One", "catalog", 2, "Fresh Books", "interested", 11, "Call after 6", nil, "system-admin", now, now))
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
		WithArgs(11, "Call after 6", "system-admin", 9).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(regexp.QuoteMeta(getByIDQuery)).
		WithArgs(9).
		WillReturnRows(sqlmock.NewRows([]string{"id", "item_id", "item_type", "item_title", "source_page", "source_rail_id", "source_rail_title", "status", "customer_id", "buyer_note", "order_amount", "last_modified_by", "l_m_at", "created_at"}).
			AddRow(9, 7, "BOOK", "Book One", "catalog", 2, "Fresh Books", "interested", 11, "Call after 6", nil, "system-admin", now, now))
	mock.ExpectCommit()

	store := NewPostgresStore(db)
	updated, alreadyConverted, err := store.ConvertToInterested(9, ConvertInput{
		CustomerID: 11,
		Note:       "Call after 6",
		ModifiedBy: "system-admin",
	})
	if err != nil {
		t.Fatalf("convert returned error: %v", err)
	}
	if alreadyConverted || updated.Status != StatusInterested || updated.CustomerID != 11 {
		t.Fatalf("unexpected convert output: already=%v, enquiry=%+v", alreadyConverted, updated)
	}

	_, alreadyConverted, err = store.ConvertToInterested(9, ConvertInput{
		CustomerID: 11,
		Note:       "Call after 6",
		ModifiedBy: "system-admin",
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

	updateQuery := `UPDATE enquiries SET status = 'interested', customer_id = $1, buyer_note = $2, last_modified_by = $3, l_m_at = NOW() WHERE id = $4 AND status = 'clicked' RETURNING id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, customer_id, buyer_note, order_amount, last_modified_by, l_m_at, created_at`
	bundleFlagQuery := `SELECT out_of_stock_on_interested FROM bundles WHERE id = $1`
	updateBundleQuery := `UPDATE bundles SET in_stock = FALSE, is_published = FALSE, unpublished_at = CASE WHEN is_published THEN NOW() ELSE unpublished_at END, unpublished_reason = 'out_of_stock', updated_at = NOW() WHERE id = $1 AND (in_stock = TRUE OR is_published = TRUE)`
	now := time.Date(2026, time.March, 9, 8, 30, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(updateQuery)).
		WithArgs(9, "Bundle", "system-admin", 10).
		WillReturnRows(sqlmock.NewRows([]string{"id", "item_id", "item_type", "item_title", "source_page", "source_rail_id", "source_rail_title", "status", "customer_id", "buyer_note", "order_amount", "last_modified_by", "l_m_at", "created_at"}).
			AddRow(10, 23, "BUNDLE", "Bundle One", "catalog", 5, "Premium bundles", "interested", 9, "Bundle", nil, "system-admin", now, now))
	mock.ExpectQuery(regexp.QuoteMeta(bundleFlagQuery)).
		WithArgs(23).
		WillReturnRows(sqlmock.NewRows([]string{"out_of_stock_on_interested"}).AddRow(true))
	mock.ExpectExec(regexp.QuoteMeta(updateBundleQuery)).
		WithArgs(23).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	store := NewPostgresStore(db)
	updated, alreadyConverted, err := store.ConvertToInterested(10, ConvertInput{
		CustomerID: 9,
		Note:       "Bundle",
		ModifiedBy: "system-admin",
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

	updateQuery := `UPDATE enquiries SET status = 'interested', customer_id = $1, buyer_note = $2, last_modified_by = $3, l_m_at = NOW() WHERE id = $4 AND status = 'clicked' RETURNING id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, customer_id, buyer_note, order_amount, last_modified_by, l_m_at, created_at`
	bookFlagQuery := `SELECT out_of_stock_on_interested FROM books WHERE id = $1`
	now := time.Date(2026, time.March, 9, 8, 45, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(updateQuery)).
		WithArgs(5, nil, "system-admin", 11).
		WillReturnRows(sqlmock.NewRows([]string{"id", "item_id", "item_type", "item_title", "source_page", "source_rail_id", "source_rail_title", "status", "customer_id", "buyer_note", "order_amount", "last_modified_by", "l_m_at", "created_at"}).
			AddRow(11, 44, "BOOK", "Book Disabled", "catalog", 3, "Deals", "interested", 5, "", nil, "system-admin", now, now))
	mock.ExpectQuery(regexp.QuoteMeta(bookFlagQuery)).
		WithArgs(44).
		WillReturnRows(sqlmock.NewRows([]string{"out_of_stock_on_interested"}).AddRow(false))
	mock.ExpectCommit()

	store := NewPostgresStore(db)
	updated, alreadyConverted, err := store.ConvertToInterested(11, ConvertInput{
		CustomerID: 5,
		Note:       "",
		ModifiedBy: "system-admin",
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

		updateQuery := `UPDATE enquiries SET status = 'interested', customer_id = $1, buyer_note = $2, last_modified_by = $3, l_m_at = NOW() WHERE id = $4 AND status = 'clicked' RETURNING id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, customer_id, buyer_note, order_amount, last_modified_by, l_m_at, created_at`
		getByIDQuery := `SELECT id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, customer_id, buyer_note, order_amount, last_modified_by, l_m_at, created_at FROM enquiries WHERE id = $1`

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(updateQuery)).
			WithArgs(1, nil, "system-admin", 42).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(regexp.QuoteMeta(getByIDQuery)).
			WithArgs(42).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectRollback()

		store := NewPostgresStore(db)
		_, alreadyConverted, err := store.ConvertToInterested(42, ConvertInput{
			CustomerID: 1,
			Note:       "",
			ModifiedBy: "system-admin",
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

		updateQuery := `UPDATE enquiries SET status = 'interested', customer_id = $1, buyer_note = $2, last_modified_by = $3, l_m_at = NOW() WHERE id = $4 AND status = 'clicked' RETURNING id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, customer_id, buyer_note, order_amount, last_modified_by, l_m_at, created_at`
		getByIDQuery := `SELECT id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, customer_id, buyer_note, order_amount, last_modified_by, l_m_at, created_at FROM enquiries WHERE id = $1`
		now := time.Date(2026, time.March, 9, 9, 0, 0, 0, time.UTC)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(updateQuery)).
			WithArgs(1, nil, "system-admin", 77).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(regexp.QuoteMeta(getByIDQuery)).
			WithArgs(77).
			WillReturnRows(sqlmock.NewRows([]string{"id", "item_id", "item_type", "item_title", "source_page", "source_rail_id", "source_rail_title", "status", "customer_id", "buyer_note", "order_amount", "last_modified_by", "l_m_at", "created_at"}).
				AddRow(77, 9, "BOOK", "Book Nine", "catalog", 1, "Fresh Books", "clicked", 1, "", nil, "", nil, now))
		mock.ExpectRollback()

		store := NewPostgresStore(db)
		_, alreadyConverted, err := store.ConvertToInterested(77, ConvertInput{
			CustomerID: 1,
			Note:       "",
			ModifiedBy: "system-admin",
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

func TestPostgresStoreConvertToOrderedSuccessAndIdempotent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	lockByIDQuery := `SELECT id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, customer_id, buyer_note, order_amount, last_modified_by, l_m_at, created_at FROM enquiries WHERE id = $1 FOR UPDATE`
	customerAddressQuery := `SELECT address FROM customers WHERE id = $1 FOR UPDATE`
	updateOrderedQuery := `UPDATE enquiries SET status = 'ordered', order_amount = $1, buyer_note = $2, last_modified_by = $3, l_m_at = NOW() WHERE id = $4 AND status = 'interested' RETURNING id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, customer_id, buyer_note, order_amount, last_modified_by, l_m_at, created_at`
	now := time.Date(2026, time.March, 9, 10, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(lockByIDQuery)).
		WithArgs(55).
		WillReturnRows(sqlmock.NewRows([]string{"id", "item_id", "item_type", "item_title", "source_page", "source_rail_id", "source_rail_title", "status", "customer_id", "buyer_note", "order_amount", "last_modified_by", "l_m_at", "created_at"}).
			AddRow(55, 7, "BOOK", "Book One", "catalog", 2, "Fresh Books", "interested", 11, "Interested", nil, "system-admin", now, now))
	mock.ExpectQuery(regexp.QuoteMeta(customerAddressQuery)).
		WithArgs(11).
		WillReturnRows(sqlmock.NewRows([]string{"address"}).AddRow("Block A, MG Road"))
	mock.ExpectQuery(regexp.QuoteMeta(updateOrderedQuery)).
		WithArgs(499, "Paid in cash", "system-admin", 55).
		WillReturnRows(sqlmock.NewRows([]string{"id", "item_id", "item_type", "item_title", "source_page", "source_rail_id", "source_rail_title", "status", "customer_id", "buyer_note", "order_amount", "last_modified_by", "l_m_at", "created_at"}).
			AddRow(55, 7, "BOOK", "Book One", "catalog", 2, "Fresh Books", "ordered", 11, "Paid in cash", 499, "system-admin", now, now))
	mock.ExpectCommit()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(lockByIDQuery)).
		WithArgs(55).
		WillReturnRows(sqlmock.NewRows([]string{"id", "item_id", "item_type", "item_title", "source_page", "source_rail_id", "source_rail_title", "status", "customer_id", "buyer_note", "order_amount", "last_modified_by", "l_m_at", "created_at"}).
			AddRow(55, 7, "BOOK", "Book One", "catalog", 2, "Fresh Books", "ordered", 11, "Paid in cash", 499, "system-admin", now, now))
	mock.ExpectCommit()

	store := NewPostgresStore(db)
	updated, alreadyOrdered, err := store.ConvertToOrdered(55, OrderInput{
		OrderAmount: 499,
		Note:        "Paid in cash",
		ModifiedBy:  "system-admin",
	})
	if err != nil {
		t.Fatalf("convert to ordered returned error: %v", err)
	}
	if alreadyOrdered || updated.Status != StatusOrdered {
		t.Fatalf("unexpected convert output: already=%v, enquiry=%+v", alreadyOrdered, updated)
	}
	if updated.OrderAmount == nil || *updated.OrderAmount != 499 {
		t.Fatalf("expected order amount 499, got %+v", updated.OrderAmount)
	}

	_, alreadyOrdered, err = store.ConvertToOrdered(55, OrderInput{
		OrderAmount: 999,
		Note:        "Should not overwrite",
		ModifiedBy:  "system-admin",
	})
	if err != nil {
		t.Fatalf("second convert to ordered returned error: %v", err)
	}
	if !alreadyOrdered {
		t.Fatalf("expected alreadyOrdered=true on second conversion")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresStoreConvertToOrderedAddressAndTransitionGuards(t *testing.T) {
	t.Run("requires address when linked customer address is missing", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("failed to create sqlmock: %v", err)
		}
		defer db.Close()

		lockByIDQuery := `SELECT id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, customer_id, buyer_note, order_amount, last_modified_by, l_m_at, created_at FROM enquiries WHERE id = $1 FOR UPDATE`
		customerAddressQuery := `SELECT address FROM customers WHERE id = $1 FOR UPDATE`
		now := time.Date(2026, time.March, 9, 10, 15, 0, 0, time.UTC)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(lockByIDQuery)).
			WithArgs(71).
			WillReturnRows(sqlmock.NewRows([]string{"id", "item_id", "item_type", "item_title", "source_page", "source_rail_id", "source_rail_title", "status", "customer_id", "buyer_note", "order_amount", "last_modified_by", "l_m_at", "created_at"}).
				AddRow(71, 44, "BOOK", "Book Forty Four", "catalog", 3, "Deals", "interested", 9, "", nil, "system-admin", now, now))
		mock.ExpectQuery(regexp.QuoteMeta(customerAddressQuery)).
			WithArgs(9).
			WillReturnRows(sqlmock.NewRows([]string{"address"}).AddRow(nil))
		mock.ExpectRollback()

		store := NewPostgresStore(db)
		_, alreadyOrdered, err := store.ConvertToOrdered(71, OrderInput{
			OrderAmount: 399,
			ModifiedBy:  "system-admin",
		})
		if err != ErrAddressRequired {
			t.Fatalf("expected ErrAddressRequired, got %v", err)
		}
		if alreadyOrdered {
			t.Fatalf("expected alreadyOrdered=false")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet SQL expectations: %v", err)
		}
	})

	t.Run("rejects clicked to ordered transition", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("failed to create sqlmock: %v", err)
		}
		defer db.Close()

		lockByIDQuery := `SELECT id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, customer_id, buyer_note, order_amount, last_modified_by, l_m_at, created_at FROM enquiries WHERE id = $1 FOR UPDATE`
		now := time.Date(2026, time.March, 9, 10, 20, 0, 0, time.UTC)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(lockByIDQuery)).
			WithArgs(72).
			WillReturnRows(sqlmock.NewRows([]string{"id", "item_id", "item_type", "item_title", "source_page", "source_rail_id", "source_rail_title", "status", "customer_id", "buyer_note", "order_amount", "last_modified_by", "l_m_at", "created_at"}).
				AddRow(72, 12, "BUNDLE", "Bundle Twelve", "catalog", 4, "Premium", "clicked", 5, "", nil, "system-admin", now, now))
		mock.ExpectRollback()

		store := NewPostgresStore(db)
		_, alreadyOrdered, err := store.ConvertToOrdered(72, OrderInput{
			OrderAmount: 299,
			ModifiedBy:  "system-admin",
		})
		if err != ErrInvalidTransition {
			t.Fatalf("expected ErrInvalidTransition, got %v", err)
		}
		if alreadyOrdered {
			t.Fatalf("expected alreadyOrdered=false")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet SQL expectations: %v", err)
		}
	})
}

func TestPostgresStoreConvertToOrderedStoresEmptyNoteAsEmptyString(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	lockByIDQuery := `SELECT id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, customer_id, buyer_note, order_amount, last_modified_by, l_m_at, created_at FROM enquiries WHERE id = $1 FOR UPDATE`
	customerAddressQuery := `SELECT address FROM customers WHERE id = $1 FOR UPDATE`
	updateOrderedQuery := `UPDATE enquiries SET status = 'ordered', order_amount = $1, buyer_note = $2, last_modified_by = $3, l_m_at = NOW() WHERE id = $4 AND status = 'interested' RETURNING id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, customer_id, buyer_note, order_amount, last_modified_by, l_m_at, created_at`
	now := time.Date(2026, time.March, 10, 6, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(lockByIDQuery)).
		WithArgs(81).
		WillReturnRows(sqlmock.NewRows([]string{"id", "item_id", "item_type", "item_title", "source_page", "source_rail_id", "source_rail_title", "status", "customer_id", "buyer_note", "order_amount", "last_modified_by", "l_m_at", "created_at"}).
			AddRow(81, 5, "BOOK", "Book Five", "catalog", 1, "Fresh", "interested", 4, "prior", nil, "system-admin", now, now))
	mock.ExpectQuery(regexp.QuoteMeta(customerAddressQuery)).
		WithArgs(4).
		WillReturnRows(sqlmock.NewRows([]string{"address"}).AddRow("Addr"))
	mock.ExpectQuery(regexp.QuoteMeta(updateOrderedQuery)).
		WithArgs(219, nil, "system-admin", 81).
		WillReturnRows(sqlmock.NewRows([]string{"id", "item_id", "item_type", "item_title", "source_page", "source_rail_id", "source_rail_title", "status", "customer_id", "buyer_note", "order_amount", "last_modified_by", "l_m_at", "created_at"}).
			AddRow(81, 5, "BOOK", "Book Five", "catalog", 1, "Fresh", "ordered", 4, nil, 219, "system-admin", now, now))
	mock.ExpectCommit()

	store := NewPostgresStore(db)
	updated, alreadyOrdered, err := store.ConvertToOrdered(81, OrderInput{
		OrderAmount: 219,
		Note:        "",
		ModifiedBy:  "system-admin",
	})
	if err != nil {
		t.Fatalf("convert to ordered returned error: %v", err)
	}
	if alreadyOrdered {
		t.Fatalf("expected alreadyOrdered=false")
	}
	if updated.Note != "" {
		t.Fatalf("expected empty note, got %q", updated.Note)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}
