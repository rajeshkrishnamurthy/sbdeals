package books

import (
	"errors"
	"testing"
)

func TestMemoryStoreCreateDefaultsInStockAndList(t *testing.T) {
	store := NewMemoryStore()
	bundlePrice := 150.0

	created, err := store.Create(CreateInput{
		Title:       "The Hobbit",
		Cover:       Cover{Data: []byte("img-bytes"), MimeType: "image/jpeg"},
		SupplierID:  11,
		IsBoxSet:    true,
		Category:    "Fiction",
		Format:      "Paperback",
		Condition:   "Very good",
		MRP:         499,
		MyPrice:     299,
		BundlePrice: &bundlePrice,
		Author:      "J.R.R. Tolkien",
		Notes:       "Classic fantasy",
	})
	if err != nil {
		t.Fatalf("create returned error: %v", err)
	}
	if created.ID != 1 {
		t.Fatalf("expected ID 1, got %d", created.ID)
	}
	if !created.InStock {
		t.Fatalf("expected created book to default InStock=true")
	}
	if created.IsPublished {
		t.Fatalf("expected created book to default IsPublished=false")
	}
	if created.PublishedAt != nil {
		t.Fatalf("expected created book PublishedAt=nil")
	}
	if !created.IsBoxSet {
		t.Fatalf("expected created book IsBoxSet=true")
	}
	if created.UnpublishedAt == nil {
		t.Fatalf("expected created book UnpublishedAt to be initialized")
	}

	items, err := store.List()
	if err != nil {
		t.Fatalf("list returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Title != "The Hobbit" {
		t.Fatalf("unexpected title in list: %q", items[0].Title)
	}
	if !items[0].HasCover {
		t.Fatalf("expected list item to report cover presence")
	}
}

func TestMemoryStorePublishUnpublishAndPublishRule(t *testing.T) {
	store := NewMemoryStore()
	created, err := store.Create(CreateInput{
		Title:      "Dune",
		Cover:      Cover{Data: []byte("cover-1"), MimeType: "image/png"},
		SupplierID: 2,
		Category:   "Fiction",
		Format:     "Hardcover",
		Condition:  "Good as new",
		MRP:        899,
		MyPrice:    499,
	})
	if err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	published, err := store.Publish(created.ID)
	if err != nil {
		t.Fatalf("publish returned error: %v", err)
	}
	if !published.IsPublished || published.PublishedAt == nil {
		t.Fatalf("expected published state, got %+v", published)
	}

	unpublished, err := store.Unpublish(created.ID)
	if err != nil {
		t.Fatalf("unpublish returned error: %v", err)
	}
	if unpublished.IsPublished {
		t.Fatalf("expected unpublished state")
	}
	if unpublished.UnpublishedAt == nil {
		t.Fatalf("expected unpublished timestamp")
	}

	if _, err := store.SetInStock(created.ID, false); err != nil {
		t.Fatalf("set in stock false: %v", err)
	}
	_, err = store.Publish(created.ID)
	if !errors.Is(err, ErrCannotPublishOutOfStock) {
		t.Fatalf("expected ErrCannotPublishOutOfStock, got %v", err)
	}
}

func TestMemoryStoreSetInStockFalseAutoUnpublishes(t *testing.T) {
	store := NewMemoryStore()
	created, err := store.Create(CreateInput{
		Title:      "Dune",
		Cover:      Cover{Data: []byte("cover-1"), MimeType: "image/png"},
		SupplierID: 2,
		Category:   "Fiction",
		Format:     "Hardcover",
		Condition:  "Good as new",
		MRP:        899,
		MyPrice:    499,
	})
	if err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	if _, err := store.Publish(created.ID); err != nil {
		t.Fatalf("publish returned error: %v", err)
	}

	updated, err := store.SetInStock(created.ID, false)
	if err != nil {
		t.Fatalf("set in stock returned error: %v", err)
	}
	if updated.InStock {
		t.Fatalf("expected in-stock=false")
	}
	if updated.IsPublished {
		t.Fatalf("expected published=false after setting out-of-stock")
	}
	if updated.UnpublishedAt == nil {
		t.Fatalf("expected unpublished timestamp after setting out-of-stock")
	}
}

func TestMemoryStoreGetCoverUpdateAndStockToggle(t *testing.T) {
	store := NewMemoryStore()
	created, err := store.Create(CreateInput{
		Title:      "Dune",
		Cover:      Cover{Data: []byte("cover-1"), MimeType: "image/png"},
		SupplierID: 2,
		Category:   "Fiction",
		Format:     "Hardcover",
		Condition:  "Good as new",
		MRP:        899,
		MyPrice:    499,
	})
	if err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	cover, err := store.GetCover(created.ID)
	if err != nil {
		t.Fatalf("get cover returned error: %v", err)
	}
	if string(cover.Data) != "cover-1" {
		t.Fatalf("unexpected cover bytes")
	}

	replacedCover := &Cover{Data: []byte("cover-2"), MimeType: "image/webp"}
	updated, err := store.Update(created.ID, UpdateInput{
		Title:      "Dune Messiah",
		Cover:      replacedCover,
		SupplierID: 3,
		IsBoxSet:   true,
		Category:   "Fiction",
		Format:     "Paperback",
		Condition:  "Used",
		MRP:        599,
		MyPrice:    399,
		Author:     "Frank Herbert",
		Notes:      "Second book",
		InStock:    false,
	})
	if err != nil {
		t.Fatalf("update returned error: %v", err)
	}
	if updated.Title != "Dune Messiah" || updated.SupplierID != 3 {
		t.Fatalf("unexpected updated payload: %+v", updated)
	}
	if !updated.IsBoxSet {
		t.Fatalf("expected updated IsBoxSet=true")
	}
	if updated.InStock {
		t.Fatalf("expected updated in-stock false")
	}

	cover, err = store.GetCover(created.ID)
	if err != nil {
		t.Fatalf("get cover after update returned error: %v", err)
	}
	if string(cover.Data) != "cover-2" || cover.MimeType != "image/webp" {
		t.Fatalf("cover replacement did not persist")
	}

	toggled, err := store.SetInStock(created.ID, true)
	if err != nil {
		t.Fatalf("set in stock returned error: %v", err)
	}
	if !toggled.InStock {
		t.Fatalf("expected in-stock true after toggle")
	}
}

func TestMemoryStoreNotFoundCases(t *testing.T) {
	store := NewMemoryStore()
	if _, err := store.Get(99); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for get, got %v", err)
	}
	if _, err := store.GetCover(99); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for get cover, got %v", err)
	}
	if _, err := store.Update(99, UpdateInput{Title: "x"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for update, got %v", err)
	}
	if _, err := store.SetInStock(99, true); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for set in stock, got %v", err)
	}
}
