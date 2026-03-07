# Sprint 1 — Feature 02: Admin — Books (supplier-specific listings) (MVP)

## Goal
Allow Srikar to quickly add and maintain **book listings** that appear in the public catalog.

Important MVP assumption: the catalog is made of **supplier-specific listings** (the “same book” may appear multiple times if sourced from multiple suppliers). Each listing is attached to exactly **one** supplier.

## Primary user
- Srikar (Admin)

## Core business rules (relevant to this feature)
- Each book listing is attached to **one supplier** (required).
- Each book listing has an **In-stock** boolean.
  - Defaults to **Yes** on create (automatic in MVP).
- Public catalog will later show **only In-stock** listings.
- MVP admin must be able to set **In-stock Yes/No** on the book edit screen.

## Data to capture (Book Listing)
### Required (MVP)
- **Title** (book name)
- **Cover image** (file picker; mandatory)
- **Supplier** (select from existing suppliers)
- **MRP**
- **My price** (selling price)
- **Discount** (auto-computed from MRP and My price; not manually entered)
- **Book condition** (dropdown: Good as new / Very good / Gently used / Used)
- **Book format** (dropdown: Paperback / Hardcover)
- **Category** (dropdown: Children / Young Adults / Fiction / Non-Fiction)

### Optional (MVP)
- **My price (in bundle)** (NOT mandatory; input field, not derived)
- **Author** (free text)
- **Notes/Description** (free text)

## UI cues (MVP, to reduce Codex guesswork)
- Keep layout consistent with Suppliers admin pages (simple table list + simple form pages).
- In Add/Edit Book form, order fields roughly as: Cover image → Title → Supplier → Category → Format → Condition → MRP → My price → (auto) Discount → (optional) My price (in bundle) → Author → Notes.
- Discount is shown as read-only and auto-computed from MRP and My price.
- Books list thumbnail (explicit):
  - Keep row height tight.
  - Use a fixed-size thumbnail box sized for typical portrait book covers:
    - **Height: 48px, Width: 32px** (approx 2:3 aspect ratio)
  - Render the image inside the box with **contain/fit** behavior:
    - preserve aspect ratio
    - no cropping
    - no stretching
    - centered
  - Letterboxing is acceptable; use subtle background/border so it looks intentional.
  - If cover image is missing, show a neutral placeholder (same box size).

## Screens & flows
### 1) Books List (Admin)
- Menu item: **Books**
- List/table should be **low clutter**.
- Show columns:
  - Cover thumbnail
  - Title
  - Author
  - Category
  - My price
  - In-stock (Yes/No) — inline toggle/edit
  - Action: View/Edit
- Avoid showing too many columns in the list.
- Primary action: **Add Book**
- Optional: simple search by title

### 2) Add Book (Admin)
- Form fields (required + optional as above)
- Supplier is a dropdown sourced from the Suppliers directory.
- On Save:
  - Book listing is created
  - In-stock defaults to **Yes** automatically
  - Show success confirmation
  - Return to list or go to View/Edit (either is fine; be consistent)

### 3) View/Edit Book (Admin)
- View mode can be a **compact summary** (not all fields).
- Edit mode shows **all fields**.
- Include a simple control to set **In-stock: Yes/No**.

## Validation (MVP)
- Title required
- Cover image required
- Supplier required
- Category/Condition/Format required (dropdown)
- MRP required and must be a non-negative number
- My price required and must be a non-negative number
- Discount is computed automatically (no manual input)
- My price (in bundle) is optional; if provided, must be a non-negative number
- No need to enforce formatting of free text fields

## Explicit out of scope (MVP)
- Bulk import / fast-add optimizations (we will spec separately)
- Numeric stock counts
- Any de-duplication or “merge duplicate books across suppliers” logic
- Advanced taxonomy (genres, ISBN, etc.) unless trivial

## Acceptance criteria
- Admin can create a book listing with the required fields (including cover image).
- Newly created listing shows **In-stock = Yes**.
- Admin can view and edit a book listing, including toggling In-stock Yes/No.
- Supplier selection is required and comes from Suppliers.
- Books list displays created listings.
