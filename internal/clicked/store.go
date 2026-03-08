package clicked

import "time"

type ItemType string

const (
	ItemTypeBook   ItemType = "BOOK"
	ItemTypeBundle ItemType = "BUNDLE"
)

type Event struct {
	ID           int
	ItemID       int
	ItemType     ItemType
	ItemTitle    string
	SourcePage   string
	SourceRailID int
	SourceRail   string
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

type Store interface {
	Create(input CreateInput) (Event, error)
}
