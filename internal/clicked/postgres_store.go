package clicked

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

const enquiryScanColumns = "id, item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, customer_id, buyer_note, order_amount, last_modified_by, l_m_at, created_at"

func (s *PostgresStore) CreateClicked(input CreateInput) (Enquiry, error) {
	query := `INSERT INTO enquiries (item_id, item_type, item_title, source_page, source_rail_id, source_rail_title, status, customer_id) VALUES ($1, $2, $3, $4, $5, $6, 'clicked', 1) RETURNING ` + enquiryScanColumns
	row := s.db.QueryRowContext(context.Background(), query, input.ItemID, string(input.ItemType), input.ItemTitle, input.SourcePage, input.SourceRailID, input.SourceRail)
	return scanEnquiry(row)
}

func (s *PostgresStore) ListByStatus(status Status) ([]Enquiry, error) {
	query := `SELECT ` + enquiryScanColumns + ` FROM enquiries WHERE status = $1 ORDER BY created_at DESC, id DESC`
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

	query := `UPDATE enquiries SET status = 'interested', customer_id = $1, buyer_note = $2, last_modified_by = $3, l_m_at = NOW() WHERE id = $4 AND status = 'clicked' RETURNING ` + enquiryScanColumns
	row := tx.QueryRowContext(ctx, query, input.CustomerID, nullableTrimmed(input.Note), input.ModifiedBy, id)
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

func (s *PostgresStore) ConvertToOrdered(id int, input OrderInput) (Enquiry, bool, error) {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Enquiry{}, false, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	current, err := getByIDForUpdate(ctx, tx, id)
	if err != nil {
		return Enquiry{}, false, err
	}
	if current.Status == StatusOrdered {
		if err := tx.Commit(); err != nil {
			return Enquiry{}, false, err
		}
		return current, true, nil
	}
	if current.Status != StatusInterested || current.CustomerID <= 0 {
		return Enquiry{}, false, ErrInvalidTransition
	}
	if err := ensureCustomerAddress(ctx, tx, current.CustomerID, input.Address); err != nil {
		return Enquiry{}, false, err
	}

	query := `UPDATE enquiries SET status = 'ordered', order_amount = $1, buyer_note = $2, last_modified_by = $3, l_m_at = NOW() WHERE id = $4 AND status = 'interested' RETURNING ` + enquiryScanColumns
	row := tx.QueryRowContext(ctx, query, input.OrderAmount, nullableTrimmed(input.Note), input.ModifiedBy, id)
	updated, err := scanEnquiry(row)
	if err != nil {
		return Enquiry{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return Enquiry{}, false, err
	}
	return updated, false, nil
}

func (s *PostgresStore) Get(id int) (Enquiry, error) {
	return getByID(context.Background(), s.db, id)
}

func getByID(ctx context.Context, db queryRowContextExecutor, id int) (Enquiry, error) {
	query := `SELECT ` + enquiryScanColumns + ` FROM enquiries WHERE id = $1`
	row := db.QueryRowContext(ctx, query, id)
	return scanEnquiry(row)
}

func getByIDForUpdate(ctx context.Context, tx *sql.Tx, id int) (Enquiry, error) {
	query := `SELECT ` + enquiryScanColumns + ` FROM enquiries WHERE id = $1 FOR UPDATE`
	row := tx.QueryRowContext(ctx, query, id)
	return scanEnquiry(row)
}

func ensureCustomerAddress(ctx context.Context, tx *sql.Tx, customerID int, providedAddress string) error {
	var currentAddress sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT address FROM customers WHERE id = $1 FOR UPDATE`, customerID).Scan(&currentAddress); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInvalidTransition
		}
		return err
	}

	if currentAddress.Valid && strings.TrimSpace(currentAddress.String) != "" {
		return nil
	}

	address := strings.TrimSpace(providedAddress)
	if address == "" {
		return ErrAddressRequired
	}
	_, err := tx.ExecContext(ctx, `UPDATE customers SET address = $1, updated_at = NOW() WHERE id = $2`, address, customerID)
	return err
}

func nullableTrimmed(raw string) any {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	return trimmed
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
	if err := syncDerivedBundleStockByBook(ctx, tx, itemID); err != nil {
		return err
	}
	return unpublishRailsWithNoPublishedItems(ctx, tx)
}

func applyBundleInterestedSideEffects(ctx context.Context, tx *sql.Tx, itemID int) error {
	enabled, err := resolveBundleOutOfStockOnInterested(ctx, tx, itemID)
	if err != nil || !enabled {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE bundles SET in_stock = FALSE, is_published = FALSE, unpublished_at = CASE WHEN is_published THEN NOW() ELSE unpublished_at END, unpublished_reason = 'out_of_stock', updated_at = NOW() WHERE id = $1 AND (in_stock = TRUE OR is_published = TRUE)`, itemID); err != nil {
		return err
	}
	return unpublishRailsWithNoPublishedItems(ctx, tx)
}

func syncDerivedBundleStockByBook(ctx context.Context, tx *sql.Tx, bookID int) error {
	query := `WITH affected AS (SELECT DISTINCT bb.bundle_id FROM bundle_books bb WHERE bb.book_id = $1), derived AS (SELECT a.bundle_id, COALESCE(bool_and(bk.in_stock), TRUE) AS all_in_stock FROM affected a LEFT JOIN bundle_books bb ON bb.bundle_id = a.bundle_id LEFT JOIN books bk ON bk.id = bb.book_id GROUP BY a.bundle_id) UPDATE bundles b SET in_stock = derived.all_in_stock, is_published = CASE WHEN derived.all_in_stock THEN b.is_published ELSE FALSE END, unpublished_at = CASE WHEN NOT derived.all_in_stock AND b.is_published THEN NOW() ELSE b.unpublished_at END, unpublished_reason = CASE WHEN NOT derived.all_in_stock AND b.is_published THEN 'out_of_stock' ELSE b.unpublished_reason END, updated_at = NOW() FROM derived WHERE b.id = derived.bundle_id`
	_, err := tx.ExecContext(ctx, query, bookID)
	return err
}

func unpublishRailsWithNoPublishedItems(ctx context.Context, tx *sql.Tx) error {
	bookRailsQuery := `UPDATE rails r SET is_published = FALSE, unpublished_at = NOW(), updated_at = NOW() WHERE r.is_published = TRUE AND r.rail_type = 'BOOK' AND NOT EXISTS (SELECT 1 FROM rail_items ri JOIN books b ON b.id = ri.item_id WHERE ri.rail_id = r.id AND b.is_published = TRUE)`
	if _, err := tx.ExecContext(ctx, bookRailsQuery); err != nil {
		return err
	}

	bundleRailsQuery := `UPDATE rails r SET is_published = FALSE, unpublished_at = NOW(), updated_at = NOW() WHERE r.is_published = TRUE AND r.rail_type = 'BUNDLE' AND NOT EXISTS (SELECT 1 FROM rail_items ri JOIN bundles b ON b.id = ri.item_id WHERE ri.rail_id = r.id AND b.is_published = TRUE)`
	_, err := tx.ExecContext(ctx, bundleRailsQuery)
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
	var customerID sql.NullInt64
	var note sql.NullString
	var orderAmount sql.NullInt64
	var modifiedBy sql.NullString
	var modifiedAt sql.NullTime
	if err := scanner.Scan(
		&item.ID,
		&item.ItemID,
		&itemType,
		&item.ItemTitle,
		&item.SourcePage,
		&item.SourceRailID,
		&item.SourceRail,
		&status,
		&customerID,
		&note,
		&orderAmount,
		&modifiedBy,
		&modifiedAt,
		&item.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Enquiry{}, ErrNotFound
		}
		return Enquiry{}, err
	}
	item.ItemType = ItemType(itemType)
	item.Status = Status(status)
	if customerID.Valid {
		item.CustomerID = int(customerID.Int64)
	}
	if note.Valid {
		item.Note = note.String
	}
	if orderAmount.Valid {
		value := int(orderAmount.Int64)
		item.OrderAmount = &value
	}
	if modifiedBy.Valid {
		item.LastModifiedBy = modifiedBy.String
	}
	if modifiedAt.Valid {
		item.LastModifiedAt = &modifiedAt.Time
	}
	return item, nil
}
