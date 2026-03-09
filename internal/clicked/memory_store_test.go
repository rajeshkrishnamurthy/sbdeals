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
		CustomerID: 1,
		Note:       "Call after 6",
		ModifiedBy: "system-admin",
	})
	if err != nil {
		t.Fatalf("convert returned error: %v", err)
	}
	if alreadyConverted {
		t.Fatalf("expected first conversion to not be already converted")
	}
	if converted.Status != StatusInterested || converted.LastModifiedAt == nil {
		t.Fatalf("unexpected converted enquiry: %+v", converted)
	}
	if converted.CustomerID != 1 {
		t.Fatalf("expected customer linkage, got %+v", converted)
	}

	_, alreadyConverted, err = store.ConvertToInterested(created.ID, ConvertInput{
		CustomerID: 1,
		ModifiedBy: "system-admin",
	})
	if err != nil {
		t.Fatalf("second convert returned error: %v", err)
	}
	if !alreadyConverted {
		t.Fatalf("expected idempotent already-converted flag")
	}
}

func TestMemoryStoreConvertToOrdered(t *testing.T) {
	store := NewMemoryStore()
	created, err := store.CreateClicked(CreateInput{
		ItemID:     18,
		ItemType:   ItemTypeBundle,
		ItemTitle:  "Bundle Eighteen",
		SourcePage: "catalog",
	})
	if err != nil {
		t.Fatalf("create returned error: %v", err)
	}
	_, _, err = store.ConvertToInterested(created.ID, ConvertInput{
		CustomerID: 99,
		Note:       "Interested",
		ModifiedBy: "system-admin",
	})
	if err != nil {
		t.Fatalf("convert to interested returned error: %v", err)
	}

	ordered, alreadyOrdered, err := store.ConvertToOrdered(created.ID, OrderInput{
		OrderAmount: 499,
		Note:        "Paid in cash",
		ModifiedBy:  "system-admin",
	})
	if err != nil {
		t.Fatalf("convert to ordered returned error: %v", err)
	}
	if alreadyOrdered {
		t.Fatalf("expected first order conversion to not be already ordered")
	}
	if ordered.Status != StatusOrdered {
		t.Fatalf("expected ordered status, got %q", ordered.Status)
	}
	if ordered.OrderAmount == nil || *ordered.OrderAmount != 499 {
		t.Fatalf("expected order amount 499, got %+v", ordered.OrderAmount)
	}
	if ordered.LastModifiedAt == nil {
		t.Fatalf("expected modified timestamp to be set")
	}

	_, alreadyOrdered, err = store.ConvertToOrdered(created.ID, OrderInput{
		OrderAmount: 999,
		Note:        "Should not overwrite",
		ModifiedBy:  "system-admin",
	})
	if err != nil {
		t.Fatalf("second convert to ordered returned error: %v", err)
	}
	if !alreadyOrdered {
		t.Fatalf("expected idempotent already-ordered flag")
	}
}

func TestMemoryStoreConvertToOrderedRejectsInvalidTransition(t *testing.T) {
	store := NewMemoryStore()
	created, err := store.CreateClicked(CreateInput{
		ItemID:     22,
		ItemType:   ItemTypeBook,
		ItemTitle:  "Book Twenty-Two",
		SourcePage: "catalog",
	})
	if err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	_, _, err = store.ConvertToOrdered(created.ID, OrderInput{
		OrderAmount: 199,
		ModifiedBy:  "system-admin",
	})
	if err != ErrInvalidTransition {
		t.Fatalf("expected ErrInvalidTransition, got %v", err)
	}
}

func TestStatusAndItemTypeValidation(t *testing.T) {
	if !IsValidItemType(ItemTypeBook) || !IsValidItemType(ItemTypeBundle) || IsValidItemType(ItemType("X")) {
		t.Fatalf("unexpected item-type validation behavior")
	}
	if !IsValidStatus(StatusClicked) || !IsValidStatus(StatusInterested) || !IsValidStatus(StatusOrdered) || IsValidStatus(Status("unknown")) {
		t.Fatalf("unexpected status validation behavior")
	}
}
