package books

import "errors"

var ErrNotFound = errors.New("book not found")

// Store defines persistence operations for admin books flows.
type Store interface {
	List() ([]ListItem, error)
	Create(input CreateInput) (Book, error)
	Get(id int) (Book, error)
	GetCover(id int) (Cover, error)
	Update(id int, input UpdateInput) (Book, error)
	SetInStock(id int, inStock bool) (Book, error)
}
