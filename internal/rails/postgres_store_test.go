package rails

import (
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestPostgresStoreCreateAndGet(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	now := time.Date(2026, time.March, 7, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS(SELECT 1 FROM rails WHERE LOWER(TRIM(title)) = LOWER(TRIM($1)) AND id <> $2)`)).
		WithArgs("Top Picks", 0).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO rails (title, rail_type, admin_note, position) VALUES ($1, $2, $3, COALESCE((SELECT MAX(position) + 1 FROM rails), 1)) RETURNING id, title, rail_type, admin_note, position, is_published, published_at, unpublished_at`)).
		WithArgs("Top Picks", "BOOK", "Internal note").
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "rail_type", "admin_note", "position", "is_published", "published_at", "unpublished_at"}).
			AddRow(1, "Top Picks", "BOOK", "Internal note", 1, false, nil, now))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, title, rail_type, admin_note, position, is_published, published_at, unpublished_at FROM rails WHERE id = $1`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "rail_type", "admin_note", "position", "is_published", "published_at", "unpublished_at"}).
			AddRow(1, "Top Picks", "BOOK", "Internal note", 1, false, nil, now))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT item_id FROM rail_items WHERE rail_id = $1 ORDER BY created_at ASC, item_id ASC`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"item_id"}))

	store := NewPostgresStore(db)
	created, err := store.Create(CreateInput{Title: "Top Picks", Type: RailTypeBook, AdminNote: "Internal note"})
	if err != nil {
		t.Fatalf("create returned error: %v", err)
	}
	if created.Type != RailTypeBook || created.IsPublished {
		t.Fatalf("unexpected created rail: %+v", created)
	}
	if created.AdminNote != "Internal note" {
		t.Fatalf("expected admin note to persist, got %q", created.AdminNote)
	}

	fetched, err := store.Get(1)
	if err != nil {
		t.Fatalf("get returned error: %v", err)
	}
	if fetched.ID != 1 || fetched.Type != RailTypeBook {
		t.Fatalf("unexpected fetched rail: %+v", fetched)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresStoreTitleValidationAndDuplicateAdd(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS(SELECT 1 FROM rails WHERE LOWER(TRIM(title)) = LOWER(TRIM($1)) AND id <> $2)`)).
		WithArgs("Top Picks", 0).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	store := NewPostgresStore(db)
	if _, err := store.Create(CreateInput{Title: "Top Picks", Type: RailTypeBook}); !errors.Is(err, ErrDuplicateTitle) {
		t.Fatalf("expected ErrDuplicateTitle, got %v", err)
	}

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO rail_items (rail_id, item_id) VALUES ($1, $2) ON CONFLICT (rail_id, item_id) DO NOTHING`)).
		WithArgs(1, 10).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, title, rail_type, admin_note, position, is_published, published_at, unpublished_at FROM rails WHERE id = $1`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "rail_type", "admin_note", "position", "is_published", "published_at", "unpublished_at"}).
			AddRow(1, "Top Picks", "BOOK", "note", 1, false, nil, time.Date(2026, time.March, 7, 0, 0, 0, 0, time.UTC)))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT item_id FROM rail_items WHERE rail_id = $1 ORDER BY created_at ASC, item_id ASC`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"item_id"}))

	if _, err := store.AddItem(1, 10); !errors.Is(err, ErrDuplicateItem) {
		t.Fatalf("expected ErrDuplicateItem, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresStorePublishUnpublishMove(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	now := time.Date(2026, time.March, 7, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE rails SET is_published = TRUE, published_at = NOW(), updated_at = NOW() WHERE id = $1 RETURNING id, title, rail_type, admin_note, position, is_published, published_at, unpublished_at`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "rail_type", "admin_note", "position", "is_published", "published_at", "unpublished_at"}).
			AddRow(1, "Top Picks", "BOOK", "note", 1, true, now, now))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT item_id FROM rail_items WHERE rail_id = $1 ORDER BY created_at ASC, item_id ASC`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"item_id"}).AddRow(10))

	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE rails SET is_published = FALSE, unpublished_at = NOW(), updated_at = NOW() WHERE id = $1 RETURNING id, title, rail_type, admin_note, position, is_published, published_at, unpublished_at`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "rail_type", "admin_note", "position", "is_published", "published_at", "unpublished_at"}).
			AddRow(1, "Top Picks", "BOOK", "note", 1, false, now, now))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT item_id FROM rail_items WHERE rail_id = $1 ORDER BY created_at ASC, item_id ASC`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"item_id"}).AddRow(10))

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, position FROM rails WHERE id = $1`)).
		WithArgs(2).
		WillReturnRows(sqlmock.NewRows([]string{"id", "position"}).AddRow(2, 2))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, position FROM rails WHERE position < $1 ORDER BY position DESC, id DESC LIMIT 1`)).
		WithArgs(2).
		WillReturnRows(sqlmock.NewRows([]string{"id", "position"}).AddRow(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE rails SET position = $1, updated_at = NOW() WHERE id = $2`)).
		WithArgs(1, 2).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE rails SET position = $1, updated_at = NOW() WHERE id = $2`)).
		WithArgs(2, 1).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	store := NewPostgresStore(db)
	published, err := store.Publish(1)
	if err != nil {
		t.Fatalf("publish returned error: %v", err)
	}
	if !published.IsPublished {
		t.Fatalf("expected published")
	}
	unpublished, err := store.Unpublish(1)
	if err != nil {
		t.Fatalf("unpublish returned error: %v", err)
	}
	if unpublished.IsPublished {
		t.Fatalf("expected unpublished")
	}
	if err := store.MoveUp(2); err != nil {
		t.Fatalf("move up returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresStoreUpdateAdminNote(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS(SELECT 1 FROM rails WHERE LOWER(TRIM(title)) = LOWER(TRIM($1)) AND id <> $2)`)).
		WithArgs("Top Picks", 1).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE rails SET title = $1, admin_note = $2, updated_at = NOW() WHERE id = $3 RETURNING id, title, rail_type, admin_note, position, is_published, published_at, unpublished_at`)).
		WithArgs("Top Picks", "Updated note", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "rail_type", "admin_note", "position", "is_published", "published_at", "unpublished_at"}).
			AddRow(1, "Top Picks", "BOOK", "Updated note", 1, false, nil, time.Date(2026, time.March, 7, 0, 0, 0, 0, time.UTC)))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT item_id FROM rail_items WHERE rail_id = $1 ORDER BY created_at ASC, item_id ASC`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"item_id"}))

	store := NewPostgresStore(db)
	updated, err := store.Update(1, UpdateInput{Title: "Top Picks", AdminNote: "Updated note"})
	if err != nil {
		t.Fatalf("update returned error: %v", err)
	}
	if updated.AdminNote != "Updated note" {
		t.Fatalf("expected admin note to persist, got %q", updated.AdminNote)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresStoreMoveNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, position FROM rails WHERE id = $1`)).
		WithArgs(99).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	store := NewPostgresStore(db)
	if err := store.MoveDown(99); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresStoreListRowsError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	query := `SELECT r.id, r.title, r.admin_note, r.rail_type, r.position, r.is_published, r.published_at, r.unpublished_at, COUNT(ri.item_id) AS item_count FROM rails r LEFT JOIN rail_items ri ON ri.rail_id = r.id GROUP BY r.id, r.title, r.admin_note, r.rail_type, r.position, r.is_published, r.published_at, r.unpublished_at ORDER BY r.position ASC, r.id ASC`
	rows := sqlmock.NewRows([]string{"id", "title", "admin_note", "rail_type", "position", "is_published", "published_at", "unpublished_at", "item_count"}).
		AddRow(1, "Top Picks", "note", "BOOK", 1, false, nil, nil, 0).
		RowError(0, errors.New("row iteration failed"))
	mock.ExpectQuery(regexp.QuoteMeta(query)).WillReturnRows(rows)

	store := NewPostgresStore(db)
	_, err = store.List()
	if err == nil {
		t.Fatalf("expected list rows error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresStoreListSuccessAndQueryError(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("failed to create sqlmock: %v", err)
		}
		defer db.Close()

		query := `SELECT r.id, r.title, r.admin_note, r.rail_type, r.position, r.is_published, r.published_at, r.unpublished_at, COUNT(ri.item_id) AS item_count FROM rails r LEFT JOIN rail_items ri ON ri.rail_id = r.id GROUP BY r.id, r.title, r.admin_note, r.rail_type, r.position, r.is_published, r.published_at, r.unpublished_at ORDER BY r.position ASC, r.id ASC`
		now := time.Date(2026, time.March, 8, 10, 0, 0, 0, time.UTC)
		rows := sqlmock.NewRows([]string{"id", "title", "admin_note", "rail_type", "position", "is_published", "published_at", "unpublished_at", "item_count"}).
			AddRow(1, "Books Rail", "note", "BOOK", 1, true, now, now, 3).
			AddRow(2, "Bundles Rail", "", "BUNDLE", 2, false, nil, now, 0)
		mock.ExpectQuery(regexp.QuoteMeta(query)).WillReturnRows(rows)

		store := NewPostgresStore(db)
		items, err := store.List()
		if err != nil {
			t.Fatalf("list returned error: %v", err)
		}
		if len(items) != 2 {
			t.Fatalf("expected 2 list rows, got %d", len(items))
		}
		if items[0].Type != RailTypeBook || !items[0].IsPublished || items[0].PublishedAt == nil {
			t.Fatalf("unexpected first row: %+v", items[0])
		}
		if items[1].Type != RailTypeBundle || items[1].PublishedAt != nil || items[1].UnpublishedAt == nil {
			t.Fatalf("unexpected second row: %+v", items[1])
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet SQL expectations: %v", err)
		}
	})

	t.Run("query error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("failed to create sqlmock: %v", err)
		}
		defer db.Close()

		query := `SELECT r.id, r.title, r.admin_note, r.rail_type, r.position, r.is_published, r.published_at, r.unpublished_at, COUNT(ri.item_id) AS item_count FROM rails r LEFT JOIN rail_items ri ON ri.rail_id = r.id GROUP BY r.id, r.title, r.admin_note, r.rail_type, r.position, r.is_published, r.published_at, r.unpublished_at ORDER BY r.position ASC, r.id ASC`
		mock.ExpectQuery(regexp.QuoteMeta(query)).WillReturnError(errors.New("query failed"))

		store := NewPostgresStore(db)
		_, err = store.List()
		if err == nil {
			t.Fatalf("expected query error")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet SQL expectations: %v", err)
		}
	})
}

func TestPostgresStoreRemoveItemErrorPaths(t *testing.T) {
	t.Run("delete failure", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("failed to create sqlmock: %v", err)
		}
		defer db.Close()

		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM rail_items WHERE rail_id = $1 AND item_id = $2`)).
			WithArgs(1, 10).
			WillReturnError(errors.New("delete failed"))

		store := NewPostgresStore(db)
		_, err = store.RemoveItem(1, 10)
		if err == nil {
			t.Fatalf("expected delete error")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet SQL expectations: %v", err)
		}
	})

	t.Run("post-delete rail missing", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("failed to create sqlmock: %v", err)
		}
		defer db.Close()

		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM rail_items WHERE rail_id = $1 AND item_id = $2`)).
			WithArgs(1, 10).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, title, rail_type, admin_note, position, is_published, published_at, unpublished_at FROM rails WHERE id = $1`)).
			WithArgs(1).
			WillReturnError(sql.ErrNoRows)

		store := NewPostgresStore(db)
		_, err = store.RemoveItem(1, 10)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet SQL expectations: %v", err)
		}
	})
}

func TestPostgresStoreMoveTransactionFailurePaths(t *testing.T) {
	t.Run("second update fails", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("failed to create sqlmock: %v", err)
		}
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, position FROM rails WHERE id = $1`)).
			WithArgs(2).
			WillReturnRows(sqlmock.NewRows([]string{"id", "position"}).AddRow(2, 2))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, position FROM rails WHERE position < $1 ORDER BY position DESC, id DESC LIMIT 1`)).
			WithArgs(2).
			WillReturnRows(sqlmock.NewRows([]string{"id", "position"}).AddRow(1, 1))
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE rails SET position = $1, updated_at = NOW() WHERE id = $2`)).
			WithArgs(1, 2).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE rails SET position = $1, updated_at = NOW() WHERE id = $2`)).
			WithArgs(2, 1).
			WillReturnError(errors.New("swap failed"))
		mock.ExpectRollback()

		store := NewPostgresStore(db)
		if err := store.MoveUp(2); err == nil {
			t.Fatalf("expected move-up error")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet SQL expectations: %v", err)
		}
	})

	t.Run("commit fails", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("failed to create sqlmock: %v", err)
		}
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, position FROM rails WHERE id = $1`)).
			WithArgs(2).
			WillReturnRows(sqlmock.NewRows([]string{"id", "position"}).AddRow(2, 2))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, position FROM rails WHERE position < $1 ORDER BY position DESC, id DESC LIMIT 1`)).
			WithArgs(2).
			WillReturnRows(sqlmock.NewRows([]string{"id", "position"}).AddRow(1, 1))
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE rails SET position = $1, updated_at = NOW() WHERE id = $2`)).
			WithArgs(1, 2).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE rails SET position = $1, updated_at = NOW() WHERE id = $2`)).
			WithArgs(2, 1).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

		store := NewPostgresStore(db)
		if err := store.MoveUp(2); err == nil {
			t.Fatalf("expected commit error")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet SQL expectations: %v", err)
		}
	})
}

func TestPostgresStoreAddItemRowsAffectedError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO rail_items (rail_id, item_id) VALUES ($1, $2) ON CONFLICT (rail_id, item_id) DO NOTHING`)).
		WithArgs(1, 10).
		WillReturnResult(sqlmock.NewErrorResult(errors.New("rows-affected failed")))

	store := NewPostgresStore(db)
	_, err = store.AddItem(1, 10)
	if err == nil {
		t.Fatalf("expected rows-affected error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}
