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
		Image:             Image{Data: []byte("bundle-image"), MimeType: "image/png"},
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
	if created.IsPublished {
		t.Fatalf("expected created bundle to default IsPublished=false")
	}
	if created.PublishedAt != nil {
		t.Fatalf("expected created bundle PublishedAt=nil")
	}
	if created.UnpublishedAt == nil {
		t.Fatalf("expected created bundle UnpublishedAt initialized")
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
	if !items[0].HasImage {
		t.Fatalf("expected list item to indicate image presence")
	}
	if items[0].BundleMRP != 900 {
		t.Fatalf("expected bundle mrp sum 900, got %v", items[0].BundleMRP)
	}

	fetched, err := store.Get(created.ID)
	if err != nil {
		t.Fatalf("get returned error: %v", err)
	}
	if fetched.Name != "Starter Bundle" || fetched.Category != "Fiction" {
		t.Fatalf("unexpected fetched bundle: %+v", fetched)
	}
}

func TestMemoryStorePublishUnpublishAndPublishRule(t *testing.T) {
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

	storeWithOutOfStock := NewMemoryStore(
		map[int]string{1: "Supplier One"},
		[]PickerBook{
			{BookID: 10, Title: "Book A", Author: "Author A", SupplierID: 1, Category: "Fiction", Condition: "Very good", MRP: 400, MyPrice: 250, InStock: true},
			{BookID: 11, Title: "Book B", Author: "Author B", SupplierID: 1, Category: "Fiction", Condition: "Good as new", MRP: 500, MyPrice: 300, InStock: false},
		},
	)
	createdOutOfStock, err := storeWithOutOfStock.Create(CreateInput{
		Name:              "Out Bundle",
		SupplierID:        1,
		Category:          "Fiction",
		AllowedConditions: []string{"Very good", "Good as new"},
		BookIDs:           []int{10, 11},
		BundlePrice:       399,
	})
	if err != nil {
		t.Fatalf("create with out-of-stock returned error: %v", err)
	}
	_, err = storeWithOutOfStock.Publish(createdOutOfStock.ID)
	var outOfStockErr *ErrCannotPublishWithOutOfStockBooks
	if !errors.As(err, &outOfStockErr) {
		t.Fatalf("expected ErrCannotPublishWithOutOfStockBooks, got %v", err)
	}
	if len(outOfStockErr.BookTitles) != 1 || outOfStockErr.BookTitles[0] != "Book B" {
		t.Fatalf("unexpected out-of-stock titles: %+v", outOfStockErr.BookTitles)
	}

	storeWithOutOfStock.bundles[0].Books[1].InStock = true
	storeWithOutOfStock.bundles[0].InStock = false
	_, err = storeWithOutOfStock.Publish(createdOutOfStock.ID)
	if !errors.Is(err, ErrCannotPublishOutOfStock) {
		t.Fatalf("expected ErrCannotPublishOutOfStock, got %v", err)
	}
}

func TestMemoryStoreSyncDerivedStockByBook(t *testing.T) {
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
	if !created.InStock {
		t.Fatalf("expected created bundle to be in-stock")
	}
	if _, err := store.Publish(created.ID); err != nil {
		t.Fatalf("publish returned error: %v", err)
	}

	if err := store.SyncDerivedStockByBook(10, false); err != nil {
		t.Fatalf("sync returned error: %v", err)
	}
	afterOut, err := store.Get(created.ID)
	if err != nil {
		t.Fatalf("get after out-of-stock: %v", err)
	}
	if afterOut.InStock {
		t.Fatalf("expected bundle in-stock=false after related book out-of-stock")
	}
	if afterOut.IsPublished {
		t.Fatalf("expected bundle unpublished after becoming out-of-stock")
	}

	if err := store.SyncDerivedStockByBook(10, true); err != nil {
		t.Fatalf("sync returned error: %v", err)
	}
	afterIn, err := store.Get(created.ID)
	if err != nil {
		t.Fatalf("get after in-stock restore: %v", err)
	}
	if !afterIn.InStock {
		t.Fatalf("expected bundle in-stock=true after all books restored")
	}
	if afterIn.IsPublished {
		t.Fatalf("expected no auto-publish when stock is restored")
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
		Image:             Image{Data: []byte("initial-image"), MimeType: "image/png"},
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
		Image:             &Image{Data: []byte("updated-image"), MimeType: "image/jpeg"},
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
	if updated.ImageMimeType != "image/jpeg" {
		t.Fatalf("expected image mime update, got %q", updated.ImageMimeType)
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
	if _, err := store.GetImage(99); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for image, got %v", err)
	}
}

func TestMemoryStoreGetImage(t *testing.T) {
	store := newMemoryStoreFixture()
	created, err := store.Create(CreateInput{
		Name:              "Starter Bundle",
		SupplierID:        1,
		Category:          "Fiction",
		AllowedConditions: []string{"Very good", "Good as new"},
		BookIDs:           []int{10, 11},
		BundlePrice:       499,
		Image:             Image{Data: []byte("bundle-image"), MimeType: "image/png"},
	})
	if err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	image, err := store.GetImage(created.ID)
	if err != nil {
		t.Fatalf("get image returned error: %v", err)
	}
	if string(image.Data) != "bundle-image" || image.MimeType != "image/png" {
		t.Fatalf("unexpected image payload: %+v", image)
	}
}

func TestOutOfStockTitlesFromBooksSorted(t *testing.T) {
	books := []BundleBook{
		{Title: "Gamma", InStock: false},
		{Title: "Alpha", InStock: false},
		{Title: "Beta", InStock: true},
	}
	titles := outOfStockTitlesFromBooks(books)
	expected := []string{"Alpha", "Gamma"}
	if !reflect.DeepEqual(titles, expected) {
		t.Fatalf("expected sorted out-of-stock titles %v, got %v", expected, titles)
	}
}

func TestCloneFloatPointer(t *testing.T) {
	if cloneFloatPointer(nil) != nil {
		t.Fatalf("expected nil clone for nil pointer")
	}
	value := 12.5
	cloned := cloneFloatPointer(&value)
	if cloned == nil || *cloned != value {
		t.Fatalf("expected cloned pointer with same value")
	}
	if cloned == &value {
		t.Fatalf("expected different pointer address")
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
			{BookID: 10, Title: "Book A", Author: "Author A", SupplierID: 1, Category: "Fiction", Condition: "Very good", MRP: 400, MyPrice: 250, InStock: true},
			{BookID: 11, Title: "Book B", Author: "Author B", SupplierID: 1, Category: "Fiction", Condition: "Good as new", MRP: 500, MyPrice: 300, InStock: true},
			{BookID: 12, Title: "Book C", Author: "Author C", SupplierID: 1, Category: "Fiction", Condition: "Used", MRP: 350, MyPrice: 200, InStock: true},
		},
	)
}
