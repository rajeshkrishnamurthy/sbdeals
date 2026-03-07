# Feature 5 — Rail Item Picker with Deterministic Filters and Visual Bundle Preview

## Objective
Improve **Add item to rail** so admins can curate reliably using deterministic filters (not title-only search), while keeping MVP implementation simple and consistent with existing admin patterns.

## In Scope (MVP)

### 1) Rail behavior
- Rail remains a presentation container.
- No rail-level rule engine.
- Admin intent can be captured via an optional **Admin Note** on the rail.

### 2) Picker filters
Enhance picker with:
- Title search (existing)
- Category filter (single-select dropdown)
- Bundle price range filter: `min`, `max`
- Discount range filter (percentage): `min`, `max`

Behavior:
- Filters apply only on explicit **Apply Filters** click.
- **Reset Filters** clears all filters and returns to default result list.
- For price/discount filters, both min and max are mandatory **when that filter is used**.

Validation:
- If `min > max`, block apply and show inline validation error.

### 3) Category source
- Use same category taxonomy/source as existing book-category flow.
- Current values include: **Children, Fiction, Non-fiction**.

### 4) Item-type context
- Picker supports existing item-type context/tabs (Books/Bundles) where applicable.
- Price and discount filters apply in bundle context.

### 5) Result list behavior
- Reuse existing UX pattern used in “select books for bundle”:
  - Fixed-height container
  - Internal scroll

### 6) Result row display (common columns)
Use neutral/common headers across item types:
- Image
- Title
- Category
- Price
- Discount

Display rules:
- For bundles: Price = bundle price, Discount = bundle discount %.
- For books: Price = book price, Discount = book discount %.
- Category comes from the existing taxonomy source.

Image interaction:
- On mouse hover over thumbnail, show a simple enlarged preview (applies to both books and bundles).

### 7) Already-added items
- Items already present in the current rail must be excluded from picker results.

### 8) Rail Admin Note
- Add optional `adminNote` field on rail create/edit.
- Plain text only.
- Max length enforced (250 chars).
- Editable at create and edit time.
- Admin-only/internal; not customer-facing.

## API / Data Contract Expectations
- Picker query should support optional params:
  - `q`
  - `category`
  - `priceMin`, `priceMax`
  - `discountMin`, `discountMax`
  - item-type/context indicator if required by existing API pattern
- Response should include required display fields for bundles:
  - `id`, `title`, `thumbnailUrl`, `category`, `bundlePrice`, `discountPct`
- Excluding already-added items may be implemented server-side (preferred) or client-side fallback.

## Out of Scope (Do Not Implement in this feature)
- Rail-level rules/classification logic
- Parsing rail title/semantic inference
- Sticky filters (per rail or global)
- Multi-select category
- Comparator operator UI (`<`, `>`, etc.)
- Pagination/infinite scroll redesign
- Advanced image preview (lightbox/side panel)

## Acceptance Criteria
1. Admin can filter picker results by title + category + price range + discount range.
2. Category filter is single-select dropdown using existing category taxonomy.
3. Apply Filters is explicit (no live filtering).
4. Reset Filters resets filter state and results.
5. Invalid range (`min > max`) blocks apply and shows inline error.
6. Result rows use neutral/common columns: Image, Title, Category, Price, Discount (for both books and bundles).
7. Thumbnail hover shows enlarged preview.
8. Already-added rail items do not appear in results.
9. Result panel uses fixed-height internal-scroll pattern consistent with bundle-book selector.
10. Rail Admin Note is saved, editable, length-capped, and admin-only.

## Test Checklist (MVP)
- Price: 199–299 returns only matching bundles.
- Discount: 20–40 returns only matching bundles.
- Category=Fiction returns only Fiction.
- Combined filters return intersection.
- Price `min > max` shows error and blocks apply.
- Discount `min > max` shows error and blocks apply.
- Reset clears all fields and resets list.
- Already-selected rail item is absent in picker.
- In bundle context, row maps to common columns (Image/Title/Category/Price/Discount) using bundle values.
- In book context, row maps to common columns (Image/Title/Category/Price/Discount) using book values.
- Hover preview appears/disappears correctly.
- Admin Note persists across create/edit and enforces max length.
