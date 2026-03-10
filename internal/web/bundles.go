package web

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/rajeshkrishnamurthy/sbdeals/internal/bundles"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/suppliers"
)

var bundleValidationFieldOrder = []string{
	"image",
	"supplier_id",
	"category",
	"allowed_conditions",
	"book_ids",
	"bundle_price",
	"out_of_stock_on_interested",
}

func (s *Server) handleBundlesCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.renderBundlesList(w, r)
	case http.MethodPost:
		s.createBundle(w, r)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (s *Server) handleBundleNew(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	suppliersList, pickerBooks, err := s.loadBundleFormDependencies()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.renderBundleForm(w, buildBundleFormView(bundleFormViewOptions{
		PageTitle:        "Add Bundle",
		Action:           "/admin/bundles",
		SubmitLabel:      "Save Bundle",
		ActiveSection:    "bundles",
		Input:            bundleFormInput{OutOfStockOnInterested: "yes"},
		SupplierOptions:  suppliersList,
		CandidateBooks:   pickerBooks,
		SelectedBooks:    []bundles.PickerBook{},
		Errors:           map[string]string{},
		HasExistingImage: false,
		BundleID:         0,
		ShowSummary:      false,
	}))
}

func (s *Server) handleBundleItem(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/admin/bundles/" {
		http.Redirect(w, r, "/admin/bundles", http.StatusMovedPermanently)
		return
	}

	bundleID, action, ok := parseBundlePath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	switch action {
	case "image":
		s.handleBundleImageAction(w, r, bundleID)
	case "publish":
		s.handleBundlePublishAction(w, r, bundleID)
	case "unpublish":
		s.handleBundleUnpublishAction(w, r, bundleID)
	case "":
		s.handleBundleDetailAction(w, r, bundleID)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleBundleImageAction(w http.ResponseWriter, r *http.Request, bundleID int) {
	if !methodAllowed(r.Method, http.MethodGet) {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	s.serveBundleImage(w, r, bundleID)
}

func (s *Server) handleBundlePublishAction(w http.ResponseWriter, r *http.Request, bundleID int) {
	if !methodAllowed(r.Method, http.MethodPost, http.MethodPatch) {
		writeMethodNotAllowed(w, http.MethodPost, http.MethodPatch)
		return
	}
	s.publishBundle(w, r, bundleID)
}

func (s *Server) handleBundleUnpublishAction(w http.ResponseWriter, r *http.Request, bundleID int) {
	if !methodAllowed(r.Method, http.MethodPost, http.MethodPatch) {
		writeMethodNotAllowed(w, http.MethodPost, http.MethodPatch)
		return
	}
	s.unpublishBundle(w, r, bundleID)
}

func (s *Server) handleBundleDetailAction(w http.ResponseWriter, r *http.Request, bundleID int) {
	switch r.Method {
	case http.MethodGet:
		s.renderBundleDetail(w, r, bundleID)
	case http.MethodPost:
		s.updateBundle(w, r, bundleID)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (s *Server) renderBundlesList(w http.ResponseWriter, r *http.Request) {
	items, err := s.bundleStore.List()
	if err != nil {
		http.Error(w, "failed to load bundles", http.StatusInternalServerError)
		return
	}

	view := bundlesListViewModel{
		Flash:           r.URL.Query().Get("flash"),
		ValidationToast: strings.TrimSpace(r.URL.Query().Get("error")),
		ActiveSection:   "bundles",
		Bundles:         items,
	}
	if err := bundlesListTemplate.Execute(w, view); err != nil {
		http.Error(w, "failed to render bundles list", http.StatusInternalServerError)
	}
}

func (s *Server) createBundle(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}

	suppliersList, pickerBooks, err := s.loadBundleFormDependencies()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	input := readBundleFormInput(r)
	image, imageProvided, err := readBundleImageFromRequest(r)
	if err != nil {
		http.Error(w, "failed to read bundle image", http.StatusBadRequest)
		return
	}
	parsed, fieldErrors, selectedBooks := validateBundleFormInput(input, suppliersList, pickerBooks, true, imageProvided)
	if len(fieldErrors) > 0 {
		w.WriteHeader(http.StatusBadRequest)
		s.renderBundleForm(w, buildBundleFormView(bundleFormViewOptions{
			PageTitle:       "Add Bundle",
			Action:          "/admin/bundles",
			SubmitLabel:     "Save Bundle",
			ActiveSection:   "bundles",
			Input:           input,
			SupplierOptions: suppliersList,
			CandidateBooks:  pickerBooks,
			SelectedBooks:   selectedBooks,
			Errors:          fieldErrors,
			ValidationToast: buildValidationToast(fieldErrors, bundleValidationFieldOrder),
			ShowSummary:     false,
		}))
		return
	}

	_, err = s.bundleStore.Create(bundles.CreateInput{
		Name:                   parsed.Name,
		SupplierID:             parsed.SupplierID,
		Category:               parsed.Category,
		AllowedConditions:      parsed.AllowedConditions,
		BookIDs:                parsed.BookIDs,
		BundlePrice:            parsed.BundlePrice,
		Notes:                  parsed.Notes,
		Image:                  image,
		OutOfStockOnInterested: parsed.OutOfStockOnInterested,
	})
	if err != nil {
		http.Error(w, "failed to create bundle", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/bundles?flash="+url.QueryEscape("Bundle created successfully."), http.StatusSeeOther)
}

func (s *Server) renderBundleDetail(w http.ResponseWriter, r *http.Request, bundleID int) {
	bundle, err := s.bundleStore.Get(bundleID)
	if errors.Is(err, bundles.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "failed to load bundle", http.StatusInternalServerError)
		return
	}

	suppliersList, pickerBooks, err := s.loadBundleFormDependencies()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	selectedBooks := toPickerBooksFromBundle(bundle.Books)
	input := bundleFormInput{
		Name:                   bundle.Name,
		SupplierID:             strconv.Itoa(bundle.SupplierID),
		Category:               bundle.Category,
		AllowedConditions:      append([]string(nil), bundle.AllowedConditions...),
		BookIDValues:           intSliceToStringSlice(bundle.BookIDs),
		BundlePrice:            formatDecimal(bundle.BundlePrice),
		Notes:                  bundle.Notes,
		OutOfStockOnInterested: boolToYesNo(bundle.OutOfStockOnInterested),
	}

	summary := &bundleSummaryViewModel{
		Label:        bundleLabel(bundle.Name, bundle.ID),
		SupplierName: bundle.SupplierName,
		Category:     bundle.Category,
		BookCount:    len(bundle.BookIDs),
		BundlePrice:  formatDecimal(bundle.BundlePrice),
	}

	s.renderBundleForm(w, buildBundleFormView(bundleFormViewOptions{
		PageTitle:         "View/Edit Bundle",
		Action:            fmt.Sprintf("/admin/bundles/%d", bundleID),
		SubmitLabel:       "Save Changes",
		ActiveSection:     "bundles",
		Flash:             r.URL.Query().Get("flash"),
		ValidationToast:   strings.TrimSpace(r.URL.Query().Get("error")),
		Input:             input,
		SupplierOptions:   suppliersList,
		CandidateBooks:    pickerBooks,
		SelectedBooks:     selectedBooks,
		Errors:            map[string]string{},
		HasExistingImage:  strings.TrimSpace(bundle.ImageMimeType) != "",
		BundleID:          bundleID,
		ShowSummary:       true,
		ShowPublishToggle: true,
		PublishAction:     fmt.Sprintf("/admin/bundles/%d/%s?from=edit", bundleID, toggleActionPath(bundle.IsPublished)),
		PublishLabel:      publishStateLabel(bundle.IsPublished),
		PublishRecency:    publishRecencyLabel(bundle.IsPublished, bundle.PublishedAt, bundle.UnpublishedAt),
		Summary:           summary,
	}))
}

func (s *Server) updateBundle(w http.ResponseWriter, r *http.Request, bundleID int) {
	currentBundle, err := s.bundleStore.Get(bundleID)
	if errors.Is(err, bundles.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "failed to load bundle", http.StatusInternalServerError)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}

	suppliersList, pickerBooks, err := s.loadBundleFormDependencies()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	input := readBundleFormInput(r)
	image, imageProvided, err := readBundleImageFromRequest(r)
	if err != nil {
		http.Error(w, "failed to read bundle image", http.StatusBadRequest)
		return
	}
	parsed, fieldErrors, selectedBooks := validateBundleFormInput(input, suppliersList, pickerBooks, false, imageProvided)
	if len(fieldErrors) > 0 {
		w.WriteHeader(http.StatusBadRequest)
		summary := &bundleSummaryViewModel{
			Label:        bundleLabel(input.Name, bundleID),
			SupplierName: supplierNameByIDFromSuppliers(input.SupplierID, suppliersList),
			Category:     input.Category,
			BookCount:    len(selectedBooks),
			BundlePrice:  input.BundlePrice,
		}
		s.renderBundleForm(w, buildBundleFormView(bundleFormViewOptions{
			PageTitle:         "View/Edit Bundle",
			Action:            fmt.Sprintf("/admin/bundles/%d", bundleID),
			SubmitLabel:       "Save Changes",
			ActiveSection:     "bundles",
			Input:             input,
			SupplierOptions:   suppliersList,
			CandidateBooks:    pickerBooks,
			SelectedBooks:     selectedBooks,
			Errors:            fieldErrors,
			ValidationToast:   buildValidationToast(fieldErrors, bundleValidationFieldOrder),
			HasExistingImage:  strings.TrimSpace(currentBundle.ImageMimeType) != "",
			BundleID:          bundleID,
			ShowSummary:       true,
			ShowPublishToggle: true,
			PublishAction:     fmt.Sprintf("/admin/bundles/%d/%s?from=edit", bundleID, toggleActionPath(currentBundle.IsPublished)),
			PublishLabel:      publishStateLabel(currentBundle.IsPublished),
			PublishRecency:    publishRecencyLabel(currentBundle.IsPublished, currentBundle.PublishedAt, currentBundle.UnpublishedAt),
			Summary:           summary,
		}))
		return
	}

	_, err = s.bundleStore.Update(bundleID, bundles.UpdateInput{
		Name:                   parsed.Name,
		SupplierID:             parsed.SupplierID,
		Category:               parsed.Category,
		AllowedConditions:      parsed.AllowedConditions,
		BookIDs:                parsed.BookIDs,
		BundlePrice:            parsed.BundlePrice,
		Notes:                  parsed.Notes,
		Image:                  optionalBundleImage(image, imageProvided),
		OutOfStockOnInterested: parsed.OutOfStockOnInterested,
	})
	if err != nil {
		if errors.Is(err, bundles.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "failed to update bundle", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/bundles/%d?flash=%s", bundleID, url.QueryEscape("Bundle updated successfully.")), http.StatusSeeOther)
}

func (s *Server) publishBundle(w http.ResponseWriter, r *http.Request, bundleID int) {
	if _, err := s.bundleStore.Publish(bundleID); err != nil {
		if errors.Is(err, bundles.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		var outOfStockErr *bundles.ErrCannotPublishWithOutOfStockBooks
		if errors.As(err, &outOfStockErr) {
			http.Redirect(w, r, bundlePublishRedirectPath(r, bundleID, "", bundleOutOfStockMessage(outOfStockErr.BookTitles)), http.StatusSeeOther)
			return
		}
		http.Error(w, "failed to publish bundle", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, bundlePublishRedirectPath(r, bundleID, "Bundle published successfully.", ""), http.StatusSeeOther)
}

func (s *Server) unpublishBundle(w http.ResponseWriter, r *http.Request, bundleID int) {
	if _, err := s.bundleStore.Unpublish(bundleID); err != nil {
		if errors.Is(err, bundles.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "failed to unpublish bundle", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, bundlePublishRedirectPath(r, bundleID, "Bundle unpublished successfully.", ""), http.StatusSeeOther)
}

func optionalBundleImage(image bundles.Image, provided bool) *bundles.Image {
	if !provided {
		return nil
	}
	copyImage := image
	return &copyImage
}

func bundlePublishRedirectPath(r *http.Request, bundleID int, flash string, errorMessage string) string {
	base := "/admin/bundles"
	if r.URL.Query().Get("from") == "edit" {
		base = fmt.Sprintf("/admin/bundles/%d", bundleID)
	}
	switch {
	case flash != "":
		return base + "?flash=" + url.QueryEscape(flash)
	case errorMessage != "":
		return base + "?error=" + url.QueryEscape(errorMessage)
	default:
		return base
	}
}

func bundleOutOfStockMessage(titles []string) string {
	if len(titles) == 0 {
		return "Cannot publish bundle because one or more included books are out of stock."
	}
	return "Cannot publish bundle because these books are out of stock: " + strings.Join(titles, ", ") + "."
}

func (s *Server) loadBundleFormDependencies() ([]suppliers.Supplier, []bundles.PickerBook, error) {
	suppliersList, err := s.store.List()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load suppliers")
	}
	pickerBooks, err := s.bundleStore.ListBooksForPicker()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load books")
	}
	return suppliersList, pickerBooks, nil
}

func readBundleFormInput(r *http.Request) bundleFormInput {
	bookIDs := r.Form["book_ids"]
	allowedConditions := r.Form["allowed_conditions"]
	sort.Strings(allowedConditions)
	return bundleFormInput{
		Name:                   strings.TrimSpace(r.Form.Get("name")),
		SupplierID:             strings.TrimSpace(r.Form.Get("supplier_id")),
		Category:               strings.TrimSpace(r.Form.Get("category")),
		AllowedConditions:      trimAndCompact(allowedConditions),
		BookIDValues:           trimAndCompact(bookIDs),
		BundlePrice:            strings.TrimSpace(r.Form.Get("bundle_price")),
		Notes:                  strings.TrimSpace(r.Form.Get("notes")),
		OutOfStockOnInterested: strings.TrimSpace(r.Form.Get("out_of_stock_on_interested")),
	}
}

func readBundleImageFromRequest(r *http.Request) (bundles.Image, bool, error) {
	file, fileHeader, err := r.FormFile("image")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return bundles.Image{}, false, nil
		}
		return bundles.Image{}, false, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return bundles.Image{}, false, err
	}
	mimeType := strings.TrimSpace(fileHeader.Header.Get("Content-Type"))
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}
	return bundles.Image{Data: data, MimeType: mimeType}, true, nil
}

type parsedBundleForm struct {
	Name                   string
	SupplierID             int
	Category               string
	AllowedConditions      []string
	BookIDs                []int
	BundlePrice            float64
	Notes                  string
	OutOfStockOnInterested bool
}

func validateBundleFormInput(input bundleFormInput, suppliersList []suppliers.Supplier, pickerBooks []bundles.PickerBook, requireImage bool, imageProvided bool) (parsedBundleForm, map[string]string, []bundles.PickerBook) {
	result := parsedBundleForm{
		Name:                   input.Name,
		Category:               input.Category,
		AllowedConditions:      append([]string(nil), input.AllowedConditions...),
		Notes:                  input.Notes,
		OutOfStockOnInterested: true,
	}
	errorsByField := map[string]string{}
	if requireImage && !imageProvided {
		errorsByField["image"] = "Bundle image is required."
	}

	supplierID, supplierErr := parseSupplierIDForBundle(input.SupplierID, suppliersList)
	if supplierErr != "" {
		errorsByField["supplier_id"] = supplierErr
	} else {
		result.SupplierID = supplierID
	}

	if errText := validateOption(input.Category, bookCategoryOptions, "Category is required.", "Please choose a valid category."); errText != "" {
		errorsByField["category"] = errText
	}

	allowedConditions, conditionErr := parseAllowedConditions(input.AllowedConditions)
	if conditionErr != "" {
		errorsByField["allowed_conditions"] = conditionErr
	} else {
		result.AllowedConditions = allowedConditions
	}

	bookIDs, selectedBooks, booksErr := parseAndValidateSelectedBooks(input.BookIDValues, pickerBooks)
	if booksErr != "" {
		errorsByField["book_ids"] = booksErr
	} else {
		result.BookIDs = bookIDs
	}

	if !hasMinimumBundleItems(selectedBooks) {
		errorsByField["book_ids"] = "Minimum 2 items required unless one selected item is marked Box Set."
	}

	bundlePricePtr, priceErr := parseNonNegativeNumber(input.BundlePrice, true)
	if priceErr != "" {
		errorsByField["bundle_price"] = "Bundle price is required and must be a non-negative number."
	} else {
		result.BundlePrice = *bundlePricePtr
	}
	if value, ok := parseRequiredYesNo(input.OutOfStockOnInterested); ok {
		result.OutOfStockOnInterested = value
	} else {
		errorsByField["out_of_stock_on_interested"] = "Please choose a valid Out of stock on interested value."
	}

	if len(errorsByField) == 0 {
		if !selectedBooksMatchFilters(selectedBooks, result.SupplierID, result.Category, result.AllowedConditions) {
			errorsByField["book_ids"] = "Selected books must match the chosen supplier, category, and allowed conditions."
		}
	}

	return result, errorsByField, selectedBooks
}

func parseSupplierIDForBundle(raw string, suppliersList []suppliers.Supplier) (int, string) {
	if strings.TrimSpace(raw) == "" {
		return 0, "Supplier is required."
	}
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		return 0, "Please choose a valid supplier."
	}
	for _, supplier := range suppliersList {
		if supplier.ID == id {
			return id, ""
		}
	}
	return 0, "Please choose a valid supplier."
}

func parseAllowedConditions(values []string) ([]string, string) {
	conditions := trimAndCompact(values)
	if len(conditions) == 0 {
		return nil, "At least one allowed book condition is required."
	}
	for _, condition := range conditions {
		if validateOption(condition, bookConditionOptions, "", "invalid") != "" {
			return nil, "Please choose valid allowed book condition(s)."
		}
	}
	return conditions, ""
}

func parseAndValidateSelectedBooks(values []string, pickerBooks []bundles.PickerBook) ([]int, []bundles.PickerBook, string) {
	bookMap := make(map[int]bundles.PickerBook, len(pickerBooks))
	for _, book := range pickerBooks {
		bookMap[book.BookID] = book
	}

	ids := make([]int, 0, len(values))
	seen := map[int]struct{}{}
	for _, raw := range values {
		id, err := strconv.Atoi(raw)
		if err != nil || id <= 0 {
			return nil, nil, "Please select valid books for the bundle."
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	selected := make([]bundles.PickerBook, 0, len(ids))
	for _, id := range ids {
		book, ok := bookMap[id]
		if !ok {
			return nil, nil, "Please select valid books for the bundle."
		}
		selected = append(selected, book)
	}
	return ids, selected, ""
}

func selectedBooksMatchFilters(selected []bundles.PickerBook, supplierID int, category string, allowedConditions []string) bool {
	allowedSet := map[string]struct{}{}
	for _, condition := range allowedConditions {
		allowedSet[condition] = struct{}{}
	}
	for _, book := range selected {
		if book.SupplierID != supplierID {
			return false
		}
		if book.Category != category {
			return false
		}
		if _, ok := allowedSet[book.Condition]; !ok {
			return false
		}
	}
	return true
}

func hasMinimumBundleItems(selected []bundles.PickerBook) bool {
	if len(selected) >= 2 {
		return true
	}
	if len(selected) != 1 {
		return false
	}
	return selected[0].IsBoxSet
}

func trimAndCompact(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func intSliceToStringSlice(values []int) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, strconv.Itoa(value))
	}
	return out
}

func toPickerBooksFromBundle(bundleBooks []bundles.BundleBook) []bundles.PickerBook {
	items := make([]bundles.PickerBook, 0, len(bundleBooks))
	for _, book := range bundleBooks {
		items = append(items, bundles.PickerBook{
			BookID:      book.BookID,
			Title:       book.Title,
			Author:      book.Author,
			SupplierID:  book.SupplierID,
			IsBoxSet:    book.IsBoxSet,
			Category:    book.Category,
			Condition:   book.Condition,
			MRP:         book.MRP,
			MyPrice:     book.MyPrice,
			BundlePrice: book.BundlePrice,
			InStock:     book.InStock,
		})
	}
	return items
}

func parseBundlePath(path string) (int, string, bool) {
	prefix := "/admin/bundles/"
	if !strings.HasPrefix(path, prefix) {
		return 0, "", false
	}
	rest := strings.TrimPrefix(path, prefix)
	if rest == "" {
		return 0, "", false
	}
	parts := strings.Split(rest, "/")
	if len(parts) > 2 {
		return 0, "", false
	}
	id, err := strconv.Atoi(parts[0])
	if err != nil || id <= 0 {
		return 0, "", false
	}
	if len(parts) == 1 {
		return id, "", true
	}
	action := parts[1]
	if action != "publish" && action != "unpublish" && action != "image" {
		return 0, "", false
	}
	return id, action, true
}

func (s *Server) serveBundleImage(w http.ResponseWriter, r *http.Request, bundleID int) {
	image, err := s.bundleStore.GetImage(bundleID)
	if errors.Is(err, bundles.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "failed to load bundle image", http.StatusInternalServerError)
		return
	}
	mimeType := strings.TrimSpace(image.MimeType)
	if mimeType == "" {
		mimeType = http.DetectContentType(image.Data)
	}
	w.Header().Set("Content-Type", mimeType)
	_, _ = w.Write(image.Data)
}

func bundleLabel(name string, id int) string {
	trimmed := strings.TrimSpace(name)
	if trimmed != "" {
		return trimmed
	}
	return fmt.Sprintf("Bundle #%d", id)
}

func supplierNameByIDFromSuppliers(rawID string, suppliersList []suppliers.Supplier) string {
	id, err := strconv.Atoi(strings.TrimSpace(rawID))
	if err != nil || id <= 0 {
		return ""
	}
	for _, supplier := range suppliersList {
		if supplier.ID == id {
			return supplier.Name
		}
	}
	return ""
}

type bundleTotals struct {
	BundleMRP        float64
	SumMyPrice       float64
	SumMyBundlePrice float64
	Discount         float64
}

func calculateBundleTotals(selected []bundles.PickerBook, bundlePrice float64) bundleTotals {
	totals := bundleTotals{}
	for _, book := range selected {
		totals.BundleMRP += book.MRP
		totals.SumMyPrice += book.MyPrice
		if book.BundlePrice != nil {
			totals.SumMyBundlePrice += *book.BundlePrice
		} else {
			totals.SumMyBundlePrice += book.MyPrice
		}
	}
	if totals.BundleMRP > 0 {
		totals.Discount = ((totals.BundleMRP - bundlePrice) / totals.BundleMRP) * 100
	}
	return totals
}

type bundlesListViewModel struct {
	Flash           string
	ValidationToast string
	ActiveSection   string
	Bundles         []bundles.ListItem
}

type bundleSummaryViewModel struct {
	Label        string
	SupplierName string
	Category     string
	BookCount    int
	BundlePrice  string
}

type bundleFormInput struct {
	Name                   string
	SupplierID             string
	Category               string
	AllowedConditions      []string
	BookIDValues           []string
	BundlePrice            string
	Notes                  string
	OutOfStockOnInterested string
}

type bundleFormViewModel struct {
	PageTitle         string
	Action            string
	SubmitLabel       string
	ActiveSection     string
	Flash             string
	ValidationToast   string
	ShowPublishToggle bool
	PublishAction     string
	PublishLabel      string
	PublishRecency    string
	Input             bundleFormInput
	SupplierOptions   []suppliers.Supplier
	CategoryOptions   []string
	ConditionOptions  []string
	CandidateBooks    []bundles.PickerBook
	SelectedBooks     []bundles.PickerBook
	Errors            map[string]string
	HasExistingImage  bool
	BundleID          int
	BundleMRPText     string
	SumMyPriceText    string
	SumMyBundleText   string
	DiscountText      string
	ShowSummary       bool
	Summary           *bundleSummaryViewModel
}

func (m bundleFormViewModel) HasError(field string) bool {
	_, ok := m.Errors[field]
	return ok
}

func (m bundleFormViewModel) Error(field string) string {
	return m.Errors[field]
}

func (m bundleFormViewModel) ConditionChecked(condition string) bool {
	for _, value := range m.Input.AllowedConditions {
		if value == condition {
			return true
		}
	}
	return false
}

type bundleFormViewOptions struct {
	PageTitle         string
	Action            string
	SubmitLabel       string
	ActiveSection     string
	Flash             string
	ValidationToast   string
	ShowPublishToggle bool
	PublishAction     string
	PublishLabel      string
	PublishRecency    string
	Input             bundleFormInput
	SupplierOptions   []suppliers.Supplier
	CandidateBooks    []bundles.PickerBook
	SelectedBooks     []bundles.PickerBook
	Errors            map[string]string
	HasExistingImage  bool
	BundleID          int
	ShowSummary       bool
	Summary           *bundleSummaryViewModel
}

func buildBundleFormView(options bundleFormViewOptions) bundleFormViewModel {
	bundlePrice := 0.0
	if parsed, errText := parseNonNegativeNumber(options.Input.BundlePrice, false); errText == "" && parsed != nil {
		bundlePrice = *parsed
	}
	totals := calculateBundleTotals(options.SelectedBooks, bundlePrice)

	return bundleFormViewModel{
		PageTitle:         options.PageTitle,
		Action:            options.Action,
		SubmitLabel:       options.SubmitLabel,
		ActiveSection:     options.ActiveSection,
		Flash:             options.Flash,
		ValidationToast:   options.ValidationToast,
		ShowPublishToggle: options.ShowPublishToggle,
		PublishAction:     options.PublishAction,
		PublishLabel:      options.PublishLabel,
		PublishRecency:    options.PublishRecency,
		Input:             options.Input,
		SupplierOptions:   options.SupplierOptions,
		CategoryOptions:   bookCategoryOptions,
		ConditionOptions:  bookConditionOptions,
		CandidateBooks:    options.CandidateBooks,
		SelectedBooks:     options.SelectedBooks,
		Errors:            options.Errors,
		HasExistingImage:  options.HasExistingImage,
		BundleID:          options.BundleID,
		BundleMRPText:     formatDecimal(totals.BundleMRP),
		SumMyPriceText:    formatDecimal(totals.SumMyPrice),
		SumMyBundleText:   formatDecimal(totals.SumMyBundlePrice),
		DiscountText:      formatRoundedDiscount(totals.Discount),
		ShowSummary:       options.ShowSummary,
		Summary:           options.Summary,
	}
}

func (s *Server) renderBundleForm(w http.ResponseWriter, data bundleFormViewModel) {
	if err := bundleFormTemplate.Execute(w, data); err != nil {
		http.Error(w, "failed to render bundle form", http.StatusInternalServerError)
	}
}

func formatBundleDiscountPercent(mrp float64, bundlePrice float64) string {
	if mrp <= 0 {
		return "—"
	}
	discount := ((mrp - bundlePrice) / mrp) * 100
	return fmt.Sprintf("%d%%", int(math.Round(discount)))
}

var bundlesListTemplate = template.Must(template.New("bundles-list").Funcs(template.FuncMap{
	"adminNav":         adminNav,
	"bundleLabel":      bundleLabel,
	"money":            func(v float64) string { return fmt.Sprintf("%.2f", v) },
	"bundleDiscount":   formatBundleDiscountPercent,
	"publishRecency":   publishRecencyLabel,
	"publishState":     publishStateLabel,
	"toggleActionPath": toggleActionPath,
}).Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Bundles</title>
  <style>
    :root { --bg:#f6f8fb; --card:#fff; --line:#d9e1ea; --text:#1f2937; --accent:#0f766e; --muted:#4b5563; }
    * { box-sizing: border-box; }
    body { margin:0; font-family: "Segoe UI", Tahoma, sans-serif; background: var(--bg); color: var(--text); }
    header { background: var(--card); border-bottom:1px solid var(--line); }
    .shell { width:min(1100px, 94vw); margin:0 auto; padding:16px; }
    .admin-nav { display:flex; gap:14px; }
    .admin-nav-link { color: var(--accent); font-weight:600; text-decoration:none; padding:6px 10px; border-radius:8px; }
    .admin-nav-link.active { background:#e6f4f2; color:#0a5f57; }
    .toolbar { display:flex; align-items:center; justify-content:space-between; margin:16px 0; }
    .button { background: var(--accent); color:white; padding:10px 14px; border-radius:8px; text-decoration:none; font-weight:600; border:none; }
    table { width:100%; border-collapse:collapse; background: var(--card); border:1px solid var(--line); border-radius:10px; overflow:hidden; }
    th, td { padding:10px; text-align:left; border-bottom:1px solid var(--line); vertical-align:middle; }
    th { font-size:0.9rem; color:var(--muted); }
    .flash { margin:12px 0; padding:10px 12px; border-radius:8px; background:#d1fae5; color:#065f46; border:1px solid #6ee7b7; }
    .toast-error { position:fixed; top:16px; right:16px; max-width:min(420px, 90vw); z-index:999; margin:0; padding:10px 12px; border-radius:10px; background:#fee2e2; color:#991b1b; border:1px solid #fecaca; box-shadow:0 8px 24px rgba(0,0,0,0.12); }
    .inline-switch { display:flex; gap:8px; align-items:center; }
    .switch {
      border:0;
      background:transparent;
      padding:0;
      display:inline-flex;
      align-items:center;
      cursor:pointer;
    }
    .switch-track {
      width:38px;
      height:22px;
      border-radius:999px;
      background:#d1d5db;
      box-shadow: inset 0 0 0 1px #c0c7d1;
      position:relative;
      display:inline-block;
      flex-shrink:0;
    }
    .switch.on .switch-track { background:#86efac; box-shadow: inset 0 0 0 1px #16a34a; }
    .switch-knob {
      width:18px;
      height:18px;
      border-radius:50%;
      position:absolute;
      top:2px;
      left:2px;
      background:#ffffff;
      box-shadow:0 1px 3px rgba(0,0,0,0.3);
      transition:left 0.15s ease;
    }
    .switch.on .switch-knob { left:18px; }
    .recency { color:var(--muted); font-size:0.8rem; }
    .row-link { color: var(--accent); font-weight: 600; }
    .thumb-box { width:32px; height:48px; border:1px solid #d4dce6; background:#f2f4f7; display:flex; align-items:center; justify-content:center; border-radius:4px; }
    .thumb-image { width:32px; height:48px; object-fit:contain; object-position:center; display:block; }
    .thumb-placeholder { font-size:9px; color:#6b7280; text-align:center; line-height:1.1; }
  </style>
</head>
<body>
  <header>
    <div class="shell">{{adminNav .ActiveSection}}</div>
  </header>
  <main class="shell">
    <div class="toolbar">
      <h1>Bundles</h1>
      <a class="button" href="/admin/bundles/new">Add Bundle</a>
    </div>
    {{if .Flash}}<p class="flash">{{.Flash}}</p>{{end}}
    {{if .ValidationToast}}<p class="toast-error" role="alert">{{.ValidationToast}}</p>{{end}}
    <table>
      <thead>
        <tr>
          <th>Image</th>
          <th>Bundle</th>
          <th>Supplier</th>
          <th>Category</th>
          <th># of books</th>
          <th>Bundle price</th>
          <th>Discount %</th>
          <th>Publish</th>
          <th>Action</th>
        </tr>
      </thead>
      <tbody>
      {{if .Bundles}}
        {{range .Bundles}}
        <tr>
          <td>
            <div class="thumb-box">
            {{if .HasImage}}
              <img class="thumb-image" src="/admin/bundles/{{.ID}}/image" alt="bundle image">
            {{else}}
              <span class="thumb-placeholder">No image</span>
            {{end}}
            </div>
          </td>
          <td>{{bundleLabel .Name .ID}}</td>
          <td>{{.SupplierName}}</td>
          <td>{{.Category}}</td>
          <td>{{.BookCount}}</td>
          <td>{{money .BundlePrice}}</td>
          <td>{{bundleDiscount .BundleMRP .BundlePrice}}</td>
          <td>
            <form class="inline-switch" method="post" action="/admin/bundles/{{.ID}}/{{toggleActionPath .IsPublished}}">
              <button class="switch {{if .IsPublished}}on{{else}}off{{end}}" type="submit" aria-label="Toggle publish for {{bundleLabel .Name .ID}}">
                <span class="switch-track"><span class="switch-knob"></span></span>
              </button>
              <span class="recency">{{publishRecency .IsPublished .PublishedAt .UnpublishedAt}}</span>
            </form>
          </td>
          <td><a class="row-link" href="/admin/bundles/{{.ID}}">View/Edit</a></td>
        </tr>
        {{end}}
      {{else}}
        <tr><td colspan="9">No bundles yet. Click "Add Bundle" to create one.</td></tr>
      {{end}}
      </tbody>
    </table>
  </main>
</body>
</html>
`))

var bundleFormTemplate = template.Must(template.New("bundle-form").Funcs(template.FuncMap{
	"adminNav": adminNav,
	"money":    func(v float64) string { return fmt.Sprintf("%.2f", v) },
	"containsBookID": func(values []string, id int) bool {
		target := strconv.Itoa(id)
		for _, value := range values {
			if value == target {
				return true
			}
		}
		return false
	},
}).Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.PageTitle}}</title>
  <style>
    :root { --bg:#f6f8fb; --card:#fff; --line:#d9e1ea; --text:#1f2937; --accent:#0f766e; --muted:#4b5563; --error:#b91c1c; }
    * { box-sizing: border-box; }
    body { margin:0; font-family: "Segoe UI", Tahoma, sans-serif; background: var(--bg); color: var(--text); }
    header { background: var(--card); border-bottom:1px solid var(--line); }
    .shell { width:min(1100px, 94vw); margin:0 auto; padding:16px; }
    .admin-nav { display:flex; gap:14px; }
    .admin-nav-link { color: var(--accent); font-weight:600; text-decoration:none; padding:6px 10px; border-radius:8px; }
    .admin-nav-link.active { background:#e6f4f2; color:#0a5f57; }
    .card { background:var(--card); border:1px solid var(--line); border-radius:10px; padding:20px; }
    .field { margin:0 0 14px; }
    label { display:block; font-weight:600; margin-bottom:6px; }
    input, select, textarea { width:100%; padding:10px 12px; border:1px solid var(--line); border-radius:8px; font:inherit; }
    input[readonly] { background:#f3f4f6; }
    textarea { min-height:90px; resize:vertical; }
    .error { color: var(--error); margin-top:6px; font-size:0.9rem; }
    .button { background: var(--accent); color:white; padding:10px 14px; border-radius:8px; text-decoration:none; border:none; font-weight:600; cursor:pointer; }
    .row { display:flex; gap:10px; align-items:center; }
    .secondary { color:var(--accent); text-decoration:none; font-weight:600; }
    .flash { margin:12px 0; padding:10px 12px; border-radius:8px; background:#d1fae5; color:#065f46; border:1px solid #6ee7b7; }
    .publish-box { margin:0 0 14px; padding:12px; background:#f8fafc; border:1px solid var(--line); border-radius:8px; display:flex; gap:10px; align-items:center; }
    .toggle { padding:6px 10px; border-radius:999px; border:1px solid var(--line); cursor:pointer; font-weight:600; font-size:0.85rem; background:white; }
    .toggle.on { background:#dcfce7; color:#166534; border-color:#86efac; }
    .toggle.off { background:#f3f4f6; color:#374151; border-color:#d1d5db; }
    .recency { color:var(--muted); font-size:0.85rem; }
    .summary { margin:0 0 14px; padding:12px; background:#eef6f4; border:1px solid #d5ebe6; border-radius:8px; color:#0f3b36; }
    .step { margin-bottom:18px; }
    .step h2 { margin:0 0 8px; font-size:1.05rem; }
    .conditions { display:flex; flex-wrap:wrap; gap:10px; }
    .conditions label { display:flex; align-items:center; gap:6px; font-weight:500; }
    .conditions input[type=checkbox] { width:auto; }
    .picker-grid { display:grid; grid-template-columns: 1fr 1fr; gap:14px; }
    .eligible-scroll { max-height:420px; overflow-y:auto; }
    table { width:100%; border-collapse:collapse; border:1px solid var(--line); border-radius:8px; overflow:hidden; }
    th, td { border-bottom:1px solid var(--line); padding:8px; text-align:left; vertical-align:middle; }
    th { background:#f8fafc; color:var(--muted); font-size:0.85rem; }
    .tiny-btn { padding:6px 10px; border:1px solid var(--line); border-radius:6px; background:#fff; cursor:pointer; }
    .totals { display:grid; grid-template-columns: repeat(2, minmax(0,1fr)); gap:10px; }
    .total-card { border:1px solid var(--line); border-radius:8px; padding:10px; background:#fafcfe; }
    .total-label { color:var(--muted); font-size:0.85rem; }
    .total-value { font-weight:700; }
    .hidden { display:none; }
    .thumb-box { width:32px; height:48px; border:1px solid #d4dce6; background:#f2f4f7; display:flex; align-items:center; justify-content:center; border-radius:4px; margin-bottom:8px; }
    .thumb-image { width:32px; height:48px; object-fit:contain; object-position:center; display:block; }
    .thumb-placeholder { font-size:9px; color:#6b7280; text-align:center; line-height:1.1; }
    .toast-error { position:fixed; top:16px; right:16px; max-width:min(420px, 90vw); z-index:999; margin:0; padding:10px 12px; border-radius:10px; background:#fee2e2; color:#991b1b; border:1px solid #fecaca; box-shadow:0 8px 24px rgba(0,0,0,0.12); }
    @media (max-width: 960px) { .picker-grid { grid-template-columns: 1fr; } .totals { grid-template-columns: 1fr; } }
  </style>
</head>
<body>
  <header>
    <div class="shell">{{adminNav .ActiveSection}}</div>
  </header>
  <main class="shell">
    <h1>{{.PageTitle}}</h1>
    {{if .Flash}}<p class="flash">{{.Flash}}</p>{{end}}
    {{if .ValidationToast}}<p class="toast-error" role="alert">{{.ValidationToast}}</p>{{end}}
    {{if .ShowPublishToggle}}
    <div class="publish-box">
      <form method="post" action="{{.PublishAction}}">
        <button class="toggle {{if eq .PublishLabel "Published"}}on{{else}}off{{end}}" type="submit">{{.PublishLabel}}</button>
      </form>
      <span class="recency">{{.PublishRecency}}</span>
    </div>
    {{end}}
    {{if .ShowSummary}}
    <div class="summary"><strong>{{.Summary.Label}}</strong><br>Supplier: {{.Summary.SupplierName}} | Category: {{.Summary.Category}} | # of books: {{.Summary.BookCount}} | Bundle price: {{.Summary.BundlePrice}}</div>
    {{end}}

    <form class="card" method="post" action="{{.Action}}" enctype="multipart/form-data">
      <div class="step">
        <h2>Step 1: Choose Supplier</h2>
        <div class="field">
          <label for="supplier_id">Supplier</label>
          <select id="supplier_id" name="supplier_id" required>
            <option value="">Select supplier</option>
            {{range .SupplierOptions}}
            <option value="{{.ID}}" {{if eq $.Input.SupplierID (printf "%d" .ID)}}selected{{end}}>{{.Name}}</option>
            {{end}}
          </select>
          {{if .HasError "supplier_id"}}<div class="error">{{.Error "supplier_id"}}</div>{{end}}
        </div>
      </div>

      <div class="step">
        <h2>Step 2: Choose Category</h2>
        <div class="field">
          <label for="category">Category</label>
          <select id="category" name="category" required>
            <option value="">Select category</option>
            {{range .CategoryOptions}}
            <option value="{{.}}" {{if eq $.Input.Category .}}selected{{end}}>{{.}}</option>
            {{end}}
          </select>
          {{if .HasError "category"}}<div class="error">{{.Error "category"}}</div>{{end}}
        </div>
      </div>

      <div class="step">
        <h2>Step 3: Allowed Book Condition(s)</h2>
        <div class="conditions">
          {{range .ConditionOptions}}
          <label><input type="checkbox" name="allowed_conditions" value="{{.}}" {{if $.ConditionChecked .}}checked{{end}}> {{.}}</label>
          {{end}}
        </div>
        {{if .HasError "allowed_conditions"}}<div class="error">{{.Error "allowed_conditions"}}</div>{{end}}
      </div>

      <div class="step">
        <h2>Step 4: Add Books to Bundle</h2>
        <p>Minimum 2 items required unless one selected item is marked Box Set.</p>
        <div class="field">
          <label for="bundle-book-search">Search eligible books (title/author)</label>
          <input id="bundle-book-search" placeholder="Search title or author">
        </div>
        <div class="picker-grid">
          <div>
            <h3>Eligible books</h3>
            <div class="eligible-scroll">
              <table>
                <thead>
                  <tr><th>Title</th><th>Author</th><th>Condition</th><th>MRP</th><th>My price</th><th></th></tr>
                </thead>
                <tbody id="bundle-picker-body">
                {{range .CandidateBooks}}
                  <tr data-picker-book-row
                      data-book-id="{{.BookID}}"
                      data-title="{{.Title}}"
                      data-author="{{.Author}}"
                      data-supplier-id="{{.SupplierID}}"
                      data-category="{{.Category}}"
                      data-condition="{{.Condition}}"
                      data-mrp="{{money .MRP}}"
                      data-my-price="{{money .MyPrice}}"
                      data-bundle-effective="{{if .BundlePrice}}{{money .BundlePrice}}{{else}}{{money .MyPrice}}{{end}}">
                    <td>{{.Title}}</td>
                    <td>{{.Author}}</td>
                    <td>{{.Condition}}</td>
                    <td>{{money .MRP}}</td>
                    <td>{{money .MyPrice}}</td>
                    <td><button class="tiny-btn" type="button" data-add-book="{{.BookID}}">Add</button></td>
                  </tr>
                {{end}}
                </tbody>
              </table>
            </div>
          </div>
          <div>
            <h3>Selected books</h3>
            <div id="bundle-book-ids" class="hidden">
              {{range .Input.BookIDValues}}<input type="hidden" name="book_ids" value="{{.}}">{{end}}
            </div>
            <table>
              <thead>
                <tr><th>Title</th><th>Author</th><th>Condition</th><th>MRP</th><th>My price</th><th></th></tr>
              </thead>
              <tbody id="bundle-selected-body">
              {{range .SelectedBooks}}
                <tr data-selected-book="{{.BookID}}">
                  <td>{{.Title}}</td>
                  <td>{{.Author}}</td>
                  <td>{{.Condition}}</td>
                  <td>{{money .MRP}}</td>
                  <td>{{money .MyPrice}}</td>
                  <td><button class="tiny-btn" type="button" data-remove-book="{{.BookID}}">Remove</button></td>
                </tr>
              {{end}}
              </tbody>
            </table>
            {{if .HasError "book_ids"}}<div class="error">{{.Error "book_ids"}}</div>{{end}}
          </div>
        </div>
      </div>

      <div class="step">
        <h2>Step 5: Pricing Support</h2>
        <div class="totals">
          <div class="total-card"><div class="total-label">Bundle MRP</div><div id="bundle-total-mrp" class="total-value">{{.BundleMRPText}}</div></div>
          <div class="total-card"><div class="total-label">Sum(My price)</div><div id="bundle-total-my-price" class="total-value">{{.SumMyPriceText}}</div></div>
          <div class="total-card"><div class="total-label">Sum(My price in bundle, fallback)</div><div id="bundle-total-my-bundle" class="total-value">{{.SumMyBundleText}}</div></div>
          <div class="total-card"><div class="total-label">Bundle discount (on MRP)</div><div id="bundle-total-discount" class="total-value">{{.DiscountText}}</div></div>
        </div>
        <div class="field">
          <label for="bundle_price">Bundle price</label>
          <input id="bundle_price" name="bundle_price" value="{{.Input.BundlePrice}}" required>
          {{if .HasError "bundle_price"}}<div class="error">{{.Error "bundle_price"}}</div>{{end}}
        </div>
      </div>

      <div class="step">
        <h2>Step 6: Optional Details + Save</h2>
        <div class="field">
          <label for="image">Bundle image</label>
          <div class="thumb-box">
            <img id="bundle-image-preview" class="thumb-image {{if .HasExistingImage}}{{else}}hidden{{end}}" src="{{if .HasExistingImage}}/admin/bundles/{{.BundleID}}/image{{end}}" alt="bundle image preview">
            <span id="bundle-image-placeholder" class="thumb-placeholder {{if .HasExistingImage}}hidden{{end}}">No image</span>
          </div>
          <input id="image" name="image" type="file" accept="image/*" {{if .HasExistingImage}}{{else}}required{{end}}>
          {{if .HasError "image"}}<div class="error">{{.Error "image"}}</div>{{end}}
        </div>
        <div class="field"><label for="name">Bundle label/name (optional)</label><input id="name" name="name" value="{{.Input.Name}}"></div>
        <div class="field"><label for="notes">Notes/description (optional)</label><textarea id="notes" name="notes">{{.Input.Notes}}</textarea></div>
        <div class="field">
          <label for="out_of_stock_on_interested">Out of stock on interested</label>
          <select id="out_of_stock_on_interested" name="out_of_stock_on_interested">
            <option value="yes" {{if eq .Input.OutOfStockOnInterested "yes"}}selected{{end}}>Yes</option>
            <option value="no" {{if eq .Input.OutOfStockOnInterested "no"}}selected{{end}}>No</option>
          </select>
          {{if .HasError "out_of_stock_on_interested"}}<div class="error">{{.Error "out_of_stock_on_interested"}}</div>{{end}}
        </div>
      </div>

      <div class="row">
        <button class="button" type="submit">{{.SubmitLabel}}</button>
        <a class="secondary" href="/admin/bundles">Back to Bundles</a>
      </div>
    </form>
  </main>
  <script src="/assets/bundles-form.js" defer></script>
</body>
</html>
`))
