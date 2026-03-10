# PATCH SPEC — Bundle Stock Consistency on Book Stock Change

## Patch Title
Enforce derived bundle stock from child book stock (application-logic enforcement)

## Objective
Guarantee data integrity by ensuring `bundle.in_stock` always reflects the stock state of its constituent books.

## Scope

### In scope
- Application-layer enforcement whenever `book.in_stock` changes.
- Both directions of change (`false -> true` and `true -> false`).

### Out of scope
- Cron/reconciliation jobs.
- Changes to publish/unpublish workflow.
- Manual override pathway for `bundle.in_stock`.

## Invariant (Hard Rule)
For every bundle:
- `bundle.in_stock = true` **iff** all books in the bundle have `in_stock = true`.
- Otherwise, `bundle.in_stock = false`.
- No exceptions.

## Trigger
Run enforcement logic whenever `book.in_stock` flips value in application flow.

Current operational assumption:
- Today this flip occurs via manual admin interface.
- Implementation should still be attached to stock-flip logic (not UI-only), so behavior remains correct if new update paths are added later.

## Required Behavior

### 1) Book changes `in_stock: false -> true`
- Find all bundles containing this book.
- For each bundle, recompute child-book stock condition.
- If all books are in stock, set `bundle.in_stock = true`.
- If not all books are in stock, keep/set `bundle.in_stock = false`.

### 2) Book changes `in_stock: true -> false`
- Find all bundles containing this book.
- Set each such bundle `in_stock = false` (or equivalent recompute result).

## Publish State Interaction
- Do **not** auto-publish bundles.
- Existing manual `Unpublished -> Published` behavior remains unchanged.
- This patch only updates stock consistency.

## Data Integrity Expectations
After any successful book stock update transaction:
- No bundle should remain `in_stock = true` if any child book is out of stock.
- Any bundle whose all child books are in stock must be `in_stock = true`.

## Error/Transaction Requirement
Bundle stock updates must execute in the same logical transaction boundary as the initiating book stock update (or fail safely so no partial inconsistent state is persisted).

## Acceptance Criteria
- **AC1:** Flipping a book to in-stock updates all containing bundles to in-stock when all their books are in-stock.
- **AC2:** Flipping a book to out-of-stock immediately makes all containing bundles out-of-stock.
- **AC3:** No API/admin path can persist a bundle stock value that violates the invariant.
- **AC4:** Bundle publish state is unchanged by this patch.
- **AC5:** No scheduled reconciliation job is introduced.

## Test Cases (Minimum)
- Single bundle, single book: flip both ways.
- Multi-book bundle:
  - Last out-of-stock book flips to in-stock -> bundle becomes in-stock.
  - Any one in-stock book flips to out-of-stock -> bundle becomes out-of-stock.
- One book in multiple bundles: all impacted bundles are updated.
- No-op update (`book.in_stock` unchanged): no bundle recomputation side effects.
- Publish state remains unchanged across all above cases.
