package rails

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("rail not found")
var ErrDuplicateTitle = errors.New("rail title already exists")
var ErrDuplicateItem = errors.New("item already in rail")

type RailType string

const (
	RailTypeBook   RailType = "BOOK"
	RailTypeBundle RailType = "BUNDLE"
)

func (t RailType) IsValid() bool {
	return t == RailTypeBook || t == RailTypeBundle
}

type Rail struct {
	ID            int
	Title         string
	AdminNote     string
	Type          RailType
	ItemIDs       []int
	IsPublished   bool
	PublishedAt   *time.Time
	UnpublishedAt *time.Time
	Position      int
}

type ListItem struct {
	ID            int
	Title         string
	AdminNote     string
	Type          RailType
	ItemCount     int
	IsPublished   bool
	PublishedAt   *time.Time
	UnpublishedAt *time.Time
	Position      int
}

type CreateInput struct {
	Title     string
	Type      RailType
	AdminNote string
}

type UpdateInput struct {
	Title     string
	AdminNote string
}

type Store interface {
	List() ([]ListItem, error)
	Create(input CreateInput) (Rail, error)
	Get(id int) (Rail, error)
	Update(id int, input UpdateInput) (Rail, error)
	AddItem(id int, itemID int) (Rail, error)
	RemoveItem(id int, itemID int) (Rail, error)
	Publish(id int) (Rail, error)
	Unpublish(id int) (Rail, error)
	MoveUp(id int) error
	MoveDown(id int) error
}
