# Patch — Box Set Flag + Conditional Bundle Minimum Items

## Objective
Support publisher-defined box sets (single-MRP products) as books, without introducing a new product model, while preserving stricter bundle composition rules for regular books.

## Scope (In)

### 1) Book model extension
- Add boolean field on books: `is_box_set`
- Default: `false`
- Admin label: **Box Set** (Yes/No)

### 2) Conditional bundle minimum-items rule
- Keep current baseline rule for regular books:
  - Minimum selected items required = 2
- Add exception:
  - If at least one selected item has `is_box_set = true`, minimum selected items required = 1

### 3) Admin UX updates
- Add `Box Set` control to Add/Edit Book form.
- Update bundle validation/help text to reflect conditional rule:
  - “Minimum 2 items required unless one selected item is marked Box Set.”

### 4) Conceptual behavior
- Box set remains a **Book** (no new entity/type hierarchy).
- Box-set books continue to participate in existing bundle and rail flows through standard book selection.

## Data / Migration
- Add `is_box_set` column to books table:
  - type: boolean
  - non-null
  - default: false
- Backfill existing rows to false.

## Out of Scope (Do Not Implement)
- No new BoxSet entity/table.
- No direct box-set-only rail path.
- No additional composition guardrails (e.g., no mixing constraints).
- No storefront redesign or special customer-facing box-set treatment in this patch.

## Acceptance Criteria
1. Admin can set/unset Box Set on a book record.
2. Bundle save with only regular books still requires at least 2 selected items.
3. Bundle save with at least one box-set book allows 1 selected item.
4. Existing multi-book bundle behavior remains unchanged.
5. Existing rails behavior remains unchanged.

## Test Checklist
- Create regular book (`is_box_set=false`), try single-item bundle → blocked with validation.
- Create regular+regular bundle (2 items) → allowed.
- Create box-set book (`is_box_set=true`), single-item bundle → allowed.
- Create mixed bundle (box-set + regular) with 1 item (box-set only) → allowed.
- Edit legacy books (default false) and verify no regressions.
