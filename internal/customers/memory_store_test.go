package customers

import (
	"errors"
	"testing"
)

func TestMemoryStoreCreateDuplicateAndGet(t *testing.T) {
	store := NewMemoryStore()

	created, err := store.Create(CreateInput{
		Name:   "Rajesh K",
		Mobile: "98765-43210",
	})
	if err != nil {
		t.Fatalf("create returned error: %v", err)
	}
	if created.Mobile != "9876543210" {
		t.Fatalf("expected normalized mobile, got %q", created.Mobile)
	}

	_, err = store.Create(CreateInput{
		Name:   "Another",
		Mobile: "98765 43210",
	})
	var dupErr *DuplicateMobileError
	if !errors.As(err, &dupErr) {
		t.Fatalf("expected DuplicateMobileError, got %v", err)
	}
	if dupErr.CustomerID != created.ID {
		t.Fatalf("expected duplicate id %d, got %d", created.ID, dupErr.CustomerID)
	}

	got, err := store.Get(created.ID)
	if err != nil {
		t.Fatalf("get returned error: %v", err)
	}
	if got.Name != created.Name {
		t.Fatalf("unexpected customer: %+v", got)
	}
}

func TestMemoryStoreListSearchAndCityFilter(t *testing.T) {
	store := NewMemoryStore()
	_, _ = store.Create(CreateInput{Name: "Asha", Mobile: "9999999991", CityName: strPtr("Bengaluru"), ApartmentName: strPtr("Skyline")})
	_, _ = store.Create(CreateInput{Name: "Rohan", Mobile: "9999999992", CityName: strPtr("Chennai"), ApartmentName: strPtr("Marina")})

	all, err := store.List(ListFilter{})
	if err != nil {
		t.Fatalf("list all returned error: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(all))
	}

	search, err := store.List(ListFilter{Search: "9992"})
	if err != nil {
		t.Fatalf("search returned error: %v", err)
	}
	if len(search) != 1 || search[0].Name != "Rohan" {
		t.Fatalf("unexpected search result: %+v", search)
	}

	cities, err := store.ListCities()
	if err != nil {
		t.Fatalf("list cities returned error: %v", err)
	}
	var bengaluruID int
	for _, city := range cities {
		if city.Name == "Bengaluru" {
			bengaluruID = city.ID
		}
	}
	if bengaluruID == 0 {
		t.Fatalf("expected Bengaluru city id")
	}
	byCity, err := store.List(ListFilter{CityID: &bengaluruID})
	if err != nil {
		t.Fatalf("city filter returned error: %v", err)
	}
	if len(byCity) != 1 || byCity[0].CityName != "Bengaluru" {
		t.Fatalf("unexpected city filter result: %+v", byCity)
	}
}

func strPtr(value string) *string {
	return &value
}
