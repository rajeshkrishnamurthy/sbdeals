# Sprint 3 — WhatsApp Reservation Flow (High-Level Context)

## Sprint Goal
Turn catalog browsing into a trackable reservation pipeline by wiring item-level WhatsApp intent capture, fallback path, and admin conversion flow with stock reservation side effects.

This sprint should optimize for deterministic behavior, data integrity, and low-friction customer flow.

---

## Why this sprint exists
Sprint 2 delivered a functional catalog UI with placeholder CTA behavior.
This sprint converts that placeholder into a real, end-to-end reservation workflow foundation.

---

## Scope Strategy (Feature-by-feature inside sprint)
Implement as separate features to reduce Codex error surface and improve QA isolation.

### Feature 01 — Reserve CTA wiring + Clicked event
- Replace placeholder CTA behavior with real action.
- On Reserve tap:
  - create Clicked record (item, timestamp, source metadata)
  - open WhatsApp with prefilled message
- Handle failure paths gracefully (e.g., WhatsApp unavailable).

### Feature 02 — Admin Clicked → Interested conversion
- Admin view for Clicked records (recent-first baseline).
- Convert Clicked to Interested with required buyer details.
- Preserve audit timestamps and source traceability.

### Feature 03 - Reservation stock side effects
- On transition to Interested:
  - set item In-stock = No
  - enforce catalog hiding rules
- Define reversible behavior for cancellation/restoration (exact rule to be locked during feature spec).

### Feature 04 — Non-WhatsApp fallback capture
- Provide fallback path when WhatsApp cannot be used.
- Capture name + phone (+ optional note).
- Create Interested record directly from fallback submission.
---

## In Scope (sprint-level)
- WhatsApp deep-link/prefill generation for per-item reservation intent.
- Clicked and Interested creation paths.
- Admin conversion path from Clicked.
- Stock reservation side effects tied to Interested.
- Minimal error handling and retry UX for critical API calls.

## Out of Scope (sprint-level)
- Automated WhatsApp bot/API messaging.
- Multi-item cart/checkout flow.
- Payment and shipping integrations.
- Heavy analytics/reporting.
- UI polish beyond functional clarity.

---

## Key Design Principles
- WhatsApp-first with fallback, not fallback-first.
- One enquiry maps to one item in MVP.
- Deterministic state transitions with explicit ownership.
- Keep implementation modular; avoid tightly coupled mega-feature.

---

## Acceptance at Sprint Level
- Reserve CTA no longer shows placeholder; initiates real flow.
- Every reservation attempt is traceable (Clicked or Interested).
- Admin can operationally convert demand from Clicked to Interested.
- Stock reservation and catalog visibility are consistent after Interested.

---

## Execution Order (locked for current planning)
1. Feature 01 (CTA + Clicked)
2. Feature 02 (Admin conversion)
3. Feature 03 (Interested side effects: stock + catalog visibility)
4. Feature 04 (fallback path)

Rationale: lock canonical Interested transition behavior (including side effects) before implementing alternate Interested-entry paths (fallback).
