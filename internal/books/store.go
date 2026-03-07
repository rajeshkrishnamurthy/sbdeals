package books

import "errors"

var ErrNotFound = errors.New("book not found")
var ErrCannotPublishOutOfStock = errors.New("book cannot be published while out of stock")

// Store defines persistence operations for admin books flows.
type Store interface {
	List() ([]ListItem, error)
	Create(input CreateInput) (Book, error)
	Get(id int) (Book, error)
	GetCover(id int) (Cover, error)
	Update(id int, input UpdateInput) (Book, error)
	SetInStock(id int, inStock bool) (Book, error)
	Publish(id int) (Book, error)
	Unpublish(id int) (Book, error)
}
