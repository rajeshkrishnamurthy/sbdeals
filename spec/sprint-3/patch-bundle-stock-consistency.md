# PATCH SPEC — Stock/Publish Cascading Consistency

## Objective
Enforce simple, deterministic consistency rules across books, bundles, and rails.

## Canonical Rules

1. **Stock cascades from Book -> Bundle**
   - Bundle stock is derived from child books.
   - If all books in a bundle are in-stock, bundle is in-stock.
   - If any book in a bundle is out-of-stock, bundle is out-of-stock.

2. **Out-of-stock implies unpublished (Book and Bundle)**
   - When a **book** becomes out-of-stock, it is automatically unpublished.
   - When a **bundle** becomes out-of-stock, it is automatically unpublished.

3. **Rail unpublish on zero published items**
   - When all items in a rail are unpublished, the rail is automatically unpublished.

## Reverse Direction Rule
- Auto-publish is **not** allowed.
- `Unpublished -> Published` for book, bundle, and rail remains a manual admin action.

## Enforcement Mode
- Application logic only.
- No reconciliation cron/job in this patch.

## Trigger Points
Apply these rules whenever relevant stock/publish state changes in app logic (not UI-only), including:
- Book stock flip (`in-stock <-> out-of-stock`)
- Any change that can affect bundle stock
- Item publish changes that can affect rail publish status

## Acceptance Criteria
- AC1: Bundle stock always matches aggregate child-book stock.
- AC2: Book out-of-stock always forces book unpublished.
- AC3: Bundle out-of-stock always forces bundle unpublished.
- AC4: Rail with all items unpublished is always unpublished.
- AC5: No auto-publish occurs for book/bundle/rail.

## Minimum Test Coverage
- Book out-of-stock -> book unpublished.
- Book in-stock/out-of-stock flips correctly cascade bundle stock.
- Bundle becomes out-of-stock -> bundle unpublished.
- Rail where last published item becomes unpublished -> rail unpublished.
- Re-publishing book/bundle/rail remains manual only.
