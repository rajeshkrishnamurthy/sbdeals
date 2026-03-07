package suppliers

import (
	"errors"
	"sync"
)

var ErrNotFound = errors.New("supplier not found")

// Store holds suppliers for admin CRUD operations used in MVP.
type Store interface {
	List() ([]Supplier, error)
	Create(input Input) (Supplier, error)
	Get(id int) (Supplier, error)
	Update(id int, input Input) (Supplier, error)
}

// MemoryStore keeps suppliers in process memory for MVP simplicity.
type MemoryStore struct {
	mu        sync.RWMutex
	nextID    int
	suppliers []Supplier
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{nextID: 1}
}

func (s *MemoryStore) List() ([]Supplier, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]Supplier, len(s.suppliers))
	copy(items, s.suppliers)
	return items, nil
}

func (s *MemoryStore) Create(input Input) (Supplier, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item := Supplier{
		ID:       s.nextID,
		Name:     input.Name,
		WhatsApp: input.WhatsApp,
		Location: input.Location,
		Notes:    input.Notes,
	}
	s.nextID++
	s.suppliers = append(s.suppliers, item)
	return item, nil
}

func (s *MemoryStore) Get(id int) (Supplier, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, item := range s.suppliers {
		if item.ID == id {
			return item, nil
		}
	}
	return Supplier{}, ErrNotFound
}

func (s *MemoryStore) Update(id int, input Input) (Supplier, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, item := range s.suppliers {
		if item.ID != id {
			continue
		}
		item.Name = input.Name
		item.WhatsApp = input.WhatsApp
		item.Location = input.Location
		item.Notes = input.Notes
		s.suppliers[i] = item
		return item, nil
	}

	return Supplier{}, ErrNotFound
}
