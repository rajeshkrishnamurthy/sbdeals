package bundles

import (
	"sort"
	"sync"
	"time"
)

// MemoryStore keeps bundles in memory for tests and local fallback use.
type MemoryStore struct {
	mu           sync.RWMutex
	nextID       int
	supplierName map[int]string
	pickerByID   map[int]PickerBook
	bundles      []Bundle
}

func NewMemoryStore(supplierName map[int]string, pickerBooks []PickerBook) *MemoryStore {
	clonedNames := map[int]string{}
	for id, name := range supplierName {
		clonedNames[id] = name
	}

	pickerByID := map[int]PickerBook{}
	for _, book := range pickerBooks {
		pickerByID[book.BookID] = clonePickerBook(book)
	}

	return &MemoryStore{
		nextID:       1,
		supplierName: clonedNames,
		pickerByID:   pickerByID,
		bundles:      make([]Bundle, 0),
	}
}

func (s *MemoryStore) List() ([]ListItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]ListItem, 0, len(s.bundles))
	for _, bundle := range s.bundles {
		items = append(items, ListItem{
			ID:                bundle.ID,
			Name:              bundle.Name,
			SupplierName:      bundle.SupplierName,
			Category:          bundle.Category,
			AllowedConditions: cloneStringSlice(bundle.AllowedConditions),
			BookCount:         len(bundle.Books),
			BundlePrice:       bundle.BundlePrice,
			IsPublished:       bundle.IsPublished,
			PublishedAt:       cloneTimePointer(bundle.PublishedAt),
			UnpublishedAt:     cloneTimePointer(bundle.UnpublishedAt),
		})
	}
	return items, nil
}

func (s *MemoryStore) Create(input CreateInput) (Bundle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	bundle := s.bundleFromInput(s.nextID, input)
	s.nextID++
	s.bundles = append(s.bundles, bundle)
	return cloneBundle(bundle), nil
}

func (s *MemoryStore) Get(id int) (Bundle, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	index := s.indexByID(id)
	if index < 0 {
		return Bundle{}, ErrNotFound
	}
	return cloneBundle(s.bundles[index]), nil
}

func (s *MemoryStore) Update(id int, input UpdateInput) (Bundle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.indexByID(id)
	if index < 0 {
		return Bundle{}, ErrNotFound
	}

	updated := s.bundleFromInput(id, CreateInput{
		Name:              input.Name,
		SupplierID:        input.SupplierID,
		Category:          input.Category,
		AllowedConditions: input.AllowedConditions,
		BookIDs:           input.BookIDs,
		BundlePrice:       input.BundlePrice,
		Notes:             input.Notes,
	})
	updated.IsPublished = s.bundles[index].IsPublished
	updated.PublishedAt = cloneTimePointer(s.bundles[index].PublishedAt)
	updated.UnpublishedAt = cloneTimePointer(s.bundles[index].UnpublishedAt)
	s.bundles[index] = updated
	return cloneBundle(updated), nil
}

func (s *MemoryStore) Publish(id int) (Bundle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.indexByID(id)
	if index < 0 {
		return Bundle{}, ErrNotFound
	}
	outOfStock := outOfStockTitlesFromBooks(s.bundles[index].Books)
	if len(outOfStock) > 0 {
		return Bundle{}, &ErrCannotPublishWithOutOfStockBooks{BookTitles: outOfStock}
	}
	now := time.Now().UTC()
	s.bundles[index].IsPublished = true
	s.bundles[index].PublishedAt = &now
	return cloneBundle(s.bundles[index]), nil
}

func (s *MemoryStore) Unpublish(id int) (Bundle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.indexByID(id)
	if index < 0 {
		return Bundle{}, ErrNotFound
	}
	now := time.Now().UTC()
	s.bundles[index].IsPublished = false
	s.bundles[index].UnpublishedAt = &now
	return cloneBundle(s.bundles[index]), nil
}

func (s *MemoryStore) ListBooksForPicker() ([]PickerBook, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	books := make([]PickerBook, 0, len(s.pickerByID))
	for _, book := range s.pickerByID {
		books = append(books, clonePickerBook(book))
	}
	sort.Slice(books, func(i, j int) bool {
		return books[i].BookID < books[j].BookID
	})
	return books, nil
}

func (s *MemoryStore) bundleFromInput(id int, input CreateInput) Bundle {
	books := make([]BundleBook, 0, len(input.BookIDs))
	for _, bookID := range input.BookIDs {
		if candidate, ok := s.pickerByID[bookID]; ok {
			books = append(books, BundleBook{
				BookID:      candidate.BookID,
				Title:       candidate.Title,
				Author:      candidate.Author,
				SupplierID:  candidate.SupplierID,
				Category:    candidate.Category,
				Condition:   candidate.Condition,
				MRP:         candidate.MRP,
				MyPrice:     candidate.MyPrice,
				BundlePrice: cloneFloatPointer(candidate.BundlePrice),
				InStock:     candidate.InStock,
			})
		}
	}
	now := time.Now().UTC()

	return Bundle{
		ID:                id,
		Name:              input.Name,
		SupplierID:        input.SupplierID,
		SupplierName:      s.supplierName[input.SupplierID],
		Category:          input.Category,
		AllowedConditions: cloneStringSlice(input.AllowedConditions),
		BundlePrice:       input.BundlePrice,
		Notes:             input.Notes,
		BookIDs:           cloneIntSlice(input.BookIDs),
		Books:             books,
		IsPublished:       false,
		PublishedAt:       nil,
		UnpublishedAt:     &now,
	}
}

func (s *MemoryStore) indexByID(id int) int {
	for i, bundle := range s.bundles {
		if bundle.ID == id {
			return i
		}
	}
	return -1
}

func cloneBundle(in Bundle) Bundle {
	out := in
	out.AllowedConditions = cloneStringSlice(in.AllowedConditions)
	out.BookIDs = cloneIntSlice(in.BookIDs)
	out.Books = make([]BundleBook, len(in.Books))
	for i, book := range in.Books {
		copyBook := book
		copyBook.BundlePrice = cloneFloatPointer(book.BundlePrice)
		out.Books[i] = copyBook
	}
	out.PublishedAt = cloneTimePointer(in.PublishedAt)
	out.UnpublishedAt = cloneTimePointer(in.UnpublishedAt)
	return out
}

func clonePickerBook(in PickerBook) PickerBook {
	out := in
	out.BundlePrice = cloneFloatPointer(in.BundlePrice)
	return out
}

func cloneStringSlice(values []string) []string {
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func cloneIntSlice(values []int) []int {
	out := make([]int, len(values))
	copy(out, values)
	return out
}

func cloneFloatPointer(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func outOfStockTitlesFromBooks(books []BundleBook) []string {
	titles := make([]string, 0)
	for _, book := range books {
		if !book.InStock {
			titles = append(titles, book.Title)
		}
	}
	sort.Strings(titles)
	return titles
}
