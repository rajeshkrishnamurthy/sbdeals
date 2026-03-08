package clicked

import "testing"

func TestMemoryStoreCreateListAndConvert(t *testing.T) {
	store := NewMemoryStore()

	created, err := store.CreateClicked(CreateInput{
		ItemID:       7,
		ItemType:     ItemTypeBook,
		ItemTitle:    "Book One",
		SourcePage:   "catalog",
		SourceRailID: 2,
		SourceRail:   "Fresh Books",
	})
	if err != nil {
		t.Fatalf("create returned error: %v", err)
	}
	if created.Status != StatusClicked {
		t.Fatalf("expected clicked status, got %q", created.Status)
	}

	clickedItems, err := store.ListByStatus(StatusClicked)
	if err != nil {
		t.Fatalf("list clicked returned error: %v", err)
	}
	if len(clickedItems) != 1 || clickedItems[0].ID != created.ID {
		t.Fatalf("unexpected clicked list: %+v", clickedItems)
	}

	converted, alreadyConverted, err := store.ConvertToInterested(created.ID, ConvertInput{
		BuyerName:   "Rajesh",
		BuyerPhone:  "+919999999999",
		BuyerNote:   "Call after 6",
		ConvertedBy: "system-admin",
	})
	if err != nil {
		t.Fatalf("convert returned error: %v", err)
	}
	if alreadyConverted {
		t.Fatalf("expected first conversion to not be already converted")
	}
	if converted.Status != StatusInterested || converted.ConvertedAt == nil {
		t.Fatalf("unexpected converted enquiry: %+v", converted)
	}

	_, alreadyConverted, err = store.ConvertToInterested(created.ID, ConvertInput{
		BuyerName:   "Rajesh",
		BuyerPhone:  "+919999999999",
		ConvertedBy: "system-admin",
	})
	if err != nil {
		t.Fatalf("second convert returned error: %v", err)
	}
	if !alreadyConverted {
		t.Fatalf("expected idempotent already-converted flag")
	}
}

func TestNormalizeIndiaPhone(t *testing.T) {
	normalized, ok := NormalizeIndiaPhone(" 98765 43210 ")
	if !ok || normalized != "+919876543210" {
		t.Fatalf("unexpected normalization: %q %v", normalized, ok)
	}
	if _, ok := NormalizeIndiaPhone("12345"); ok {
		t.Fatalf("expected invalid short number")
	}
	if _, ok := NormalizeIndiaPhone("5234567890"); ok {
		t.Fatalf("expected invalid starting digit")
	}
}
