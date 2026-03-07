# SBD Positioning & Catalog Presentation — Notes (Draft)

## Context (current operations)
- Srikar currently sells via WhatsApp community; posts ads ~3–4x/week.
- Community currently has 3 groups: Children, Fiction, Non-Fiction (~200–250 members).
- Ads are also posted into apartment communities via apps like MyGate (via friends/well-wishers).
- Inventory is a few hundred books sourced by Srikar + a larger long-tail from a Mumbai supplier (Durai Book House, Matunga).
- Supplier is unorganised but established; strong relationship with Rajesh/Srikar; keen to expand online but non-technical.

## Core positioning insight
SBD should avoid competing on “infinite catalog” (Amazon-style). It should compete on:
- **curation**, **deal/value**, and **community-style drops**
- a service-like experience that turns “not found” into a lead

## Proposed primary catalog approach (front-end)
- Make the default customer experience a **curated deals drop** (e.g., “This week’s deals”), not a full search-the-world catalog.
- Surface only a **limited rotating set** of in-stock books at any point in time.
- Use strong pricing lanes where feasible (e.g., ₹99 / ₹199 / ₹299) to reinforce the “deals” brand.

## Key risk & mitigation
Risk: users search for a specific title, don’t find it, and assume the site is useless.
Mitigation: add a prominent **“Request a book / sourcing help”** CTA when users don’t find what they want.
- Must be framed honestly as “we’ll try to source” (not a guarantee).

## MVP implementation stance (current decision)
- Defer user-facing catalog work (public catalog list + presentation) to a future sprint.
- Continue building admin/inventory foundations first.

## Open choice (when we implement it)
- MVP curation mechanism:
  - simplest: per-item **Featured/Included in deals** toggle
  - later: named Drops/Collections (e.g., “Week of Mar 7”) with explicit membership
