package clicked

import (
	"sync"
	"time"
)

type MemoryStore struct {
	mu     sync.Mutex
	nextID int
	items  []Enquiry
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		nextID: 1,
		items:  make([]Enquiry, 0),
	}
}

func (s *MemoryStore) CreateClicked(input CreateInput) (Enquiry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item := Enquiry{
		ID:           s.nextID,
		ItemID:       input.ItemID,
		ItemType:     input.ItemType,
		ItemTitle:    input.ItemTitle,
		SourcePage:   input.SourcePage,
		SourceRailID: input.SourceRailID,
		SourceRail:   input.SourceRail,
		Status:       StatusClicked,
		CreatedAt:    time.Now().UTC(),
	}
	s.nextID++
	s.items = append(s.items, item)
	return cloneEnquiry(item), nil
}

func (s *MemoryStore) ListByStatus(status Status) ([]Enquiry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]Enquiry, 0)
	for i := len(s.items) - 1; i >= 0; i-- {
		if s.items[i].Status != status {
			continue
		}
		out = append(out, cloneEnquiry(s.items[i]))
	}
	return out, nil
}

func (s *MemoryStore) Get(id int) (Enquiry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.indexByID(id)
	if index < 0 {
		return Enquiry{}, ErrNotFound
	}
	return cloneEnquiry(s.items[index]), nil
}

func (s *MemoryStore) ConvertToInterested(id int, input ConvertInput) (Enquiry, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.indexByID(id)
	if index < 0 {
		return Enquiry{}, false, ErrNotFound
	}
	current := s.items[index]
	if current.Status == StatusInterested {
		return cloneEnquiry(current), true, nil
	}
	now := time.Now().UTC()
	current.Status = StatusInterested
	current.CustomerID = input.CustomerID
	current.Note = input.Note
	current.LastModifiedBy = input.ModifiedBy
	current.LastModifiedAt = &now
	s.items[index] = current
	return cloneEnquiry(current), false, nil
}

func (s *MemoryStore) ConvertToOrdered(id int, input OrderInput) (Enquiry, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.indexByID(id)
	if index < 0 {
		return Enquiry{}, false, ErrNotFound
	}
	current := s.items[index]
	if current.Status == StatusOrdered {
		return cloneEnquiry(current), true, nil
	}
	if current.Status != StatusInterested {
		return Enquiry{}, false, ErrInvalidTransition
	}

	now := time.Now().UTC()
	orderAmount := input.OrderAmount
	current.Status = StatusOrdered
	current.OrderAmount = &orderAmount
	current.Note = input.Note
	current.LastModifiedBy = input.ModifiedBy
	current.LastModifiedAt = &now
	s.items[index] = current
	return cloneEnquiry(current), false, nil
}

func (s *MemoryStore) indexByID(id int) int {
	for i := range s.items {
		if s.items[i].ID == id {
			return i
		}
	}
	return -1
}

func cloneEnquiry(in Enquiry) Enquiry {
	out := in
	if in.LastModifiedAt != nil {
		ts := *in.LastModifiedAt
		out.LastModifiedAt = &ts
	}
	if in.OrderAmount != nil {
		amount := *in.OrderAmount
		out.OrderAmount = &amount
	}
	return out
}
