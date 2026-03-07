package rails

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

func (s *PostgresStore) List() ([]ListItem, error) {
	query := `SELECT r.id, r.title, r.rail_type, r.position, r.is_published, r.published_at, r.unpublished_at, COUNT(ri.item_id) AS item_count FROM rails r LEFT JOIN rail_items ri ON ri.rail_id = r.id GROUP BY r.id, r.title, r.rail_type, r.position, r.is_published, r.published_at, r.unpublished_at ORDER BY r.position ASC, r.id ASC`
	rows, err := s.db.QueryContext(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ListItem, 0)
	for rows.Next() {
		var item ListItem
		var railType string
		var publishedAt sql.NullTime
		var unpublishedAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.Title, &railType, &item.Position, &item.IsPublished, &publishedAt, &unpublishedAt, &item.ItemCount); err != nil {
			return nil, err
		}
		item.Type = RailType(railType)
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

func (s *PostgresStore) Create(input CreateInput) (Rail, error) {
	title := strings.TrimSpace(input.Title)
	if exists, err := s.titleExists(title, 0); err != nil {
		return Rail{}, err
	} else if exists {
		return Rail{}, ErrDuplicateTitle
	}

	query := `INSERT INTO rails (title, rail_type, position) VALUES ($1, $2, COALESCE((SELECT MAX(position) + 1 FROM rails), 1)) RETURNING id, title, rail_type, position, is_published, published_at, unpublished_at`
	row := s.db.QueryRowContext(context.Background(), query, title, string(input.Type))
	created, err := scanRail(row)
	if err != nil {
		return Rail{}, err
	}
	created.ItemIDs = []int{}
	return created, nil
}

func (s *PostgresStore) Get(id int) (Rail, error) {
	query := `SELECT id, title, rail_type, position, is_published, published_at, unpublished_at FROM rails WHERE id = $1`
	row := s.db.QueryRowContext(context.Background(), query, id)
	rail, err := scanRail(row)
	if err != nil {
		return Rail{}, err
	}
	itemIDs, err := s.fetchItemIDs(id)
	if err != nil {
		return Rail{}, err
	}
	rail.ItemIDs = itemIDs
	return rail, nil
}

func (s *PostgresStore) Update(id int, input UpdateInput) (Rail, error) {
	title := strings.TrimSpace(input.Title)
	if exists, err := s.titleExists(title, id); err != nil {
		return Rail{}, err
	} else if exists {
		return Rail{}, ErrDuplicateTitle
	}

	query := `UPDATE rails SET title = $1, updated_at = NOW() WHERE id = $2 RETURNING id, title, rail_type, position, is_published, published_at, unpublished_at`
	row := s.db.QueryRowContext(context.Background(), query, title, id)
	updated, err := scanRail(row)
	if err != nil {
		return Rail{}, err
	}
	itemIDs, err := s.fetchItemIDs(id)
	if err != nil {
		return Rail{}, err
	}
	updated.ItemIDs = itemIDs
	return updated, nil
}

func (s *PostgresStore) AddItem(id int, itemID int) (Rail, error) {
	query := `INSERT INTO rail_items (rail_id, item_id) VALUES ($1, $2) ON CONFLICT (rail_id, item_id) DO NOTHING`
	result, err := s.db.ExecContext(context.Background(), query, id, itemID)
	if err != nil {
		return Rail{}, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return Rail{}, err
	}
	if rows == 0 {
		if _, err := s.Get(id); errors.Is(err, ErrNotFound) {
			return Rail{}, ErrNotFound
		}
		return Rail{}, ErrDuplicateItem
	}
	return s.Get(id)
}

func (s *PostgresStore) RemoveItem(id int, itemID int) (Rail, error) {
	query := `DELETE FROM rail_items WHERE rail_id = $1 AND item_id = $2`
	if _, err := s.db.ExecContext(context.Background(), query, id, itemID); err != nil {
		return Rail{}, err
	}
	return s.Get(id)
}

func (s *PostgresStore) Publish(id int) (Rail, error) {
	query := `UPDATE rails SET is_published = TRUE, published_at = NOW(), updated_at = NOW() WHERE id = $1 RETURNING id, title, rail_type, position, is_published, published_at, unpublished_at`
	row := s.db.QueryRowContext(context.Background(), query, id)
	updated, err := scanRail(row)
	if err != nil {
		return Rail{}, err
	}
	itemIDs, err := s.fetchItemIDs(id)
	if err != nil {
		return Rail{}, err
	}
	updated.ItemIDs = itemIDs
	return updated, nil
}

func (s *PostgresStore) Unpublish(id int) (Rail, error) {
	query := `UPDATE rails SET is_published = FALSE, unpublished_at = NOW(), updated_at = NOW() WHERE id = $1 RETURNING id, title, rail_type, position, is_published, published_at, unpublished_at`
	row := s.db.QueryRowContext(context.Background(), query, id)
	updated, err := scanRail(row)
	if err != nil {
		return Rail{}, err
	}
	itemIDs, err := s.fetchItemIDs(id)
	if err != nil {
		return Rail{}, err
	}
	updated.ItemIDs = itemIDs
	return updated, nil
}

func (s *PostgresStore) MoveUp(id int) error {
	return s.move(id, true)
}

func (s *PostgresStore) MoveDown(id int) error {
	return s.move(id, false)
}

func (s *PostgresStore) move(id int, up bool) error {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var currentID int
	var currentPosition int
	if err := tx.QueryRowContext(context.Background(), `SELECT id, position FROM rails WHERE id = $1`, id).Scan(&currentID, &currentPosition); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	neighborQuery := `SELECT id, position FROM rails WHERE position < $1 ORDER BY position DESC, id DESC LIMIT 1`
	if !up {
		neighborQuery = `SELECT id, position FROM rails WHERE position > $1 ORDER BY position ASC, id ASC LIMIT 1`
	}

	var neighborID int
	var neighborPosition int
	if err := tx.QueryRowContext(context.Background(), neighborQuery, currentPosition).Scan(&neighborID, &neighborPosition); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return tx.Commit()
		}
		return err
	}

	if _, err := tx.ExecContext(context.Background(), `UPDATE rails SET position = $1, updated_at = NOW() WHERE id = $2`, neighborPosition, currentID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(context.Background(), `UPDATE rails SET position = $1, updated_at = NOW() WHERE id = $2`, currentPosition, neighborID); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *PostgresStore) fetchItemIDs(id int) ([]int, error) {
	query := `SELECT item_id FROM rail_items WHERE rail_id = $1 ORDER BY created_at ASC, item_id ASC`
	rows, err := s.db.QueryContext(context.Background(), query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]int, 0)
	for rows.Next() {
		var itemID int
		if err := rows.Scan(&itemID); err != nil {
			return nil, err
		}
		ids = append(ids, itemID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

func (s *PostgresStore) titleExists(title string, excludeID int) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM rails WHERE LOWER(TRIM(title)) = LOWER(TRIM($1)) AND id <> $2)`
	var exists bool
	if err := s.db.QueryRowContext(context.Background(), query, title, excludeID).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func scanRail(scanner interface{ Scan(dest ...any) error }) (Rail, error) {
	var rail Rail
	var railType string
	var publishedAt sql.NullTime
	var unpublishedAt sql.NullTime
	if err := scanner.Scan(&rail.ID, &rail.Title, &railType, &rail.Position, &rail.IsPublished, &publishedAt, &unpublishedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Rail{}, ErrNotFound
		}
		return Rail{}, err
	}
	rail.Type = RailType(railType)
	if publishedAt.Valid {
		rail.PublishedAt = &publishedAt.Time
	}
	if unpublishedAt.Valid {
		rail.UnpublishedAt = &unpublishedAt.Time
	}
	return rail, nil
}
