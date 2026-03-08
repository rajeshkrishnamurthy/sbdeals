package books

import "time"

// Book is the full admin view model for a supplier-specific book listing.
type Book struct {
	ID                     int
	Title                  string
	SupplierID             int
	CoverMimeType          string
	IsBoxSet               bool
	Category               string
	Format                 string
	Condition              string
	MRP                    float64
	MyPrice                float64
	BundlePrice            *float64
	Author                 string
	Notes                  string
	InStock                bool
	OutOfStockOnInterested bool
	IsPublished            bool
	PublishedAt            *time.Time
	UnpublishedAt          *time.Time
}

// Cover contains binary image bytes and associated MIME type.
type Cover struct {
	Data     []byte
	MimeType string
}

// ListItem is a low-clutter row projection for the admin books list.
type ListItem struct {
	ID            int
	Title         string
	Author        string
	Category      string
	MyPrice       float64
	InStock       bool
	HasCover      bool
	IsPublished   bool
	PublishedAt   *time.Time
	UnpublishedAt *time.Time
}

// CreateInput captures book fields for create flow.
type CreateInput struct {
	Title                  string
	Cover                  Cover
	SupplierID             int
	IsBoxSet               bool
	Category               string
	Format                 string
	Condition              string
	MRP                    float64
	MyPrice                float64
	BundlePrice            *float64
	Author                 string
	Notes                  string
	OutOfStockOnInterested bool
}

// UpdateInput captures editable book fields for edit flow.
type UpdateInput struct {
	Title                  string
	Cover                  *Cover
	SupplierID             int
	IsBoxSet               bool
	Category               string
	Format                 string
	Condition              string
	MRP                    float64
	MyPrice                float64
	BundlePrice            *float64
	Author                 string
	Notes                  string
	InStock                bool
	OutOfStockOnInterested bool
}

func ComputeDiscount(mrp, myPrice float64) float64 {
	if mrp <= 0 {
		return 0
	}
	return ((mrp - myPrice) / mrp) * 100
}
