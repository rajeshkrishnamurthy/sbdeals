package bundles

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

const conditionsDelimiter = "||"

// PostgresStore persists bundles in PostgreSQL.
type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) List() ([]ListItem, error) {
	query := `SELECT b.id, b.name, s.name AS supplier_name, b.category, array_to_string(b.allowed_conditions, '||') AS allowed_conditions, b.bundle_price, COUNT(bb.book_id) AS book_count, COALESCE(SUM(bk.mrp), 0) AS bundle_mrp, COALESCE(OCTET_LENGTH(b.bundle_image), 0) > 0 AS has_image, b.is_published, b.published_at, b.unpublished_at FROM bundles b JOIN suppliers s ON s.id = b.supplier_id LEFT JOIN bundle_books bb ON bb.bundle_id = b.id LEFT JOIN books bk ON bk.id = bb.book_id GROUP BY b.id, b.name, s.name, b.category, b.allowed_conditions, b.bundle_price, b.bundle_image, b.is_published, b.published_at, b.unpublished_at ORDER BY b.id ASC`
	rows, err := s.db.QueryContext(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ListItem, 0)
	for rows.Next() {
		var item ListItem
		var allowedConditions string
		var publishedAt sql.NullTime
		var unpublishedAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.Name, &item.SupplierName, &item.Category, &allowedConditions, &item.BundlePrice, &item.BookCount, &item.BundleMRP, &item.HasImage, &item.IsPublished, &publishedAt, &unpublishedAt); err != nil {
			return nil, err
		}
		item.AllowedConditions = splitConditions(allowedConditions)
		if publishedAt.Valid {
			item.PublishedAt = &publishedAt.Time
		}
		if unpublishedAt.Valid {
			item.UnpublishedAt = &unpublishedAt.Time
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *PostgresStore) Create(input CreateInput) (Bundle, error) {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Bundle{}, err
	}

	insertBundle := `INSERT INTO bundles (name, supplier_id, category, allowed_conditions, bundle_price, notes, bundle_image, bundle_image_mime_type, out_of_stock_on_interested) VALUES ($1, $2, $3, string_to_array($4, '||'), $5, $6, $7, $8, $9) RETURNING id`
	conditions := joinConditions(input.AllowedConditions)
	var bundleID int
	if err := tx.QueryRowContext(ctx, insertBundle, input.Name, input.SupplierID, input.Category, conditions, input.BundlePrice, input.Notes, input.Image.Data, input.Image.MimeType, input.OutOfStockOnInterested).Scan(&bundleID); err != nil {
		_ = tx.Rollback()
		return Bundle{}, err
	}

	if err := insertBundleBooks(ctx, tx, bundleID, input.BookIDs); err != nil {
		_ = tx.Rollback()
		return Bundle{}, err
	}

	if err := tx.Commit(); err != nil {
		return Bundle{}, err
	}

	return s.Get(bundleID)
}

func (s *PostgresStore) Get(id int) (Bundle, error) {
	ctx := context.Background()
	bundle, err := queryBundleByID(ctx, s.db, id)
	if err != nil {
		return Bundle{}, err
	}
	books, bookIDs, err := queryBundleBooks(ctx, s.db, id)
	if err != nil {
		return Bundle{}, err
	}
	bundle.Books = books
	bundle.BookIDs = bookIDs
	return bundle, nil
}

func (s *PostgresStore) Update(id int, input UpdateInput) (Bundle, error) {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Bundle{}, err
	}

	updateQuery := `UPDATE bundles SET name = $1, supplier_id = $2, category = $3, allowed_conditions = string_to_array($4, '||'), bundle_price = $5, notes = $6, out_of_stock_on_interested = $7, updated_at = NOW() WHERE id = $8`
	args := []any{input.Name, input.SupplierID, input.Category, joinConditions(input.AllowedConditions), input.BundlePrice, input.Notes, input.OutOfStockOnInterested, id}
	if input.Image != nil {
		updateQuery = `UPDATE bundles SET name = $1, supplier_id = $2, category = $3, allowed_conditions = string_to_array($4, '||'), bundle_price = $5, notes = $6, bundle_image = $7, bundle_image_mime_type = $8, out_of_stock_on_interested = $9, updated_at = NOW() WHERE id = $10`
		args = []any{input.Name, input.SupplierID, input.Category, joinConditions(input.AllowedConditions), input.BundlePrice, input.Notes, input.Image.Data, input.Image.MimeType, input.OutOfStockOnInterested, id}
	}
	res, err := tx.ExecContext(ctx, updateQuery, args...)
	if err != nil {
		_ = tx.Rollback()
		return Bundle{}, err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return Bundle{}, err
	}
	if rowsAffected == 0 {
		_ = tx.Rollback()
		return Bundle{}, ErrNotFound
	}

	deleteQuery := `DELETE FROM bundle_books WHERE bundle_id = $1`
	if _, err := tx.ExecContext(ctx, deleteQuery, id); err != nil {
		_ = tx.Rollback()
		return Bundle{}, err
	}

	if err := insertBundleBooks(ctx, tx, id, input.BookIDs); err != nil {
		_ = tx.Rollback()
		return Bundle{}, err
	}

	if err := tx.Commit(); err != nil {
		return Bundle{}, err
	}

	return s.Get(id)
}

func (s *PostgresStore) Publish(id int) (Bundle, error) {
	ctx := context.Background()
	query := `UPDATE bundles b SET is_published = TRUE, published_at = NOW(), unpublished_reason = '', updated_at = NOW() WHERE b.id = $1 AND b.in_stock = TRUE AND NOT EXISTS (SELECT 1 FROM bundle_books bb JOIN books bk ON bk.id = bb.book_id WHERE bb.bundle_id = b.id AND bk.in_stock = FALSE) RETURNING b.id`

	var bundleID int
	if err := s.db.QueryRowContext(ctx, query, id).Scan(&bundleID); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return Bundle{}, err
		}
		bundle, getErr := s.Get(id)
		if errors.Is(getErr, ErrNotFound) {
			return Bundle{}, ErrNotFound
		}
		if getErr != nil {
			return Bundle{}, getErr
		}
		outOfStockTitles, titlesErr := s.outOfStockBookTitles(bundle.ID)
		if titlesErr != nil {
			return Bundle{}, titlesErr
		}
		if len(outOfStockTitles) > 0 {
			return Bundle{}, &ErrCannotPublishWithOutOfStockBooks{BookTitles: outOfStockTitles}
		}
		if !bundle.InStock {
			return Bundle{}, ErrCannotPublishOutOfStock
		}
		return Bundle{}, ErrNotFound
	}

	return s.Get(bundleID)
}

func (s *PostgresStore) Unpublish(id int) (Bundle, error) {
	ctx := context.Background()
	query := `UPDATE bundles SET is_published = FALSE, unpublished_at = NOW(), unpublished_reason = '', updated_at = NOW() WHERE id = $1 RETURNING id`
	var bundleID int
	if err := s.db.QueryRowContext(ctx, query, id).Scan(&bundleID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Bundle{}, ErrNotFound
		}
		return Bundle{}, err
	}
	return s.Get(bundleID)
}

func (s *PostgresStore) ListBooksForPicker() ([]PickerBook, error) {
	query := `SELECT id, title, author, supplier_id, is_box_set, category, condition, mrp, my_price, bundle_price, in_stock FROM books ORDER BY id ASC`
	rows, err := s.db.QueryContext(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]PickerBook, 0)
	for rows.Next() {
		var item PickerBook
		var bundlePrice sql.NullFloat64
		if err := rows.Scan(&item.BookID, &item.Title, &item.Author, &item.SupplierID, &item.IsBoxSet, &item.Category, &item.Condition, &item.MRP, &item.MyPrice, &bundlePrice, &item.InStock); err != nil {
			return nil, err
		}
		if bundlePrice.Valid {
			item.BundlePrice = &bundlePrice.Float64
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *PostgresStore) GetImage(id int) (Image, error) {
	query := `SELECT bundle_image, bundle_image_mime_type FROM bundles WHERE id = $1`
	row := s.db.QueryRowContext(context.Background(), query, id)
	var image Image
	if err := row.Scan(&image.Data, &image.MimeType); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Image{}, ErrNotFound
		}
		return Image{}, err
	}
	if len(image.Data) == 0 {
		return Image{}, ErrNotFound
	}
	return image, nil
}

func queryBundleByID(ctx context.Context, db queryRowContextExecutor, id int) (Bundle, error) {
	query := `SELECT b.id, b.name, b.supplier_id, s.name AS supplier_name, b.category, array_to_string(b.allowed_conditions, '||') AS allowed_conditions, b.bundle_price, b.notes, b.in_stock, b.out_of_stock_on_interested, b.bundle_image_mime_type, b.is_published, b.published_at, b.unpublished_at FROM bundles b JOIN suppliers s ON s.id = b.supplier_id WHERE b.id = $1`
	row := db.QueryRowContext(ctx, query, id)

	var bundle Bundle
	var allowedConditions string
	var publishedAt sql.NullTime
	var unpublishedAt sql.NullTime
	if err := row.Scan(&bundle.ID, &bundle.Name, &bundle.SupplierID, &bundle.SupplierName, &bundle.Category, &allowedConditions, &bundle.BundlePrice, &bundle.Notes, &bundle.InStock, &bundle.OutOfStockOnInterested, &bundle.ImageMimeType, &bundle.IsPublished, &publishedAt, &unpublishedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Bundle{}, ErrNotFound
		}
		return Bundle{}, err
	}
	bundle.AllowedConditions = splitConditions(allowedConditions)
	if publishedAt.Valid {
		bundle.PublishedAt = &publishedAt.Time
	}
	if unpublishedAt.Valid {
		bundle.UnpublishedAt = &unpublishedAt.Time
	}
	return bundle, nil
}

func queryBundleBooks(ctx context.Context, db queryContextExecutor, bundleID int) ([]BundleBook, []int, error) {
	query := `SELECT bb.book_id, b.title, b.author, b.supplier_id, b.is_box_set, b.category, b.condition, b.mrp, b.my_price, b.bundle_price, b.in_stock FROM bundle_books bb JOIN books b ON b.id = bb.book_id WHERE bb.bundle_id = $1 ORDER BY bb.position ASC`
	rows, err := db.QueryContext(ctx, query, bundleID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	booksList := make([]BundleBook, 0)
	bookIDs := make([]int, 0)
	for rows.Next() {
		var item BundleBook
		var bundlePrice sql.NullFloat64
		if err := rows.Scan(&item.BookID, &item.Title, &item.Author, &item.SupplierID, &item.IsBoxSet, &item.Category, &item.Condition, &item.MRP, &item.MyPrice, &bundlePrice, &item.InStock); err != nil {
			return nil, nil, err
		}
		if bundlePrice.Valid {
			item.BundlePrice = &bundlePrice.Float64
		}
		booksList = append(booksList, item)
		bookIDs = append(bookIDs, item.BookID)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return booksList, bookIDs, nil
}

func insertBundleBooks(ctx context.Context, db execContextExecutor, bundleID int, bookIDs []int) error {
	query := `INSERT INTO bundle_books (bundle_id, book_id, position) VALUES ($1, $2, $3)`
	for index, bookID := range bookIDs {
		if _, err := db.ExecContext(ctx, query, bundleID, bookID, index); err != nil {
			return fmt.Errorf("insert bundle book %d: %w", bookID, err)
		}
	}
	return nil
}

func (s *PostgresStore) outOfStockBookTitles(bundleID int) ([]string, error) {
	query := `SELECT b.title FROM bundle_books bb JOIN books b ON b.id = bb.book_id WHERE bb.bundle_id = $1 AND b.in_stock = FALSE ORDER BY b.title ASC`
	rows, err := s.db.QueryContext(context.Background(), query, bundleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	titles := make([]string, 0)
	for rows.Next() {
		var title string
		if err := rows.Scan(&title); err != nil {
			return nil, err
		}
		titles = append(titles, title)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return titles, nil
}

func joinConditions(values []string) string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		cleaned = append(cleaned, trimmed)
	}
	return strings.Join(cleaned, conditionsDelimiter)
}

func splitConditions(serialized string) []string {
	serialized = strings.TrimSpace(serialized)
	if serialized == "" {
		return []string{}
	}
	parts := strings.Split(serialized, conditionsDelimiter)
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		cleaned = append(cleaned, trimmed)
	}
	return cleaned
}

type queryRowContextExecutor interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type queryContextExecutor interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type execContextExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}
