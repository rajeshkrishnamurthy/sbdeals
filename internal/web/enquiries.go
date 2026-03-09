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
	if action != "convert" || r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}
	s.convertEnquiryToInterested(w, r, id)
}

func parseEnquiryPath(path string) (int, string, bool) {
	prefix := "/admin/enquiries/"
	if !strings.HasPrefix(path, prefix) {
		return 0, "", false
	}
	rest := strings.TrimPrefix(path, prefix)
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || parts[1] != "convert" {
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

	vm := enquiriesListViewModel{
		ActiveSection:   "enquiries",
		SelectedStatus:  status,
		Rows:            items,
		Flash:           strings.TrimSpace(r.URL.Query().Get("flash")),
		ValidationToast: strings.TrimSpace(r.URL.Query().Get("error")),
		OpenConvertModal: strings.TrimSpace(r.URL.Query().Get("open_convert_modal")) == "1",
		ModalEnquiryID:   parseModalEnquiryID(r.URL.Query().Get("modal_enquiry_id")),
		ModalCustomerID:  strings.TrimSpace(r.URL.Query().Get("customer_id")),
		ModalQuickName:   strings.TrimSpace(r.URL.Query().Get("quick_customer_name")),
		ModalQuickMobile: strings.TrimSpace(r.URL.Query().Get("quick_customer_mobile")),
		ModalNote:        strings.TrimSpace(r.URL.Query().Get("note")),
		ModalSearch:      strings.TrimSpace(r.URL.Query().Get("customer_search")),
		CustomerOptions: customerOptions,
		customerByID:    customerByID,
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

func parseModalEnquiryID(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return 0
	}
	return value
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
	ActiveSection   string
	SelectedStatus  clicked.Status
	Rows            []clicked.Enquiry
	Flash           string
	ValidationToast string
	OpenConvertModal bool
	ModalEnquiryID   int
	ModalCustomerID  string
	ModalQuickName   string
	ModalQuickMobile string
	ModalNote        string
	ModalSearch      string
	CustomerOptions []customers.ListItem
	customerByID    map[int]customers.ListItem
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

func statusLabel(status clicked.Status) string {
	switch status {
	case clicked.StatusClicked:
		return "Clicked"
	case clicked.StatusInterested:
		return "Interested"
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
    </div>
    {{if .Flash}}<div class="toast success">{{.Flash}}</div>{{end}}
    {{if .ValidationToast}}{{if not .OpenConvertModal}}<div class="toast error">{{.ValidationToast}}</div>{{end}}{{end}}
    <table>
      <thead>
        <tr>
          <th>Created</th>
          <th>Item</th>
          <th>Source</th>
          <th>Status</th>
          <th>Buyer</th>
          {{if .IsClickedTab}}<th>Action</th>{{end}}
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
            {{else}}
              <span class="muted">Not captured</span>
            {{end}}
          </td>
          {{if $.IsClickedTab}}
          <td>
            <button type="button" class="open-convert-dialog" data-enquiry-id="{{.ID}}">Convert to Interested</button>
          </td>
          {{end}}
        </tr>
        {{else}}
        <tr>
          <td colspan="{{if .IsClickedTab}}6{{else}}5{{end}}" class="muted">No enquiries found for this status.</td>
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
  <script>
    (function () {
      var dialog = document.getElementById("convert-dialog");
      var form = document.getElementById("convert-form");
      var cancel = document.getElementById("cancel-convert");
      var search = document.getElementById("customer-search");
      var select = document.getElementById("customer-id");
      var quickName = document.getElementById("quick-customer-name");
      var quickMobile = document.getElementById("quick-customer-mobile");
      var note = document.getElementById("enquiry-note");
      if (!dialog || !form || !cancel || !search || !select) return;

      var originalOptions = Array.prototype.slice.call(select.querySelectorAll("option"));

      function filterOptions() {
        var q = (search.value || "").toLowerCase().trim();
        select.innerHTML = "";
        originalOptions.forEach(function (option) {
          if (!option.value) {
            select.appendChild(option.cloneNode(true));
            return;
          }
          var label = (option.getAttribute("data-customer-label") || "").toLowerCase();
          if (!q || label.indexOf(q) >= 0) {
            select.appendChild(option.cloneNode(true));
          }
        });
      }

      function openDialog(state) {
        if (!state || !state.enquiryID) return;
        form.setAttribute("action", "/admin/enquiries/" + state.enquiryID + "/convert");
        form.reset();
        search.value = state.search || "";
        if (quickName) quickName.value = state.quickName || "";
        if (quickMobile) quickMobile.value = state.quickMobile || "";
        if (note) note.value = state.note || "";
        filterOptions();
        if (state.customerID) {
          select.value = state.customerID;
        }
        dialog.showModal();
      }

      document.querySelectorAll(".open-convert-dialog").forEach(function (button) {
        button.addEventListener("click", function () {
          openDialog({ enquiryID: button.getAttribute("data-enquiry-id") });
        });
      });

      cancel.addEventListener("click", function () {
        dialog.close();
      });

      search.addEventListener("input", filterOptions);

      if (form.getAttribute("data-open-on-load") === "1") {
        openDialog({
          enquiryID: form.getAttribute("data-enquiry-id"),
          customerID: form.getAttribute("data-customer-id"),
          quickName: form.getAttribute("data-quick-customer-name"),
          quickMobile: form.getAttribute("data-quick-customer-mobile"),
          note: form.getAttribute("data-note"),
          search: form.getAttribute("data-customer-search")
        });
      }
    })();
  </script>
  {{end}}
</body>
</html>`))
