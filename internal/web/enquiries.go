package web

import (
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/rajeshkrishnamurthy/sbdeals/internal/clicked"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/customers"
)

const defaultConvertedBy = "system-admin"

func (s *Server) handleEnquiriesCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.renderEnquiriesList(w, r)
	default:
		writeMethodNotAllowed(w, http.MethodGet)
	}
}

func (s *Server) handleEnquiryItem(w http.ResponseWriter, r *http.Request) {
	id, action, ok := parseEnquiryPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}
	switch action {
	case "convert":
		s.convertEnquiryToInterested(w, r, id)
	case "order":
		s.convertEnquiryToOrdered(w, r, id)
	default:
		http.NotFound(w, r)
	}
}

func parseEnquiryPath(path string) (int, string, bool) {
	prefix := "/admin/enquiries/"
	if !strings.HasPrefix(path, prefix) {
		return 0, "", false
	}
	rest := strings.TrimPrefix(path, prefix)
	parts := strings.Split(rest, "/")
	if len(parts) != 2 {
		return 0, "", false
	}
	if parts[1] != "convert" && parts[1] != "order" {
		return 0, "", false
	}
	id, err := strconv.Atoi(parts[0])
	if err != nil || id <= 0 {
		return 0, "", false
	}
	return id, parts[1], true
}

func (s *Server) renderEnquiriesList(w http.ResponseWriter, r *http.Request) {
	status := clicked.Status(strings.TrimSpace(r.URL.Query().Get("status")))
	if !clicked.IsValidStatus(status) {
		status = clicked.StatusClicked
	}

	items, err := s.clickedStore.ListByStatus(status)
	if err != nil {
		http.Error(w, "failed to load enquiries", http.StatusInternalServerError)
		return
	}
	customerOptions, err := s.customerStore.List(customers.ListFilter{})
	if err != nil {
		http.Error(w, "failed to load customers", http.StatusInternalServerError)
		return
	}
	customerByID := make(map[int]customers.ListItem, len(customerOptions))
	for _, customer := range customerOptions {
		customerByID[customer.ID] = customer
	}
	customerAddressByID := map[int]string{}
	for _, item := range items {
		if item.CustomerID <= 0 {
			continue
		}
		customer, err := s.customerStore.Get(item.CustomerID)
		if err != nil {
			continue
		}
		if _, exists := customerByID[item.CustomerID]; !exists {
			customerByID[item.CustomerID] = customers.ListItem{
				ID:            customer.ID,
				Name:          customer.Name,
				Mobile:        customer.Mobile,
				CityName:      customer.CityName,
				ApartmentName: customer.ApartmentName,
			}
		}
		if customer.Address != nil {
			address := strings.TrimSpace(*customer.Address)
			if address != "" {
				customerAddressByID[item.CustomerID] = address
			}
		}
	}

	vm := enquiriesListViewModel{
		ActiveSection:        "enquiries",
		SelectedStatus:       status,
		Rows:                 items,
		Flash:                strings.TrimSpace(r.URL.Query().Get("flash")),
		ValidationToast:      strings.TrimSpace(r.URL.Query().Get("error")),
		OpenConvertModal:     strings.TrimSpace(r.URL.Query().Get("open_convert_modal")) == "1",
		ModalEnquiryID:       parseModalEnquiryID(r.URL.Query().Get("modal_enquiry_id")),
		ModalCustomerID:      strings.TrimSpace(r.URL.Query().Get("customer_id")),
		ModalQuickName:       strings.TrimSpace(r.URL.Query().Get("quick_customer_name")),
		ModalQuickMobile:     strings.TrimSpace(r.URL.Query().Get("quick_customer_mobile")),
		ModalNote:            strings.TrimSpace(r.URL.Query().Get("note")),
		ModalSearch:          strings.TrimSpace(r.URL.Query().Get("customer_search")),
		OpenOrderModal:       parseOptionalBool(r.URL.Query().Get("open_order_modal")),
		OrderModalEnquiryID:  parseModalEnquiryID(r.URL.Query().Get("modal_order_enquiry_id")),
		OrderModalCustomer:   strings.TrimSpace(r.URL.Query().Get("modal_order_customer_name")),
		OrderModalMobile:     strings.TrimSpace(r.URL.Query().Get("modal_order_customer_mobile")),
		OrderModalHasAddress: parseOptionalBool(r.URL.Query().Get("modal_order_has_address")),
		OrderModalAddressReq: parseOptionalBool(r.URL.Query().Get("modal_order_require_address")),
		OrderModalAmount:     strings.TrimSpace(r.URL.Query().Get("order_amount")),
		OrderModalOrderNote:  strings.TrimSpace(r.URL.Query().Get("note")),
		OrderModalAddress:    strings.TrimSpace(r.URL.Query().Get("address")),
		OrderAmountError:     strings.TrimSpace(r.URL.Query().Get("order_amount_error")),
		OrderAddressError:    strings.TrimSpace(r.URL.Query().Get("address_error")),
		CustomerOptions:      customerOptions,
		customerByID:         customerByID,
		customerAddressByID:  customerAddressByID,
	}
	if err := enquiriesListTemplate.Execute(w, vm); err != nil {
		http.Error(w, "failed to render enquiries list", http.StatusInternalServerError)
	}
}

func (s *Server) convertEnquiryToInterested(w http.ResponseWriter, r *http.Request, enquiryID int) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}

	customerID, customerErr := resolveCustomerIDForConversion(r, s.customerStore)
	if customerErr != "" {
		http.Redirect(w, r, enquiriesValidationRedirect(clicked.StatusClicked, enquiryID, customerErr, r), http.StatusSeeOther)
		return
	}
	note := strings.TrimSpace(r.FormValue("note"))
	if len(note) > 500 {
		http.Redirect(w, r, enquiriesValidationRedirect(clicked.StatusClicked, enquiryID, "Note must be 500 characters or fewer.", r), http.StatusSeeOther)
		return
	}
	_, alreadyConverted, err := s.clickedStore.ConvertToInterested(enquiryID, clicked.ConvertInput{
		CustomerID: customerID,
		Note:       note,
		ModifiedBy: defaultConvertedBy,
	})
	if err != nil {
		if err == clicked.ErrNotFound {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "failed to convert enquiry", http.StatusInternalServerError)
		return
	}
	if alreadyConverted {
		http.Redirect(w, r, enquiriesRedirect(clicked.StatusClicked, "Already converted.", ""), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, enquiriesRedirect(clicked.StatusClicked, "Enquiry converted to Interested.", ""), http.StatusSeeOther)
}

func (s *Server) convertEnquiryToOrdered(w http.ResponseWriter, r *http.Request, enquiryID int) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}

	current, customer, alreadyOrdered, err := s.resolveOrderConversionTarget(enquiryID)
	if handled := s.handleOrderTargetError(w, r, current, alreadyOrdered, err); handled {
		return
	}

	hasAddress := customerHasAddress(customer)
	orderAmount, note, addressInput, fieldErrors := parseOrderSubmission(r, hasAddress)
	if len(fieldErrors) > 0 {
		toast := buildValidationToast(fieldErrors, []string{"order_amount", "address"})
		http.Redirect(w, r, enquiriesOrderValidationRedirect(clicked.StatusInterested, enquiryID, toast, r, customer.Name, customer.Mobile, hasAddress, fieldErrors["address"] != "", fieldErrors), http.StatusSeeOther)
		return
	}

	if err := s.persistCustomerAddressForOrdered(customer, hasAddress, addressInput); err != nil {
		http.Error(w, "failed to update customer address", http.StatusInternalServerError)
		return
	}

	_, alreadyOrdered, err = s.clickedStore.ConvertToOrdered(enquiryID, clicked.OrderInput{
		OrderAmount: orderAmount,
		Note:        note,
		Address:     addressInput,
		ModifiedBy:  defaultConvertedBy,
	})
	if handled := s.handleOrderConversionStoreError(w, r, enquiryID, customer, hasAddress, alreadyOrdered, err); handled {
		return
	}
	http.Redirect(w, r, enquiriesRedirect(clicked.StatusInterested, "Enquiry converted to Ordered.", ""), http.StatusSeeOther)
}

func (s *Server) resolveOrderConversionTarget(enquiryID int) (clicked.Enquiry, customers.Customer, bool, error) {
	current, err := s.clickedStore.Get(enquiryID)
	if err != nil {
		return clicked.Enquiry{}, customers.Customer{}, false, err
	}
	if current.Status == clicked.StatusOrdered {
		return current, customers.Customer{}, true, nil
	}
	if current.Status != clicked.StatusInterested || current.CustomerID <= 0 {
		return current, customers.Customer{}, false, clicked.ErrInvalidTransition
	}

	customer, err := s.customerStore.Get(current.CustomerID)
	if err != nil {
		return current, customers.Customer{}, false, err
	}
	return current, customer, false, nil
}

func (s *Server) handleOrderTargetError(w http.ResponseWriter, r *http.Request, enquiry clicked.Enquiry, alreadyOrdered bool, err error) bool {
	if err == nil {
		return false
	}
	if err == clicked.ErrNotFound {
		http.NotFound(w, r)
		return true
	}
	if alreadyOrdered {
		http.Redirect(w, r, enquiriesRedirect(clicked.StatusOrdered, "", "Already ordered"), http.StatusSeeOther)
		return true
	}
	if err == clicked.ErrInvalidTransition {
		http.Redirect(w, r, enquiriesRedirect(normalizeEnquiryStatusForRedirect(enquiry.Status), "", "Only interested enquiries can be ordered."), http.StatusSeeOther)
		return true
	}
	http.Error(w, "failed to load enquiry", http.StatusInternalServerError)
	return true
}

func parseOrderSubmission(r *http.Request, hasAddress bool) (int, string, string, map[string]string) {
	orderAmount, fieldErrors := parseOrderAmountField(r.FormValue("order_amount"))
	note := strings.TrimSpace(r.FormValue("note"))
	addressInput := strings.TrimSpace(r.FormValue("address"))
	if !hasAddress && addressInput == "" {
		fieldErrors["address"] = "Address is required to convert to Ordered."
	}
	return orderAmount, note, addressInput, fieldErrors
}

func (s *Server) persistCustomerAddressForOrdered(customer customers.Customer, hasAddress bool, addressInput string) error {
	if hasAddress {
		return nil
	}
	_, err := s.customerStore.Update(customer.ID, customers.UpdateInput{
		Name:          customer.Name,
		Address:       stringPointer(addressInput),
		CityName:      stringPointer(customer.CityName),
		ApartmentName: stringPointer(customer.ApartmentName),
		Notes:         customer.Notes,
	})
	return err
}

func (s *Server) handleOrderConversionStoreError(w http.ResponseWriter, r *http.Request, enquiryID int, customer customers.Customer, hasAddress bool, alreadyOrdered bool, err error) bool {
	if err == nil && !alreadyOrdered {
		return false
	}
	if alreadyOrdered {
		http.Redirect(w, r, enquiriesRedirect(clicked.StatusOrdered, "", "Already ordered"), http.StatusSeeOther)
		return true
	}
	if err == clicked.ErrNotFound {
		http.NotFound(w, r)
		return true
	}
	if err == clicked.ErrInvalidTransition {
		http.Redirect(w, r, enquiriesRedirect(clicked.StatusInterested, "", "Only interested enquiries can be ordered."), http.StatusSeeOther)
		return true
	}
	if err == clicked.ErrAddressRequired {
		fieldErrs := map[string]string{"address": "Address is required to convert to Ordered."}
		toast := buildValidationToast(fieldErrs, []string{"address"})
		http.Redirect(w, r, enquiriesOrderValidationRedirect(clicked.StatusInterested, enquiryID, toast, r, customer.Name, customer.Mobile, hasAddress, true, fieldErrs), http.StatusSeeOther)
		return true
	}
	http.Error(w, "failed to convert enquiry", http.StatusInternalServerError)
	return true
}

func enquiriesRedirect(status clicked.Status, flash string, errMsg string) string {
	base := "/admin/enquiries?status=" + url.QueryEscape(string(status))
	if flash != "" {
		return base + "&flash=" + url.QueryEscape(flash)
	}
	if errMsg != "" {
		return base + "&error=" + url.QueryEscape(errMsg)
	}
	return base
}

func enquiriesValidationRedirect(status clicked.Status, enquiryID int, errMsg string, r *http.Request) string {
	redirect := enquiriesRedirect(status, "", errMsg)
	return redirect +
		"&open_convert_modal=1" +
		"&modal_enquiry_id=" + url.QueryEscape(strconv.Itoa(enquiryID)) +
		"&customer_id=" + url.QueryEscape(strings.TrimSpace(r.FormValue("customer_id"))) +
		"&quick_customer_name=" + url.QueryEscape(strings.TrimSpace(r.FormValue("quick_customer_name"))) +
		"&quick_customer_mobile=" + url.QueryEscape(strings.TrimSpace(r.FormValue("quick_customer_mobile"))) +
		"&note=" + url.QueryEscape(strings.TrimSpace(r.FormValue("note"))) +
		"&customer_search=" + url.QueryEscape(strings.TrimSpace(r.FormValue("customer_search")))
}

func enquiriesOrderValidationRedirect(status clicked.Status, enquiryID int, validationToast string, r *http.Request, customerName string, customerMobile string, hasAddress bool, requireAddress bool, fieldErrors map[string]string) string {
	values := url.Values{}
	values.Set("status", string(status))
	if validationToast != "" {
		values.Set("error", validationToast)
	}
	values.Set("open_order_modal", "1")
	values.Set("modal_order_enquiry_id", strconv.Itoa(enquiryID))
	values.Set("modal_order_customer_name", strings.TrimSpace(customerName))
	values.Set("modal_order_customer_mobile", strings.TrimSpace(customerMobile))
	if hasAddress {
		values.Set("modal_order_has_address", "1")
	} else {
		values.Set("modal_order_has_address", "0")
	}
	if requireAddress {
		values.Set("modal_order_require_address", "1")
	}
	values.Set("order_amount", strings.TrimSpace(r.FormValue("order_amount")))
	values.Set("note", strings.TrimSpace(r.FormValue("note")))
	values.Set("address", strings.TrimSpace(r.FormValue("address")))
	if msg := strings.TrimSpace(fieldErrors["order_amount"]); msg != "" {
		values.Set("order_amount_error", msg)
	}
	if msg := strings.TrimSpace(fieldErrors["address"]); msg != "" {
		values.Set("address_error", msg)
	}
	return "/admin/enquiries?" + values.Encode()
}

func parseModalEnquiryID(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return 0
	}
	return value
}

func parseOptionalBool(raw string) bool {
	value := strings.TrimSpace(raw)
	return value == "1" || strings.EqualFold(value, "true")
}

func parseOrderAmountField(raw string) (int, map[string]string) {
	out := map[string]string{}
	value := strings.TrimSpace(raw)
	amount, err := strconv.Atoi(value)
	if err != nil || amount <= 0 {
		out["order_amount"] = "Order amount must be a whole number greater than 0."
		return 0, out
	}
	return amount, out
}

func normalizeEnquiryStatusForRedirect(status clicked.Status) clicked.Status {
	if clicked.IsValidStatus(status) {
		return status
	}
	return clicked.StatusClicked
}

func customerHasAddress(customer customers.Customer) bool {
	if customer.Address == nil {
		return false
	}
	return strings.TrimSpace(*customer.Address) != ""
}

func stringPointer(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func resolveCustomerIDForConversion(r *http.Request, customerStore customers.Store) (int, string) {
	quickName := strings.TrimSpace(r.FormValue("quick_customer_name"))
	quickMobile := strings.TrimSpace(r.FormValue("quick_customer_mobile"))
	if customerID, ok := parseSelectedCustomerID(r.FormValue("customer_id")); ok {
		if quickName != "" || quickMobile != "" {
			return 0, "Choose either existing customer or quick-create details."
		}
		if _, err := customerStore.Get(customerID); err == nil {
			return customerID, ""
		}
		return 0, "Please choose a valid customer."
	}
	return resolveQuickCreateCustomerID(customerStore, quickName, quickMobile)
}

func parseSelectedCustomerID(raw string) (int, bool) {
	customerID, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || customerID <= 0 {
		return 0, false
	}
	return customerID, true
}

func resolveQuickCreateCustomerID(customerStore customers.Store, quickName string, quickMobile string) (int, string) {
	if quickName == "" && quickMobile == "" {
		return 0, "Customer selection is required."
	}
	if quickName == "" || quickMobile == "" {
		return 0, "Quick-create requires both customer name and mobile."
	}
	if len(quickName) < 2 || len(quickName) > 100 {
		return 0, "Quick-create customer name must be between 2 and 100 characters."
	}
	normalizedMobile := customers.NormalizeMobile(quickMobile)
	if len(normalizedMobile) != 10 {
		return 0, "Quick-create customer mobile must be exactly 10 digits."
	}

	created, err := customerStore.Create(customers.CreateInput{Name: quickName, Mobile: quickMobile})
	if err == nil {
		return created.ID, ""
	}
	return resolveDuplicateQuickCreate(customerStore, err)
}

func resolveDuplicateQuickCreate(customerStore customers.Store, createErr error) (int, string) {
	var dupErr *customers.DuplicateMobileError
	if !errors.As(createErr, &dupErr) || dupErr.CustomerID <= 0 {
		return 0, "Failed to quick-create customer."
	}
	if _, err := customerStore.Get(dupErr.CustomerID); err != nil {
		return 0, "Failed to quick-create customer."
	}
	return dupErr.CustomerID, ""
}

type enquiriesListViewModel struct {
	ActiveSection        string
	SelectedStatus       clicked.Status
	Rows                 []clicked.Enquiry
	Flash                string
	ValidationToast      string
	OpenConvertModal     bool
	ModalEnquiryID       int
	ModalCustomerID      string
	ModalQuickName       string
	ModalQuickMobile     string
	ModalNote            string
	ModalSearch          string
	OpenOrderModal       bool
	OrderModalEnquiryID  int
	OrderModalCustomer   string
	OrderModalMobile     string
	OrderModalHasAddress bool
	OrderModalAddressReq bool
	OrderModalAmount     string
	OrderModalOrderNote  string
	OrderModalAddress    string
	OrderAmountError     string
	OrderAddressError    string
	CustomerOptions      []customers.ListItem
	customerByID         map[int]customers.ListItem
	customerAddressByID  map[int]string
}

func (v enquiriesListViewModel) TabClass(status clicked.Status) string {
	if v.SelectedStatus == status {
		return "status-tab active"
	}
	return "status-tab"
}

func (v enquiriesListViewModel) IsClickedTab() bool {
	return v.SelectedStatus == clicked.StatusClicked
}

func (v enquiriesListViewModel) IsInterestedTab() bool {
	return v.SelectedStatus == clicked.StatusInterested
}

func (v enquiriesListViewModel) IsOrderedTab() bool {
	return v.SelectedStatus == clicked.StatusOrdered
}

func (v enquiriesListViewModel) HasActionColumn() bool {
	return v.IsClickedTab() || v.IsInterestedTab()
}

func statusLabel(status clicked.Status) string {
	switch status {
	case clicked.StatusClicked:
		return "Clicked"
	case clicked.StatusInterested:
		return "Interested"
	case clicked.StatusOrdered:
		return "Ordered"
	default:
		return string(status)
	}
}

func (v enquiriesListViewModel) CustomerName(id int) string {
	item, ok := v.customerByID[id]
	if !ok {
		return ""
	}
	return item.Name
}

func (v enquiriesListViewModel) CustomerMobile(id int) string {
	item, ok := v.customerByID[id]
	if !ok {
		return ""
	}
	return item.Mobile
}

func (v enquiriesListViewModel) CustomerHasAddress(id int) bool {
	return strings.TrimSpace(v.customerAddressByID[id]) != ""
}

var enquiriesListTemplate = template.Must(template.New("enquiries-list").Funcs(template.FuncMap{
	"adminNav":    adminNav,
	"statusLabel": statusLabel,
}).Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Enquiries</title>
  <style>
    :root { --bg:#f6f8fb; --card:#fff; --line:#d9e1ea; --text:#1f2937; --accent:#0f766e; --muted:#4b5563; }
    * { box-sizing: border-box; }
    body { margin:0; font-family: "Segoe UI", Tahoma, sans-serif; background: var(--bg); color: var(--text); }
    header { background: var(--card); border-bottom:1px solid var(--line); }
    .shell { width:min(1100px, 94vw); margin:0 auto; padding:16px; }
    .admin-nav { display:flex; gap:14px; }
    .admin-nav-link { color: var(--accent); font-weight:600; text-decoration:none; padding:6px 10px; border-radius:8px; }
    .admin-nav-link.active { background:#e6f4f2; color:#0a5f57; }
    .header { display:flex; justify-content:space-between; align-items:center; gap:12px; margin-bottom:16px; }
    h1 { margin:0; font-size:1.8rem; }
    .tabs { display:flex; gap:8px; margin-bottom:16px; }
    .status-tab { text-decoration:none; padding:8px 12px; border-radius:999px; background:#e5e7eb; color:#1f2937; font-weight:600; }
    .status-tab.active { background:#0f766e; color:#fff; }
    .toast { margin-bottom:14px; padding:10px 12px; border-radius:8px; font-weight:600; }
    .toast.success { background:#d1fae5; color:#065f46; border:1px solid #6ee7b7; }
    .toast.error { background:#fee2e2; color:#991b1b; border:1px solid #fecaca; }
    table { width:100%; border-collapse:collapse; background:#fff; border:1px solid #e5e7eb; border-radius:10px; overflow:hidden; }
    th, td { text-align:left; padding:10px; border-bottom:1px solid #f0f1f5; vertical-align:top; }
    th { background:#f9fafb; font-size:0.9rem; color:#4b5563; }
    .muted { color:#6b7280; font-size:0.9rem; }
    button { border:none; border-radius:8px; background:#0f766e; color:#fff; padding:9px 12px; font-weight:700; cursor:pointer; }
    dialog { border:1px solid #d1d5db; border-radius:12px; width:min(720px, 96vw); padding:0; }
    .dialog-card { padding:16px; }
    .dialog-title { margin:0 0 10px; font-size:1.15rem; }
    .form-grid { display:grid; gap:10px; }
    .field { display:grid; gap:6px; }
    .field label { font-weight:600; }
    .field input, .field select, .field textarea { width:100%; border:1px solid #d1d5db; border-radius:8px; padding:8px; font:inherit; }
    .field textarea { min-height:70px; resize:vertical; }
    .section { border:1px solid #e5e7eb; border-radius:10px; padding:10px; background:#f9fafb; display:grid; gap:10px; }
    .section-title { margin:0; font-size:0.95rem; font-weight:700; color:#374151; text-transform:uppercase; letter-spacing:0.02em; }
    .section-divider { height:1px; background:#e5e7eb; margin:2px 0; }
    .dialog-row { display:flex; gap:8px; justify-content:flex-end; margin-top:8px; }
    .button-secondary { background:#e5e7eb; color:#1f2937; }
    .help { color:#6b7280; font-size:0.85rem; }
    .help.error { color:#991b1b; }
    .readonly { background:#f3f4f6; color:#4b5563; }
    .hidden { display:none; }
  </style>
</head>
<body>
  <header>
    <div class="shell">{{adminNav .ActiveSection}}</div>
  </header>
  <main class="shell">
    <div class="header">
      <h1>Enquiries</h1>
    </div>
    <div class="tabs">
      <a href="/admin/enquiries?status=clicked" class="{{.TabClass "clicked"}}">Clicked</a>
      <a href="/admin/enquiries?status=interested" class="{{.TabClass "interested"}}">Interested</a>
      <a href="/admin/enquiries?status=ordered" class="{{.TabClass "ordered"}}">Ordered</a>
    </div>
    {{if .Flash}}<div class="toast success">{{.Flash}}</div>{{end}}
    {{if .ValidationToast}}{{if and (not .OpenConvertModal) (not .OpenOrderModal)}}<div class="toast error">{{.ValidationToast}}</div>{{end}}{{end}}
    <table>
      <thead>
        <tr>
          <th>Created</th>
          <th>Item</th>
          <th>Source</th>
          <th>Status</th>
          <th>Buyer</th>
          {{if .HasActionColumn}}<th>Action</th>{{end}}
        </tr>
      </thead>
      <tbody>
        {{range .Rows}}
        <tr>
          <td>{{.CreatedAt.Format "2006-01-02 15:04"}}</td>
          <td>
            <div><strong>{{.ItemTitle}}</strong></div>
            <div class="muted">{{.ItemType}} #{{.ItemID}}</div>
          </td>
          <td>
            <div>{{.SourcePage}}</div>
            <div class="muted">{{.SourceRail}}</div>
          </td>
          <td>{{statusLabel .Status}}</td>
          <td>
            {{if eq .Status "interested"}}
              <div><strong>{{$.CustomerName .CustomerID}}</strong></div>
              <div class="muted">{{$.CustomerMobile .CustomerID}}</div>
              {{if .Note}}<div class="muted">{{.Note}}</div>{{end}}
              <div class="muted">By {{.LastModifiedBy}}</div>
            {{else if eq .Status "ordered"}}
              <div><strong>{{$.CustomerName .CustomerID}}</strong></div>
              <div class="muted">{{$.CustomerMobile .CustomerID}}</div>
              {{if .OrderAmount}}<div class="muted">Order Amount: ₹{{.OrderAmount}}</div>{{end}}
              {{if .Note}}<div class="muted">{{.Note}}</div>{{end}}
              <div class="muted">By {{.LastModifiedBy}}</div>
            {{else}}
              <span class="muted">Not captured</span>
            {{end}}
          </td>
          {{if $.HasActionColumn}}
          <td>
            {{if $.IsClickedTab}}
            <button type="button" class="open-convert-dialog" data-enquiry-id="{{.ID}}">Convert to Interested</button>
            {{else if $.IsInterestedTab}}
            <button
              type="button"
              class="open-order-dialog"
              data-enquiry-id="{{.ID}}"
              data-customer-name="{{$.CustomerName .CustomerID}}"
              data-customer-mobile="{{$.CustomerMobile .CustomerID}}"
              data-has-address="{{if $.CustomerHasAddress .CustomerID}}1{{else}}0{{end}}"
            >Convert to Ordered</button>
            {{end}}
          </td>
          {{end}}
        </tr>
        {{else}}
        <tr>
          <td colspan="{{if .HasActionColumn}}6{{else}}5{{end}}" class="muted">No enquiries found for this status.</td>
        </tr>
        {{end}}
      </tbody>
    </table>
  </main>
  {{if .IsClickedTab}}
  <dialog id="convert-dialog">
    <form
      id="convert-form"
      class="dialog-card"
      method="post"
      action=""
      data-open-on-load="{{if .OpenConvertModal}}1{{else}}0{{end}}"
      data-enquiry-id="{{.ModalEnquiryID}}"
      data-customer-id="{{.ModalCustomerID}}"
      data-quick-customer-name="{{.ModalQuickName}}"
      data-quick-customer-mobile="{{.ModalQuickMobile}}"
      data-note="{{.ModalNote}}"
      data-customer-search="{{.ModalSearch}}"
    >
      <h2 class="dialog-title">Convert to Interested</h2>
      {{if .OpenConvertModal}}{{if .ValidationToast}}<div class="toast error" role="alert">{{.ValidationToast}}</div>{{end}}{{end}}
      <div class="form-grid">
        <div class="section">
          <p class="section-title">Search existing customer</p>
          <div class="field">
            <label for="customer-search">Search customer</label>
            <input id="customer-search" name="customer_search" placeholder="Search by name or mobile">
          </div>
          <div class="field">
            <label for="customer-id">Customer (required)</label>
            <select id="customer-id" name="customer_id">
              <option value="">Select existing customer</option>
              {{range .CustomerOptions}}
              <option value="{{.ID}}" data-customer-label="{{.Name}} {{.Mobile}}">{{.Name}} — {{.Mobile}}</option>
              {{end}}
            </select>
          </div>
        </div>
        <div class="section-divider"></div>
        <div class="section">
          <p class="section-title">Quick-create customer (if no match)</p>
          <div class="field">
            <label for="quick-customer-name">Quick-create customer: Name</label>
            <input id="quick-customer-name" name="quick_customer_name" placeholder="Required only when creating new">
          </div>
          <div class="field">
            <label for="quick-customer-mobile">Quick-create customer: Mobile</label>
            <input id="quick-customer-mobile" name="quick_customer_mobile" placeholder="10-digit mobile">
          </div>
        </div>
        <div class="field">
          <label for="enquiry-note">Note (optional)</label>
          <textarea id="enquiry-note" name="note" placeholder="Optional note"></textarea>
        </div>
      </div>
      <div class="dialog-row">
        <button type="button" id="cancel-convert" class="button-secondary">Cancel</button>
        <button type="submit">Convert</button>
      </div>
    </form>
  </dialog>
  {{end}}
  {{if .IsInterestedTab}}
  <dialog id="order-dialog">
    <form
      id="order-form"
      class="dialog-card"
      method="post"
      action=""
      data-open-on-load="{{if .OpenOrderModal}}1{{else}}0{{end}}"
      data-enquiry-id="{{.OrderModalEnquiryID}}"
      data-customer-name="{{.OrderModalCustomer}}"
      data-customer-mobile="{{.OrderModalMobile}}"
      data-has-address="{{if .OrderModalHasAddress}}1{{else}}0{{end}}"
      data-require-address="{{if .OrderModalAddressReq}}1{{else}}0{{end}}"
      data-order-amount="{{.OrderModalAmount}}"
      data-note="{{.OrderModalOrderNote}}"
      data-address="{{.OrderModalAddress}}"
    >
      <h2 class="dialog-title">Convert to Ordered</h2>
      {{if .OpenOrderModal}}{{if .ValidationToast}}<div class="toast error" role="alert">{{.ValidationToast}}</div>{{end}}{{end}}
      <div class="form-grid">
        <div class="field">
          <label for="order-customer-name">Customer</label>
          <input id="order-customer-name" class="readonly" readonly>
        </div>
        <div class="field">
          <label for="order-customer-mobile">Mobile</label>
          <input id="order-customer-mobile" class="readonly" readonly>
        </div>
        <div class="field">
          <label for="order-amount">Order Amount (required)</label>
          <input id="order-amount" name="order_amount" inputmode="numeric" placeholder="Enter order amount">
          {{if .OrderAmountError}}<div class="help error">{{.OrderAmountError}}</div>{{end}}
        </div>
        <div class="field hidden" id="order-address-field">
          <label for="order-address">Address</label>
          <textarea id="order-address" name="address" placeholder="Enter customer address"></textarea>
          {{if .OrderAddressError}}<div class="help error">{{.OrderAddressError}}</div>{{end}}
        </div>
        <div class="field">
          <label for="order-note">Note (optional)</label>
          <textarea id="order-note" name="note" placeholder="Optional note"></textarea>
        </div>
      </div>
      <input type="hidden" id="order-customer-name-hidden" name="order_customer_name" value="">
      <input type="hidden" id="order-customer-mobile-hidden" name="order_customer_mobile" value="">
      <input type="hidden" id="order-customer-has-address-hidden" name="order_customer_has_address" value="0">
      <div class="dialog-row">
        <button type="button" id="cancel-order" class="button-secondary">Cancel</button>
        <button type="submit">Convert</button>
      </div>
    </form>
  </dialog>
  {{end}}
  {{if or .IsClickedTab .IsInterestedTab}}
  <script src="/assets/enquiries-form.js" defer></script>
  {{end}}
</body>
</html>`))
