# Sprint 1 — Feature 01: Admin — Suppliers (MVP)

## Goal
Allow Srikar to maintain a simple supplier directory so:
- every book listing can be attached to a supplier
- later workflows can message suppliers via WhatsApp and generate supplier action links

## Primary user
- Srikar (Admin)

## Data to capture (Supplier)
Required:
- Supplier Name
- WhatsApp Number (single number only in MVP)
- Location (chosen from a dropdown)

Optional:
- Notes (free text)

## Location handling (MVP)
- In the Supplier form, Location is selected from a dropdown.
- MVP can maintain the dropdown list internally (no separate admin screen required to manage locations).

## Screens & flows
### 1) Suppliers List
- Menu item: **Suppliers**
- List/table shows: Name, WhatsApp number, Location
- Primary action: **Add Supplier**
- Row action: **View/Edit**

### 2) Add Supplier
- Form fields: Name (required), WhatsApp number (required), Location (required dropdown), Notes (optional)
- Save creates supplier and returns to list or detail with confirmation.

### 3) View/Edit Supplier
- Show same fields
- Edit + Save

## Constraints / Out of scope (MVP)
- No supplier deactivation
- No multiple WhatsApp numbers
- No location-management admin screen
- Avoid delete unless trivial; if supported, only allow delete when not referenced by any book listings

## Acceptance criteria
- Admin can create a supplier with Name + WhatsApp number + Location.
- Admin can view and edit supplier details.
- Suppliers list displays the created suppliers.
- Location is a dropdown at supplier create/edit time.
- No deactivation controls are present.
