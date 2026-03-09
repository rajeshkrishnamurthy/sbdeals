# Sprint 3 / Feature XX-A — Customer Master Management (Add / View / Edit)

## 1) Scope
Build a reusable customer master with:
- Add Customer
- View Customers (list + detail)
- Edit Customer

### Out of scope (DO NOT IMPLEMENT)
- Customer lifecycle state machine (prospect/buyer/repeater/loyal)
- Mapping customer lifecycle to enquiry/order lifecycle
- Auto-promotion based on order history
- Order/enquiry side-effects triggered by customer updates

---

## 2) MVP Data Model

### 2.1 Customer
- `id` (system-generated)
- `name` (required)
- `mobile` (required, unique, immutable after create)
- `address` (optional)
- `city_id` (optional, reference to City Master)
- `apartment_complex_id` (optional, reference to Apartment Complex Master)
- `notes` (optional)
- `created_at`, `updated_at` (internal fields; not required in list UI)

### 2.2 City Master
- `id`
- `name` (unique on normalized value)

### 2.3 Apartment Complex Master
- `id`
- `city_id` (required)
- `name`
- uniqueness: (`city_id`, normalized `name`)

> Domain requirement: Apartment complex is a master scoped by city.

---

## 3) Validation + Normalization Rules

### 3.1 Customer fields
- `name`:
  - required
  - trim whitespace
  - min length: 2
  - max length: 100
- `mobile`:
  - required
  - normalize by stripping non-digits (`+`, spaces, dashes, etc.)
  - must be exactly 10 digits after normalization
  - unique on normalized value
- `address`:
  - optional
  - max length: 250
- `city` (via master):
  - optional
  - if provided, must map to a City Master record
- `apartment complex` (via master):
  - optional
  - if provided, must map to Apartment Complex Master record under selected city
  - max length for new master name input: 120
- `notes`:
  - optional
  - max length: 500

### 3.2 Common behavior
- Trim all text fields.
- For optional fields, empty-after-trim => store as `null`.

---

## 4) Add Customer

## 4.1 UI behavior
- Mandatory on create: `name`, `mobile`.
- Optional on create: `city`, `apartment complex`, `address`, `notes`.
- City-first dependency for apartment complex:
  1. User selects city first.
  2. Apartment complex input is then enabled and scoped to that city.
  3. Apartment complex UI can be autocomplete or dropdown (implementation may choose whichever is simpler).

## 4.2 Dedupe behavior (mobile)
- On create, if normalized mobile already exists:
  - block create (hard stop)
  - show message: customer already exists
  - provide actions: **View Existing** and **Edit Existing**
- No duplicate customer record is created.

## 4.3 Master handling
- City and apartment complex masters are reusable entities.
- If UI supports creating new master entries inline, enforce normalization and uniqueness constraints.

---

## 5) View Customers

## 5.1 Customer list (MVP)
- Columns:
  - Name
  - Mobile
  - City
  - Apartment Complex
- Search:
  - Name
  - Mobile
- Filter:
  - City
- No `Updated At` column in list.
- No pagination in MVP.

## 5.2 Customer detail
- Display all customer fields.
- Provide Edit action.

---

## 6) Edit Customer

### 6.1 Editable fields
- `name`
- `address`
- `city`
- `apartment complex`
- `notes`

### 6.2 Non-editable field
- `mobile` is immutable in MVP.

---

## 7) Audit Expectations
- Auditability is low priority for this feature.
- Minimal/basic audit is sufficient (no mandatory field-level before/after history).

---

## 8) Error + Success Handling

### 8.1 Errors
- Validation errors: field-level messages.
- Duplicate mobile on create: blocking error + clear action to open existing customer.
- Master lookup mismatch (e.g., apartment not under selected city): validation error.

### 8.2 Success
- On successful create/edit, show explicit success feedback.
- Post-save navigation behavior should follow existing admin UI conventions.

---

## 9) API/Consistency Expectations
- No explicit idempotency key support in MVP (KISS).
- Consistency guarantees rely on:
  - normalized unique mobile constraint
  - transactional handling for create/update operations

---

## 10) Acceptance Criteria (MVP)
1. Admin can create customer with only Name + Mobile.
2. Mobile is normalized and unique; duplicate create is blocked.
3. On duplicate mobile, UI provides View/Edit existing customer path.
4. City must be selected before apartment complex selection.
5. Apartment complex options are scoped by selected city.
6. Admin can view customer list with Name/Mobile/City/Apartment columns.
7. Search by Name/Mobile works in list.
8. Filter by City works in list.
9. No pagination appears in customer list.
10. Admin can edit all customer fields except mobile.
11. Basic success/error states are shown for add/edit flows.
12. No lifecycle/integration side-effects are implemented.
