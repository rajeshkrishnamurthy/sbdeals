package web

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/rajeshkrishnamurthy/sbdeals/internal/customers"
)

var customerValidationFieldOrder = []string{
	"name",
	"mobile",
	"city_name",
	"apartment_name",
	"address",
	"notes",
}

func (s *Server) handleCustomersCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.renderCustomersList(w, r)
	case http.MethodPost:
		s.createCustomer(w, r)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (s *Server) handleCustomerNew(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	cities, err := s.customerStore.ListCities()
	if err != nil {
		http.Error(w, "failed to load cities", http.StatusInternalServerError)
		return
	}
	apartmentMap, err := s.loadApartmentMap(cities)
	if err != nil {
		http.Error(w, "failed to load apartment complexes", http.StatusInternalServerError)
		return
	}
	s.renderCustomerForm(w, customerFormViewModel{
		PageTitle:       "Add Customer",
		Action:          "/admin/customers",
		SubmitLabel:     "Save Customer",
		ActiveSection:   "customers",
		Input:           customerFormInput{},
		Errors:          map[string]string{},
		CityOptions:     cities,
		ApartmentByCity: apartmentMap,
		MobileReadOnly:  false,
	})
}

func (s *Server) handleCustomerItem(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/admin/customers/" {
		http.Redirect(w, r, "/admin/customers", http.StatusMovedPermanently)
		return
	}
	id, ok := parseCustomerPathID(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.renderCustomerDetail(w, r, id)
	case http.MethodPost:
		s.updateCustomer(w, r, id)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (s *Server) renderCustomersList(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	cityID, cityRaw := parseOptionalPositiveInt(r.URL.Query().Get("city_id"))
	filter := customers.ListFilter{Search: search, CityID: cityID}
	items, err := s.customerStore.List(filter)
	if err != nil {
		http.Error(w, "failed to load customers", http.StatusInternalServerError)
		return
	}
	cities, err := s.customerStore.ListCities()
	if err != nil {
		http.Error(w, "failed to load cities", http.StatusInternalServerError)
		return
	}

	view := customersListViewModel{
		ActiveSection:   "customers",
		Flash:           r.URL.Query().Get("flash"),
		ValidationToast: strings.TrimSpace(r.URL.Query().Get("error")),
		Rows:            items,
		CityOptions:     cities,
		Search:          search,
		SelectedCityID:  cityRaw,
	}
	if err := customersListTemplate.Execute(w, view); err != nil {
		http.Error(w, "failed to render customer list", http.StatusInternalServerError)
	}
}

func (s *Server) createCustomer(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}
	cities, err := s.customerStore.ListCities()
	if err != nil {
		http.Error(w, "failed to load cities", http.StatusInternalServerError)
		return
	}
	apartmentMap, err := s.loadApartmentMap(cities)
	if err != nil {
		http.Error(w, "failed to load apartment complexes", http.StatusInternalServerError)
		return
	}

	input := readCustomerFormInput(r)
	parsed, errs := validateCustomerForm(input, false)
	if len(errs) > 0 {
		w.WriteHeader(http.StatusBadRequest)
		s.renderCustomerForm(w, customerFormViewModel{
			PageTitle:       "Add Customer",
			Action:          "/admin/customers",
			SubmitLabel:     "Save Customer",
			ActiveSection:   "customers",
			Input:           input,
			Errors:          errs,
			CityOptions:     cities,
			ApartmentByCity: apartmentMap,
			MobileReadOnly:  false,
			ValidationToast: buildValidationToast(errs, customerValidationFieldOrder),
		})
		return
	}

	created, err := s.customerStore.Create(customers.CreateInput{
		Name:          parsed.Name,
		Mobile:        parsed.Mobile,
		Address:       parsed.Address,
		CityName:      parsed.CityName,
		ApartmentName: parsed.ApartmentName,
		Notes:         parsed.Notes,
	})
	if err != nil {
		var dupErr *customers.DuplicateMobileError
		if errors.As(err, &dupErr) {
			errs = map[string]string{"mobile": "Customer already exists."}
			w.WriteHeader(http.StatusBadRequest)
			s.renderCustomerForm(w, customerFormViewModel{
				PageTitle:           "Add Customer",
				Action:              "/admin/customers",
				SubmitLabel:         "Save Customer",
				ActiveSection:       "customers",
				Input:               input,
				Errors:              errs,
				CityOptions:         cities,
				ApartmentByCity:     apartmentMap,
				MobileReadOnly:      false,
				DuplicateCustomerID: &dupErr.CustomerID,
				ValidationToast:     "Please fix: Customer already exists.",
			})
			return
		}
		http.Error(w, "failed to create customer", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/customers/%d?flash=%s", created.ID, url.QueryEscape("Customer created successfully.")), http.StatusSeeOther)
}

func (s *Server) renderCustomerDetail(w http.ResponseWriter, r *http.Request, id int) {
	customer, err := s.customerStore.Get(id)
	if errors.Is(err, customers.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "failed to load customer", http.StatusInternalServerError)
		return
	}
	cities, err := s.customerStore.ListCities()
	if err != nil {
		http.Error(w, "failed to load cities", http.StatusInternalServerError)
		return
	}
	apartmentMap, err := s.loadApartmentMap(cities)
	if err != nil {
		http.Error(w, "failed to load apartment complexes", http.StatusInternalServerError)
		return
	}
	s.renderCustomerForm(w, customerFormViewModel{
		PageTitle:       "View/Edit Customer",
		Action:          fmt.Sprintf("/admin/customers/%d", id),
		SubmitLabel:     "Save Changes",
		ActiveSection:   "customers",
		Flash:           r.URL.Query().Get("flash"),
		ValidationToast: strings.TrimSpace(r.URL.Query().Get("error")),
		Input: customerFormInput{
			Name:          customer.Name,
			Mobile:        customer.Mobile,
			CityName:      customer.CityName,
			ApartmentName: customer.ApartmentName,
			Address:       ptrToString(customer.Address),
			Notes:         ptrToString(customer.Notes),
		},
		Errors:          map[string]string{},
		CityOptions:     cities,
		ApartmentByCity: apartmentMap,
		MobileReadOnly:  true,
	})
}

func (s *Server) updateCustomer(w http.ResponseWriter, r *http.Request, id int) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}
	if _, err := s.customerStore.Get(id); errors.Is(err, customers.ErrNotFound) {
		http.NotFound(w, r)
		return
	}

	cities, err := s.customerStore.ListCities()
	if err != nil {
		http.Error(w, "failed to load cities", http.StatusInternalServerError)
		return
	}
	apartmentMap, err := s.loadApartmentMap(cities)
	if err != nil {
		http.Error(w, "failed to load apartment complexes", http.StatusInternalServerError)
		return
	}

	input := readCustomerFormInput(r)
	parsed, errs := validateCustomerForm(input, true)
	if len(errs) > 0 {
		w.WriteHeader(http.StatusBadRequest)
		s.renderCustomerForm(w, customerFormViewModel{
			PageTitle:       "View/Edit Customer",
			Action:          fmt.Sprintf("/admin/customers/%d", id),
			SubmitLabel:     "Save Changes",
			ActiveSection:   "customers",
			Input:           input,
			Errors:          errs,
			CityOptions:     cities,
			ApartmentByCity: apartmentMap,
			MobileReadOnly:  true,
			ValidationToast: buildValidationToast(errs, customerValidationFieldOrder),
		})
		return
	}

	_, err = s.customerStore.Update(id, customers.UpdateInput{
		Name:          parsed.Name,
		Address:       parsed.Address,
		CityName:      parsed.CityName,
		ApartmentName: parsed.ApartmentName,
		Notes:         parsed.Notes,
	})
	if errors.Is(err, customers.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "failed to update customer", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/admin/customers/%d?flash=%s", id, url.QueryEscape("Customer updated successfully.")), http.StatusSeeOther)
}

func parseCustomerPathID(path string) (int, bool) {
	prefix := "/admin/customers/"
	if !strings.HasPrefix(path, prefix) {
		return 0, false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(path, prefix))
	if rest == "" || strings.Contains(rest, "/") {
		return 0, false
	}
	id, err := strconv.Atoi(rest)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func parseOptionalPositiveInt(raw string) (*int, string) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, ""
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return nil, ""
	}
	return &parsed, value
}

func ptrToString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func (s *Server) loadApartmentMap(cities []customers.City) (map[int][]customers.ApartmentComplex, error) {
	out := make(map[int][]customers.ApartmentComplex, len(cities))
	for _, city := range cities {
		items, err := s.customerStore.ListApartmentComplexesByCityID(city.ID)
		if err != nil {
			return nil, err
		}
		out[city.ID] = items
	}
	return out, nil
}

type customerFormInput struct {
	Name          string
	Mobile        string
	CityName      string
	ApartmentName string
	Address       string
	Notes         string
}

func readCustomerFormInput(r *http.Request) customerFormInput {
	return customerFormInput{
		Name:          strings.TrimSpace(r.FormValue("name")),
		Mobile:        strings.TrimSpace(r.FormValue("mobile")),
		CityName:      strings.TrimSpace(r.FormValue("city_name")),
		ApartmentName: strings.TrimSpace(r.FormValue("apartment_name")),
		Address:       strings.TrimSpace(r.FormValue("address")),
		Notes:         strings.TrimSpace(r.FormValue("notes")),
	}
}

type parsedCustomerForm struct {
	Name          string
	Mobile        string
	CityName      *string
	ApartmentName *string
	Address       *string
	Notes         *string
}

func validateCustomerForm(input customerFormInput, mobileReadOnly bool) (parsedCustomerForm, map[string]string) {
	out := parsedCustomerForm{
		Name:          input.Name,
		Address:       optionalText(input.Address),
		Notes:         optionalText(input.Notes),
		CityName:      optionalText(input.CityName),
		ApartmentName: optionalText(input.ApartmentName),
	}
	errs := map[string]string{}

	if len(out.Name) < 2 || len(out.Name) > 100 {
		errs["name"] = "Name is required and must be between 2 and 100 characters."
	}

	if !mobileReadOnly {
		out.Mobile = customers.NormalizeMobile(input.Mobile)
		if len(out.Mobile) != 10 {
			errs["mobile"] = "Mobile is required and must be exactly 10 digits."
		}
	}

	if out.Address != nil && len(*out.Address) > 250 {
		errs["address"] = "Address must be 250 characters or fewer."
	}
	if out.Notes != nil && len(*out.Notes) > 500 {
		errs["notes"] = "Notes must be 500 characters or fewer."
	}
	if out.ApartmentName != nil && len(*out.ApartmentName) > 120 {
		errs["apartment_name"] = "Apartment complex must be 120 characters or fewer."
	}
	if out.CityName == nil && out.ApartmentName != nil {
		errs["apartment_name"] = "Select city before apartment complex."
	}
	return out, errs
}

func optionalText(raw string) *string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

type customersListViewModel struct {
	ActiveSection   string
	Flash           string
	ValidationToast string
	Rows            []customers.ListItem
	CityOptions     []customers.City
	Search          string
	SelectedCityID  string
}

type customerFormViewModel struct {
	PageTitle           string
	Action              string
	SubmitLabel         string
	ActiveSection       string
	Flash               string
	ValidationToast     string
	Input               customerFormInput
	Errors              map[string]string
	CityOptions         []customers.City
	ApartmentByCity     map[int][]customers.ApartmentComplex
	MobileReadOnly      bool
	DuplicateCustomerID *int
}

func (m customerFormViewModel) HasError(field string) bool {
	_, ok := m.Errors[field]
	return ok
}

func (m customerFormViewModel) Error(field string) string {
	return m.Errors[field]
}

func (m customerFormViewModel) SelectedCityID() string {
	inputCity := strings.TrimSpace(m.Input.CityName)
	if inputCity == "" {
		return ""
	}
	for _, city := range m.CityOptions {
		if strings.EqualFold(strings.TrimSpace(city.Name), inputCity) {
			return strconv.Itoa(city.ID)
		}
	}
	return ""
}

func (m customerFormViewModel) DuplicateID() string {
	if m.DuplicateCustomerID == nil || *m.DuplicateCustomerID <= 0 {
		return ""
	}
	return strconv.Itoa(*m.DuplicateCustomerID)
}

var customersListTemplate = template.Must(template.New("customers-list").Funcs(template.FuncMap{
	"adminNav": adminNav,
}).Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Customers</title>
  <style>
    :root { --bg:#f6f8fb; --card:#fff; --line:#d9e1ea; --text:#1f2937; --accent:#0f766e; --muted:#4b5563; }
    * { box-sizing: border-box; }
    body { margin:0; font-family: "Segoe UI", Tahoma, sans-serif; background:var(--bg); color:var(--text); }
    header { background:var(--card); border-bottom:1px solid var(--line); }
    .shell { width:min(1100px, 94vw); margin:0 auto; padding:16px; }
    .admin-nav { display:flex; gap:14px; }
    .admin-nav-link { color:var(--accent); font-weight:600; text-decoration:none; padding:6px 10px; border-radius:8px; }
    .admin-nav-link.active { background:#e6f4f2; color:#0a5f57; }
    .toolbar { display:flex; align-items:center; justify-content:space-between; margin:16px 0; }
    .filters { display:grid; grid-template-columns:2fr 1fr auto; gap:10px; margin:0 0 14px; }
    input, select { width:100%; padding:10px 12px; border:1px solid var(--line); border-radius:8px; font:inherit; }
    .button { background:var(--accent); color:#fff; border:none; border-radius:8px; padding:10px 14px; text-decoration:none; font-weight:600; cursor:pointer; }
    .secondary { color:var(--accent); text-decoration:none; font-weight:600; align-self:center; }
    table { width:100%; border-collapse:collapse; background:var(--card); border:1px solid var(--line); border-radius:10px; overflow:hidden; }
    th, td { padding:9px 10px; text-align:left; border-bottom:1px solid var(--line); vertical-align:middle; }
    th { font-size:0.9rem; color:var(--muted); }
    .row-link { color:var(--accent); font-weight:600; }
    .flash { margin:12px 0; padding:10px 12px; border-radius:8px; background:#d1fae5; color:#065f46; border:1px solid #6ee7b7; }
    .toast-error { position:fixed; top:16px; right:16px; max-width:min(420px, 90vw); z-index:999; margin:0; padding:10px 12px; border-radius:10px; background:#fee2e2; color:#991b1b; border:1px solid #fecaca; box-shadow:0 8px 24px rgba(0,0,0,0.12); }
  </style>
</head>
<body>
  <header><div class="shell">{{adminNav .ActiveSection}}</div></header>
  <main class="shell">
    <div class="toolbar">
      <h1>Customers</h1>
      <a class="button" href="/admin/customers/new">Add Customer</a>
    </div>
    {{if .Flash}}<p class="flash">{{.Flash}}</p>{{end}}
    {{if .ValidationToast}}<p class="toast-error" role="alert">{{.ValidationToast}}</p>{{end}}
    <form class="filters" method="get" action="/admin/customers">
      <input name="q" value="{{.Search}}" placeholder="Search by name or mobile">
      <select name="city_id">
        <option value="">All cities</option>
        {{range .CityOptions}}
        <option value="{{.ID}}" {{if eq $.SelectedCityID (printf "%d" .ID)}}selected{{end}}>{{.Name}}</option>
        {{end}}
      </select>
      <button class="button" type="submit">Apply</button>
    </form>
    <table>
      <thead>
        <tr><th>Name</th><th>Mobile</th><th>City</th><th>Apartment Complex</th><th>Action</th></tr>
      </thead>
      <tbody>
      {{if .Rows}}
        {{range .Rows}}
        <tr>
          <td>{{.Name}}</td>
          <td>{{.Mobile}}</td>
          <td>{{.CityName}}</td>
          <td>{{.ApartmentName}}</td>
          <td><a class="row-link" href="/admin/customers/{{.ID}}">View/Edit</a></td>
        </tr>
        {{end}}
      {{else}}
        <tr><td colspan="5">No customers found.</td></tr>
      {{end}}
      </tbody>
    </table>
  </main>
</body>
</html>
`))

var customersFormTemplate = template.Must(template.New("customers-form").Funcs(template.FuncMap{
	"adminNav": adminNav,
}).Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.PageTitle}}</title>
  <style>
    :root { --bg:#f6f8fb; --card:#fff; --line:#d9e1ea; --text:#1f2937; --accent:#0f766e; --muted:#4b5563; --error:#b91c1c; }
    * { box-sizing: border-box; }
    body { margin:0; font-family:"Segoe UI", Tahoma, sans-serif; background:var(--bg); color:var(--text); }
    header { background:var(--card); border-bottom:1px solid var(--line); }
    .shell { width:min(1100px, 94vw); margin:0 auto; padding:16px; }
    .admin-nav { display:flex; gap:14px; }
    .admin-nav-link { color:var(--accent); font-weight:600; text-decoration:none; padding:6px 10px; border-radius:8px; }
    .admin-nav-link.active { background:#e6f4f2; color:#0a5f57; }
    .card { background:var(--card); border:1px solid var(--line); border-radius:10px; padding:20px; }
    .field { margin:0 0 14px; }
    label { display:block; font-weight:600; margin-bottom:6px; }
    input, textarea { width:100%; padding:10px 12px; border:1px solid var(--line); border-radius:8px; font:inherit; }
    textarea { min-height:100px; resize:vertical; }
    .error { color:var(--error); margin-top:6px; font-size:0.9rem; }
    .button { background:var(--accent); color:#fff; border:none; border-radius:8px; padding:10px 14px; text-decoration:none; font-weight:600; cursor:pointer; }
    .secondary { color:var(--accent); text-decoration:none; font-weight:600; }
    .row { display:flex; gap:10px; align-items:center; }
    .flash { margin:12px 0; padding:10px 12px; border-radius:8px; background:#d1fae5; color:#065f46; border:1px solid #6ee7b7; }
    .toast-error { position:fixed; top:16px; right:16px; max-width:min(420px, 90vw); z-index:999; margin:0; padding:10px 12px; border-radius:10px; background:#fee2e2; color:#991b1b; border:1px solid #fecaca; box-shadow:0 8px 24px rgba(0,0,0,0.12); }
    .dup-actions { display:flex; gap:12px; margin-top:6px; }
    .dup-actions a { color:var(--accent); font-weight:600; text-decoration:none; }
  </style>
</head>
<body>
  <header><div class="shell">{{adminNav .ActiveSection}}</div></header>
  <main class="shell">
    <h1>{{.PageTitle}}</h1>
    {{if .Flash}}<p class="flash">{{.Flash}}</p>{{end}}
    {{if .ValidationToast}}<p class="toast-error" role="alert">{{.ValidationToast}}</p>{{end}}
    <form class="card" method="post" action="{{.Action}}">
      <div class="field">
        <label for="name">Name</label>
        <input id="name" name="name" value="{{.Input.Name}}" required>
        {{if .HasError "name"}}<div class="error">{{.Error "name"}}</div>{{end}}
      </div>
      <div class="field">
        <label for="mobile">Mobile</label>
        <input id="mobile" name="mobile" value="{{.Input.Mobile}}" {{if .MobileReadOnly}}readonly{{end}} required>
        {{if .HasError "mobile"}}<div class="error">{{.Error "mobile"}}</div>{{end}}
        {{if .DuplicateCustomerID}}
        <div class="dup-actions">
          <a href="/admin/customers/{{.DuplicateID}}">View Existing</a>
          <a href="/admin/customers/{{.DuplicateID}}">Edit Existing</a>
        </div>
        {{end}}
      </div>
      <div class="field">
        <label for="city_name">City</label>
        <input id="city_name" name="city_name" list="city-options" value="{{.Input.CityName}}" autocomplete="off">
        <datalist id="city-options">
          {{range .CityOptions}}
          <option value="{{.Name}}"></option>
          {{end}}
        </datalist>
        {{if .HasError "city_name"}}<div class="error">{{.Error "city_name"}}</div>{{end}}
      </div>
      <div class="field">
        <label for="apartment_name">Apartment Complex</label>
        <input id="apartment_name" name="apartment_name" list="apartment-options" value="{{.Input.ApartmentName}}" autocomplete="off">
        <datalist id="apartment-options"></datalist>
        {{if .HasError "apartment_name"}}<div class="error">{{.Error "apartment_name"}}</div>{{end}}
      </div>
      <div class="field">
        <label for="address">Address (optional)</label>
        <input id="address" name="address" value="{{.Input.Address}}">
        {{if .HasError "address"}}<div class="error">{{.Error "address"}}</div>{{end}}
      </div>
      <div class="field">
        <label for="notes">Notes (optional)</label>
        <textarea id="notes" name="notes">{{.Input.Notes}}</textarea>
        {{if .HasError "notes"}}<div class="error">{{.Error "notes"}}</div>{{end}}
      </div>
      <div class="row">
        <button class="button" type="submit">{{.SubmitLabel}}</button>
        <a class="secondary" href="/admin/customers">Back to Customers</a>
      </div>
    </form>
  </main>
  <script>
    window.CUSTOMER_FORM_DATA = {
      selectedCityId: "{{.SelectedCityID}}",
      cityByName: {
        {{range .CityOptions}}"{{js .Name}}":"{{.ID}}",{{end}}
      },
      apartmentsByCity: {
        {{range .CityOptions}}
        "{{.ID}}":[{{range index $.ApartmentByCity .ID}}"{{js .Name}}",{{end}}],
        {{end}}
      }
    };
  </script>
  <script src="/assets/customers-form.js" defer></script>
</body>
</html>
`))

func (s *Server) renderCustomerForm(w http.ResponseWriter, view customerFormViewModel) {
	if err := customersFormTemplate.Execute(w, view); err != nil {
		http.Error(w, "failed to render customer form", http.StatusInternalServerError)
	}
}

func (s *Server) handleCustomersFormJSAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	_, _ = w.Write([]byte(customersFormJS))
}

const customersFormJS = `(function () {
  const cityInput = document.getElementById("city_name");
  const apartmentInput = document.getElementById("apartment_name");
  const apartmentOptions = document.getElementById("apartment-options");
  const data = window.CUSTOMER_FORM_DATA || {};
  if (!cityInput || !apartmentInput || !apartmentOptions) return;

  function cityId() {
    const name = (cityInput.value || "").trim();
    if (!name) return "";
    return String((data.cityByName || {})[name] || "");
  }

  function syncApartmentOptions() {
    const id = cityId();
    const names = id ? ((data.apartmentsByCity || {})[id] || []) : [];
    apartmentOptions.innerHTML = names.map((name) => "<option value=\"" + String(name).replace(/"/g, "&quot;") + "\"></option>").join("");
    const hasCityInput = Boolean((cityInput.value || "").trim());
    apartmentInput.disabled = !hasCityInput;
    if (!hasCityInput) apartmentInput.value = "";
  }

  cityInput.addEventListener("change", syncApartmentOptions);
  cityInput.addEventListener("input", syncApartmentOptions);
  syncApartmentOptions();
})();`
