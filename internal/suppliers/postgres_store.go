package suppliers

import (
	"context"
	"database/sql"
	"errors"
)

// PostgresStore persists suppliers in PostgreSQL.
type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) List() ([]Supplier, error) {
	rows, err := s.db.QueryContext(context.Background(), `SELECT id, name, whatsapp, location, notes FROM suppliers ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Supplier, 0)
	for rows.Next() {
		var item Supplier
		if err := rows.Scan(&item.ID, &item.Name, &item.WhatsApp, &item.Location, &item.Notes); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (s *PostgresStore) Create(input Input) (Supplier, error) {
	var item Supplier
	query := `INSERT INTO suppliers (name, whatsapp, location, notes) VALUES ($1, $2, $3, $4) RETURNING id, name, whatsapp, location, notes`
	err := s.db.QueryRowContext(context.Background(), query, input.Name, input.WhatsApp, input.Location, input.Notes).
		Scan(&item.ID, &item.Name, &item.WhatsApp, &item.Location, &item.Notes)
	if err != nil {
		return Supplier{}, err
	}
	return item, nil
}

func (s *PostgresStore) Get(id int) (Supplier, error) {
	var item Supplier
	err := s.db.QueryRowContext(context.Background(), `SELECT id, name, whatsapp, location, notes FROM suppliers WHERE id = $1`, id).
		Scan(&item.ID, &item.Name, &item.WhatsApp, &item.Location, &item.Notes)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Supplier{}, ErrNotFound
		}
		return Supplier{}, err
	}
	return item, nil
}

func (s *PostgresStore) Update(id int, input Input) (Supplier, error) {
	var item Supplier
	query := `UPDATE suppliers SET name = $1, whatsapp = $2, location = $3, notes = $4, updated_at = NOW() WHERE id = $5 RETURNING id, name, whatsapp, location, notes`
	err := s.db.QueryRowContext(context.Background(), query, input.Name, input.WhatsApp, input.Location, input.Notes, id).
		Scan(&item.ID, &item.Name, &item.WhatsApp, &item.Location, &item.Notes)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Supplier{}, ErrNotFound
		}
		return Supplier{}, err
	}
	return item, nil
}
