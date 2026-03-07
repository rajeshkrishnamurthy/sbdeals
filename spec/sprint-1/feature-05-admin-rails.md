# Sprint 1 — Feature 05: Admin — Rails (Curation + Ordering) (MVP)

## Goal
Allow admin to curate catalog rails by creating rails, assigning items, and controlling rail order for future catalog rendering.

## Primary user
- Srikar (Admin)

## Scope (MVP)
1) Create rail with:
   - Title
   - Rail Type (`BOOK` or `BUNDLE`) — immutable after creation
2) Edit rail:
   - Title editable
   - Add/remove items of matching type
3) Rail-level publish/unpublish:
   - Unpublished/Published toggle
   - Same instant toggle style as existing admin publish controls
4) Rail ordering (between rails):
   - Move Up / Move Down actions
5) Rails list and rail detail/edit pages

## Explicit decisions locked
- Rail can contain either books or bundles, never mixed.
- Rail type is immutable after creation.
- Rail title is editable.
- No item ordering controls within a rail in MVP (system/default order acceptable).
- Item can appear in multiple rails.
- Same item cannot be added twice within the same rail.
- Rail titles must be unique.
- No rail delete in MVP.
- No minimum item count required to publish a rail (manual control).

## Publish/unpublish behavior (rail level)
- On create:
  - Rail starts as **Unpublished**
- On publish:
  - Rail becomes **Published**
- On unpublish:
  - Rail becomes **Unpublished**
- Recency indicator:
  - Published rail: `(Xd)` from published timestamp
  - Unpublished rail: `(Xd)` from unpublished timestamp

## Admin UI / UX

### 1) Rails list (`/admin/rails`)
Columns:
- Rail Title
- Rail Type (Books/Bundles)
- # Items
- Status (Unpublished/Published) + recency `(Xd)`
- Order controls (Move Up / Move Down)
- Action: View/Edit

Actions:
- Add Rail
- Publish/unpublish toggle from list row (instant, no confirm)

### 2) Add Rail (`/admin/rails/new`)
Fields:
- Title (required, unique)
- Type (required, Books or Bundles, immutable once created)
- Initial status defaults to **Unpublished**

### 3) View/Edit Rail (`/admin/rails/:id`)
- Edit title
- Show immutable type
- Publish/unpublish toggle + recency indicator
- Item assignment panel:
  - Available items list (matching rail type only)
  - Simple title search on available items
  - Add to rail / Remove from rail
  - Selected items list clearly visible

## Validation & feedback
- Duplicate rail title → reject with toast error.
- Duplicate add within same rail → reject with toast error.
- Type mismatch item add (e.g., bundle into book rail) → reject with toast error.
- Validation failures are never silent.
- Success toasts for create/edit/add/remove/publish/unpublish.

(Conforms to `spec/ADMIN-UI-STANDARDS.md` toast requirement.)

## Reuse guidance (to reduce drift)
- **Publish/unpublish behavior:** reuse existing implementation patterns from Sprint 1 Feature 04 (toggle behavior, recency handling, toast UX) so admin behavior remains consistent.
- **Item search behavior:** reuse the simple search interaction pattern already used in bundle management where feasible, so filtering UX feels consistent across admin screens.

## Out of scope (this feature)
- Customer catalog rendering rules (e.g., storefront filtering logic)
- Item ordering within a rail
- Rail delete
- Drag-and-drop ordering
- Accessibility/security hardening pass

## Acceptance criteria
1) Admin can create a rail with unique title + type.
2) Rail type cannot be changed after creation.
3) Admin can edit rail title.
4) Admin can publish/unpublish rails from list and edit views.
5) Recency `(Xd)` appears for rail publish state on list and edit views.
6) Admin can add/remove matching-type items from rail using available list + search.
7) Same item cannot be added more than once to the same rail.
8) Same item can exist in multiple different rails.
9) Admin can reorder rails via Move Up / Move Down.
10) No rail delete is exposed in MVP.
