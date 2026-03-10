package web

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/rajeshkrishnamurthy/sbdeals/internal/books"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/suppliers"
)

var (
	bookCategoryOptions      = []string{"Children", "Young Adults", "Fiction", "Non-Fiction"}
	bookFormatOptions        = []string{"Paperback", "Hardcover"}
	bookConditionOptions     = []string{"Good as new", "Very good", "Gently used", "Used"}
	bookValidationFieldOrder = []string{
		"cover",
		"title",
		"supplier_id",
		"is_box_set",
		"category",
		"format",
		"condition",
		"mrp",
		"my_price",
		"bundle_price",
		"out_of_stock_on_interested",
		"in_stock",
	}
)

func (s *Server) handleBooksCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.renderBooksList(w, r)
	case http.MethodPost:
		s.createBook(w, r)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (s *Server) handleBookNew(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	suppliersList, err := s.store.List()
	if err != nil {
		http.Error(w, "failed to load suppliers", http.StatusInternalServerError)
		return
	}

	s.renderBookForm(w, bookFormViewModel{
		PageTitle:         "Add Book",
		Action:            "/admin/books",
		SubmitLabel:       "Save Book",
		ActiveSection:     "books",
		Input:             bookFormInput{InStock: "yes", IsBoxSet: "no", OutOfStockOnInterested: "yes"},
		SupplierOptions:   suppliersList,
		CategoryOptions:   bookCategoryOptions,
		FormatOptions:     bookFormatOptions,
		ConditionOptions:  bookConditionOptions,
		Errors:            map[string]string{},
		DiscountReadOnly:  "0.00%",
		ShowInStockEditor: false,
	})
}

func (s *Server) handleBookItem(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/admin/books/" {
		http.Redirect(w, r, "/admin/books", http.StatusMovedPermanently)
		return
	}

	id, action, ok := parseBookPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	if !s.handleBookItemAction(w, r, id, action) {
		http.NotFound(w, r)
	}
}

func (s *Server) handleBookItemAction(w http.ResponseWriter, r *http.Request, id int, action string) bool {
	switch action {
	case "cover":
		s.handleBookCoverAction(w, r, id)
		return true
	case "stock":
		s.handleBookStockAction(w, r, id)
		return true
	case "publish":
		s.handleBookPublishAction(w, r, id)
		return true
	case "unpublish":
		s.handleBookUnpublishAction(w, r, id)
		return true
	case "":
		s.handleBookDetailAction(w, r, id)
		return true
	}
	return false
}

func (s *Server) handleBookCoverAction(w http.ResponseWriter, r *http.Request, id int) {
	if !methodAllowed(r.Method, http.MethodGet) {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	s.serveBookCover(w, r, id)
}

func (s *Server) handleBookStockAction(w http.ResponseWriter, r *http.Request, id int) {
	if !methodAllowed(r.Method, http.MethodPost) {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}
	s.updateBookInStockInline(w, r, id)
}

func (s *Server) handleBookPublishAction(w http.ResponseWriter, r *http.Request, id int) {
	if !methodAllowed(r.Method, http.MethodPost, http.MethodPatch) {
		writeMethodNotAllowed(w, http.MethodPost, http.MethodPatch)
		return
	}
	s.publishBook(w, r, id)
}

func (s *Server) handleBookUnpublishAction(w http.ResponseWriter, r *http.Request, id int) {
	if !methodAllowed(r.Method, http.MethodPost, http.MethodPatch) {
		writeMethodNotAllowed(w, http.MethodPost, http.MethodPatch)
		return
	}
	s.unpublishBook(w, r, id)
}

func (s *Server) handleBookDetailAction(w http.ResponseWriter, r *http.Request, id int) {
	switch r.Method {
	case http.MethodGet:
		s.renderBookDetail(w, r, id)
	case http.MethodPost:
		s.updateBook(w, r, id)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (s *Server) renderBooksList(w http.ResponseWriter, r *http.Request) {
	items, err := s.bookStore.List()
	if err != nil {
		http.Error(w, "failed to load books", http.StatusInternalServerError)
		return
	}

	data := booksListViewModel{
		Flash:           r.URL.Query().Get("flash"),
		ValidationToast: strings.TrimSpace(r.URL.Query().Get("error")),
		ActiveSection:   "books",
		Books:           items,
	}
	if err := booksListTemplate.Execute(w, data); err != nil {
		http.Error(w, "failed to render books list", http.StatusInternalServerError)
	}
}

func (s *Server) createBook(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}

	suppliersList, err := s.store.List()
	if err != nil {
		http.Error(w, "failed to load suppliers", http.StatusInternalServerError)
		return
	}

	inputRaw := readBookFormInput(r)
	cover, coverProvided, err := readCoverFromRequest(r)
	if err != nil {
		http.Error(w, "failed to read cover image", http.StatusBadRequest)
		return
	}

	parsed, fieldErrors := validateBookForm(inputRaw, suppliersList, true, coverProvided, false)
	if len(fieldErrors) > 0 {
		w.WriteHeader(http.StatusBadRequest)
		s.renderBookForm(w, buildBookFormView(bookFormViewOptions{
			PageTitle:         "Add Book",
			Action:            "/admin/books",
			SubmitLabel:       "Save Book",
			ActiveSection:     "books",
			Input:             inputRaw,
			SupplierOptions:   suppliersList,
			Errors:            fieldErrors,
			ValidationToast:   buildValidationToast(fieldErrors, bookValidationFieldOrder),
			ShowInStockEditor: false,
			HasExistingCover:  false,
			Summary:           nil,
		}))
		return
	}

	_, err = s.bookStore.Create(books.CreateInput{
		Title:                  parsed.Title,
		Cover:                  cover,
		SupplierID:             parsed.SupplierID,
		IsBoxSet:               parsed.IsBoxSet,
		Category:               parsed.Category,
		Format:                 parsed.Format,
		Condition:              parsed.Condition,
		MRP:                    parsed.MRP,
		MyPrice:                parsed.MyPrice,
		BundlePrice:            parsed.BundlePrice,
		Author:                 parsed.Author,
		Notes:                  parsed.Notes,
		OutOfStockOnInterested: parsed.OutOfStockOnInterested,
	})
	if err != nil {
		http.Error(w, "failed to create book", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/books?flash="+url.QueryEscape("Book created successfully."), http.StatusSeeOther)
}

func (s *Server) renderBookDetail(w http.ResponseWriter, r *http.Request, bookID int) {
	book, err := s.bookStore.Get(bookID)
	if errors.Is(err, books.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "failed to load book", http.StatusInternalServerError)
		return
	}

	suppliersList, err := s.store.List()
	if err != nil {
		http.Error(w, "failed to load suppliers", http.StatusInternalServerError)
		return
	}

	input := bookFormInput{
		Title:                  book.Title,
		SupplierID:             strconv.Itoa(book.SupplierID),
		IsBoxSet:               boolToYesNo(book.IsBoxSet),
		Category:               book.Category,
		Format:                 book.Format,
		Condition:              book.Condition,
		MRP:                    formatDecimal(book.MRP),
		MyPrice:                formatDecimal(book.MyPrice),
		BundlePrice:            optionalDecimal(book.BundlePrice),
		Author:                 book.Author,
		Notes:                  book.Notes,
		InStock:                boolToStockValue(book.InStock),
		OutOfStockOnInterested: boolToYesNo(book.OutOfStockOnInterested),
	}

	summary := &bookSummaryViewModel{
		Title:        book.Title,
		SupplierName: supplierNameByID(book.SupplierID, suppliersList),
		Category:     book.Category,
		MyPrice:      formatDecimal(book.MyPrice),
		InStock:      boolToStockLabel(book.InStock),
	}

	s.renderBookForm(w, buildBookFormView(bookFormViewOptions{
		PageTitle:         "View/Edit Book",
		Action:            fmt.Sprintf("/admin/books/%d", bookID),
		SubmitLabel:       "Save Changes",
		ActiveSection:     "books",
		Flash:             r.URL.Query().Get("flash"),
		ValidationToast:   strings.TrimSpace(r.URL.Query().Get("error")),
		Input:             input,
		SupplierOptions:   suppliersList,
		Errors:            map[string]string{},
		ShowInStockEditor: true,
		HasExistingCover:  true,
		BookID:            bookID,
		ShowPublishToggle: true,
		PublishAction:     fmt.Sprintf("/admin/books/%d/%s?from=edit", bookID, toggleActionPath(book.IsPublished)),
		PublishLabel:      publishStateLabel(book.IsPublished),
		PublishRecency:    publishRecencyLabel(book.IsPublished, book.PublishedAt, book.UnpublishedAt),
		Summary:           summary,
	}))
}

func (s *Server) updateBook(w http.ResponseWriter, r *http.Request, bookID int) {
	currentBook, err := s.bookStore.Get(bookID)
	if errors.Is(err, books.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "failed to load book", http.StatusInternalServerError)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}

	suppliersList, err := s.store.List()
	if err != nil {
		http.Error(w, "failed to load suppliers", http.StatusInternalServerError)
		return
	}

	inputRaw := readBookFormInput(r)
	cover, coverProvided, err := readCoverFromRequest(r)
	if err != nil {
		http.Error(w, "failed to read cover image", http.StatusBadRequest)
		return
	}

	parsed, fieldErrors := validateBookForm(inputRaw, suppliersList, false, coverProvided, true)
	if len(fieldErrors) > 0 {
		w.WriteHeader(http.StatusBadRequest)
		summary := &bookSummaryViewModel{
			Title:        inputRaw.Title,
			SupplierName: supplierNameByID(parsed.SupplierID, suppliersList),
			Category:     inputRaw.Category,
			MyPrice:      inputRaw.MyPrice,
			InStock:      stockValueToLabel(inputRaw.InStock),
		}
		s.renderBookForm(w, buildBookFormView(bookFormViewOptions{
			PageTitle:         "View/Edit Book",
			Action:            fmt.Sprintf("/admin/books/%d", bookID),
			SubmitLabel:       "Save Changes",
			ActiveSection:     "books",
			Input:             inputRaw,
			SupplierOptions:   suppliersList,
			Errors:            fieldErrors,
			ValidationToast:   buildValidationToast(fieldErrors, bookValidationFieldOrder),
			ShowInStockEditor: true,
			HasExistingCover:  true,
			BookID:            bookID,
			ShowPublishToggle: true,
			PublishAction:     fmt.Sprintf("/admin/books/%d/%s?from=edit", bookID, toggleActionPath(currentBook.IsPublished)),
			PublishLabel:      publishStateLabel(currentBook.IsPublished),
			PublishRecency:    publishRecencyLabel(currentBook.IsPublished, currentBook.PublishedAt, currentBook.UnpublishedAt),
			Summary:           summary,
		}))
		return
	}

	var coverPtr *books.Cover
	if coverProvided {
		coverPtr = &cover
	}

	_, err = s.bookStore.Update(bookID, books.UpdateInput{
		Title:                  parsed.Title,
		Cover:                  coverPtr,
		SupplierID:             parsed.SupplierID,
		IsBoxSet:               parsed.IsBoxSet,
		Category:               parsed.Category,
		Format:                 parsed.Format,
		Condition:              parsed.Condition,
		MRP:                    parsed.MRP,
		MyPrice:                parsed.MyPrice,
		BundlePrice:            parsed.BundlePrice,
		Author:                 parsed.Author,
		Notes:                  parsed.Notes,
		InStock:                parsed.InStock,
		OutOfStockOnInterested: parsed.OutOfStockOnInterested,
	})
	if err != nil {
		if errors.Is(err, books.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "failed to update book", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/books/%d?flash=%s", bookID, url.QueryEscape("Book updated successfully.")), http.StatusSeeOther)
}

func (s *Server) updateBookInStockInline(w http.ResponseWriter, r *http.Request, bookID int) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/admin/books?error="+url.QueryEscape("Please submit a valid in-stock value."), http.StatusSeeOther)
		return
	}

	inStock, ok := parseStockValue(strings.TrimSpace(r.Form.Get("in_stock")))
	if !ok {
		http.Redirect(w, r, "/admin/books?error="+url.QueryEscape("Please choose a valid in-stock value."), http.StatusSeeOther)
		return
	}

	if _, err := s.bookStore.SetInStock(bookID, inStock); err != nil {
		if errors.Is(err, books.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "failed to update in-stock", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/books?flash="+url.QueryEscape("Book stock updated successfully."), http.StatusSeeOther)
}

func (s *Server) publishBook(w http.ResponseWriter, r *http.Request, bookID int) {
	if _, err := s.bookStore.Publish(bookID); err != nil {
		if errors.Is(err, books.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		if errors.Is(err, books.ErrCannotPublishOutOfStock) {
			http.Redirect(w, r, bookPublishRedirectPath(r, bookID, "", "Cannot publish book because it is out of stock."), http.StatusSeeOther)
			return
		}
		http.Error(w, "failed to publish book", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, bookPublishRedirectPath(r, bookID, "Book published successfully.", ""), http.StatusSeeOther)
}

func (s *Server) unpublishBook(w http.ResponseWriter, r *http.Request, bookID int) {
	if _, err := s.bookStore.Unpublish(bookID); err != nil {
		if errors.Is(err, books.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "failed to unpublish book", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, bookPublishRedirectPath(r, bookID, "Book unpublished successfully.", ""), http.StatusSeeOther)
}

func bookPublishRedirectPath(r *http.Request, bookID int, flash string, errorMessage string) string {
	base := "/admin/books"
	if r.URL.Query().Get("from") == "edit" {
		base = fmt.Sprintf("/admin/books/%d", bookID)
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

func (s *Server) serveBookCover(w http.ResponseWriter, r *http.Request, bookID int) {
	cover, err := s.bookStore.GetCover(bookID)
	if errors.Is(err, books.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "failed to load cover", http.StatusInternalServerError)
		return
	}

	mimeType := strings.TrimSpace(cover.MimeType)
	if mimeType == "" {
		mimeType = http.DetectContentType(cover.Data)
	}
	w.Header().Set("Content-Type", mimeType)
	_, _ = w.Write(cover.Data)
}

func parseBookPath(path string) (int, string, bool) {
	prefix := "/admin/books/"
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
	if action != "cover" && action != "stock" && action != "publish" && action != "unpublish" {
		return 0, "", false
	}
	return id, action, true
}

func readBookFormInput(r *http.Request) bookFormInput {
	return bookFormInput{
		Title:                  strings.TrimSpace(r.FormValue("title")),
		SupplierID:             strings.TrimSpace(r.FormValue("supplier_id")),
		IsBoxSet:               strings.TrimSpace(r.FormValue("is_box_set")),
		Category:               strings.TrimSpace(r.FormValue("category")),
		Format:                 strings.TrimSpace(r.FormValue("format")),
		Condition:              strings.TrimSpace(r.FormValue("condition")),
		MRP:                    strings.TrimSpace(r.FormValue("mrp")),
		MyPrice:                strings.TrimSpace(r.FormValue("my_price")),
		BundlePrice:            strings.TrimSpace(r.FormValue("bundle_price")),
		Author:                 strings.TrimSpace(r.FormValue("author")),
		Notes:                  strings.TrimSpace(r.FormValue("notes")),
		InStock:                strings.TrimSpace(r.FormValue("in_stock")),
		OutOfStockOnInterested: strings.TrimSpace(r.FormValue("out_of_stock_on_interested")),
	}
}

func readCoverFromRequest(r *http.Request) (books.Cover, bool, error) {
	file, fileHeader, err := r.FormFile("cover")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return books.Cover{}, false, nil
		}
		return books.Cover{}, false, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return books.Cover{}, false, err
	}
	mimeType := strings.TrimSpace(fileHeader.Header.Get("Content-Type"))
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}
	return books.Cover{Data: data, MimeType: mimeType}, true, nil
}

type parsedBookForm struct {
	Title                  string
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

func validateBookForm(input bookFormInput, suppliersList []suppliers.Supplier, requireCover bool, coverProvided bool, requireInStock bool) (parsedBookForm, map[string]string) {
	result := parsedBookForm{
		Title:                  input.Title,
		Category:               input.Category,
		Format:                 input.Format,
		Condition:              input.Condition,
		Author:                 input.Author,
		Notes:                  input.Notes,
		InStock:                true,
		OutOfStockOnInterested: true,
	}
	errs := map[string]string{}

	validateBookIdentity(input, suppliersList, requireCover, coverProvided, &result, errs)
	validateBookClassification(input, errs)
	validateBookPricing(input, &result, errs)
	validateBookOutOfStockOnInterested(input, &result, errs)
	validateBookStock(input, requireInStock, &result, errs)

	return result, errs
}

func validateBookIdentity(input bookFormInput, suppliersList []suppliers.Supplier, requireCover bool, coverProvided bool, result *parsedBookForm, errs map[string]string) {
	if input.Title == "" {
		errs["title"] = "Title is required."
	}
	if requireCover && !coverProvided {
		errs["cover"] = "Cover image is required."
	}

	supplierID, supplierErr := parseSupplierIDForBook(input.SupplierID, suppliersList)
	if supplierErr != "" {
		errs["supplier_id"] = supplierErr
	} else {
		result.SupplierID = supplierID
	}

	if boxSetValue, ok := parseYesNo(input.IsBoxSet); ok {
		result.IsBoxSet = boxSetValue
	} else {
		errs["is_box_set"] = "Please choose a valid Box Set value."
	}
}

func validateBookClassification(input bookFormInput, errs map[string]string) {
	if errText := validateOption(input.Category, bookCategoryOptions, "Category is required.", "Please choose a valid category."); errText != "" {
		errs["category"] = errText
	}
	if errText := validateOption(input.Format, bookFormatOptions, "Book format is required.", "Please choose a valid format."); errText != "" {
		errs["format"] = errText
	}
	if errText := validateOption(input.Condition, bookConditionOptions, "Book condition is required.", "Please choose a valid condition."); errText != "" {
		errs["condition"] = errText
	}
}

func validateBookPricing(input bookFormInput, result *parsedBookForm, errs map[string]string) {
	mrp, mrpErr := parseNonNegativeNumber(input.MRP, true)
	if mrpErr != "" {
		errs["mrp"] = "MRP is required and must be a non-negative number."
	} else {
		result.MRP = *mrp
	}

	myPrice, myPriceErr := parseNonNegativeNumber(input.MyPrice, true)
	if myPriceErr != "" {
		errs["my_price"] = "My price is required and must be a non-negative number."
	} else {
		result.MyPrice = *myPrice
	}

	bundlePrice, bundleErr := parseNonNegativeNumber(input.BundlePrice, false)
	if bundleErr != "" {
		errs["bundle_price"] = "My price (in bundle), if provided, must be a non-negative number."
	} else {
		result.BundlePrice = bundlePrice
	}
}

func validateBookStock(input bookFormInput, requireInStock bool, result *parsedBookForm, errs map[string]string) {
	if !requireInStock {
		return
	}
	if value, ok := parseStockValue(input.InStock); ok {
		result.InStock = value
		return
	}
	errs["in_stock"] = "Please choose a valid in-stock value."
}

func validateBookOutOfStockOnInterested(input bookFormInput, result *parsedBookForm, errs map[string]string) {
	if value, ok := parseRequiredYesNo(input.OutOfStockOnInterested); ok {
		result.OutOfStockOnInterested = value
		return
	}
	errs["out_of_stock_on_interested"] = "Please choose a valid Out of stock on interested value."
}

func validateOption(value string, allowed []string, emptyMsg string, invalidMsg string) string {
	if value == "" {
		return emptyMsg
	}
	for _, option := range allowed {
		if option == value {
			return ""
		}
	}
	return invalidMsg
}

func parseSupplierIDForBook(raw string, suppliersList []suppliers.Supplier) (int, string) {
	if strings.TrimSpace(raw) == "" {
		return 0, "Supplier is required."
	}
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		return 0, "Please choose a valid supplier."
	}
	for _, item := range suppliersList {
		if item.ID == id {
			return id, ""
		}
	}
	return 0, "Please choose a valid supplier."
}

func parseNonNegativeNumber(raw string, required bool) (*float64, string) {
	value := strings.TrimSpace(raw)
	if value == "" {
		if required {
			return nil, "required"
		}
		return nil, ""
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || parsed < 0 {
		return nil, "invalid"
	}
	return &parsed, ""
}

func parseStockValue(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "yes":
		return true, true
	case "no":
		return false, true
	default:
		return false, false
	}
}

func parseYesNo(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return false, true
	case "yes":
		return true, true
	case "no":
		return false, true
	default:
		return false, false
	}
}

func parseRequiredYesNo(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "yes":
		return true, true
	case "no":
		return false, true
	default:
		return false, false
	}
}

func boolToYesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func boolToStockValue(inStock bool) string {
	if inStock {
		return "yes"
	}
	return "no"
}

func boolToStockLabel(inStock bool) string {
	if inStock {
		return "Yes"
	}
	return "No"
}

func stockValueToLabel(value string) string {
	if strings.EqualFold(value, "no") {
		return "No"
	}
	return "Yes"
}

func toggleActionPath(isPublished bool) string {
	if isPublished {
		return "unpublish"
	}
	return "publish"
}

func formatDecimal(value float64) string {
	return strconv.FormatFloat(value, 'f', 2, 64)
}

func optionalDecimal(value *float64) string {
	if value == nil {
		return ""
	}
	return formatDecimal(*value)
}

func supplierNameByID(id int, suppliersList []suppliers.Supplier) string {
	for _, item := range suppliersList {
		if item.ID == id {
			return item.Name
		}
	}
	return ""
}

type booksListViewModel struct {
	Flash           string
	ValidationToast string
	ActiveSection   string
	Books           []books.ListItem
}

type bookSummaryViewModel struct {
	Title        string
	SupplierName string
	Category     string
	MyPrice      string
	InStock      string
}

type bookFormInput struct {
	Title                  string
	SupplierID             string
	IsBoxSet               string
	Category               string
	Format                 string
	Condition              string
	MRP                    string
	MyPrice                string
	BundlePrice            string
	Author                 string
	Notes                  string
	InStock                string
	OutOfStockOnInterested string
}

type bookFormViewModel struct {
	PageTitle         string
	Action            string
	SubmitLabel       string
	ActiveSection     string
	Flash             string
	ValidationToast   string
	Input             bookFormInput
	SupplierOptions   []suppliers.Supplier
	CategoryOptions   []string
	FormatOptions     []string
	ConditionOptions  []string
	Errors            map[string]string
	DiscountReadOnly  string
	ShowInStockEditor bool
	HasExistingCover  bool
	BookID            int
	ShowPublishToggle bool
	PublishAction     string
	PublishLabel      string
	PublishRecency    string
	Summary           *bookSummaryViewModel
}

func (m bookFormViewModel) HasError(field string) bool {
	_, ok := m.Errors[field]
	return ok
}

func (m bookFormViewModel) Error(field string) string {
	return m.Errors[field]
}

type bookFormViewOptions struct {
	PageTitle         string
	Action            string
	SubmitLabel       string
	ActiveSection     string
	Flash             string
	ValidationToast   string
	Input             bookFormInput
	SupplierOptions   []suppliers.Supplier
	Errors            map[string]string
	ShowInStockEditor bool
	HasExistingCover  bool
	BookID            int
	ShowPublishToggle bool
	PublishAction     string
	PublishLabel      string
	PublishRecency    string
	Summary           *bookSummaryViewModel
}

func buildBookFormView(options bookFormViewOptions) bookFormViewModel {
	mrp, _ := parseNonNegativeNumber(options.Input.MRP, false)
	myPrice, _ := parseNonNegativeNumber(options.Input.MyPrice, false)
	discount := "0%"
	if mrp != nil && myPrice != nil {
		discount = formatRoundedDiscount(books.ComputeDiscount(*mrp, *myPrice))
	}

	return bookFormViewModel{
		PageTitle:         options.PageTitle,
		Action:            options.Action,
		SubmitLabel:       options.SubmitLabel,
		ActiveSection:     options.ActiveSection,
		Flash:             options.Flash,
		ValidationToast:   options.ValidationToast,
		Input:             options.Input,
		SupplierOptions:   options.SupplierOptions,
		CategoryOptions:   bookCategoryOptions,
		FormatOptions:     bookFormatOptions,
		ConditionOptions:  bookConditionOptions,
		Errors:            options.Errors,
		DiscountReadOnly:  discount,
		ShowInStockEditor: options.ShowInStockEditor,
		HasExistingCover:  options.HasExistingCover,
		BookID:            options.BookID,
		ShowPublishToggle: options.ShowPublishToggle,
		PublishAction:     options.PublishAction,
		PublishLabel:      options.PublishLabel,
		PublishRecency:    options.PublishRecency,
		Summary:           options.Summary,
	}
}

func formatRoundedDiscount(value float64) string {
	return fmt.Sprintf("%d%%", int(math.Round(value)))
}

func (s *Server) renderBookForm(w http.ResponseWriter, data bookFormViewModel) {
	if err := booksFormTemplate.Execute(w, data); err != nil {
		http.Error(w, "failed to render book form", http.StatusInternalServerError)
	}
}

var booksListTemplate = template.Must(template.New("books-list").Funcs(template.FuncMap{
	"adminNav":         adminNav,
	"money":            func(v float64) string { return fmt.Sprintf("%.2f", v) },
	"publishRecency":   publishRecencyLabel,
	"publishState":     publishStateLabel,
	"toggleActionPath": toggleActionPath,
}).Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Books</title>
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
    th, td { padding:8px 10px; text-align:left; border-bottom:1px solid var(--line); vertical-align:middle; }
    th { font-size:0.9rem; color:var(--muted); }
    .flash { margin:12px 0; padding:10px 12px; border-radius:8px; background:#d1fae5; color:#065f46; border:1px solid #6ee7b7; }
    .thumb-box { width:32px; height:48px; border:1px solid #d4dce6; background:#f2f4f7; display:flex; align-items:center; justify-content:center; border-radius:4px; }
    .thumb-image { width:32px; height:48px; object-fit:contain; object-position:center; display:block; }
    .thumb-placeholder { font-size:9px; color:#6b7280; text-align:center; line-height:1.1; }
    .row-link { color: var(--accent); font-weight: 600; }
    .inline-switch { display:flex; gap:8px; align-items:center; }
    .switch {
      min-width:72px;
      border:0;
      border-radius:999px;
      padding:3px 8px 3px 4px;
      display:inline-flex;
      align-items:center;
      gap:6px;
      font-size:0.74rem;
      font-weight:700;
      letter-spacing:0.02em;
      text-transform:uppercase;
      color:#6b7280;
      background:#e5e7eb;
      cursor:pointer;
      box-shadow: inset 0 0 0 1px #d1d5db;
    }
    .switch.on { color:#065f46; background:#22c55e; box-shadow: inset 0 0 0 1px #16a34a; }
    .switch-track {
      width:34px;
      height:20px;
      border-radius:999px;
      background:rgba(255,255,255,0.35);
      position:relative;
      display:inline-block;
      flex-shrink:0;
    }
    .switch-knob {
      width:16px;
      height:16px;
      border-radius:50%;
      position:absolute;
      top:2px;
      left:2px;
      background:#ffffff;
      box-shadow:0 1px 3px rgba(0,0,0,0.3);
      transition:left 0.15s ease;
    }
    .switch.on .switch-knob { left:16px; }
    .switch-text { line-height:1; }
    .recency { color:var(--muted); font-size:0.8rem; }
    .price { font-variant-numeric: tabular-nums; }
    .toast-error { position:fixed; top:16px; right:16px; max-width:min(420px, 90vw); z-index:999; margin:0; padding:10px 12px; border-radius:10px; background:#fee2e2; color:#991b1b; border:1px solid #fecaca; box-shadow:0 8px 24px rgba(0,0,0,0.12); }
  </style>
</head>
<body>
  <header>
    <div class="shell">
      {{adminNav .ActiveSection}}
    </div>
  </header>
  <main class="shell">
    <div class="toolbar">
      <h1>Books</h1>
      <a class="button" href="/admin/books/new">Add Book</a>
    </div>
    {{if .Flash}}<p class="flash">{{.Flash}}</p>{{end}}
    {{if .ValidationToast}}<p class="toast-error" role="alert">{{.ValidationToast}}</p>{{end}}
    <table>
      <thead>
        <tr>
          <th>Cover</th>
          <th>Title</th>
          <th>Author</th>
          <th>Category</th>
          <th>My price</th>
          <th>In-stock</th>
          <th>Publish</th>
          <th>Action</th>
        </tr>
      </thead>
      <tbody>
      {{if .Books}}
        {{range .Books}}
        <tr>
          <td>
            <div class="thumb-box">
              {{if .HasCover}}
              <img class="thumb-image" src="/admin/books/{{.ID}}/cover" alt="book cover">
              {{else}}
              <span class="thumb-placeholder">No image</span>
              {{end}}
            </div>
          </td>
          <td>{{.Title}}</td>
          <td>{{.Author}}</td>
          <td>{{.Category}}</td>
          <td class="price">{{money .MyPrice}}</td>
          <td>
            <form class="inline-switch" method="post" action="/admin/books/{{.ID}}/stock">
              <input type="hidden" name="in_stock" value="{{if .InStock}}no{{else}}yes{{end}}">
              <button class="switch {{if .InStock}}on{{else}}off{{end}}" type="submit" aria-label="Toggle in-stock for {{.Title}}">
                <span class="switch-track"><span class="switch-knob"></span></span>
                <span class="switch-text">{{if .InStock}}ON{{else}}OFF{{end}}</span>
              </button>
            </form>
          </td>
          <td>
            <form class="inline-switch" method="post" action="/admin/books/{{.ID}}/{{toggleActionPath .IsPublished}}">
              <button class="switch {{if .IsPublished}}on{{else}}off{{end}}" type="submit" aria-label="Toggle publish for {{.Title}}">
                <span class="switch-track"><span class="switch-knob"></span></span>
                <span class="switch-text">{{if .IsPublished}}ON{{else}}OFF{{end}}</span>
              </button>
              <span class="recency">{{publishRecency .IsPublished .PublishedAt .UnpublishedAt}}</span>
            </form>
          </td>
          <td><a class="row-link" href="/admin/books/{{.ID}}">View/Edit</a></td>
        </tr>
        {{end}}
      {{else}}
        <tr>
          <td colspan="8">No books yet. Click "Add Book" to create one.</td>
        </tr>
      {{end}}
      </tbody>
    </table>
  </main>
</body>
</html>
`))

var booksFormTemplate = template.Must(template.New("books-form").Funcs(template.FuncMap{"adminNav": adminNav}).Parse(`<!doctype html>
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
    textarea { min-height:100px; resize:vertical; }
    .error { color: var(--error); margin-top:6px; font-size:0.9rem; }
    .button { background: var(--accent); color:white; padding:10px 14px; border-radius:8px; text-decoration:none; border:none; font-weight:600; cursor:pointer; }
    .row { display:flex; gap:10px; align-items:center; }
    .secondary { color:var(--accent); text-decoration:none; font-weight:600; }
    .flash { margin:12px 0; padding:10px 12px; border-radius:8px; background:#d1fae5; color:#065f46; border:1px solid #6ee7b7; }
    .summary { margin:0 0 14px; padding:12px; background:#eef6f4; border:1px solid #d5ebe6; border-radius:8px; color:#0f3b36; }
    .thumb-box { width:32px; height:48px; border:1px solid #d4dce6; background:#f2f4f7; display:flex; align-items:center; justify-content:center; border-radius:4px; margin-bottom:8px; }
    .thumb-image { width:32px; height:48px; object-fit:contain; object-position:center; display:block; }
    .thumb-placeholder { font-size:9px; color:#6b7280; text-align:center; line-height:1.1; }
    .publish-box { margin:0 0 14px; padding:12px; background:#f8fafc; border:1px solid var(--line); border-radius:8px; display:flex; gap:10px; align-items:center; }
    .toggle { padding:6px 10px; border-radius:999px; border:1px solid var(--line); cursor:pointer; font-weight:600; font-size:0.85rem; background:white; }
    .toggle.on { background:#dcfce7; color:#166534; border-color:#86efac; }
    .toggle.off { background:#f3f4f6; color:#374151; border-color:#d1d5db; }
    .recency { color:var(--muted); font-size:0.85rem; }
    .hidden { display:none; }
    .toast-error { position:fixed; top:16px; right:16px; max-width:min(420px, 90vw); z-index:999; margin:0; padding:10px 12px; border-radius:10px; background:#fee2e2; color:#991b1b; border:1px solid #fecaca; box-shadow:0 8px 24px rgba(0,0,0,0.12); }
  </style>
</head>
<body>
  <header>
    <div class="shell">
      {{adminNav .ActiveSection}}
    </div>
  </header>
  <main class="shell">
    <h1>{{.PageTitle}}</h1>
    {{if .Flash}}<p class="flash">{{.Flash}}</p>{{end}}
    {{if .ValidationToast}}<p class="toast-error" role="alert">{{.ValidationToast}}</p>{{end}}

    {{if .Summary}}
    <div class="summary">
      <strong>{{.Summary.Title}}</strong><br>
      Supplier: {{.Summary.SupplierName}} | Category: {{.Summary.Category}} | My price: {{.Summary.MyPrice}} | In-stock: {{.Summary.InStock}}
    </div>
    {{end}}

    {{if .ShowPublishToggle}}
    <div class="publish-box">
      <form method="post" action="{{.PublishAction}}">
        <button class="toggle {{if eq .PublishLabel "Published"}}on{{else}}off{{end}}" type="submit">{{.PublishLabel}}</button>
      </form>
      <span class="recency">{{.PublishRecency}}</span>
    </div>
    {{end}}

    <form class="card" method="post" action="{{.Action}}" enctype="multipart/form-data">
      <div class="field">
        <label for="cover">Cover image</label>
        <div class="thumb-box">
          <img id="book-cover-preview" class="thumb-image {{if .HasExistingCover}}{{else}}hidden{{end}}" src="{{if .HasExistingCover}}/admin/books/{{.BookID}}/cover{{end}}" alt="book cover preview">
          <span id="book-cover-placeholder" class="thumb-placeholder {{if .HasExistingCover}}hidden{{end}}">No image</span>
        </div>
        <input id="cover" name="cover" type="file" accept="image/*">
        {{if .HasError "cover"}}<div class="error">{{.Error "cover"}}</div>{{end}}
      </div>

      <div class="field">
        <label for="title">Title</label>
        <input id="title" name="title" value="{{.Input.Title}}" required>
        {{if .HasError "title"}}<div class="error">{{.Error "title"}}</div>{{end}}
      </div>

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

      <div class="field">
        <label for="is_box_set">Box Set</label>
        <select id="is_box_set" name="is_box_set" required>
          <option value="no" {{if eq .Input.IsBoxSet "no"}}selected{{end}}>No</option>
          <option value="yes" {{if eq .Input.IsBoxSet "yes"}}selected{{end}}>Yes</option>
        </select>
        {{if .HasError "is_box_set"}}<div class="error">{{.Error "is_box_set"}}</div>{{end}}
      </div>

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

      <div class="field">
        <label for="format">Book format</label>
        <select id="format" name="format" required>
          <option value="">Select format</option>
          {{range .FormatOptions}}
          <option value="{{.}}" {{if eq $.Input.Format .}}selected{{end}}>{{.}}</option>
          {{end}}
        </select>
        {{if .HasError "format"}}<div class="error">{{.Error "format"}}</div>{{end}}
      </div>

      <div class="field">
        <label for="condition">Book condition</label>
        <select id="condition" name="condition" required>
          <option value="">Select condition</option>
          {{range .ConditionOptions}}
          <option value="{{.}}" {{if eq $.Input.Condition .}}selected{{end}}>{{.}}</option>
          {{end}}
        </select>
        {{if .HasError "condition"}}<div class="error">{{.Error "condition"}}</div>{{end}}
      </div>

      <div class="field">
        <label for="mrp">MRP</label>
        <input id="mrp" name="mrp" value="{{.Input.MRP}}" required>
        {{if .HasError "mrp"}}<div class="error">{{.Error "mrp"}}</div>{{end}}
      </div>

      <div class="field">
        <label for="my_price">My price</label>
        <input id="my_price" name="my_price" value="{{.Input.MyPrice}}" required>
        {{if .HasError "my_price"}}<div class="error">{{.Error "my_price"}}</div>{{end}}
      </div>

      <div class="field">
        <label for="discount">Discount (auto-computed)</label>
        <input id="discount" name="discount" value="{{.DiscountReadOnly}}" readonly>
      </div>

      <div class="field">
        <label for="bundle_price">My price (in bundle) (optional)</label>
        <input id="bundle_price" name="bundle_price" value="{{.Input.BundlePrice}}">
        {{if .HasError "bundle_price"}}<div class="error">{{.Error "bundle_price"}}</div>{{end}}
      </div>

      <div class="field">
        <label for="author">Author (optional)</label>
        <input id="author" name="author" value="{{.Input.Author}}">
      </div>

      <div class="field">
        <label for="notes">Notes/Description (optional)</label>
        <textarea id="notes" name="notes">{{.Input.Notes}}</textarea>
      </div>

      <div class="field">
        <label for="out_of_stock_on_interested">Out of stock on interested</label>
        <select id="out_of_stock_on_interested" name="out_of_stock_on_interested">
          <option value="yes" {{if eq .Input.OutOfStockOnInterested "yes"}}selected{{end}}>Yes</option>
          <option value="no" {{if eq .Input.OutOfStockOnInterested "no"}}selected{{end}}>No</option>
        </select>
        {{if .HasError "out_of_stock_on_interested"}}<div class="error">{{.Error "out_of_stock_on_interested"}}</div>{{end}}
      </div>

      {{if .ShowInStockEditor}}
      <div class="field">
        <label for="in_stock">In-stock</label>
        <select id="in_stock" name="in_stock">
          <option value="yes" {{if eq .Input.InStock "yes"}}selected{{end}}>Yes</option>
          <option value="no" {{if eq .Input.InStock "no"}}selected{{end}}>No</option>
        </select>
        {{if .HasError "in_stock"}}<div class="error">{{.Error "in_stock"}}</div>{{end}}
      </div>
      {{end}}

      <div class="row">
        <button class="button" type="submit">{{.SubmitLabel}}</button>
        <a class="secondary" href="/admin/books">Back to Books</a>
      </div>
    </form>
  </main>
  <script src="/assets/books-form.js" defer></script>
</body>
</html>
`))
