package rails

import (
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
)

type MemoryStore struct {
	mu     sync.RWMutex
	nextID int
	rails  []Rail
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{nextID: 1, rails: make([]Rail, 0)}
}

func (s *MemoryStore) List() ([]ListItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ordered := s.orderedRailsLocked()
	items := make([]ListItem, 0, len(ordered))
	for _, rail := range ordered {
		items = append(items, ListItem{
			ID:            rail.ID,
			Title:         rail.Title,
			AdminNote:     rail.AdminNote,
			Type:          rail.Type,
			ItemCount:     len(rail.ItemIDs),
			IsPublished:   rail.IsPublished,
			PublishedAt:   cloneTime(rail.PublishedAt),
			UnpublishedAt: cloneTime(rail.UnpublishedAt),
			Position:      rail.Position,
		})
	}
	return items, nil
}

func (s *MemoryStore) Create(input CreateInput) (Rail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.titleExistsLocked(input.Title, 0) {
		return Rail{}, ErrDuplicateTitle
	}

	now := time.Now().UTC()
	rail := Rail{
		ID:            s.nextID,
		Title:         strings.TrimSpace(input.Title),
		AdminNote:     strings.TrimSpace(input.AdminNote),
		Type:          input.Type,
		ItemIDs:       []int{},
		IsPublished:   false,
		PublishedAt:   nil,
		UnpublishedAt: &now,
		Position:      len(s.rails) + 1,
	}
	s.nextID++
	s.rails = append(s.rails, rail)
	return cloneRail(rail), nil
}

func (s *MemoryStore) Get(id int) (Rail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	index := s.indexByIDLocked(id)
	if index < 0 {
		return Rail{}, ErrNotFound
	}
	return cloneRail(s.rails[index]), nil
}

func (s *MemoryStore) Update(id int, input UpdateInput) (Rail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.indexByIDLocked(id)
	if index < 0 {
		return Rail{}, ErrNotFound
	}
	if s.titleExistsLocked(input.Title, id) {
		return Rail{}, ErrDuplicateTitle
	}
	s.rails[index].Title = strings.TrimSpace(input.Title)
	s.rails[index].AdminNote = strings.TrimSpace(input.AdminNote)
	return cloneRail(s.rails[index]), nil
}

func (s *MemoryStore) AddItem(id int, itemID int) (Rail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.indexByIDLocked(id)
	if index < 0 {
		return Rail{}, ErrNotFound
	}
	if slices.Contains(s.rails[index].ItemIDs, itemID) {
		return Rail{}, ErrDuplicateItem
	}
	s.rails[index].ItemIDs = append(s.rails[index].ItemIDs, itemID)
	return cloneRail(s.rails[index]), nil
}

func (s *MemoryStore) RemoveItem(id int, itemID int) (Rail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.indexByIDLocked(id)
	if index < 0 {
		return Rail{}, ErrNotFound
	}
	current := s.rails[index].ItemIDs
	filtered := make([]int, 0, len(current))
	for _, existing := range current {
		if existing == itemID {
			continue
		}
		filtered = append(filtered, existing)
	}
	s.rails[index].ItemIDs = filtered
	return cloneRail(s.rails[index]), nil
}

func (s *MemoryStore) Publish(id int) (Rail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.indexByIDLocked(id)
	if index < 0 {
		return Rail{}, ErrNotFound
	}
	now := time.Now().UTC()
	s.rails[index].IsPublished = true
	s.rails[index].PublishedAt = &now
	return cloneRail(s.rails[index]), nil
}

func (s *MemoryStore) Unpublish(id int) (Rail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.indexByIDLocked(id)
	if index < 0 {
		return Rail{}, ErrNotFound
	}
	now := time.Now().UTC()
	s.rails[index].IsPublished = false
	s.rails[index].UnpublishedAt = &now
	return cloneRail(s.rails[index]), nil
}

func (s *MemoryStore) MoveUp(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.indexByIDLocked(id)
	if index < 0 {
		return ErrNotFound
	}
	neighbor := s.neighborIndexLocked(index, true)
	if neighbor < 0 {
		return nil
	}
	s.swapPositionLocked(index, neighbor)
	return nil
}

func (s *MemoryStore) MoveDown(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.indexByIDLocked(id)
	if index < 0 {
		return ErrNotFound
	}
	neighbor := s.neighborIndexLocked(index, false)
	if neighbor < 0 {
		return nil
	}
	s.swapPositionLocked(index, neighbor)
	return nil
}

func (s *MemoryStore) neighborIndexLocked(index int, up bool) int {
	target := -1
	for i := range s.rails {
		if i == index {
			continue
		}
		if up {
			if s.rails[i].Position < s.rails[index].Position && (target < 0 || s.rails[i].Position > s.rails[target].Position) {
				target = i
			}
		} else {
			if s.rails[i].Position > s.rails[index].Position && (target < 0 || s.rails[i].Position < s.rails[target].Position) {
				target = i
			}
		}
	}
	return target
}

func (s *MemoryStore) swapPositionLocked(a, b int) {
	s.rails[a].Position, s.rails[b].Position = s.rails[b].Position, s.rails[a].Position
}

func (s *MemoryStore) titleExistsLocked(title string, excludeID int) bool {
	normalized := strings.TrimSpace(strings.ToLower(title))
	for _, rail := range s.rails {
		if rail.ID == excludeID {
			continue
		}
		if strings.TrimSpace(strings.ToLower(rail.Title)) == normalized {
			return true
		}
	}
	return false
}

func (s *MemoryStore) indexByIDLocked(id int) int {
	for i, rail := range s.rails {
		if rail.ID == id {
			return i
		}
	}
	return -1
}

func (s *MemoryStore) orderedRailsLocked() []Rail {
	ordered := make([]Rail, len(s.rails))
	copy(ordered, s.rails)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Position == ordered[j].Position {
			return ordered[i].ID < ordered[j].ID
		}
		return ordered[i].Position < ordered[j].Position
	})
	return ordered
}

func cloneRail(rail Rail) Rail {
	out := rail
	out.ItemIDs = append([]int(nil), rail.ItemIDs...)
	out.PublishedAt = cloneTime(rail.PublishedAt)
	out.UnpublishedAt = cloneTime(rail.UnpublishedAt)
	return out
}

func cloneTime(ts *time.Time) *time.Time {
	if ts == nil {
		return nil
	}
	copied := *ts
	return &copied
}
