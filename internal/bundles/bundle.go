package bundles

import "errors"

var ErrNotFound = errors.New("bundle not found")

// BundleBook is a book entry inside a bundle.
type BundleBook struct {
	BookID       int
	Title        string
	Author       string
	SupplierID   int
	Category     string
	Condition    string
	MRP          float64
	MyPrice      float64
	BundlePrice  *float64
}

// PickerBook represents an eligible book candidate for the bundle picker.
type PickerBook struct {
	BookID      int
	Title       string
	Author      string
	SupplierID  int
	Category    string
	Condition   string
	MRP         float64
	MyPrice     float64
	BundlePrice *float64
}

// Bundle is the detailed bundle aggregate used by add/edit screens.
type Bundle struct {
	ID                int
	Name              string
	SupplierID        int
	SupplierName      string
	Category          string
	AllowedConditions []string
	BundlePrice       float64
	Notes             string
	BookIDs           []int
	Books             []BundleBook
}

// ListItem is the low-clutter projection for bundles list page.
type ListItem struct {
	ID                int
	Name              string
	SupplierName      string
	Category          string
	AllowedConditions []string
	BookCount         int
	BundlePrice       float64
}

// CreateInput captures required and optional fields for bundle creation.
type CreateInput struct {
	Name              string
	SupplierID        int
	Category          string
	AllowedConditions []string
	BookIDs           []int
	BundlePrice       float64
	Notes             string
}

// UpdateInput captures editable fields for bundle updates.
type UpdateInput struct {
	Name              string
	SupplierID        int
	Category          string
	AllowedConditions []string
	BookIDs           []int
	BundlePrice       float64
	Notes             string
}

// Store defines persistence operations for bundles.
type Store interface {
	List() ([]ListItem, error)
	Create(input CreateInput) (Bundle, error)
	Get(id int) (Bundle, error)
	Update(id int, input UpdateInput) (Bundle, error)
	ListBooksForPicker() ([]PickerBook, error)
}
