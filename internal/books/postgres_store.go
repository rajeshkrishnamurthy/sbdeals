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
	rows, err := s.db.QueryContext(context.Background(), `SELECT id, title, author, category, my_price, in_stock, OCTET_LENGTH(cover_image) > 0 AS has_cover FROM books ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ListItem, 0)
	for rows.Next() {
		var item ListItem
		if err := rows.Scan(&item.ID, &item.Title, &item.Author, &item.Category, &item.MyPrice, &item.InStock, &item.HasCover); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *PostgresStore) Create(input CreateInput) (Book, error) {
	query := `INSERT INTO books (title, cover_image, cover_mime_type, supplier_id, category, format, condition, mrp, my_price, bundle_price, author, notes) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING id, title, supplier_id, cover_mime_type, category, format, condition, mrp, my_price, bundle_price, author, notes, in_stock`
	row := s.db.QueryRowContext(context.Background(), query,
		input.Title,
		input.Cover.Data,
		input.Cover.MimeType,
		input.SupplierID,
		input.Category,
		input.Format,
		input.Condition,
		input.MRP,
		input.MyPrice,
		input.BundlePrice,
		input.Author,
		input.Notes,
	)

	return scanBook(row)
}

func (s *PostgresStore) Get(id int) (Book, error) {
	query := `SELECT id, title, supplier_id, cover_mime_type, category, format, condition, mrp, my_price, bundle_price, author, notes, in_stock FROM books WHERE id = $1`
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
	return cover, nil
}

func (s *PostgresStore) Update(id int, input UpdateInput) (Book, error) {
	if input.Cover != nil {
		query := `UPDATE books SET title = $1, cover_image = $2, cover_mime_type = $3, supplier_id = $4, category = $5, format = $6, condition = $7, mrp = $8, my_price = $9, bundle_price = $10, author = $11, notes = $12, in_stock = $13, updated_at = NOW() WHERE id = $14 RETURNING id, title, supplier_id, cover_mime_type, category, format, condition, mrp, my_price, bundle_price, author, notes, in_stock`
		row := s.db.QueryRowContext(context.Background(), query,
			input.Title,
			input.Cover.Data,
			input.Cover.MimeType,
			input.SupplierID,
			input.Category,
			input.Format,
			input.Condition,
			input.MRP,
			input.MyPrice,
			input.BundlePrice,
			input.Author,
			input.Notes,
			input.InStock,
			id,
		)
		return scanBook(row)
	}

	query := `UPDATE books SET title = $1, supplier_id = $2, category = $3, format = $4, condition = $5, mrp = $6, my_price = $7, bundle_price = $8, author = $9, notes = $10, in_stock = $11, updated_at = NOW() WHERE id = $12 RETURNING id, title, supplier_id, cover_mime_type, category, format, condition, mrp, my_price, bundle_price, author, notes, in_stock`
	row := s.db.QueryRowContext(context.Background(), query,
		input.Title,
		input.SupplierID,
		input.Category,
		input.Format,
		input.Condition,
		input.MRP,
		input.MyPrice,
		input.BundlePrice,
		input.Author,
		input.Notes,
		input.InStock,
		id,
	)
	return scanBook(row)
}

func (s *PostgresStore) SetInStock(id int, inStock bool) (Book, error) {
	query := `UPDATE books SET in_stock = $1, updated_at = NOW() WHERE id = $2 RETURNING id, title, supplier_id, cover_mime_type, category, format, condition, mrp, my_price, bundle_price, author, notes, in_stock`
	row := s.db.QueryRowContext(context.Background(), query, inStock, id)
	return scanBook(row)
}

func scanBook(scanner interface{ Scan(dest ...any) error }) (Book, error) {
	var book Book
	var bundlePrice sql.NullFloat64
	if err := scanner.Scan(
		&book.ID,
		&book.Title,
		&book.SupplierID,
		&book.CoverMimeType,
		&book.Category,
		&book.Format,
		&book.Condition,
		&book.MRP,
		&book.MyPrice,
		&bundlePrice,
		&book.Author,
		&book.Notes,
		&book.InStock,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Book{}, ErrNotFound
		}
		return Book{}, err
	}
	if bundlePrice.Valid {
		book.BundlePrice = &bundlePrice.Float64
	}
	return book, nil
}
