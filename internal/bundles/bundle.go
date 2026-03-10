package bundles

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

var ErrNotFound = errors.New("bundle not found")
var ErrCannotPublishOutOfStock = errors.New("bundle cannot be published because it is out of stock")

type ErrCannotPublishWithOutOfStockBooks struct {
	BookTitles []string
}

func (e *ErrCannotPublishWithOutOfStockBooks) Error() string {
	if len(e.BookTitles) == 0 {
		return "bundle cannot be published because included books are out of stock"
	}
	titles := append([]string(nil), e.BookTitles...)
	sort.Strings(titles)
	return fmt.Sprintf("bundle cannot be published because these books are out of stock: %s", joinTitles(titles))
}

func joinTitles(titles []string) string {
	if len(titles) == 0 {
		return ""
	}
	out := fmt.Sprintf("%q", titles[0])
	for i := 1; i < len(titles); i++ {
		out += ", " + fmt.Sprintf("%q", titles[i])
	}
	return out
}

// BundleBook is a book entry inside a bundle.
type BundleBook struct {
	BookID      int
	Title       string
	Author      string
	SupplierID  int
	IsBoxSet    bool
	Category    string
	Condition   string
	MRP         float64
	MyPrice     float64
	BundlePrice *float64
	InStock     bool
}

type Image struct {
	Data     []byte
	MimeType string
}

// PickerBook represents an eligible book candidate for the bundle picker.
type PickerBook struct {
	BookID      int
	Title       string
	Author      string
	SupplierID  int
	IsBoxSet    bool
	Category    string
	Condition   string
	MRP         float64
	MyPrice     float64
	BundlePrice *float64
	InStock     bool
}

// Bundle is the detailed bundle aggregate used by add/edit screens.
type Bundle struct {
	ID                     int
	Name                   string
	SupplierID             int
	SupplierName           string
	Category               string
	AllowedConditions      []string
	BundlePrice            float64
	Notes                  string
	BookIDs                []int
	Books                  []BundleBook
	InStock                bool
	OutOfStockOnInterested bool
	ImageMimeType          string
	IsPublished            bool
	PublishedAt            *time.Time
	UnpublishedAt          *time.Time
}

// ListItem is the low-clutter projection for bundles list page.
type ListItem struct {
	ID                int
	Name              string
	SupplierName      string
	Category          string
	AllowedConditions []string
	BookCount         int
	BundleMRP         float64
	BundlePrice       float64
	HasImage          bool
	IsPublished       bool
	PublishedAt       *time.Time
	UnpublishedAt     *time.Time
}

// CreateInput captures required and optional fields for bundle creation.
type CreateInput struct {
	Name                   string
	SupplierID             int
	Category               string
	AllowedConditions      []string
	BookIDs                []int
	BundlePrice            float64
	Notes                  string
	Image                  Image
	OutOfStockOnInterested bool
}

// UpdateInput captures editable fields for bundle updates.
type UpdateInput struct {
	Name                   string
	SupplierID             int
	Category               string
	AllowedConditions      []string
	BookIDs                []int
	BundlePrice            float64
	Notes                  string
	Image                  *Image
	OutOfStockOnInterested bool
}

// Store defines persistence operations for bundles.
type Store interface {
	List() ([]ListItem, error)
	Create(input CreateInput) (Bundle, error)
	Get(id int) (Bundle, error)
	Update(id int, input UpdateInput) (Bundle, error)
	Publish(id int) (Bundle, error)
	Unpublish(id int) (Bundle, error)
	ListBooksForPicker() ([]PickerBook, error)
	GetImage(id int) (Image, error)
}
