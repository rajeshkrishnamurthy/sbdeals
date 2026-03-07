package bundles

import (
	"errors"
	"reflect"
	"testing"
)

func TestMemoryStoreCreateListAndGet(t *testing.T) {
	store := newMemoryStoreFixture()

	created, err := store.Create(CreateInput{
		Name:              "Starter Bundle",
		SupplierID:        1,
		Category:          "Fiction",
		AllowedConditions: []string{"Very good", "Good as new"},
		BookIDs:           []int{10, 11},
		BundlePrice:       499,
		Notes:             "Weekend deal",
	})
	if err != nil {
		t.Fatalf("create returned error: %v", err)
	}
	if created.ID != 1 {
		t.Fatalf("expected ID 1, got %d", created.ID)
	}
	if len(created.Books) != 2 {
		t.Fatalf("expected 2 books, got %d", len(created.Books))
	}

	items, err := store.List()
	if err != nil {
		t.Fatalf("list returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 list item, got %d", len(items))
	}
	if items[0].SupplierName != "Supplier One" || items[0].BookCount != 2 {
		t.Fatalf("unexpected list item: %+v", items[0])
	}

	fetched, err := store.Get(created.ID)
	if err != nil {
		t.Fatalf("get returned error: %v", err)
	}
	if fetched.Name != "Starter Bundle" || fetched.Category != "Fiction" {
		t.Fatalf("unexpected fetched bundle: %+v", fetched)
	}
}

func TestMemoryStoreUpdate(t *testing.T) {
	store := newMemoryStoreFixture()
	created, err := store.Create(CreateInput{
		Name:              "Starter Bundle",
		SupplierID:        1,
		Category:          "Fiction",
		AllowedConditions: []string{"Very good", "Good as new"},
		BookIDs:           []int{10, 11},
		BundlePrice:       499,
	})
	if err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	updated, err := store.Update(created.ID, UpdateInput{
		Name:              "Updated Bundle",
		SupplierID:        1,
		Category:          "Fiction",
		AllowedConditions: []string{"Used", "Very good"},
		BookIDs:           []int{10, 12},
		BundlePrice:       399,
		Notes:             "Updated notes",
	})
	if err != nil {
		t.Fatalf("update returned error: %v", err)
	}
	if updated.Name != "Updated Bundle" || updated.BundlePrice != 399 {
		t.Fatalf("unexpected updated bundle: %+v", updated)
	}
	if !reflect.DeepEqual(updated.BookIDs, []int{10, 12}) {
		t.Fatalf("unexpected updated book ids: %+v", updated.BookIDs)
	}
}

func TestMemoryStoreNotFound(t *testing.T) {
	store := NewMemoryStore(nil, nil)
	if _, err := store.Get(99); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for get, got %v", err)
	}
	if _, err := store.Update(99, UpdateInput{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for update, got %v", err)
	}
}

func TestMemoryStoreListBooksForPicker(t *testing.T) {
	store := NewMemoryStore(nil, []PickerBook{
		{BookID: 20, Title: "B"},
		{BookID: 10, Title: "A"},
	})

	books, err := store.ListBooksForPicker()
	if err != nil {
		t.Fatalf("list books for picker returned error: %v", err)
	}
	if len(books) != 2 {
		t.Fatalf("expected 2 books, got %d", len(books))
	}
	if books[0].BookID != 10 || books[1].BookID != 20 {
		t.Fatalf("expected books sorted by ID, got %+v", books)
	}
}

func newMemoryStoreFixture() *MemoryStore {
	return NewMemoryStore(
		map[int]string{1: "Supplier One"},
		[]PickerBook{
			{BookID: 10, Title: "Book A", Author: "Author A", SupplierID: 1, Category: "Fiction", Condition: "Very good", MRP: 400, MyPrice: 250},
			{BookID: 11, Title: "Book B", Author: "Author B", SupplierID: 1, Category: "Fiction", Condition: "Good as new", MRP: 500, MyPrice: 300},
			{BookID: 12, Title: "Book C", Author: "Author C", SupplierID: 1, Category: "Fiction", Condition: "Used", MRP: 350, MyPrice: 200},
		},
	)
}
