package web

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/rajeshkrishnamurthy/sbdeals/internal/bundles"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/suppliers"
)

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
		PageTitle:       "Add Bundle",
		Action:          "/admin/bundles",
		SubmitLabel:     "Save Bundle",
		ActiveSection:   "bundles",
		Input:           bundleFormInput{},
		SupplierOptions: suppliersList,
		CandidateBooks:  pickerBooks,
		SelectedBooks:   []bundles.PickerBook{},
		Errors:          map[string]string{},
		ShowSummary:     false,
	}))
}

func (s *Server) handleBundleItem(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/admin/bundles/" {
		http.Redirect(w, r, "/admin/bundles", http.StatusMovedPermanently)
		return
	}

	bundleID, ok := parseBundlePath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

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
		Flash:         r.URL.Query().Get("flash"),
		ActiveSection: "bundles",
		Bundles:       items,
	}
	if err := bundlesListTemplate.Execute(w, view); err != nil {
		http.Error(w, "failed to render bundles list", http.StatusInternalServerError)
	}
}

func (s *Server) createBundle(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}

	suppliersList, pickerBooks, err := s.loadBundleFormDependencies()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	input := readBundleFormInput(r)
	parsed, fieldErrors, selectedBooks := validateBundleFormInput(input, suppliersList, pickerBooks)
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
			ShowSummary:     false,
		}))
		return
	}

	_, err = s.bundleStore.Create(bundles.CreateInput{
		Name:              parsed.Name,
		SupplierID:        parsed.SupplierID,
		Category:          parsed.Category,
		AllowedConditions: parsed.AllowedConditions,
		BookIDs:           parsed.BookIDs,
		BundlePrice:       parsed.BundlePrice,
		Notes:             parsed.Notes,
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
		Name:              bundle.Name,
		SupplierID:        strconv.Itoa(bundle.SupplierID),
		Category:          bundle.Category,
		AllowedConditions: append([]string(nil), bundle.AllowedConditions...),
		BookIDValues:      intSliceToStringSlice(bundle.BookIDs),
		BundlePrice:       formatDecimal(bundle.BundlePrice),
		Notes:             bundle.Notes,
	}

	summary := &bundleSummaryViewModel{
		Label:        bundleLabel(bundle.Name, bundle.ID),
		SupplierName: bundle.SupplierName,
		Category:     bundle.Category,
		BookCount:    len(bundle.BookIDs),
		BundlePrice:  formatDecimal(bundle.BundlePrice),
	}

	s.renderBundleForm(w, buildBundleFormView(bundleFormViewOptions{
		PageTitle:       "View/Edit Bundle",
		Action:          fmt.Sprintf("/admin/bundles/%d", bundleID),
		SubmitLabel:     "Save Changes",
		ActiveSection:   "bundles",
		Flash:           r.URL.Query().Get("flash"),
		Input:           input,
		SupplierOptions: suppliersList,
		CandidateBooks:  pickerBooks,
		SelectedBooks:   selectedBooks,
		Errors:          map[string]string{},
		ShowSummary:     true,
		Summary:         summary,
	}))
}

func (s *Server) updateBundle(w http.ResponseWriter, r *http.Request, bundleID int) {
	if _, err := s.bundleStore.Get(bundleID); errors.Is(err, bundles.ErrNotFound) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, "failed to load bundle", http.StatusInternalServerError)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}

	suppliersList, pickerBooks, err := s.loadBundleFormDependencies()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	input := readBundleFormInput(r)
	parsed, fieldErrors, selectedBooks := validateBundleFormInput(input, suppliersList, pickerBooks)
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
			PageTitle:       "View/Edit Bundle",
			Action:          fmt.Sprintf("/admin/bundles/%d", bundleID),
			SubmitLabel:     "Save Changes",
			ActiveSection:   "bundles",
			Input:           input,
			SupplierOptions: suppliersList,
			CandidateBooks:  pickerBooks,
			SelectedBooks:   selectedBooks,
			Errors:          fieldErrors,
			ShowSummary:     true,
			Summary:         summary,
		}))
		return
	}

	_, err = s.bundleStore.Update(bundleID, bundles.UpdateInput{
		Name:              parsed.Name,
		SupplierID:        parsed.SupplierID,
		Category:          parsed.Category,
		AllowedConditions: parsed.AllowedConditions,
		BookIDs:           parsed.BookIDs,
		BundlePrice:       parsed.BundlePrice,
		Notes:             parsed.Notes,
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
		Name:              strings.TrimSpace(r.Form.Get("name")),
		SupplierID:        strings.TrimSpace(r.Form.Get("supplier_id")),
		Category:          strings.TrimSpace(r.Form.Get("category")),
		AllowedConditions: trimAndCompact(allowedConditions),
		BookIDValues:      trimAndCompact(bookIDs),
		BundlePrice:       strings.TrimSpace(r.Form.Get("bundle_price")),
		Notes:             strings.TrimSpace(r.Form.Get("notes")),
	}
}

type parsedBundleForm struct {
	Name              string
	SupplierID        int
	Category          string
	AllowedConditions []string
	BookIDs           []int
	BundlePrice       float64
	Notes             string
}

func validateBundleFormInput(input bundleFormInput, suppliersList []suppliers.Supplier, pickerBooks []bundles.PickerBook) (parsedBundleForm, map[string]string, []bundles.PickerBook) {
	result := parsedBundleForm{Name: input.Name, Category: input.Category, AllowedConditions: append([]string(nil), input.AllowedConditions...), Notes: input.Notes}
	errorsByField := map[string]string{}

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

	if len(result.BookIDs) < 2 {
		errorsByField["book_ids"] = "Bundle must include at least 2 books."
	}

	bundlePricePtr, priceErr := parseNonNegativeNumber(input.BundlePrice, true)
	if priceErr != "" {
		errorsByField["bundle_price"] = "Bundle price is required and must be a non-negative number."
	} else {
		result.BundlePrice = *bundlePricePtr
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
			Category:    book.Category,
			Condition:   book.Condition,
			MRP:         book.MRP,
			MyPrice:     book.MyPrice,
			BundlePrice: book.BundlePrice,
		})
	}
	return items
}

func parseBundlePath(path string) (int, bool) {
	prefix := "/admin/bundles/"
	if !strings.HasPrefix(path, prefix) {
		return 0, false
	}
	rest := strings.TrimPrefix(path, prefix)
	if rest == "" || strings.Contains(rest, "/") {
		return 0, false
	}
	id, err := strconv.Atoi(rest)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
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
	Flash         string
	ActiveSection string
	Bundles       []bundles.ListItem
}

type bundleSummaryViewModel struct {
	Label        string
	SupplierName string
	Category     string
	BookCount    int
	BundlePrice  string
}

type bundleFormInput struct {
	Name              string
	SupplierID        string
	Category          string
	AllowedConditions []string
	BookIDValues      []string
	BundlePrice       string
	Notes             string
}

type bundleFormViewModel struct {
	PageTitle        string
	Action           string
	SubmitLabel      string
	ActiveSection    string
	Flash            string
	Input            bundleFormInput
	SupplierOptions  []suppliers.Supplier
	CategoryOptions  []string
	ConditionOptions []string
	CandidateBooks   []bundles.PickerBook
	SelectedBooks    []bundles.PickerBook
	Errors           map[string]string
	BundleMRPText    string
	SumMyPriceText   string
	SumMyBundleText  string
	DiscountText     string
	ShowSummary      bool
	Summary          *bundleSummaryViewModel
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
	PageTitle       string
	Action          string
	SubmitLabel     string
	ActiveSection   string
	Flash           string
	Input           bundleFormInput
	SupplierOptions []suppliers.Supplier
	CandidateBooks  []bundles.PickerBook
	SelectedBooks   []bundles.PickerBook
	Errors          map[string]string
	ShowSummary     bool
	Summary         *bundleSummaryViewModel
}

func buildBundleFormView(options bundleFormViewOptions) bundleFormViewModel {
	bundlePrice := 0.0
	if parsed, errText := parseNonNegativeNumber(options.Input.BundlePrice, false); errText == "" && parsed != nil {
		bundlePrice = *parsed
	}
	totals := calculateBundleTotals(options.SelectedBooks, bundlePrice)

	return bundleFormViewModel{
		PageTitle:        options.PageTitle,
		Action:           options.Action,
		SubmitLabel:      options.SubmitLabel,
		ActiveSection:    options.ActiveSection,
		Flash:            options.Flash,
		Input:            options.Input,
		SupplierOptions:  options.SupplierOptions,
		CategoryOptions:  bookCategoryOptions,
		ConditionOptions: bookConditionOptions,
		CandidateBooks:   options.CandidateBooks,
		SelectedBooks:    options.SelectedBooks,
		Errors:           options.Errors,
		BundleMRPText:    formatDecimal(totals.BundleMRP),
		SumMyPriceText:   formatDecimal(totals.SumMyPrice),
		SumMyBundleText:  formatDecimal(totals.SumMyBundlePrice),
		DiscountText:     formatRoundedDiscount(totals.Discount),
		ShowSummary:      options.ShowSummary,
		Summary:          options.Summary,
	}
}

func (s *Server) renderBundleForm(w http.ResponseWriter, data bundleFormViewModel) {
	if err := bundleFormTemplate.Execute(w, data); err != nil {
		http.Error(w, "failed to render bundle form", http.StatusInternalServerError)
	}
}

var bundlesListTemplate = template.Must(template.New("bundles-list").Funcs(template.FuncMap{
	"adminNav":       adminNav,
	"bundleLabel":    bundleLabel,
	"money":          func(v float64) string { return fmt.Sprintf("%.2f", v) },
	"joinConditions": func(values []string) string { return strings.Join(values, ", ") },
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
    .row-link { color: var(--accent); font-weight: 600; }
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
    <table>
      <thead>
        <tr>
          <th>Bundle</th>
          <th>Supplier</th>
          <th>Category</th>
          <th>Allowed condition(s)</th>
          <th># of books</th>
          <th>Bundle price</th>
          <th>Action</th>
        </tr>
      </thead>
      <tbody>
      {{if .Bundles}}
        {{range .Bundles}}
        <tr>
          <td>{{bundleLabel .Name .ID}}</td>
          <td>{{.SupplierName}}</td>
          <td>{{.Category}}</td>
          <td>{{joinConditions .AllowedConditions}}</td>
          <td>{{.BookCount}}</td>
          <td>{{money .BundlePrice}}</td>
          <td><a class="row-link" href="/admin/bundles/{{.ID}}">View/Edit</a></td>
        </tr>
        {{end}}
      {{else}}
        <tr><td colspan="7">No bundles yet. Click "Add Bundle" to create one.</td></tr>
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
    {{if .ShowSummary}}
    <div class="summary"><strong>{{.Summary.Label}}</strong><br>Supplier: {{.Summary.SupplierName}} | Category: {{.Summary.Category}} | # of books: {{.Summary.BookCount}} | Bundle price: {{.Summary.BundlePrice}}</div>
    {{end}}

    <form class="card" method="post" action="{{.Action}}">
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
        <div class="field"><label for="name">Bundle label/name (optional)</label><input id="name" name="name" value="{{.Input.Name}}"></div>
        <div class="field"><label for="notes">Notes/description (optional)</label><textarea id="notes" name="notes">{{.Input.Notes}}</textarea></div>
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
