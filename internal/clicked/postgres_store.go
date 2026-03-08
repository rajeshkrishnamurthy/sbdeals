package clicked

import (
	"context"
	"database/sql"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) Create(input CreateInput) (Event, error) {
	query := `INSERT INTO clicked_events (item_id, item_type, item_title, source_page, source_rail_id, source_rail_title) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, created_at`
	row := s.db.QueryRowContext(context.Background(), query, input.ItemID, string(input.ItemType), input.ItemTitle, input.SourcePage, input.SourceRailID, input.SourceRail)

	var event Event
	var itemType string
	if err := row.Scan(&event.ID, &event.ItemID, &itemType, &event.ItemTitle, &event.SourcePage, &event.SourceRailID, &event.SourceRail, &event.CreatedAt); err != nil {
		return Event{}, err
	}
	event.ItemType = ItemType(itemType)
	return event, nil
}
