# Sprint 2 — Functional Catalog UI (Rails-First)

## Sprint Goal
Deliver a functional, mobile-responsive catalog frontend that renders Admin-defined rails from backend data, so reservation/ordering workflow development can proceed.

**Priority for this sprint:** functional correctness and integration fidelity over visual polish.

---

## Scope Boundaries

### In Scope
- Render catalog as a single vertically scrollable page of rails.
- Render both **Bundle rails** and **Book rails** from backend.
- Preserve exact rail ordering from Admin/backend.
- Show only rails that are active/published.
- Support horizontal rail browsing on mobile via native swipe.
- Show a **Reserve on WhatsApp** CTA on each card.
- Reserve CTA tap shows non-wired placeholder feedback: **"Coming soon"**.
- Handle rails API failure with inline error state + retry.
- Use one initial load for all published rails/items.

### Out of Scope (Do Not Implement)
- Product detail page (PDP), card expansion, modal details.
- Search, filter, sort.
- Reserve/WhatsApp integration wiring.
- Frontend custom rail ordering.
- Pagination/incremental loading.
- Sleek/polish-heavy UI iteration.
- Dedicated accessibility/security hardening pass (deferred to later phase).

---

## Functional Requirements

### 1) Catalog Layout
- Catalog is one long page (vertical scroll).
- Rails appear in exact backend order.
- Each rail section includes:
  - rail title
  - horizontal card row

### 2) Card Behavior
- Each card displays minimum functional fields:
  - image
  - title
  - pricing block (based on backend contract)
  - Reserve on WhatsApp CTA
- Tapping card body does nothing in this sprint (no navigation).
- Tapping Reserve CTA shows a non-blocking **"Coming soon"** toast/snackbar.

### 3) Data Integration
- Frontend must use live backend rails endpoints.
- No mock/fallback data path.
- Initial page load fetches all published rails/items in one request path.

### 4) Rail Visibility Rules
- Only active/published rails are shown.
- If a rail has no active items, render the rail shell (title/container) with empty content state for that rail.
- If there are no published rails at all: no dedicated global empty-state requirement for this sprint.

### 5) Error Handling
- If API request fails:
  - show inline error state in catalog region
  - include Retry action
- Retry triggers same initial fetch again.

### 6) Mobile Responsiveness
- Rail rows must support natural horizontal swipe on mobile.
- Layout must remain usable without clipping primary information.
- No mandatory desktop-only controls (e.g., rail arrows).

---

## Acceptance Criteria
- Catalog renders both bundle and book rails when both exist.
- Rails appear in exact Admin/backend-defined order.
- Inactive/unpublished rails are not rendered.
- Rail with zero active items still shows rail shell.
- Reserve CTA appears on each card and shows **"Coming soon"** on tap.
- No PDP/detail navigation is triggered from card tap.
- No search/filter/sort controls appear.
- API failure produces inline error + retry, and retry works.
- Mobile horizontal swipe works for rail browsing.

---

## QA Checklist
- Verify behavior with:
  - only bundle rails
  - only book rails
  - mixed rails
  - rail with zero items
  - API failure and recovery via retry
  - mobile viewport widths
- Confirm:
  - backend order fidelity
  - consistent CTA placement and placeholder action
  - no accidental mock-data dependency

---

## Notes for Implementation
- This sprint exists to unblock downstream reservation/ordering workflow work.
- Prefer simple, deterministic implementation choices.
- Explicitly defer aesthetic refinements to a later polish sprint.
