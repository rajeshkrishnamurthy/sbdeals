package clicked

import (
	"errors"
	"strings"
	"time"
)

var ErrNotFound = errors.New("enquiry not found")

type ItemType string

const (
	ItemTypeBook   ItemType = "BOOK"
	ItemTypeBundle ItemType = "BUNDLE"
)

type Status string

const (
	StatusClicked    Status = "clicked"
	StatusInterested Status = "interested"
)

type Enquiry struct {
	ID           int
	ItemID       int
	ItemType     ItemType
	ItemTitle    string
	SourcePage   string
	SourceRailID int
	SourceRail   string
	Status       Status
	BuyerName    string
	BuyerPhone   string
	BuyerNote    string
	ConvertedBy  string
	ConvertedAt  *time.Time
	CreatedAt    time.Time
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
	BuyerName   string
	BuyerPhone  string
	BuyerNote   string
	ConvertedBy string
}

type Store interface {
	CreateClicked(input CreateInput) (Enquiry, error)
	ListByStatus(status Status) ([]Enquiry, error)
	ConvertToInterested(id int, input ConvertInput) (Enquiry, bool, error)
}

func IsValidItemType(itemType ItemType) bool {
	return itemType == ItemTypeBook || itemType == ItemTypeBundle
}

func IsValidStatus(status Status) bool {
	return status == StatusClicked || status == StatusInterested
}

func NormalizeIndiaPhone(local string) (string, bool) {
	digits := onlyDigits(local)
	if len(digits) != 10 {
		return "", false
	}
	if digits[0] < '6' || digits[0] > '9' {
		return "", false
	}
	return "+91" + digits, true
}

func onlyDigits(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
