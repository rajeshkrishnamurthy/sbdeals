package web

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/rajeshkrishnamurthy/sbdeals/internal/books"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/bundles"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/rails"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/suppliers"
)

func TestRailsListRendersColumnsAndActions(t *testing.T) {
	s, railStore, _, _, _ := newRailsFixture(t)
	if _, err := railStore.Create(rails.CreateInput{Title: "Top Rail", Type: rails.RailTypeBook}); err != nil {
		t.Fatalf("create rail: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/rails", nil)
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	checks := []string{"Rails", "Add Rail", "Rail Title", "Rail Type", "# Items", "Status", "Order", "Move Up", "Move Down", "View/Edit"}
	for _, check := range checks {
		if !strings.Contains(body, check) {
			t.Fatalf("expected body to contain %q", check)
		}
	}
	assertAdminNav(t, body, "/admin/rails")
}

func TestCreateRailAndDuplicateTitleValidation(t *testing.T) {
	s, railStore, _, _, _ := newRailsFixture(t)

	form := url.Values{}
	form.Set("title", "Weekend Picks")
	form.Set("type", "BOOK")
	form.Set("admin_note", "Internal note")
	req := httptest.NewRequest(http.MethodPost, "/admin/rails", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", rr.Code)
	}
	if rr.Header().Get("Location") != "/admin/rails?flash=Rail+created+successfully." {
		t.Fatalf("unexpected redirect: %s", rr.Header().Get("Location"))
	}

	created, err := railStore.Get(1)
	if err != nil {
		t.Fatalf("expected created rail: %v", err)
	}
	if created.Type != rails.RailTypeBook {
		t.Fatalf("expected BOOK type, got %s", created.Type)
	}
	if created.AdminNote != "Internal note" {
		t.Fatalf("expected admin note to persist, got %q", created.AdminNote)
	}

	dupReq := httptest.NewRequest(http.MethodPost, "/admin/rails", strings.NewReader(form.Encode()))
	dupReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	dupRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(dupRR, dupReq)
	if dupRR.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", dupRR.Code)
	}
	if !strings.Contains(dupRR.Body.String(), "Rail title must be unique.") {
		t.Fatalf("expected duplicate-title validation message")
	}
	if !strings.Contains(dupRR.Body.String(), `class="toast-error"`) {
		t.Fatalf("expected validation toast")
	}
}

func TestRailEditKeepsTypeImmutableAndSupportsPublishRecency(t *testing.T) {
	s, railStore, _, _, _ := newRailsFixture(t)
	created, err := railStore.Create(rails.CreateInput{Title: "Top Rail", Type: rails.RailTypeBook})
	if err != nil {
		t.Fatalf("create rail: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/rails/"+strconv.Itoa(created.ID), nil)
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Rail Type (immutable)") {
		t.Fatalf("expected immutable type label")
	}
	if !regexp.MustCompile(`\(\d+d\)`).MatchString(body) {
		t.Fatalf("expected recency indicator")
	}
	if !strings.Contains(body, "/assets/rails-form.js") {
		t.Fatalf("expected rails search script in detail")
	}

	form := url.Values{}
	form.Set("title", "Renamed Rail")
	form.Set("type", "BUNDLE")
	form.Set("admin_note", "Updated note")
	updateReq := httptest.NewRequest(http.MethodPost, "/admin/rails/"+strconv.Itoa(created.ID), strings.NewReader(form.Encode()))
	updateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	updateRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(updateRR, updateReq)
	if updateRR.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", updateRR.Code)
	}

	updated, err := railStore.Get(created.ID)
	if err != nil {
		t.Fatalf("get updated rail: %v", err)
	}
	if updated.Title != "Renamed Rail" {
		t.Fatalf("expected updated title, got %q", updated.Title)
	}
	if updated.Type != rails.RailTypeBook {
		t.Fatalf("expected type to remain BOOK, got %s", updated.Type)
	}
	if updated.AdminNote != "Updated note" {
		t.Fatalf("expected updated admin note, got %q", updated.AdminNote)
	}
}

func TestRailPublishUnpublishAndMoveOrdering(t *testing.T) {
	s, railStore, _, _, _ := newRailsFixture(t)
	first, err := railStore.Create(rails.CreateInput{Title: "First", Type: rails.RailTypeBook})
	if err != nil {
		t.Fatalf("create first: %v", err)
	}
	second, err := railStore.Create(rails.CreateInput{Title: "Second", Type: rails.RailTypeBundle})
	if err != nil {
		t.Fatalf("create second: %v", err)
	}

	publishReq := httptest.NewRequest(http.MethodPost, "/admin/rails/"+strconv.Itoa(first.ID)+"/publish", nil)
	publishRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(publishRR, publishReq)
	if publishRR.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", publishRR.Code)
	}
	if publishRR.Header().Get("Location") != "/admin/rails?flash=Rail+published+successfully." {
		t.Fatalf("unexpected publish redirect: %s", publishRR.Header().Get("Location"))
	}
	published, err := railStore.Get(first.ID)
	if err != nil {
		t.Fatalf("get published: %v", err)
	}
	if !published.IsPublished {
		t.Fatalf("expected published rail")
	}

	unpublishReq := httptest.NewRequest(http.MethodPost, "/admin/rails/"+strconv.Itoa(first.ID)+"/unpublish?from=edit", nil)
	unpublishRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(unpublishRR, unpublishReq)
	if unpublishRR.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", unpublishRR.Code)
	}
	if unpublishRR.Header().Get("Location") != "/admin/rails/"+strconv.Itoa(first.ID)+"?flash=Rail+unpublished+successfully." {
		t.Fatalf("unexpected unpublish redirect: %s", unpublishRR.Header().Get("Location"))
	}

	moveReq := httptest.NewRequest(http.MethodPost, "/admin/rails/"+strconv.Itoa(second.ID)+"/move-up", nil)
	moveRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(moveRR, moveReq)
	if moveRR.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", moveRR.Code)
	}
	list, err := railStore.List()
	if err != nil {
		t.Fatalf("list rails: %v", err)
	}
	if len(list) < 2 || list[0].ID != second.ID {
		t.Fatalf("expected second rail moved to top, got %+v", list)
	}
}

func TestRailItemsAddAndRemove(t *testing.T) {
	s, railStore, _, _, _ := newRailsFixture(t)
	rail, err := railStore.Create(rails.CreateInput{Title: "Books Rail", Type: rails.RailTypeBook})
	if err != nil {
		t.Fatalf("create rail: %v", err)
	}

	addRR := postRailItemRequest(s, rail.ID, "add", "1")
	if addRR.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", addRR.Code)
	}
	added, err := railStore.Get(rail.ID)
	if err != nil {
		t.Fatalf("get rail: %v", err)
	}
	if len(added.ItemIDs) != 1 || added.ItemIDs[0] != 1 {
		t.Fatalf("unexpected item IDs after add: %+v", added.ItemIDs)
	}

	removeRR := postRailItemRequest(s, rail.ID, "remove", "1")
	if removeRR.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", removeRR.Code)
	}
	removed, err := railStore.Get(rail.ID)
	if err != nil {
		t.Fatalf("get removed rail: %v", err)
	}
	if len(removed.ItemIDs) != 0 {
		t.Fatalf("expected no items after remove, got %+v", removed.ItemIDs)
	}
}

func TestRailItemsDuplicateAddShowsValidationRedirect(t *testing.T) {
	s, railStore, _, _, _ := newRailsFixture(t)
	rail, err := railStore.Create(rails.CreateInput{Title: "Books Rail", Type: rails.RailTypeBook})
	if err != nil {
		t.Fatalf("create rail: %v", err)
	}

	firstRR := postRailItemRequest(s, rail.ID, "add", "1")
	if firstRR.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", firstRR.Code)
	}

	dupRR := postRailItemRequest(s, rail.ID, "add", "1")
	if !strings.HasPrefix(dupRR.Header().Get("Location"), "/admin/rails/"+strconv.Itoa(rail.ID)+"?error=") {
		t.Fatalf("expected duplicate error redirect, got %s", dupRR.Header().Get("Location"))
	}
}

func TestRailItemsTypeMismatchShowsValidationToast(t *testing.T) {
	s, railStore, _, _, _ := newRailsFixture(t)
	rail, err := railStore.Create(rails.CreateInput{Title: "Books Rail", Type: rails.RailTypeBook})
	if err != nil {
		t.Fatalf("create rail: %v", err)
	}

	mismatchRR := postRailItemRequest(s, rail.ID, "add", "3")
	if mismatchRR.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", mismatchRR.Code)
	}
	mismatchLocation := mismatchRR.Header().Get("Location")
	if !strings.HasPrefix(mismatchLocation, "/admin/rails/"+strconv.Itoa(rail.ID)+"?error=") {
		t.Fatalf("expected mismatch error redirect, got %s", mismatchLocation)
	}

	getMismatchReq := httptest.NewRequest(http.MethodGet, mismatchLocation, nil)
	getMismatchRR := httptest.NewRecorder()
	s.Handler().ServeHTTP(getMismatchRR, getMismatchReq)
	if !strings.Contains(getMismatchRR.Body.String(), "Type mismatch: item does not match rail type.") {
		t.Fatalf("expected mismatch toast message")
	}
}

func TestRailItemsAllowSameItemAcrossRails(t *testing.T) {
	s, railStore, _, _, _ := newRailsFixture(t)
	first, err := railStore.Create(rails.CreateInput{Title: "Books Rail", Type: rails.RailTypeBook})
	if err != nil {
		t.Fatalf("create first rail: %v", err)
	}
	second, err := railStore.Create(rails.CreateInput{Title: "Another Books Rail", Type: rails.RailTypeBook})
	if err != nil {
		t.Fatalf("create second rail: %v", err)
	}

	firstRR := postRailItemRequest(s, first.ID, "add", "1")
	if firstRR.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", firstRR.Code)
	}
	secondRR := postRailItemRequest(s, second.ID, "add", "1")
	if secondRR.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", secondRR.Code)
	}

	updated, err := railStore.Get(second.ID)
	if err != nil {
		t.Fatalf("get second rail: %v", err)
	}
	if len(updated.ItemIDs) != 1 || updated.ItemIDs[0] != 1 {
		t.Fatalf("expected same item to be addable in multiple rails, got %+v", updated.ItemIDs)
	}
}

func TestRailItemPickerBundleFiltersApplyDeterministically(t *testing.T) {
	s, railStore, _, _, _ := newRailsFixture(t)
	rail, err := railStore.Create(rails.CreateInput{Title: "Bundle Rail", Type: rails.RailTypeBundle})
	if err != nil {
		t.Fatalf("create rail: %v", err)
	}

	// Apply explicit filter set that should match only "Bundle Two".
	path := "/admin/rails/" + strconv.Itoa(rail.ID) + "?q=Two&category=Fiction&priceMin=290&priceMax=305&discountMin=40&discountMax=50"
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Bundle Two") {
		t.Fatalf("expected filtered result to include Bundle Two")
	}
	if strings.Contains(body, "Bundle One") {
		t.Fatalf("expected Bundle One to be filtered out")
	}
	if !strings.Contains(body, "Apply Filters") {
		t.Fatalf("expected explicit apply filters control")
	}
}

func TestRailItemPickerInvalidRangeShowsInlineError(t *testing.T) {
	s, railStore, _, _, _ := newRailsFixture(t)
	rail, err := railStore.Create(rails.CreateInput{Title: "Bundle Rail", Type: rails.RailTypeBundle})
	if err != nil {
		t.Fatalf("create rail: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/rails/"+strconv.Itoa(rail.ID)+"?priceMin=400&priceMax=300", nil)
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Minimum price cannot be greater than maximum price.") {
		t.Fatalf("expected inline range validation error")
	}
	if !strings.Contains(body, `class="toast-error"`) {
		t.Fatalf("expected range validation toast")
	}
}

func TestRailItemPickerExcludesAlreadyAddedItems(t *testing.T) {
	s, railStore, _, _, _ := newRailsFixture(t)
	rail, err := railStore.Create(rails.CreateInput{Title: "Books Rail", Type: rails.RailTypeBook})
	if err != nil {
		t.Fatalf("create rail: %v", err)
	}

	addRR := postRailItemRequest(s, rail.ID, "add", "1")
	if addRR.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", addRR.Code)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/rails/"+strconv.Itoa(rail.ID), nil)
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	addPattern := regexp.MustCompile(fmt.Sprintf(`(?s)<form method="post" action="/admin/rails/%d/items/add">\s*<input type="hidden" name="item_id" value="1"`, rail.ID))
	if addPattern.MatchString(body) {
		t.Fatalf("expected already-added item to be excluded from available picker list")
	}
	removePattern := regexp.MustCompile(fmt.Sprintf(`(?s)<form method="post" action="/admin/rails/%d/items/remove">\s*<input type="hidden" name="item_id" value="1"`, rail.ID))
	if !removePattern.MatchString(body) {
		t.Fatalf("expected already-added item to remain in selected list")
	}
}

func TestRailItemPickerRendersCommonColumnsAndHoverPreview(t *testing.T) {
	s, railStore, _, _, _ := newRailsFixture(t)
	rail, err := railStore.Create(rails.CreateInput{Title: "Books Rail", Type: rails.RailTypeBook})
	if err != nil {
		t.Fatalf("create rail: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/rails/"+strconv.Itoa(rail.ID), nil)
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	checks := []string{"Image", "Title", "Category", "Price", "Discount", "thumb-preview", "Reset Filters", "Admin Note"}
	for _, check := range checks {
		if !strings.Contains(body, check) {
			t.Fatalf("expected body to contain %q", check)
		}
	}
	if !strings.Contains(body, `action="/admin/rails/`+strconv.Itoa(rail.ID)+`#rail-picker"`) {
		t.Fatalf("expected apply filters form to keep user anchored to picker area")
	}
}

func TestRailsFormAssetServesScript(t *testing.T) {
	s, _, _, _, _ := newRailsFixture(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/assets/rails-form.js", bytes.NewBuffer(nil))
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if strings.TrimSpace(rr.Body.String()) == "" {
		t.Fatalf("expected rails-form.js content")
	}
}

func TestRailPickerDiscountMinUsesRoundedPercent(t *testing.T) {
	min := 60.0
	max := 60.0
	criteria := railPickerFilterCriteria{
		DiscountMin: &min,
		DiscountMax: &max,
	}

	shouldMatch := railItemOption{DiscountPct: 59.6}
	if !railItemMatchesFilters(shouldMatch, criteria, rails.RailTypeBundle) {
		t.Fatalf("expected rounded 59.6%% discount to match 60-60 range")
	}

	shouldNotMatch := railItemOption{DiscountPct: 59.4}
	if railItemMatchesFilters(shouldNotMatch, criteria, rails.RailTypeBundle) {
		t.Fatalf("expected rounded 59.4%% discount to not match 60-60 range")
	}
}

func postRailItemRequest(s *Server, railID int, action string, itemID string) *httptest.ResponseRecorder {
	form := url.Values{}
	form.Set("item_id", itemID)
	req := httptest.NewRequest(http.MethodPost, "/admin/rails/"+strconv.Itoa(railID)+"/items/"+action, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	return rr
}

func newRailsFixture(t *testing.T) (*Server, *rails.MemoryStore, *books.MemoryStore, *bundles.MemoryStore, *suppliers.MemoryStore) {
	t.Helper()

	supplierStore := suppliers.NewMemoryStore()
	supplier, err := supplierStore.Create(suppliers.Input{Name: "Supplier A", WhatsApp: "+91-1", Location: "Bengaluru"})
	if err != nil {
		t.Fatalf("create supplier failed: %v", err)
	}

	bookStore := books.NewMemoryStore()
	_, err = bookStore.Create(books.CreateInput{
		Title:      "Book One",
		Cover:      books.Cover{Data: []byte("img"), MimeType: "image/png"},
		SupplierID: supplier.ID,
		Category:   "Fiction",
		Format:     "Paperback",
		Condition:  "Very good",
		MRP:        300,
		MyPrice:    200,
	})
	if err != nil {
		t.Fatalf("create book failed: %v", err)
	}
	_, err = bookStore.Create(books.CreateInput{
		Title:      "Book Two",
		Cover:      books.Cover{Data: []byte("img2"), MimeType: "image/png"},
		SupplierID: supplier.ID,
		Category:   "Children",
		Format:     "Paperback",
		Condition:  "Very good",
		MRP:        150,
		MyPrice:    90,
	})
	if err != nil {
		t.Fatalf("create book two failed: %v", err)
	}

	bundleStore := bundles.NewMemoryStore(
		map[int]string{supplier.ID: supplier.Name},
		[]bundles.PickerBook{
			{BookID: 10, Title: "Bundle Book A", SupplierID: supplier.ID, Category: "Fiction", Condition: "Very good", MRP: 300, MyPrice: 200, InStock: true},
			{BookID: 11, Title: "Bundle Book B", SupplierID: supplier.ID, Category: "Fiction", Condition: "Very good", MRP: 250, MyPrice: 180, InStock: true},
			{BookID: 12, Title: "Bundle Book C", SupplierID: supplier.ID, Category: "Children", Condition: "Very good", MRP: 150, MyPrice: 100, InStock: true},
			{BookID: 13, Title: "Bundle Book D", SupplierID: supplier.ID, Category: "Children", Condition: "Very good", MRP: 130, MyPrice: 90, InStock: true},
		},
	)
	if _, err := bundleStore.Create(bundles.CreateInput{
		Name:              "Bundle One",
		SupplierID:        supplier.ID,
		Category:          "Fiction",
		AllowedConditions: []string{"Very good"},
		BookIDs:           []int{10, 11},
		BundlePrice:       320,
	}); err != nil {
		t.Fatalf("create bundle one failed: %v", err)
	}
	if _, err := bundleStore.Create(bundles.CreateInput{
		Name:              "Bundle Two",
		SupplierID:        supplier.ID,
		Category:          "Fiction",
		AllowedConditions: []string{"Very good"},
		BookIDs:           []int{10, 11},
		BundlePrice:       300,
	}); err != nil {
		t.Fatalf("create bundle two failed: %v", err)
	}
	if _, err := bundleStore.Create(bundles.CreateInput{
		Name:              "Bundle Three",
		SupplierID:        supplier.ID,
		Category:          "Children",
		AllowedConditions: []string{"Very good"},
		BookIDs:           []int{12, 13},
		BundlePrice:       220,
	}); err != nil {
		t.Fatalf("create bundle three failed: %v", err)
	}

	railStore := rails.NewMemoryStore()
	server := NewServerWithStores(supplierStore, bookStore, bundleStore, railStore)
	return server, railStore, bookStore, bundleStore, supplierStore
}
