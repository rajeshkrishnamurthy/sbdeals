package suppliers

import (
	"database/sql"
	"errors"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestPostgresStoreList(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "name", "whatsapp", "location", "notes"}).
		AddRow(1, "A1 Books", "+91-9876543210", "Bengaluru", "note")

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, name, whatsapp, location, notes FROM suppliers ORDER BY id ASC`)).
		WillReturnRows(rows)

	store := NewPostgresStore(db)
	items, err := store.List()
	if err != nil {
		t.Fatalf("list returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Name != "A1 Books" {
		t.Fatalf("unexpected name: %q", items[0].Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresStoreCreate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	input := Input{Name: "A1 Books", WhatsApp: "+91-9999999999", Location: "Chennai", Notes: "new"}
	query := `INSERT INTO suppliers (name, whatsapp, location, notes) VALUES ($1, $2, $3, $4) RETURNING id, name, whatsapp, location, notes`
	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(input.Name, input.WhatsApp, input.Location, input.Notes).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "whatsapp", "location", "notes"}).
			AddRow(4, input.Name, input.WhatsApp, input.Location, input.Notes))

	store := NewPostgresStore(db)
	created, err := store.Create(input)
	if err != nil {
		t.Fatalf("create returned error: %v", err)
	}
	if created.ID != 4 {
		t.Fatalf("expected id 4, got %d", created.ID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresStoreGetNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, name, whatsapp, location, notes FROM suppliers WHERE id = $1`)).
		WithArgs(42).
		WillReturnError(sql.ErrNoRows)

	store := NewPostgresStore(db)
	_, err = store.Get(42)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresStoreUpdate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	input := Input{Name: "Updated", WhatsApp: "+91-9000000000", Location: "Hyderabad", Notes: "changed"}
	query := `UPDATE suppliers SET name = $1, whatsapp = $2, location = $3, notes = $4, updated_at = NOW() WHERE id = $5 RETURNING id, name, whatsapp, location, notes`
	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(input.Name, input.WhatsApp, input.Location, input.Notes, 3).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "whatsapp", "location", "notes"}).
			AddRow(3, input.Name, input.WhatsApp, input.Location, input.Notes))

	store := NewPostgresStore(db)
	updated, err := store.Update(3, input)
	if err != nil {
		t.Fatalf("update returned error: %v", err)
	}
	if updated.Name != "Updated" {
		t.Fatalf("unexpected updated name: %q", updated.Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresStoreUpdateNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	input := Input{Name: "Updated", WhatsApp: "+91-9000000000", Location: "Hyderabad", Notes: "changed"}
	query := `UPDATE suppliers SET name = $1, whatsapp = $2, location = $3, notes = $4, updated_at = NOW() WHERE id = $5 RETURNING id, name, whatsapp, location, notes`
	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(input.Name, input.WhatsApp, input.Location, input.Notes, 404).
		WillReturnError(sql.ErrNoRows)

	store := NewPostgresStore(db)
	_, err = store.Update(404, input)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}
