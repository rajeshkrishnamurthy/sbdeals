# Sprint 1 — Feature 05: Admin — Bundles (MVP)

## Goal
Allow Srikar to create and maintain **bundles** (curated sets of books) that can be sold as a single deal.

## Primary user
- Srikar (Admin)

## Core business rules (MVP)
- A bundle contains **multiple books**.
- Bundles **cannot contain other bundles**.
- **Single-supplier constraint:** all books in a bundle must be from the **same supplier**.
- Bundles must have a **Category** (same dropdown as books).
- Books available to be added to a bundle must be filtered by:
  - selected Supplier
  - selected Category
  - selected allowed set of Book Conditions (see below)

## Pricing & dynamic totals (MVP)
- Bundle has an explicit **Bundle price** (input).
- Admin UI must show **live, real-time** running totals while building a bundle (updates immediately as books are added/removed):
  - **Bundle MRP** = Sum(MRP) of all selected books
  - **Sum(My price)** of all selected books
  - Optional: also show Sum(My price in bundle) if provided per book (fallback to My price)
  - **Bundle discount (on MRP)** computed automatically from Bundle MRP vs Bundle price
- These totals are for decision support; bundle price remains manually entered.

## Condition coherence (MVP)
- While creating a bundle, admin selects an allowed set of Book Conditions (**multi-select**).
- Only books matching the selected condition(s) are eligible to be added.
- (Rationale: avoid mixing “Good as new” with “Used” within a bundle, while allowing adjacent combinations like Good as new + Very good.)

## Out-of-stock / invalid bundles
- Not part of this feature. See backlog item **BL-002** in `/BACKLOG.md`.

## Data to capture (Bundle)
### Required
- Supplier (selected first)
- Category
- Allowed Book Condition set
- Included books
- Bundle price

### Optional
- Bundle label/name (optional in MVP; can be auto-generated later)
- Notes/description

## Bundle image
- Not part of this feature. See backlog item **BL-001** in `/BACKLOG.md`.

## Admin UI standard
- Must follow `spec/ADMIN-UI-STANDARDS.md`.

## Screens & flows
### 1) Bundles List (Admin)
- Menu item: **Bundles**
- List/table should be low clutter and show at minimum:
  - Bundle label/name (if present) OR an auto label like **Bundle #123**
  - Supplier
  - Category
  - Allowed condition(s)
  - # of books
  - Bundle price
  - Action: View/Edit
- Primary action: **Add Bundle**
- Note: status/invalidity indicators are deferred until the out-of-stock feature.

### 2) Add Bundle (Admin)
- Step 1: Choose **Supplier**
- Step 2: Choose **Category**
- Step 3: Choose allowed **Book Condition(s)** (e.g., one condition or a small set)
- Step 4: Add books to bundle using a **Book picker** (do not use a hidden multi-select-only UX)
  - Provide a search/filter box within eligible books (title/author)
  - Provide an “Add” action per book
  - As books are added, they must appear immediately in a visible **Selected books** list/table
  - Selected list supports remove (books can be removed from a bundle)
- Step 5: Pricing support
  - Show **live Sum(My price)** for selected books (updates in real time)
  - Optionally show Sum(My price in bundle) with fallback logic
  - Admin inputs Bundle price explicitly
- Step 6: Save bundle

### 3) View/Edit Bundle (Admin)
- View mode can be compact summary
- Edit mode allows:
  - supplier/category/conditions (changing these may require re-validating selected books)
  - add/remove included books via the same Book picker + Selected books list
  - bundle price
  - optional label/name
  - notes

### 4) Invalid Bundles (Admin)
- Not part of this feature. See backlog item **BL-002** in `/BACKLOG.md`.

## Validation
- Supplier required.
- Category required.
- Allowed book condition(s) required.
- Bundle must contain **at least 2 books**.
- Bundle price must be non-negative.

## Out of scope
- Bundle-of-bundles
- Multi-supplier bundles
- Public bundle catalog (deferred)

## Deferred / DO NOT IMPLEMENT in this feature
- Bundle out-of-stock invalidation + Invalid Bundles admin (see **BL-002** in `/BACKLOG.md`)
- Bundle image / auto-generation (see **BL-001** in `/BACKLOG.md`)

## Acceptance criteria
- Admin can create a bundle by selecting supplier + category + allowed condition(s) → adding 2+ books → setting bundle price.
- Eligible books list is filtered by supplier, category, and condition selection.
- Selected books are always visible in a dedicated Selected books list/table (not hidden inside a dropdown).
- Live totals (Bundle MRP, Sum(My price), and computed bundle discount) update instantly as books are added/removed.
- Bundle edit enforces single-supplier constraint and re-validates selection when supplier/category/conditions change.
