package clicked

import (
	"sync"
	"time"
)

type MemoryStore struct {
	mu     sync.Mutex
	nextID int
	items  []Event
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		nextID: 1,
		items:  make([]Event, 0),
	}
}

func (s *MemoryStore) Create(input CreateInput) (Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	event := Event{
		ID:           s.nextID,
		ItemID:       input.ItemID,
		ItemType:     input.ItemType,
		ItemTitle:    input.ItemTitle,
		SourcePage:   input.SourcePage,
		SourceRailID: input.SourceRailID,
		SourceRail:   input.SourceRail,
		CreatedAt:    time.Now().UTC(),
	}
	s.nextID++
	s.items = append(s.items, event)
	return event, nil
}
