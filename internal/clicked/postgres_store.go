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
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Enquiry{}, false, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	query := `UPDATE enquiries SET status = 'interested', buyer_name = $1, buyer_phone = $2, buyer_note = $3, converted_by = $4, converted_at = NOW() WHERE id = $5 AND status = 'clicked' RETURNING id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, buyer_name, buyer_phone, buyer_note, converted_by, converted_at, created_at`
	row := tx.QueryRowContext(ctx, query, input.BuyerName, input.BuyerPhone, input.BuyerNote, input.ConvertedBy, id)
	updated, err := scanEnquiry(row)
	if err == nil {
		if err := applyInterestedTransitionSideEffects(ctx, tx, updated); err != nil {
			return Enquiry{}, false, err
		}
		if err := tx.Commit(); err != nil {
			return Enquiry{}, false, err
		}
		return updated, false, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return Enquiry{}, false, err
	}

	current, getErr := getByID(ctx, tx, id)
	if getErr != nil {
		return Enquiry{}, false, getErr
	}
	if current.Status == StatusInterested {
		if err := tx.Commit(); err != nil {
			return Enquiry{}, false, err
		}
		return current, true, nil
	}
	return Enquiry{}, false, ErrNotFound
}

func (s *PostgresStore) getByID(id int) (Enquiry, error) {
	return getByID(context.Background(), s.db, id)
}

func getByID(ctx context.Context, db queryRowContextExecutor, id int) (Enquiry, error) {
	query := `SELECT id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, buyer_name, buyer_phone, buyer_note, converted_by, converted_at, created_at FROM enquiries WHERE id = $1`
	row := db.QueryRowContext(ctx, query, id)
	return scanEnquiry(row)
}

func applyInterestedTransitionSideEffects(ctx context.Context, tx *sql.Tx, enquiry Enquiry) error {
	switch enquiry.ItemType {
	case ItemTypeBook:
		return applyBookInterestedSideEffects(ctx, tx, enquiry.ItemID)
	case ItemTypeBundle:
		return applyBundleInterestedSideEffects(ctx, tx, enquiry.ItemID)
	default:
		return nil
	}
}

func applyBookInterestedSideEffects(ctx context.Context, tx *sql.Tx, itemID int) error {
	enabled, err := resolveBookOutOfStockOnInterested(ctx, tx, itemID)
	if err != nil || !enabled {
		return err
	}

	if _, err := tx.ExecContext(ctx, `UPDATE books SET in_stock = FALSE, is_published = FALSE, unpublished_at = CASE WHEN is_published THEN NOW() ELSE unpublished_at END, unpublished_reason = 'out_of_stock', updated_at = NOW() WHERE id = $1 AND (in_stock = TRUE OR is_published = TRUE)`, itemID); err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `UPDATE bundles b SET is_published = FALSE, unpublished_at = NOW(), unpublished_reason = 'out_of_stock', updated_at = NOW() WHERE b.is_published = TRUE AND EXISTS (SELECT 1 FROM bundle_books bb WHERE bb.bundle_id = b.id AND bb.book_id = $1)`, itemID)
	return err
}

func applyBundleInterestedSideEffects(ctx context.Context, tx *sql.Tx, itemID int) error {
	enabled, err := resolveBundleOutOfStockOnInterested(ctx, tx, itemID)
	if err != nil || !enabled {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE bundles SET in_stock = FALSE, is_published = FALSE, unpublished_at = CASE WHEN is_published THEN NOW() ELSE unpublished_at END, unpublished_reason = 'out_of_stock', updated_at = NOW() WHERE id = $1 AND (in_stock = TRUE OR is_published = TRUE)`, itemID)
	return err
}

func resolveBookOutOfStockOnInterested(ctx context.Context, tx *sql.Tx, itemID int) (bool, error) {
	var enabled bool
	if err := tx.QueryRowContext(ctx, `SELECT out_of_stock_on_interested FROM books WHERE id = $1`, itemID).Scan(&enabled); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return enabled, nil
}

func resolveBundleOutOfStockOnInterested(ctx context.Context, tx *sql.Tx, itemID int) (bool, error) {
	var enabled bool
	if err := tx.QueryRowContext(ctx, `SELECT out_of_stock_on_interested FROM bundles WHERE id = $1`, itemID).Scan(&enabled); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return enabled, nil
}

type queryRowContextExecutor interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
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
