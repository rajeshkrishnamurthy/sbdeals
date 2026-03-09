package customers

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

func (s *PostgresStore) List(filter ListFilter) ([]ListItem, error) {
	query := `SELECT c.id, c.name, c.mobile, COALESCE(ci.name, ''), COALESCE(ac.name, '')
FROM customers c
LEFT JOIN cities ci ON ci.id = c.city_id
LEFT JOIN apartment_complexes ac ON ac.id = c.apartment_complex_id
WHERE ($1 = '' OR c.name ILIKE '%' || $1 || '%' OR c.mobile ILIKE '%' || $1 || '%')
  AND ($2::BIGINT IS NULL OR c.city_id = $2)
ORDER BY c.id ASC`
	var cityArg any
	if filter.CityID != nil {
		cityArg = *filter.CityID
	}
	rows, err := s.db.QueryContext(context.Background(), query, strings.TrimSpace(filter.Search), cityArg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ListItem, 0)
	for rows.Next() {
		var item ListItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Mobile, &item.CityName, &item.ApartmentName); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PostgresStore) Get(id int) (Customer, error) {
	query := `SELECT c.id, c.name, c.mobile, c.address, c.city_id, COALESCE(ci.name, ''), c.apartment_complex_id, COALESCE(ac.name, ''), c.notes, c.created_at, c.updated_at
FROM customers c
LEFT JOIN cities ci ON ci.id = c.city_id
LEFT JOIN apartment_complexes ac ON ac.id = c.apartment_complex_id
WHERE c.id = $1`
	row := s.db.QueryRowContext(context.Background(), query, id)
	return scanCustomer(row)
}

func (s *PostgresStore) Create(input CreateInput) (Customer, error) {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Customer{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	normalizedMobile := NormalizeMobile(input.Mobile)
	if existingID, found, err := findCustomerByNormalizedMobile(ctx, tx, normalizedMobile); err != nil {
		return Customer{}, err
	} else if found {
		return Customer{}, &DuplicateMobileError{CustomerID: existingID}
	}

	cityID, err := resolveCity(ctx, tx, input.CityName)
	if err != nil {
		return Customer{}, err
	}
	apartmentID, err := resolveApartment(ctx, tx, cityID, input.ApartmentName)
	if err != nil {
		return Customer{}, err
	}

	query := `INSERT INTO customers(name, mobile, normalized_mobile, address, city_id, apartment_complex_id, notes)
VALUES($1, $2, $3, $4, $5, $6, $7)
RETURNING id`
	var id int
	if err := tx.QueryRowContext(ctx, query,
		strings.TrimSpace(input.Name),
		normalizedMobile,
		normalizedMobile,
		nullString(input.Address),
		nullInt(cityID),
		nullInt(apartmentID),
		nullString(input.Notes),
	).Scan(&id); err != nil {
		return Customer{}, err
	}

	if err := tx.Commit(); err != nil {
		return Customer{}, err
	}
	return s.Get(id)
}

func (s *PostgresStore) Update(id int, input UpdateInput) (Customer, error) {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Customer{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := getCustomerIDForUpdate(ctx, tx, id); err != nil {
		return Customer{}, err
	}

	cityID, err := resolveCity(ctx, tx, input.CityName)
	if err != nil {
		return Customer{}, err
	}
	apartmentID, err := resolveApartment(ctx, tx, cityID, input.ApartmentName)
	if err != nil {
		return Customer{}, err
	}

	query := `UPDATE customers
SET name = $1, address = $2, city_id = $3, apartment_complex_id = $4, notes = $5, updated_at = NOW()
WHERE id = $6`
	if _, err := tx.ExecContext(ctx, query,
		strings.TrimSpace(input.Name),
		nullString(input.Address),
		nullInt(cityID),
		nullInt(apartmentID),
		nullString(input.Notes),
		id,
	); err != nil {
		return Customer{}, err
	}

	if err := tx.Commit(); err != nil {
		return Customer{}, err
	}
	return s.Get(id)
}

func (s *PostgresStore) ListCities() ([]City, error) {
	query := `SELECT id, name FROM cities ORDER BY LOWER(name) ASC, id ASC`
	rows, err := s.db.QueryContext(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]City, 0)
	for rows.Next() {
		var city City
		if err := rows.Scan(&city.ID, &city.Name); err != nil {
			return nil, err
		}
		items = append(items, city)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *PostgresStore) ListApartmentComplexesByCityID(cityID int) ([]ApartmentComplex, error) {
	query := `SELECT id, city_id, name FROM apartment_complexes WHERE city_id = $1 ORDER BY LOWER(name) ASC, id ASC`
	rows, err := s.db.QueryContext(context.Background(), query, cityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ApartmentComplex, 0)
	for rows.Next() {
		var apartment ApartmentComplex
		if err := rows.Scan(&apartment.ID, &apartment.CityID, &apartment.Name); err != nil {
			return nil, err
		}
		items = append(items, apartment)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func getCustomerIDForUpdate(ctx context.Context, tx *sql.Tx, id int) (int, error) {
	var existing int
	if err := tx.QueryRowContext(ctx, `SELECT id FROM customers WHERE id = $1 FOR UPDATE`, id).Scan(&existing); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	return existing, nil
}

func findCustomerByNormalizedMobile(ctx context.Context, tx *sql.Tx, normalizedMobile string) (int, bool, error) {
	var id int
	err := tx.QueryRowContext(ctx, `SELECT id FROM customers WHERE normalized_mobile = $1`, normalizedMobile).Scan(&id)
	if err == nil {
		return id, true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	return 0, false, err
}

func resolveCity(ctx context.Context, tx *sql.Tx, cityName *string) (*int, error) {
	name := normalizeOptionalText(cityName)
	if name == nil {
		return nil, nil
	}
	normalized := NormalizeMasterName(*name)

	var existingID int
	err := tx.QueryRowContext(ctx, `SELECT id FROM cities WHERE normalized_name = $1`, normalized).Scan(&existingID)
	if err == nil {
		return &existingID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	var createdID int
	insert := `INSERT INTO cities(name, normalized_name) VALUES ($1, $2)
ON CONFLICT (normalized_name) DO UPDATE SET name = EXCLUDED.name, updated_at = NOW()
RETURNING id`
	if err := tx.QueryRowContext(ctx, insert, *name, normalized).Scan(&createdID); err != nil {
		return nil, err
	}
	return &createdID, nil
}

func resolveApartment(ctx context.Context, tx *sql.Tx, cityID *int, apartmentName *string) (*int, error) {
	name := normalizeOptionalText(apartmentName)
	if name == nil {
		return nil, nil
	}
	if cityID == nil {
		return nil, nil
	}
	normalized := NormalizeMasterName(*name)

	var existingID int
	err := tx.QueryRowContext(ctx, `SELECT id FROM apartment_complexes WHERE city_id = $1 AND normalized_name = $2`, *cityID, normalized).Scan(&existingID)
	if err == nil {
		return &existingID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	var createdID int
	insert := `INSERT INTO apartment_complexes(city_id, name, normalized_name) VALUES ($1, $2, $3)
ON CONFLICT (city_id, normalized_name) DO UPDATE SET name = EXCLUDED.name, updated_at = NOW()
RETURNING id`
	if err := tx.QueryRowContext(ctx, insert, *cityID, *name, normalized).Scan(&createdID); err != nil {
		return nil, err
	}
	return &createdID, nil
}

func scanCustomer(scanner interface{ Scan(dest ...any) error }) (Customer, error) {
	var customer Customer
	var address sql.NullString
	var cityID sql.NullInt64
	var apartmentID sql.NullInt64
	var notes sql.NullString
	if err := scanner.Scan(
		&customer.ID,
		&customer.Name,
		&customer.Mobile,
		&address,
		&cityID,
		&customer.CityName,
		&apartmentID,
		&customer.ApartmentName,
		&notes,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Customer{}, ErrNotFound
		}
		return Customer{}, err
	}
	if address.Valid {
		value := strings.TrimSpace(address.String)
		customer.Address = &value
	}
	if cityID.Valid {
		value := int(cityID.Int64)
		customer.CityID = &value
	}
	if apartmentID.Valid {
		value := int(apartmentID.Int64)
		customer.ApartmentComplexID = &value
	}
	if notes.Valid {
		value := strings.TrimSpace(notes.String)
		customer.Notes = &value
	}
	return customer, nil
}

func normalizeOptionalText(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func nullString(value *string) any {
	normalized := normalizeOptionalText(value)
	if normalized == nil {
		return nil
	}
	return *normalized
}

func nullInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}
