package clicked

import "testing"

func TestMemoryStoreCreate(t *testing.T) {
	store := NewMemoryStore()

	created, err := store.Create(CreateInput{
		ItemID:       7,
		ItemType:     ItemTypeBook,
		ItemTitle:    "Book One",
		SourcePage:   "catalog",
		SourceRailID: 2,
		SourceRail:   "Fresh Books",
	})
	if err != nil {
		t.Fatalf("create returned error: %v", err)
	}
	if created.ID != 1 {
		t.Fatalf("expected ID=1, got %d", created.ID)
	}
	if created.ItemType != ItemTypeBook || created.ItemTitle != "Book One" {
		t.Fatalf("unexpected created event: %+v", created)
	}
	if created.CreatedAt.IsZero() {
		t.Fatalf("expected CreatedAt to be set")
	}
}
