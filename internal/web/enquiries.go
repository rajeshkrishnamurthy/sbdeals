package web

import (
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/rajeshkrishnamurthy/sbdeals/internal/clicked"
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

	vm := enquiriesListViewModel{
		ActiveSection:   "enquiries",
		SelectedStatus:  status,
		Rows:            items,
		Flash:           strings.TrimSpace(r.URL.Query().Get("flash")),
		ValidationToast: strings.TrimSpace(r.URL.Query().Get("error")),
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

	buyerName := strings.TrimSpace(r.FormValue("buyer_name"))
	localPhone := strings.TrimSpace(r.FormValue("buyer_phone"))
	buyerNote := strings.TrimSpace(r.FormValue("buyer_note"))

	if buyerName == "" {
		http.Redirect(w, r, enquiriesRedirect(clicked.StatusClicked, "", "Buyer name is required."), http.StatusSeeOther)
		return
	}

	phone, ok := clicked.NormalizeIndiaPhone(localPhone)
	if !ok {
		http.Redirect(w, r, enquiriesRedirect(clicked.StatusClicked, "", "Please enter a valid 10-digit India mobile number."), http.StatusSeeOther)
		return
	}

	_, alreadyConverted, err := s.clickedStore.ConvertToInterested(enquiryID, clicked.ConvertInput{
		BuyerName:   buyerName,
		BuyerPhone:  phone,
		BuyerNote:   buyerNote,
		ConvertedBy: defaultConvertedBy,
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

type enquiriesListViewModel struct {
	ActiveSection   string
	SelectedStatus  clicked.Status
	Rows            []clicked.Enquiry
	Flash           string
	ValidationToast string
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

func localPhoneDisplay(value string) string {
	trimmed := strings.TrimSpace(value)
	return strings.TrimPrefix(trimmed, "+91")
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

var enquiriesListTemplate = template.Must(template.New("enquiries-list").Funcs(template.FuncMap{
	"adminNav":          adminNav,
	"statusLabel":       statusLabel,
	"localPhoneDisplay": localPhoneDisplay,
}).Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Enquiries</title>
  <style>
    body { font-family: "Segoe UI", sans-serif; background:#f6f7fb; margin:0; color:#1f2937; }
    .container { max-width:1120px; margin:30px auto; padding:0 20px; }
    .shell { margin-bottom:16px; }
    .admin-nav { display:flex; gap:10px; flex-wrap:wrap; }
    .admin-nav-link { text-decoration:none; color:#1f2937; background:#e5e7eb; padding:8px 12px; border-radius:8px; font-weight:600; }
    .admin-nav-link.active { background:#0f766e; color:#fff; }
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
    .inline-form { display:grid; gap:8px; }
    .inline-form-row { display:flex; gap:8px; flex-wrap:wrap; align-items:center; }
    input, textarea { border:1px solid #d1d5db; border-radius:8px; padding:8px; font:inherit; }
    input { width:170px; }
    textarea { width:min(360px, 100%); min-height:44px; }
    .phone-prefix { background:#eef2ff; border:1px solid #d1d5db; border-radius:8px; padding:8px; font-weight:600; }
    button { border:none; border-radius:8px; background:#0f766e; color:#fff; padding:9px 12px; font-weight:700; cursor:pointer; }
  </style>
</head>
<body>
  <div class="container">
    <div class="shell">{{adminNav .ActiveSection}}</div>
    <div class="header">
      <h1>Enquiries</h1>
    </div>
    <div class="tabs">
      <a href="/admin/enquiries?status=clicked" class="{{.TabClass "clicked"}}">Clicked</a>
      <a href="/admin/enquiries?status=interested" class="{{.TabClass "interested"}}">Interested</a>
    </div>
    {{if .Flash}}<div class="toast success">{{.Flash}}</div>{{end}}
    {{if .ValidationToast}}<div class="toast error">{{.ValidationToast}}</div>{{end}}
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
              <div><strong>{{.BuyerName}}</strong></div>
              <div class="muted">{{.BuyerPhone}}</div>
              {{if .BuyerNote}}<div class="muted">{{.BuyerNote}}</div>{{end}}
              <div class="muted">By {{.ConvertedBy}}</div>
            {{else}}
              <span class="muted">Not captured</span>
            {{end}}
          </td>
          {{if $.IsClickedTab}}
          <td>
            <form method="post" action="{{printf "/admin/enquiries/%d/convert" .ID}}" class="inline-form">
              <div class="inline-form-row">
                <input name="buyer_name" placeholder="Buyer name" required>
                <span class="phone-prefix">+91</span>
                <input name="buyer_phone" placeholder="10-digit mobile" required>
              </div>
              <div class="inline-form-row">
                <textarea name="buyer_note" placeholder="Note (optional)"></textarea>
              </div>
              <div class="inline-form-row">
                <button type="submit">Convert to Interested</button>
              </div>
            </form>
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
  </div>
</body>
</html>`))
