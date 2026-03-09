package customers

import (
	"errors"
	"strings"
	"time"
)

var ErrNotFound = errors.New("customer not found")

type DuplicateMobileError struct {
	CustomerID int
}

func (e *DuplicateMobileError) Error() string {
	return "customer already exists"
}

type City struct {
	ID   int
	Name string
}

type ApartmentComplex struct {
	ID     int
	CityID int
	Name   string
}

type Customer struct {
	ID                 int
	Name               string
	Mobile             string
	Address            *string
	CityID             *int
	CityName           string
	ApartmentComplexID *int
	ApartmentName      string
	Notes              *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type ListItem struct {
	ID            int
	Name          string
	Mobile        string
	CityName      string
	ApartmentName string
}

type ListFilter struct {
	Search string
	CityID *int
}

type CreateInput struct {
	Name          string
	Mobile        string
	Address       *string
	CityName      *string
	ApartmentName *string
	Notes         *string
}

type UpdateInput struct {
	Name          string
	Address       *string
	CityName      *string
	ApartmentName *string
	Notes         *string
}

type Store interface {
	List(filter ListFilter) ([]ListItem, error)
	Get(id int) (Customer, error)
	Create(input CreateInput) (Customer, error)
	Update(id int, input UpdateInput) (Customer, error)
	ListCities() ([]City, error)
	ListApartmentComplexesByCityID(cityID int) ([]ApartmentComplex, error)
}

func NormalizeMobile(raw string) string {
	var b strings.Builder
	b.Grow(len(raw))
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func NormalizeMasterName(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}
