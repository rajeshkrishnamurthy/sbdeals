package books

import (
	"context"
	"database/sql"
	"errors"
)

// PostgresStore persists books in PostgreSQL.
type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) List() ([]ListItem, error) {
	rows, err := s.db.QueryContext(context.Background(), `SELECT id, supplier_id, title, author, category, mrp, my_price, in_stock, COALESCE(OCTET_LENGTH(cover_image), 0) > 0 AS has_cover, is_published, published_at, unpublished_at FROM books ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ListItem, 0)
	for rows.Next() {
		var item ListItem
		var publishedAt sql.NullTime
		var unpublishedAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.SupplierID, &item.Title, &item.Author, &item.Category, &item.MRP, &item.MyPrice, &item.InStock, &item.HasCover, &item.IsPublished, &publishedAt, &unpublishedAt); err != nil {
			return nil, err
		}
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

func (s *PostgresStore) Create(input CreateInput) (Book, error) {
	query := `INSERT INTO books (title, cover_image, cover_mime_type, supplier_id, is_box_set, category, format, condition, mrp, my_price, bundle_price, author, notes, out_of_stock_on_interested) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14) RETURNING id, title, supplier_id, cover_mime_type, is_box_set, category, format, condition, mrp, my_price, bundle_price, author, notes, in_stock, out_of_stock_on_interested, is_published, published_at, unpublished_at`
	row := s.db.QueryRowContext(context.Background(), query,
		input.Title,
		input.Cover.Data,
		input.Cover.MimeType,
		input.SupplierID,
		input.IsBoxSet,
		input.Category,
		input.Format,
		input.Condition,
		input.MRP,
		input.MyPrice,
		input.BundlePrice,
		input.Author,
		input.Notes,
		input.OutOfStockOnInterested,
	)

	return scanBook(row)
}

func (s *PostgresStore) Get(id int) (Book, error) {
	query := `SELECT id, title, supplier_id, cover_mime_type, is_box_set, category, format, condition, mrp, my_price, bundle_price, author, notes, in_stock, out_of_stock_on_interested, is_published, published_at, unpublished_at FROM books WHERE id = $1`
	row := s.db.QueryRowContext(context.Background(), query, id)
	return scanBook(row)
}

func (s *PostgresStore) GetCover(id int) (Cover, error) {
	query := `SELECT cover_image, cover_mime_type FROM books WHERE id = $1`
	row := s.db.QueryRowContext(context.Background(), query, id)

	var cover Cover
	if err := row.Scan(&cover.Data, &cover.MimeType); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Cover{}, ErrNotFound
		}
		return Cover{}, err
	}
	if len(cover.Data) == 0 {
		return Cover{}, ErrNotFound
	}
	return cover, nil
}

func (s *PostgresStore) Update(id int, input UpdateInput) (Book, error) {
	if input.Cover != nil {
		query := `UPDATE books SET title = $1, cover_image = $2, cover_mime_type = $3, supplier_id = $4, is_box_set = $5, category = $6, format = $7, condition = $8, mrp = $9, my_price = $10, bundle_price = $11, author = $12, notes = $13, in_stock = $14, is_published = CASE WHEN $14 THEN is_published ELSE FALSE END, unpublished_at = CASE WHEN $14 THEN unpublished_at ELSE CASE WHEN is_published THEN NOW() ELSE unpublished_at END END, unpublished_reason = CASE WHEN $14 THEN unpublished_reason ELSE CASE WHEN is_published THEN 'out_of_stock' ELSE unpublished_reason END END, out_of_stock_on_interested = $15, updated_at = NOW() WHERE id = $16 RETURNING id, title, supplier_id, cover_mime_type, is_box_set, category, format, condition, mrp, my_price, bundle_price, author, notes, in_stock, out_of_stock_on_interested, is_published, published_at, unpublished_at`
		row := s.db.QueryRowContext(context.Background(), query,
			input.Title,
			input.Cover.Data,
			input.Cover.MimeType,
			input.SupplierID,
			input.IsBoxSet,
			input.Category,
			input.Format,
			input.Condition,
			input.MRP,
			input.MyPrice,
			input.BundlePrice,
			input.Author,
			input.Notes,
			input.InStock,
			input.OutOfStockOnInterested,
			id,
		)
		return scanBook(row)
	}

	query := `UPDATE books SET title = $1, supplier_id = $2, is_box_set = $3, category = $4, format = $5, condition = $6, mrp = $7, my_price = $8, bundle_price = $9, author = $10, notes = $11, in_stock = $12, is_published = CASE WHEN $12 THEN is_published ELSE FALSE END, unpublished_at = CASE WHEN $12 THEN unpublished_at ELSE CASE WHEN is_published THEN NOW() ELSE unpublished_at END END, unpublished_reason = CASE WHEN $12 THEN unpublished_reason ELSE CASE WHEN is_published THEN 'out_of_stock' ELSE unpublished_reason END END, out_of_stock_on_interested = $13, updated_at = NOW() WHERE id = $14 RETURNING id, title, supplier_id, cover_mime_type, is_box_set, category, format, condition, mrp, my_price, bundle_price, author, notes, in_stock, out_of_stock_on_interested, is_published, published_at, unpublished_at`
	row := s.db.QueryRowContext(context.Background(), query,
		input.Title,
		input.SupplierID,
		input.IsBoxSet,
		input.Category,
		input.Format,
		input.Condition,
		input.MRP,
		input.MyPrice,
		input.BundlePrice,
		input.Author,
		input.Notes,
		input.InStock,
		input.OutOfStockOnInterested,
		id,
	)
	return scanBook(row)
}

func (s *PostgresStore) SetInStock(id int, inStock bool) (Book, error) {
	query := `UPDATE books SET in_stock = $1, is_published = CASE WHEN $1 THEN is_published ELSE FALSE END, unpublished_at = CASE WHEN $1 THEN unpublished_at ELSE CASE WHEN is_published THEN NOW() ELSE unpublished_at END END, unpublished_reason = CASE WHEN $1 THEN unpublished_reason ELSE CASE WHEN is_published THEN 'out_of_stock' ELSE unpublished_reason END END, updated_at = NOW() WHERE id = $2 RETURNING id, title, supplier_id, cover_mime_type, is_box_set, category, format, condition, mrp, my_price, bundle_price, author, notes, in_stock, out_of_stock_on_interested, is_published, published_at, unpublished_at`
	row := s.db.QueryRowContext(context.Background(), query, inStock, id)
	return scanBook(row)
}

func (s *PostgresStore) Publish(id int) (Book, error) {
	query := `UPDATE books SET is_published = TRUE, published_at = NOW(), unpublished_reason = '', updated_at = NOW() WHERE id = $1 AND in_stock = TRUE RETURNING id, title, supplier_id, cover_mime_type, is_box_set, category, format, condition, mrp, my_price, bundle_price, author, notes, in_stock, out_of_stock_on_interested, is_published, published_at, unpublished_at`
	row := s.db.QueryRowContext(context.Background(), query, id)
	book, err := scanBook(row)
	if !errors.Is(err, ErrNotFound) {
		return book, err
	}

	current, getErr := s.Get(id)
	if errors.Is(getErr, ErrNotFound) {
		return Book{}, ErrNotFound
	}
	if getErr != nil {
		return Book{}, getErr
	}
	if !current.InStock {
		return Book{}, ErrCannotPublishOutOfStock
	}
	return Book{}, err
}

func (s *PostgresStore) Unpublish(id int) (Book, error) {
	query := `UPDATE books SET is_published = FALSE, unpublished_at = NOW(), unpublished_reason = '', updated_at = NOW() WHERE id = $1 RETURNING id, title, supplier_id, cover_mime_type, is_box_set, category, format, condition, mrp, my_price, bundle_price, author, notes, in_stock, out_of_stock_on_interested, is_published, published_at, unpublished_at`
	row := s.db.QueryRowContext(context.Background(), query, id)
	return scanBook(row)
}

func scanBook(scanner interface{ Scan(dest ...any) error }) (Book, error) {
	var book Book
	var bundlePrice sql.NullFloat64
	var publishedAt sql.NullTime
	var unpublishedAt sql.NullTime
	if err := scanner.Scan(
		&book.ID,
		&book.Title,
		&book.SupplierID,
		&book.CoverMimeType,
		&book.IsBoxSet,
		&book.Category,
		&book.Format,
		&book.Condition,
		&book.MRP,
		&book.MyPrice,
		&bundlePrice,
		&book.Author,
		&book.Notes,
		&book.InStock,
		&book.OutOfStockOnInterested,
		&book.IsPublished,
		&publishedAt,
		&unpublishedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Book{}, ErrNotFound
		}
		return Book{}, err
	}
	if bundlePrice.Valid {
		book.BundlePrice = &bundlePrice.Float64
	}
	if publishedAt.Valid {
		book.PublishedAt = &publishedAt.Time
	}
	if unpublishedAt.Valid {
		book.UnpublishedAt = &unpublishedAt.Time
	}
	return book, nil
}
