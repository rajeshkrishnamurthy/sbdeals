package clicked

import (
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestPostgresStoreCreate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	query := `INSERT INTO clicked_events (item_id, item_type, item_title, source_page, source_rail_id, source_rail_title) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, created_at`
	now := time.Date(2026, time.March, 8, 7, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(7, "BOOK", "Book One", "catalog", 2, "Fresh Books").
		WillReturnRows(sqlmock.NewRows([]string{"id", "item_id", "item_type", "item_title", "source_page", "source_rail_id", "source_rail_title", "created_at"}).
			AddRow(1, 7, "BOOK", "Book One", "catalog", 2, "Fresh Books", now))

	store := NewPostgresStore(db)
	event, err := store.Create(CreateInput{
		ItemID:       7,
		ItemType:     ItemTypeBook,
		ItemTitle:    "Book One",
		SourcePage:   "catalog",
		SourceRailID: 2,
		SourceRail:   "Fresh Books",
	})
	if err != nil {
		t.Fatalf("create returned error: %v", err)
	}
	if event.ItemType != ItemTypeBook || event.CreatedAt != now {
		t.Fatalf("unexpected event: %+v", event)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}
