package clicked

import (
	"context"
	"database/sql"
	"errors"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) CreateClicked(input CreateInput) (Enquiry, error) {
	query := `INSERT INTO enquiries (item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status) VALUES ($1, $2, $3, $4, $5, $6, 'clicked') RETURNING id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, buyer_name, buyer_phone, buyer_note, converted_by, converted_at, created_at`
	row := s.db.QueryRowContext(context.Background(), query, input.ItemID, string(input.ItemType), input.ItemTitle, input.SourcePage, input.SourceRailID, input.SourceRail)
	return scanEnquiry(row)
}

func (s *PostgresStore) ListByStatus(status Status) ([]Enquiry, error) {
	query := `SELECT id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, buyer_name, buyer_phone, buyer_note, converted_by, converted_at, created_at FROM enquiries WHERE status = $1 ORDER BY created_at DESC, id DESC`
	rows, err := s.db.QueryContext(context.Background(), query, string(status))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Enquiry, 0)
	for rows.Next() {
		item, scanErr := scanEnquiry(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *PostgresStore) ConvertToInterested(id int, input ConvertInput) (Enquiry, bool, error) {
	query := `UPDATE enquiries SET status = 'interested', buyer_name = $1, buyer_phone = $2, buyer_note = $3, converted_by = $4, converted_at = NOW() WHERE id = $5 AND status = 'clicked' RETURNING id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, buyer_name, buyer_phone, buyer_note, converted_by, converted_at, created_at`
	row := s.db.QueryRowContext(context.Background(), query, input.BuyerName, input.BuyerPhone, input.BuyerNote, input.ConvertedBy, id)
	updated, err := scanEnquiry(row)
	if err == nil {
		return updated, false, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return Enquiry{}, false, err
	}

	current, getErr := s.getByID(id)
	if getErr != nil {
		return Enquiry{}, false, getErr
	}
	if current.Status == StatusInterested {
		return current, true, nil
	}
	return Enquiry{}, false, ErrNotFound
}

func (s *PostgresStore) getByID(id int) (Enquiry, error) {
	query := `SELECT id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, buyer_name, buyer_phone, buyer_note, converted_by, converted_at, created_at FROM enquiries WHERE id = $1`
	row := s.db.QueryRowContext(context.Background(), query, id)
	return scanEnquiry(row)
}

func scanEnquiry(scanner interface{ Scan(dest ...any) error }) (Enquiry, error) {
	var item Enquiry
	var itemType string
	var status string
	var convertedBy sql.NullString
	var convertedAt sql.NullTime
	if err := scanner.Scan(
		&item.ID,
		&item.ItemID,
		&itemType,
		&item.ItemTitle,
		&item.SourcePage,
		&item.SourceRailID,
		&item.SourceRail,
		&status,
		&item.BuyerName,
		&item.BuyerPhone,
		&item.BuyerNote,
		&convertedBy,
		&convertedAt,
		&item.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Enquiry{}, ErrNotFound
		}
		return Enquiry{}, err
	}
	item.ItemType = ItemType(itemType)
	item.Status = Status(status)
	if convertedBy.Valid {
		item.ConvertedBy = convertedBy.String
	}
	if convertedAt.Valid {
		item.ConvertedAt = &convertedAt.Time
	}
	return item, nil
}
