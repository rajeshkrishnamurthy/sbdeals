package customers

import (
	"sort"
	"strings"
	"sync"
	"time"
)

type MemoryStore struct {
	mu sync.Mutex

	nextCustomerID int
	nextCityID     int
	nextApartment  int

	customers  map[int]Customer
	cities     map[int]City
	apartments map[int]ApartmentComplex
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		nextCustomerID: 1,
		nextCityID:     1,
		nextApartment:  1,
		customers:      map[int]Customer{},
		cities:         map[int]City{},
		apartments:     map[int]ApartmentComplex{},
	}
}

func (s *MemoryStore) List(filter ListFilter) ([]ListItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	search := strings.ToLower(strings.TrimSpace(filter.Search))
	items := make([]ListItem, 0, len(s.customers))
	for _, customer := range s.customers {
		if filter.CityID != nil {
			if customer.CityID == nil || *customer.CityID != *filter.CityID {
				continue
			}
		}
		if search != "" && !matchesSearch(customer, search) {
			continue
		}
		items = append(items, ListItem{
			ID:            customer.ID,
			Name:          customer.Name,
			Mobile:        customer.Mobile,
			CityName:      customer.CityName,
			ApartmentName: customer.ApartmentName,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

func (s *MemoryStore) Get(id int) (Customer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	customer, ok := s.customers[id]
	if !ok {
		return Customer{}, ErrNotFound
	}
	return cloneCustomer(customer), nil
}

func (s *MemoryStore) Create(input CreateInput) (Customer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	normalizedMobile := NormalizeMobile(input.Mobile)
	for _, customer := range s.customers {
		if NormalizeMobile(customer.Mobile) == normalizedMobile {
			return Customer{}, &DuplicateMobileError{CustomerID: customer.ID}
		}
	}

	cityID, cityName := s.resolveCity(input.CityName)
	aptID, aptName := s.resolveApartment(cityID, input.ApartmentName)
	now := time.Now().UTC()
	customer := Customer{
		ID:                 s.nextCustomerID,
		Name:               strings.TrimSpace(input.Name),
		Mobile:             normalizedMobile,
		Address:            cloneStringPtr(input.Address),
		CityID:             cityID,
		CityName:           cityName,
		ApartmentComplexID: aptID,
		ApartmentName:      aptName,
		Notes:              cloneStringPtr(input.Notes),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	s.customers[customer.ID] = customer
	s.nextCustomerID++
	return cloneCustomer(customer), nil
}

func (s *MemoryStore) Update(id int, input UpdateInput) (Customer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.customers[id]
	if !ok {
		return Customer{}, ErrNotFound
	}

	cityID, cityName := s.resolveCity(input.CityName)
	aptID, aptName := s.resolveApartment(cityID, input.ApartmentName)
	current.Name = strings.TrimSpace(input.Name)
	current.Address = cloneStringPtr(input.Address)
	current.CityID = cityID
	current.CityName = cityName
	current.ApartmentComplexID = aptID
	current.ApartmentName = aptName
	current.Notes = cloneStringPtr(input.Notes)
	current.UpdatedAt = time.Now().UTC()
	s.customers[id] = current
	return cloneCustomer(current), nil
}

func (s *MemoryStore) ListCities() ([]City, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]City, 0, len(s.cities))
	for _, city := range s.cities {
		items = append(items, city)
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	return items, nil
}

func (s *MemoryStore) ListApartmentComplexesByCityID(cityID int) ([]ApartmentComplex, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]ApartmentComplex, 0)
	for _, apartment := range s.apartments {
		if apartment.CityID == cityID {
			items = append(items, apartment)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	return items, nil
}

func (s *MemoryStore) resolveCity(name *string) (*int, string) {
	if name == nil {
		return nil, ""
	}
	trimmed := strings.TrimSpace(*name)
	if trimmed == "" {
		return nil, ""
	}
	normalized := NormalizeMasterName(trimmed)
	for _, city := range s.cities {
		if NormalizeMasterName(city.Name) == normalized {
			id := city.ID
			return &id, city.Name
		}
	}
	id := s.nextCityID
	s.nextCityID++
	s.cities[id] = City{ID: id, Name: trimmed}
	return &id, trimmed
}

func (s *MemoryStore) resolveApartment(cityID *int, name *string) (*int, string) {
	if cityID == nil || name == nil {
		return nil, ""
	}
	trimmed := strings.TrimSpace(*name)
	if trimmed == "" {
		return nil, ""
	}
	normalized := NormalizeMasterName(trimmed)
	for _, apartment := range s.apartments {
		if apartment.CityID == *cityID && NormalizeMasterName(apartment.Name) == normalized {
			id := apartment.ID
			return &id, apartment.Name
		}
	}
	id := s.nextApartment
	s.nextApartment++
	s.apartments[id] = ApartmentComplex{
		ID:     id,
		CityID: *cityID,
		Name:   trimmed,
	}
	return &id, trimmed
}

func matchesSearch(customer Customer, search string) bool {
	return strings.Contains(strings.ToLower(customer.Name), search) ||
		strings.Contains(strings.ToLower(customer.Mobile), search)
}

func cloneStringPtr(in *string) *string {
	if in == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*in)
	if trimmed == "" {
		return nil
	}
	out := trimmed
	return &out
}

func cloneCustomer(customer Customer) Customer {
	out := customer
	out.Address = cloneStringPtr(customer.Address)
	out.Notes = cloneStringPtr(customer.Notes)
	if customer.CityID != nil {
		id := *customer.CityID
		out.CityID = &id
	}
	if customer.ApartmentComplexID != nil {
		id := *customer.ApartmentComplexID
		out.ApartmentComplexID = &id
	}
	return out
}
