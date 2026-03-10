# PATCH SPEC — Stock/Publish Consistency for Books, Bundles, and Rails

## Patch Title
Enforce derived bundle stock + one-way auto-unpublish consistency (application-logic enforcement)

## Objective
Guarantee data integrity across catalog entities by enforcing:
- Derived bundle stock from child books
- Auto-unpublish when an entity becomes out-of-stock
- Auto-unpublish of rails when all contained items are unpublished

## Scope

### In scope
- Application-layer enforcement whenever relevant stock/publish transitions occur.
- Bundle stock derivation from child books in both directions.
- One-way auto-unpublish rules for books, bundles, and rails.

### Out of scope
- Cron/reconciliation jobs.
- Any auto-publish behavior.
- Manual override pathways that violate these invariants.

## Invariants (Hard Rules)

### A) Bundle stock is derived
For every bundle:
- `bundle.in_stock = true` **iff** all books in the bundle have `in_stock = true`.
- Otherwise, `bundle.in_stock = false`.
- No exceptions.

### B) Out-of-stock implies unpublished (book/bundle)
- If a **book** flips to `in_stock = false`, it must be auto-set to `published = false`.
- If a **bundle** flips to `in_stock = false`, it must be auto-set to `published = false`.

### C) Rail auto-unpublish on zero published items
- If all items in a rail are unpublished, the rail must be auto-set to `published = false`.

### D) Publish is always manual in reverse direction
- `unpublished -> published` for **book**, **bundle**, and **rail** is always manual.
- No auto-publish from any consistency rule.

## Triggers
Run enforcement logic in application flow when any of the following transitions occur:
1. `book.in_stock` flips (`false -> true` or `true -> false`)
2. `book.published` or `bundle.published` changes in a way that can affect rail aggregate publish state
3. Any transition that can make a bundle become out-of-stock

Current operational assumption:
- Today, `book.in_stock` out->in happens manually in admin.
- Enforcement must be attached to domain logic (not UI-only), so future update paths remain consistent automatically.

## Required Behavior

### 1) Book changes `in_stock: false -> true`
- Find all bundles containing this book.
- Recompute each bundle stock from child books.
- If all child books are in stock, set `bundle.in_stock = true`.
- If not all child books are in stock, keep/set `bundle.in_stock = false`.
- Do **not** auto-publish bundle.

### 2) Book changes `in_stock: true -> false`
- Set `book.published = false`.
- Find all bundles containing this book.
- Set each such bundle `in_stock = false` (or equivalent recompute result).
- For each affected bundle now out-of-stock, set `bundle.published = false`.

### 3) Bundle becomes out-of-stock by any path
- Ensure `bundle.published = false`.

### 4) Rail publish consistency
- When item publish state changes can affect a rail, evaluate rail items.
- If all items in a rail are unpublished, set `rail.published = false`.
- If at least one item later becomes published, rail remains unchanged (manual publish required).

## Publish State Interaction Summary
- Auto-unpublish is allowed per rules above.
- Auto-publish is never allowed.
- Manual `Unpublished -> Published` remains required for books, bundles, rails.

## Data Integrity Expectations
After any successful relevant update transaction:
- No bundle may remain `in_stock = true` if any child book is out-of-stock.
- Any bundle with all child books in-stock must be `in_stock = true`.
- No out-of-stock book/bundle may remain published.
- No rail with all items unpublished may remain published.

## Error/Transaction Requirement
Consistency updates must execute within the same logical transaction boundary as the initiating change (or fail safely so partial inconsistent state is not persisted).

## Acceptance Criteria
- **AC1:** Book `false -> true` stock flip recomputes all containing bundles; bundles become in-stock only if all child books are in-stock.
- **AC2:** Book `true -> false` stock flip immediately forces containing bundles out-of-stock.
- **AC3:** Book/bundle out-of-stock state always results in published=false for that entity.
- **AC4:** Rail auto-unpublishes when all rail items are unpublished.
- **AC5:** No reverse auto-publish for book/bundle/rail.
- **AC6:** No API/admin path can persist a state violating these invariants.
- **AC7:** No scheduled reconciliation job is introduced.

## Test Cases (Minimum)
- Single bundle, single book: stock flips both ways; publish side-effects validated.
- Multi-book bundle:
  - Last out-of-stock child flips in-stock -> bundle becomes in-stock (publish unchanged).
  - Any in-stock child flips out-of-stock -> bundle out-of-stock + unpublished.
- One book in multiple bundles: all impacted bundles updated.
- Book flips out-of-stock while published=true -> book auto-unpublished.
- Bundle flips out-of-stock through any path -> bundle auto-unpublished.
- Rail with N items:
  - Gradually unpublish items until all unpublished -> rail auto-unpublished.
  - Publish one item again -> rail stays unpublished until manually republished.
- No-op update (`book.in_stock` unchanged): no unnecessary side effects.
