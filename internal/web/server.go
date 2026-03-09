package web

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/rajeshkrishnamurthy/sbdeals/internal/books"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/bundles"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/clicked"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/customers"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/rails"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/suppliers"
)

var defaultLocations = []string{
	"Bengaluru",
	"Chennai",
	"Hyderabad",
	"Mumbai",
	"Pune",
}

// Server provides HTTP handlers for the Sprint 1 Supplier admin feature.
type Server struct {
	store         suppliers.Store
	bookStore     books.Store
	bundleStore   bundles.Store
	railStore     rails.Store
	clickedStore  clicked.Store
	customerStore customers.Store
	locations     []string
	mux           *http.ServeMux
}

func NewServer(store suppliers.Store, bookStores ...books.Store) *Server {
	bookStore := books.Store(books.NewMemoryStore())
	if len(bookStores) > 0 && bookStores[0] != nil {
		bookStore = bookStores[0]
	}
	bundleStore := bundles.Store(bundles.NewMemoryStore(nil, nil))
	railStore := rails.Store(rails.NewMemoryStore())
	clickedStore := clicked.Store(clicked.NewMemoryStore())
	customerStore := customers.Store(customers.NewMemoryStore())
	return NewServerWithAllStores(store, bookStore, bundleStore, railStore, clickedStore, customerStore)
}

func NewServerWithStores(store suppliers.Store, bookStore books.Store, bundleStore bundles.Store, railStores ...rails.Store) *Server {
	railStore := rails.Store(rails.NewMemoryStore())
	if len(railStores) > 0 && railStores[0] != nil {
		railStore = railStores[0]
	}
	clickedStore := clicked.Store(clicked.NewMemoryStore())
	customerStore := customers.Store(customers.NewMemoryStore())
	return NewServerWithAllStores(store, bookStore, bundleStore, railStore, clickedStore, customerStore)
}

func NewServerWithStoresAndClicked(store suppliers.Store, bookStore books.Store, bundleStore bundles.Store, railStore rails.Store, clickedStore clicked.Store) *Server {
	customerStore := customers.Store(customers.NewMemoryStore())
	return NewServerWithAllStores(store, bookStore, bundleStore, railStore, clickedStore, customerStore)
}

func NewServerWithAllStores(store suppliers.Store, bookStore books.Store, bundleStore bundles.Store, railStore rails.Store, clickedStore clicked.Store, customerStore customers.Store) *Server {
	s := &Server{
		store:         store,
		bookStore:     bookStore,
		bundleStore:   bundleStore,
		railStore:     railStore,
		clickedStore:  clickedStore,
		customerStore: customerStore,
		locations:     append([]string(nil), defaultLocations...),
		mux:           http.NewServeMux(),
	}

	s.mux.HandleFunc("/", s.handleRoot)
	s.mux.HandleFunc("/api/catalog", s.handleCatalogData)
	s.mux.HandleFunc("/api/clicked", s.handleClickedCreate)
	s.mux.HandleFunc("/admin/suppliers", s.handleSuppliersCollection)
	s.mux.HandleFunc("/admin/suppliers/new", s.handleSupplierNew)
	s.mux.HandleFunc("/admin/suppliers/", s.handleSupplierItem)
	s.mux.HandleFunc("/admin/books", s.handleBooksCollection)
	s.mux.HandleFunc("/admin/books/new", s.handleBookNew)
	s.mux.HandleFunc("/admin/books/", s.handleBookItem)
	s.mux.HandleFunc("/admin/bundles", s.handleBundlesCollection)
	s.mux.HandleFunc("/admin/bundles/new", s.handleBundleNew)
	s.mux.HandleFunc("/admin/bundles/", s.handleBundleItem)
	s.mux.HandleFunc("/admin/rails", s.handleRailsCollection)
	s.mux.HandleFunc("/admin/rails/new", s.handleRailNew)
	s.mux.HandleFunc("/admin/rails/", s.handleRailItem)
	s.mux.HandleFunc("/admin/enquiries", s.handleEnquiriesCollection)
	s.mux.HandleFunc("/admin/enquiries/", s.handleEnquiryItem)
	s.mux.HandleFunc("/admin/customers", s.handleCustomersCollection)
	s.mux.HandleFunc("/admin/customers/new", s.handleCustomerNew)
	s.mux.HandleFunc("/admin/customers/", s.handleCustomerItem)
	s.mux.HandleFunc("/assets/books-form.js", s.handleBooksFormJSAsset)
	s.mux.HandleFunc("/assets/bundles-form.js", s.handleBundlesFormJSAsset)
	s.mux.HandleFunc("/assets/rails-form.js", s.handleRailsFormJSAsset)
	s.mux.HandleFunc("/assets/catalog.js", s.handleCatalogJSAsset)
	s.mux.HandleFunc("/assets/customers-form.js", s.handleCustomersFormJSAsset)
	s.mux.HandleFunc("/assets/enquiries-form.js", s.handleEnquiriesFormJSAsset)

	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	s.renderCatalogPage(w, r)
}

func (s *Server) handleSuppliersCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.renderSupplierList(w, r)
	case http.MethodPost:
		s.createSupplier(w, r)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (s *Server) handleSupplierNew(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	s.renderSupplierForm(w, supplierFormViewModel{
		PageTitle:     "Add Supplier",
		Action:        "/admin/suppliers",
		SubmitLabel:   "Save Supplier",
		ActiveSection: "suppliers",
		Input:         suppliers.Input{},
		Locations:     s.locations,
		Errors:        map[string]string{},
	})
}

func (s *Server) handleSupplierItem(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/admin/suppliers/" {
		http.NotFound(w, r)
		return
	}

	id, ok := parseSupplierID(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.renderSupplierDetail(w, r, id)
	case http.MethodPost:
		s.updateSupplier(w, r, id)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (s *Server) renderSupplierList(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.List()
	if err != nil {
		http.Error(w, "failed to load suppliers", http.StatusInternalServerError)
		return
	}

	data := supplierListViewModel{
		Flash:         r.URL.Query().Get("flash"),
		ActiveSection: "suppliers",
		Suppliers:     items,
	}

	if err := supplierListTemplate.Execute(w, data); err != nil {
		http.Error(w, "failed to render suppliers list", http.StatusInternalServerError)
	}
}

func (s *Server) renderSupplierDetail(w http.ResponseWriter, r *http.Request, supplierID int) {
	item, err := s.store.Get(supplierID)
	if errors.Is(err, suppliers.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "failed to load supplier", http.StatusInternalServerError)
		return
	}

	s.renderSupplierForm(w, supplierFormViewModel{
		PageTitle:     "View/Edit Supplier",
		Action:        fmt.Sprintf("/admin/suppliers/%d", supplierID),
		SubmitLabel:   "Save Changes",
		Flash:         r.URL.Query().Get("flash"),
		ActiveSection: "suppliers",
		Input: suppliers.Input{
			Name:     item.Name,
			WhatsApp: item.WhatsApp,
			Location: item.Location,
			Notes:    item.Notes,
		},
		Locations: s.locations,
		Errors:    map[string]string{},
	})
}

func (s *Server) createSupplier(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}

	input := readInputFromForm(r)
	errorsByField := validateInput(input, s.locations)
	if len(errorsByField) > 0 {
		w.WriteHeader(http.StatusBadRequest)
		s.renderSupplierForm(w, supplierFormViewModel{
			PageTitle:     "Add Supplier",
			Action:        "/admin/suppliers",
			SubmitLabel:   "Save Supplier",
			ActiveSection: "suppliers",
			Input:         input,
			Locations:     s.locations,
			Errors:        errorsByField,
			ValidationToast: buildValidationToast(errorsByField, []string{
				"name",
				"whatsapp",
				"location",
			}),
		})
		return
	}

	if _, err := s.store.Create(input); err != nil {
		http.Error(w, "failed to create supplier", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/suppliers?flash="+url.QueryEscape("Supplier created successfully."), http.StatusSeeOther)
}

func (s *Server) updateSupplier(w http.ResponseWriter, r *http.Request, supplierID int) {
	if _, err := s.store.Get(supplierID); errors.Is(err, suppliers.ErrNotFound) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, "failed to load supplier", http.StatusInternalServerError)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}

	input := readInputFromForm(r)
	errorsByField := validateInput(input, s.locations)
	if len(errorsByField) > 0 {
		w.WriteHeader(http.StatusBadRequest)
		s.renderSupplierForm(w, supplierFormViewModel{
			PageTitle:     "View/Edit Supplier",
			Action:        fmt.Sprintf("/admin/suppliers/%d", supplierID),
			SubmitLabel:   "Save Changes",
			ActiveSection: "suppliers",
			Input:         input,
			Locations:     s.locations,
			Errors:        errorsByField,
			ValidationToast: buildValidationToast(errorsByField, []string{
				"name",
				"whatsapp",
				"location",
			}),
		})
		return
	}

	if _, err := s.store.Update(supplierID, input); err != nil {
		if errors.Is(err, suppliers.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "failed to update supplier", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/suppliers/%d?flash=%s", supplierID, url.QueryEscape("Supplier updated successfully.")), http.StatusSeeOther)
}

func readInputFromForm(r *http.Request) suppliers.Input {
	return suppliers.Input{
		Name:     strings.TrimSpace(r.Form.Get("name")),
		WhatsApp: strings.TrimSpace(r.Form.Get("whatsapp")),
		Location: strings.TrimSpace(r.Form.Get("location")),
		Notes:    strings.TrimSpace(r.Form.Get("notes")),
	}
}

func validateInput(input suppliers.Input, allowedLocations []string) map[string]string {
	errs := map[string]string{}
	if input.Name == "" {
		errs["name"] = "Supplier name is required."
	}
	if input.WhatsApp == "" {
		errs["whatsapp"] = "WhatsApp number is required."
	}
	if input.Location == "" {
		errs["location"] = "Location is required."
	} else if !containsLocation(allowedLocations, input.Location) {
		errs["location"] = "Please choose a valid location from the dropdown."
	}
	return errs
}

func containsLocation(locations []string, target string) bool {
	for _, location := range locations {
		if location == target {
			return true
		}
	}
	return false
}

func parseSupplierID(path string) (int, bool) {
	prefix := "/admin/suppliers/"
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

func writeMethodNotAllowed(w http.ResponseWriter, methods ...string) {
	w.Header().Set("Allow", strings.Join(methods, ", "))
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

type supplierListViewModel struct {
	Flash         string
	ActiveSection string
	Suppliers     []suppliers.Supplier
}

type supplierFormViewModel struct {
	PageTitle       string
	Action          string
	SubmitLabel     string
	Flash           string
	ValidationToast string
	ActiveSection   string
	Input           suppliers.Input
	Locations       []string
	Errors          map[string]string
}

func (m supplierFormViewModel) HasError(field string) bool {
	_, ok := m.Errors[field]
	return ok
}

func (m supplierFormViewModel) Error(field string) string {
	return m.Errors[field]
}

var supplierListTemplate = template.Must(template.New("supplier-list").Funcs(template.FuncMap{"adminNav": adminNav}).Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Suppliers</title>
  <style>
    :root { --bg:#f6f8fb; --card:#fff; --line:#d9e1ea; --text:#1f2937; --accent:#0f766e; --muted:#4b5563; }
    * { box-sizing: border-box; }
    body { margin:0; font-family: "Segoe UI", Tahoma, sans-serif; background: var(--bg); color: var(--text); }
    header { background: var(--card); border-bottom:1px solid var(--line); }
    .shell { width:min(1100px, 94vw); margin:0 auto; padding:16px; }
    .admin-nav { display:flex; gap:14px; }
    .admin-nav-link { color: var(--accent); font-weight:600; text-decoration:none; padding:6px 10px; border-radius:8px; }
    .admin-nav-link.active { background:#e6f4f2; color:#0a5f57; }
    .toolbar { display:flex; align-items:center; justify-content:space-between; margin: 16px 0; }
    .button { background: var(--accent); color:white; padding:10px 14px; border-radius:8px; text-decoration:none; font-weight:600; }
    table { width:100%; border-collapse:collapse; background: var(--card); border:1px solid var(--line); border-radius:10px; overflow:hidden; }
    th, td { padding:12px; text-align:left; border-bottom:1px solid var(--line); }
    th { font-size:0.9rem; color:var(--muted); }
    .flash { margin: 12px 0; padding: 10px 12px; border-radius:8px; background:#d1fae5; color:#065f46; border:1px solid #6ee7b7; }
    .empty { background: var(--card); border:1px dashed var(--line); border-radius:10px; padding:20px; color:var(--muted); }
    .row-link { color: var(--accent); font-weight: 600; }
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
      <h1>Suppliers</h1>
      <a class="button" href="/admin/suppliers/new">Add Supplier</a>
    </div>
    {{if .Flash}}<p class="flash">{{.Flash}}</p>{{end}}
    <table>
      <thead>
        <tr>
          <th>Name</th>
          <th>WhatsApp number</th>
          <th>Location</th>
          <th>Action</th>
        </tr>
      </thead>
      <tbody>
        {{if .Suppliers}}
          {{range .Suppliers}}
          <tr>
            <td>{{.Name}}</td>
            <td>{{.WhatsApp}}</td>
            <td>{{.Location}}</td>
            <td><a class="row-link" href="/admin/suppliers/{{.ID}}">View/Edit</a></td>
          </tr>
          {{end}}
        {{else}}
          <tr>
            <td class="empty" colspan="4">No suppliers yet. Click "Add Supplier" to create one.</td>
          </tr>
        {{end}}
      </tbody>
    </table>
  </main>
</body>
</html>
`))

var supplierFormTemplate = template.Must(template.New("supplier-form").Funcs(template.FuncMap{"adminNav": adminNav}).Parse(`<!doctype html>
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
    .field { margin:0 0 16px; }
    label { display:block; font-weight:600; margin-bottom:6px; }
    input, select, textarea { width:100%; padding:10px 12px; border:1px solid var(--line); border-radius:8px; font:inherit; }
    textarea { min-height:120px; resize:vertical; }
    .error { color: var(--error); margin-top:6px; font-size:0.9rem; }
    .button { background: var(--accent); color:white; padding:10px 14px; border-radius:8px; text-decoration:none; border:none; font-weight:600; cursor:pointer; }
    .row { display:flex; gap:10px; align-items:center; }
    .secondary { color:var(--accent); text-decoration:none; font-weight:600; }
    .flash { margin: 12px 0; padding: 10px 12px; border-radius:8px; background:#d1fae5; color:#065f46; border:1px solid #6ee7b7; }
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
    <form class="card" method="post" action="{{.Action}}">
      <div class="field">
        <label for="name">Supplier Name</label>
        <input id="name" name="name" value="{{.Input.Name}}" required>
        {{if .HasError "name"}}<div class="error">{{.Error "name"}}</div>{{end}}
      </div>
      <div class="field">
        <label for="whatsapp">WhatsApp Number</label>
        <input id="whatsapp" name="whatsapp" value="{{.Input.WhatsApp}}" required>
        {{if .HasError "whatsapp"}}<div class="error">{{.Error "whatsapp"}}</div>{{end}}
      </div>
      <div class="field">
        <label for="location">Location</label>
        <select id="location" name="location" required>
          <option value="">Select location</option>
          {{range .Locations}}
          <option value="{{.}}" {{if eq $.Input.Location .}}selected{{end}}>{{.}}</option>
          {{end}}
        </select>
        {{if .HasError "location"}}<div class="error">{{.Error "location"}}</div>{{end}}
      </div>
      <div class="field">
        <label for="notes">Notes (optional)</label>
        <textarea id="notes" name="notes">{{.Input.Notes}}</textarea>
      </div>
      <div class="row">
        <button class="button" type="submit">{{.SubmitLabel}}</button>
        <a class="secondary" href="/admin/suppliers">Back to Suppliers</a>
      </div>
    </form>
  </main>
</body>
</html>
`))

func (s *Server) renderSupplierForm(w http.ResponseWriter, data supplierFormViewModel) {
	if err := supplierFormTemplate.Execute(w, data); err != nil {
		http.Error(w, "failed to render supplier form", http.StatusInternalServerError)
	}
}
