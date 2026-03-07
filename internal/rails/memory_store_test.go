package rails

import (
	"errors"
	"testing"
)

func TestMemoryStoreCreateDefaultsUnpublishedAndUniqueTitle(t *testing.T) {
	store := NewMemoryStore()
	created, err := store.Create(CreateInput{Title: "Top Picks", Type: RailTypeBook})
	if err != nil {
		t.Fatalf("create returned error: %v", err)
	}
	if created.IsPublished {
		t.Fatalf("expected default unpublished rail")
	}
	if created.UnpublishedAt == nil {
		t.Fatalf("expected unpublishedAt initialized")
	}
	if created.PublishedAt != nil {
		t.Fatalf("expected publishedAt nil on create")
	}
	if created.Type != RailTypeBook {
		t.Fatalf("expected type BOOK, got %s", created.Type)
	}

	_, err = store.Create(CreateInput{Title: "top picks", Type: RailTypeBundle})
	if !errors.Is(err, ErrDuplicateTitle) {
		t.Fatalf("expected ErrDuplicateTitle, got %v", err)
	}
}

func TestMemoryStoreAddRemoveItemAndDuplicate(t *testing.T) {
	store := NewMemoryStore()
	created, err := store.Create(CreateInput{Title: "Top Picks", Type: RailTypeBook})
	if err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	updated, err := store.AddItem(created.ID, 10)
	if err != nil {
		t.Fatalf("add item returned error: %v", err)
	}
	if len(updated.ItemIDs) != 1 || updated.ItemIDs[0] != 10 {
		t.Fatalf("unexpected item IDs after add: %+v", updated.ItemIDs)
	}

	if _, err := store.AddItem(created.ID, 10); !errors.Is(err, ErrDuplicateItem) {
		t.Fatalf("expected ErrDuplicateItem, got %v", err)
	}

	updated, err = store.RemoveItem(created.ID, 10)
	if err != nil {
		t.Fatalf("remove item returned error: %v", err)
	}
	if len(updated.ItemIDs) != 0 {
		t.Fatalf("expected no items after remove, got %+v", updated.ItemIDs)
	}
}

func TestMemoryStorePublishAndUnpublish(t *testing.T) {
	store := NewMemoryStore()
	created, err := store.Create(CreateInput{Title: "A", Type: RailTypeBook})
	if err != nil {
		t.Fatalf("create rail: %v", err)
	}

	pub, err := store.Publish(created.ID)
	if err != nil {
		t.Fatalf("publish returned error: %v", err)
	}
	if !pub.IsPublished || pub.PublishedAt == nil {
		t.Fatalf("expected published state")
	}
	unpub, err := store.Unpublish(created.ID)
	if err != nil {
		t.Fatalf("unpublish returned error: %v", err)
	}
	if unpub.IsPublished || unpub.UnpublishedAt == nil {
		t.Fatalf("expected unpublished state with timestamp")
	}
}

func TestMemoryStoreMoveOrdering(t *testing.T) {
	store := NewMemoryStore()
	first, err := store.Create(CreateInput{Title: "A", Type: RailTypeBook})
	if err != nil {
		t.Fatalf("create first: %v", err)
	}
	second, err := store.Create(CreateInput{Title: "B", Type: RailTypeBundle})
	if err != nil {
		t.Fatalf("create second: %v", err)
	}

	if err := store.MoveUp(second.ID); err != nil {
		t.Fatalf("move up returned error: %v", err)
	}
	assertOrderIDs(t, store, second.ID, first.ID)

	if err := store.MoveDown(second.ID); err != nil {
		t.Fatalf("move down returned error: %v", err)
	}
	assertOrderIDs(t, store, first.ID, second.ID)
}

func assertOrderIDs(t *testing.T, store *MemoryStore, expected ...int) {
	t.Helper()
	items, err := store.List()
	if err != nil {
		t.Fatalf("list returned error: %v", err)
	}
	if len(items) != len(expected) {
		t.Fatalf("expected %d rails, got %d", len(expected), len(items))
	}
	for idx, wantID := range expected {
		if items[idx].ID != wantID {
			t.Fatalf("unexpected order at index %d: got=%d want=%d", idx, items[idx].ID, wantID)
		}
	}
}

func TestMemoryStoreNotFoundPaths(t *testing.T) {
	store := NewMemoryStore()
	if _, err := store.Get(99); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for get, got %v", err)
	}
	if _, err := store.Update(99, UpdateInput{Title: "x"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for update, got %v", err)
	}
	if _, err := store.Publish(99); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for publish, got %v", err)
	}
	if err := store.MoveDown(99); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for move down, got %v", err)
	}
}
