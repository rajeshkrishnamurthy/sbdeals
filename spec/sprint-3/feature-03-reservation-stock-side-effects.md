# Sprint 3 / Feature 03 — Out-of-stock side effects on Interested transition

## Goal
When an enquiry transitions from **Clicked** to **Interested**, optionally mark the related **catalog item** (book or bundle) as out-of-stock and enforce catalog visibility using a single rule: **only Published items are shown**.

---

## Decisions Locked (Stage-3)
1. **Trigger:** on lifecycle transition **`clicked -> interested`**.

2. **Item-level control:** boolean flag
   - Field: **`out_of_stock_on_interested`**
   - Applies to both **books** and **bundles**.
   - Semantics:
     - `true` → apply out-of-stock + publish side effects.
     - `false` → no stock/publish side effects.

3. **Defaults:** Option A
   - New books default to `true`.
   - Existing books are backfilled to `true`.
   - New bundles default to `true`.
   - Existing bundles are backfilled to `true`.

4. **Catalog alignment principle:** single signal
   - Catalog continues to depend only on **`published`** state.
   - No parallel stock-based catalog branching for this feature.

5. **Primary item side effect when flag is true:**
   - Mark the enquiry target item (book or bundle) out-of-stock.
   - Auto-unpublish the enquiry target item (`published=false`).

6. **Book→bundle propagation rule:** Option A
   - If the enquiry target is a **book**, bundles containing that book are auto-unpublished (any-member rule).

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
- Add `out_of_stock_on_interested` to book and bundle models and admin create/edit UI.
- Apply side effects on `clicked -> interested` when flag is true.
- Unpublish affected books/bundles based on target type + propagation rules.
- Ensure idempotent and concurrency-safe transition behavior.
- Migrate existing books and bundles with default value.

## Out of Scope
- Cancel/reopen restoration logic (book or bundle re-publish).
- Any redesign of manual publish workflows.
- New lifecycle statuses beyond Clicked/Interested.

---

## Functional Requirements

### 1) Item Configuration
- Add boolean field: `out_of_stock_on_interested`.
- Admin UI label: **Out of stock on interested**.
- Field is editable in both create and edit flows for books and bundles.
- Default: `true`.

### 2) Trigger Point
- Evaluate this feature only on successful transition `clicked -> interested`.
- No side effects for other transitions in this feature.

### 3) Side Effects (when `out_of_stock_on_interested = true`)
On transition:
1. Resolve enquiry target type: **book** or **bundle**.
2. Mark target item out-of-stock (`in_stock=No` or equivalent stock state).
3. If target item is `published=true`, set `published=false`.
4. Record system unpublish metadata with reason `out_of_stock` (fields as per existing model, e.g. reason/by/at).
5. If target is a **book**, resolve bundles containing this book and unpublish each currently-published bundle with reason `out_of_stock`.
6. If target is a **bundle**, no additional fan-out is required in this feature.

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
- Bundle: add `out_of_stock_on_interested` (boolean, default true).

### Migration
- Backfill existing books with `out_of_stock_on_interested=true`.
- Backfill existing bundles with `out_of_stock_on_interested=true`.

### Transition Service
- Extend Clicked→Interested transition handler to:
  - gate side effects via target item’s `out_of_stock_on_interested`.
  - perform stock + publish side effects atomically (transaction/locking pattern per stack).
  - for book targets, cascade bundle unpublish using guarded updates (`published=true` only).

### Metadata
- Persist unpublish reason metadata with reason value `out_of_stock` for system-triggered unpublishes.

---

## Acceptance Criteria
1. For a **book enquiry** with `out_of_stock_on_interested=true`, converting `clicked -> interested` marks the book out-of-stock and unpublishes it.
2. For that book enquiry, published bundles containing the book are unpublished.
3. For a **bundle enquiry** with `out_of_stock_on_interested=true`, converting `clicked -> interested` marks that bundle out-of-stock and unpublishes it.
4. For either target type with `out_of_stock_on_interested=false`, conversion performs no stock/publish side effects.
5. Repeated/concurrent conversion attempts do not duplicate effects.
6. Existing books and bundles after migration have `out_of_stock_on_interested=true`.
7. Catalog behavior remains unchanged conceptually: items appear only when `published=true`.

---

## QA Checklist
- Create/edit book and bundle: toggle visible and persisted for `out_of_stock_on_interested`.
- Convert clicked enquiry for flag=true **book**:
  - book becomes out-of-stock + unpublished.
  - reason metadata reflects `out_of_stock`.
  - containing published bundles become unpublished.
- Convert clicked enquiry for flag=true **bundle**:
  - bundle becomes out-of-stock + unpublished.
  - reason metadata reflects `out_of_stock`.
- Convert clicked enquiry for flag=false target:
  - no stock or publish change.
- Replay transition / concurrent calls:
  - stable final state; no duplicate side effects.
- Migration validation:
  - pre-existing books and bundles have flag set true.

---

## Implementation Notes for Codex
- Keep lifecycle transition and side effects centralized in domain/service layer.
- Enforce server-side idempotency; do not rely only on UI button state.
- Use guarded conditional updates (`WHERE published = true`) for clean no-op behavior.
- Keep bundle dependency resolution deterministic and testable.
- Do not implement cancel/reopen restoration logic in this feature.
