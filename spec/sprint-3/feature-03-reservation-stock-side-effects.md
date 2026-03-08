# Sprint 3 / Feature 03 — Out-of-stock side effects on Interested transition

## Goal
When an enquiry transitions from **Clicked** to **Interested**, optionally mark the related book as out-of-stock and enforce catalog visibility using a single rule: **only Published items are shown**.

---

## Decisions Locked (Stage-3)
1. **Trigger:** on lifecycle transition **`clicked -> interested`**.

2. **Book-level control:** boolean flag
   - Field: **`out_of_stock_on_interested`**
   - Semantics:
     - `true` → apply out-of-stock + publish side effects.
     - `false` → no stock/publish side effects.

3. **Defaults:** Option A
   - New books default to `true`.
   - Existing books are backfilled to `true`.

4. **Catalog alignment principle:** single signal
   - Catalog continues to depend only on **`published`** state.
   - No parallel stock-based catalog branching for this feature.

5. **Book side effect when flag is true:**
   - Mark book out-of-stock.
   - Auto-unpublish book (`published=false`).

6. **Bundle propagation rule:** Option A
   - If a book is unpublished via this flow, all bundles containing that book are auto-unpublished (any-member rule).

7. **System reason metadata:**
   - Use reason value: **`out_of_stock`**.

8. **Idempotency:** Option A
   - Repeated/concurrent “mark interested” attempts are no-op after first successful effect.

9. **Write guard on unpublish:** Option A
   - Only transition records currently `published=true` to `published=false`.
   - Already unpublished records remain unchanged.

10. **Restore/reopen logic:** deferred
   - Re-publish/restore behavior for cancel/reopen is explicitly deferred to the dedicated cancel/reopen feature.

---

## In Scope
- Add `out_of_stock_on_interested` to book model and admin create/edit UI.
- Apply side effects on `clicked -> interested` when flag is true.
- Unpublish affected books and dependent bundles.
- Ensure idempotent and concurrency-safe transition behavior.
- Migrate existing books with default value.

## Out of Scope
- Cancel/reopen restoration logic (book or bundle re-publish).
- Any redesign of manual publish workflows.
- New lifecycle statuses beyond Clicked/Interested.

---

## Functional Requirements

### 1) Book Configuration
- Add boolean field: `out_of_stock_on_interested`.
- Admin UI label: **Out of stock on interested**.
- Field is editable in both create and edit flows.
- Default: `true`.

### 2) Trigger Point
- Evaluate this feature only on successful transition `clicked -> interested`.
- No side effects for other transitions in this feature.

### 3) Side Effects (when `out_of_stock_on_interested = true`)
On transition:
1. Mark the book out-of-stock (`in_stock=No` or equivalent stock state).
2. If book is `published=true`, set `published=false`.
3. Record system unpublish metadata with reason `out_of_stock` (fields as per existing model, e.g. reason/by/at).
4. Resolve bundles containing this book.
5. For each such bundle, if `published=true`, set `published=false` and record same reason metadata (`out_of_stock`).

### 4) No Side Effects Path (when flag is false)
- Transition proceeds `clicked -> interested` with no stock mutation and no publish-state change.

### 5) Idempotency + Concurrency
- Multiple retries/concurrent requests for same enquiry must produce one effective side-effect application.
- Subsequent duplicate attempts must behave as idempotent no-op success.
- System must avoid repeated publish churn and inconsistent bundle states.

---

## Data / API / Service Changes

### Data Model
- Book: add `out_of_stock_on_interested` (boolean, default true).

### Migration
- Backfill existing books with `out_of_stock_on_interested=true`.

### Transition Service
- Extend Clicked→Interested transition handler to:
  - gate side effects via `out_of_stock_on_interested`.
  - perform stock + publish side effects atomically (transaction/locking pattern per stack).
  - cascade bundle unpublish using guarded updates (`published=true` only).

### Metadata
- Persist unpublish reason metadata with reason value `out_of_stock` for system-triggered unpublishes.

---

## Acceptance Criteria
1. For a book with `out_of_stock_on_interested=true`, converting enquiry `clicked -> interested` marks it out-of-stock and unpublishes it.
2. Bundles containing that book are unpublished (if currently published).
3. For a book with `out_of_stock_on_interested=false`, conversion performs no stock/publish side effects.
4. Repeated/concurrent conversion attempts do not duplicate effects.
5. Existing books after migration have `out_of_stock_on_interested=true`.
6. Catalog behavior remains unchanged conceptually: items appear only when `published=true`.

---

## QA Checklist
- Create/edit book: toggle visible and persisted for `out_of_stock_on_interested`.
- Convert clicked enquiry for flag=true book:
  - book stock becomes out-of-stock.
  - book becomes unpublished.
  - reason metadata reflects `out_of_stock`.
- Verify all containing published bundles become unpublished with same reason.
- Convert clicked enquiry for flag=false book:
  - no stock or publish change.
- Replay transition / concurrent calls:
  - stable final state; no duplicate side effects.
- Migration validation:
  - pre-existing books have flag set true.

---

## Implementation Notes for Codex
- Keep lifecycle transition and side effects centralized in domain/service layer.
- Enforce server-side idempotency; do not rely only on UI button state.
- Use guarded conditional updates (`WHERE published = true`) for clean no-op behavior.
- Keep bundle dependency resolution deterministic and testable.
- Do not implement cancel/reopen restoration logic in this feature.
