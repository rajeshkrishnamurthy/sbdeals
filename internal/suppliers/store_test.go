package suppliers

import (
	"errors"
	"testing"
)

func TestMemoryStoreCreateListGetUpdate(t *testing.T) {
	store := NewMemoryStore()

	created, err := store.Create(Input{
		Name:     "A1 Books",
		WhatsApp: "+91-9876543210",
		Location: "Bengaluru",
		Notes:    "Prefers evening pickup",
	})
	if err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	if created.ID != 1 {
		t.Fatalf("expected ID 1, got %d", created.ID)
	}

	items, err := store.List()
	if err != nil {
		t.Fatalf("list returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 supplier, got %d", len(items))
	}
	if items[0].Name != "A1 Books" {
		t.Fatalf("expected name A1 Books, got %q", items[0].Name)
	}

	fetched, err := store.Get(created.ID)
	if err != nil {
		t.Fatalf("expected supplier to exist: %v", err)
	}
	if fetched.WhatsApp != "+91-9876543210" {
		t.Fatalf("unexpected whatsapp: %q", fetched.WhatsApp)
	}

	updated, err := store.Update(created.ID, Input{
		Name:     "A1 Books Updated",
		WhatsApp: "+91-9000000000",
		Location: "Hyderabad",
		Notes:    "Updated note",
	})
	if err != nil {
		t.Fatalf("update returned error: %v", err)
	}
	if updated.Name != "A1 Books Updated" {
		t.Fatalf("unexpected updated name: %q", updated.Name)
	}

	afterUpdate, err := store.Get(created.ID)
	if err != nil {
		t.Fatalf("expected supplier to exist after update: %v", err)
	}
	if afterUpdate.Location != "Hyderabad" {
		t.Fatalf("expected updated location Hyderabad, got %q", afterUpdate.Location)
	}
}

func TestMemoryStoreUpdateNotFound(t *testing.T) {
	store := NewMemoryStore()
	_, err := store.Update(99, Input{Name: "Nope"})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
