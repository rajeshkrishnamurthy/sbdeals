package clicked

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("enquiry not found")
var ErrInvalidTransition = errors.New("invalid enquiry transition")
var ErrAddressRequired = errors.New("customer address is required")

type ItemType string

const (
	ItemTypeBook   ItemType = "BOOK"
	ItemTypeBundle ItemType = "BUNDLE"
)

type Status string

const (
	StatusClicked    Status = "clicked"
	StatusInterested Status = "interested"
	StatusOrdered    Status = "ordered"
)

type Enquiry struct {
	ID             int
	ItemID         int
	ItemType       ItemType
	ItemTitle      string
	SourcePage     string
	SourceRailID   int
	SourceRail     string
	Status         Status
	CustomerID     int
	Note           string
	OrderAmount    *int
	LastModifiedBy string
	LastModifiedAt *time.Time
	CreatedAt      time.Time
}

type CreateInput struct {
	ItemID       int
	ItemType     ItemType
	ItemTitle    string
	SourcePage   string
	SourceRailID int
	SourceRail   string
}

type ConvertInput struct {
	CustomerID int
	Note       string
	ModifiedBy string
}

type OrderInput struct {
	OrderAmount int
	Note        string
	Address     string
	ModifiedBy  string
}

type Store interface {
	CreateClicked(input CreateInput) (Enquiry, error)
	Get(id int) (Enquiry, error)
	ListByStatus(status Status) ([]Enquiry, error)
	ConvertToInterested(id int, input ConvertInput) (Enquiry, bool, error)
	ConvertToOrdered(id int, input OrderInput) (Enquiry, bool, error)
}

func IsValidItemType(itemType ItemType) bool {
	return itemType == ItemTypeBook || itemType == ItemTypeBundle
}

func IsValidStatus(status Status) bool {
	return status == StatusClicked || status == StatusInterested || status == StatusOrdered
}
