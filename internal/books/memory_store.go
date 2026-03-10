package books

import (
	"sync"
	"time"
)

type memoryRow struct {
	book  Book
	cover Cover
}

// MemoryStore keeps books in process memory for tests and local development fallback.
type MemoryStore struct {
	mu     sync.RWMutex
	nextID int
	rows   []memoryRow
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{nextID: 1}
}

func (s *MemoryStore) List() ([]ListItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]ListItem, 0, len(s.rows))
	for _, row := range s.rows {
		items = append(items, ListItem{
			ID:            row.book.ID,
			Title:         row.book.Title,
			Author:        row.book.Author,
			Category:      row.book.Category,
			MyPrice:       row.book.MyPrice,
			InStock:       row.book.InStock,
			HasCover:      len(row.cover.Data) > 0,
			IsPublished:   row.book.IsPublished,
			PublishedAt:   cloneTimePointer(row.book.PublishedAt),
			UnpublishedAt: cloneTimePointer(row.book.UnpublishedAt),
		})
	}
	return items, nil
}

func (s *MemoryStore) Create(input CreateInput) (Book, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	book := Book{
		ID:                     s.nextID,
		Title:                  input.Title,
		SupplierID:             input.SupplierID,
		CoverMimeType:          input.Cover.MimeType,
		IsBoxSet:               input.IsBoxSet,
		Category:               input.Category,
		Format:                 input.Format,
		Condition:              input.Condition,
		MRP:                    input.MRP,
		MyPrice:                input.MyPrice,
		BundlePrice:            cloneFloatPointer(input.BundlePrice),
		Author:                 input.Author,
		Notes:                  input.Notes,
		InStock:                true,
		OutOfStockOnInterested: input.OutOfStockOnInterested,
		IsPublished:            false,
		PublishedAt:            nil,
		UnpublishedAt:          &now,
	}

	s.nextID++
	s.rows = append(s.rows, memoryRow{book: book, cover: cloneCover(input.Cover)})
	return cloneBook(book), nil
}

func (s *MemoryStore) Get(id int) (Book, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	idx := s.indexByID(id)
	if idx < 0 {
		return Book{}, ErrNotFound
	}
	return cloneBook(s.rows[idx].book), nil
}

func (s *MemoryStore) GetCover(id int) (Cover, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	idx := s.indexByID(id)
	if idx < 0 {
		return Cover{}, ErrNotFound
	}
	return cloneCover(s.rows[idx].cover), nil
}

func (s *MemoryStore) Update(id int, input UpdateInput) (Book, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.indexByID(id)
	if idx < 0 {
		return Book{}, ErrNotFound
	}

	row := s.rows[idx]
	row.book.Title = input.Title
	row.book.SupplierID = input.SupplierID
	row.book.IsBoxSet = input.IsBoxSet
	row.book.Category = input.Category
	row.book.Format = input.Format
	row.book.Condition = input.Condition
	row.book.MRP = input.MRP
	row.book.MyPrice = input.MyPrice
	row.book.BundlePrice = cloneFloatPointer(input.BundlePrice)
	row.book.Author = input.Author
	row.book.Notes = input.Notes
	applyInStockStateTransition(&row.book, input.InStock)
	row.book.OutOfStockOnInterested = input.OutOfStockOnInterested

	if input.Cover != nil {
		row.cover = cloneCover(*input.Cover)
		row.book.CoverMimeType = input.Cover.MimeType
	}

	s.rows[idx] = row
	return cloneBook(row.book), nil
}

func (s *MemoryStore) SetInStock(id int, inStock bool) (Book, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.indexByID(id)
	if idx < 0 {
		return Book{}, ErrNotFound
	}

	row := s.rows[idx]
	applyInStockStateTransition(&row.book, inStock)
	s.rows[idx] = row
	return cloneBook(row.book), nil
}

func (s *MemoryStore) Publish(id int) (Book, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.indexByID(id)
	if idx < 0 {
		return Book{}, ErrNotFound
	}
	row := s.rows[idx]
	if !row.book.InStock {
		return Book{}, ErrCannotPublishOutOfStock
	}
	now := time.Now().UTC()
	row.book.IsPublished = true
	row.book.PublishedAt = &now
	s.rows[idx] = row
	return cloneBook(row.book), nil
}

func (s *MemoryStore) Unpublish(id int) (Book, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.indexByID(id)
	if idx < 0 {
		return Book{}, ErrNotFound
	}
	row := s.rows[idx]
	now := time.Now().UTC()
	row.book.IsPublished = false
	row.book.UnpublishedAt = &now
	s.rows[idx] = row
	return cloneBook(row.book), nil
}

func (s *MemoryStore) indexByID(id int) int {
	for i, row := range s.rows {
		if row.book.ID == id {
			return i
		}
	}
	return -1
}

func cloneBook(book Book) Book {
	book.BundlePrice = cloneFloatPointer(book.BundlePrice)
	book.PublishedAt = cloneTimePointer(book.PublishedAt)
	book.UnpublishedAt = cloneTimePointer(book.UnpublishedAt)
	return book
}

func cloneFloatPointer(v *float64) *float64 {
	if v == nil {
		return nil
	}
	cloned := *v
	return &cloned
}

func cloneCover(cover Cover) Cover {
	data := make([]byte, len(cover.Data))
	copy(data, cover.Data)
	return Cover{Data: data, MimeType: cover.MimeType}
}

func cloneTimePointer(v *time.Time) *time.Time {
	if v == nil {
		return nil
	}
	cloned := *v
	return &cloned
}

func applyInStockStateTransition(book *Book, inStock bool) {
	book.InStock = inStock
	if inStock {
		return
	}
	now := time.Now().UTC()
	book.IsPublished = false
	book.UnpublishedAt = &now
}
