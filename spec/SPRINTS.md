# SBD MVP — Sprint Split (proposed)

## Sprint 1 — Admin foundations (Suppliers, Books, Bundles) + stock validity
**Goal:** Build the admin/inventory foundations first; customer-facing catalog work is deferred to a future sprint based on positioning decisions.

- Public pages (catalog/presentation) are **deferred**; Sprint 1 focuses on admin foundations.
- Book management (admin): add/edit, pricing (MRP + My price + optional bundle price), attach supplier, **In-stock defaults to Yes**
- Bundle management (admin): create/edit bundle, enforce “no bundle-of-bundles” + **single-supplier**, set explicit price (default suggestion)
- Bundle validity rules wired to stock (bundle becomes invalid if any included book is **not In-stock**) + **Invalid Bundles** admin view (basic list is fine)

## Sprint 2 — Customer “Clicked/Interested” pipeline (WhatsApp + fallback form) + stock reservation
**Goal:** Turn browsing into trackable demand; reserve stock at Interested.

- Per-item CTA: **“WhatsApp Srikar”** with prefilled message (includes item link + short code)
- Create **Clicked** record on CTA click; admin list of Clicked (recent)
- Convert Clicked → Interested (capture buyer name + phone + notes)
- Non-WhatsApp fallback form: name + phone → creates **Interested** directly (linked to item)
- Stock reservation rules: when item becomes **Interested**, set **In-stock = No**; catalog hides it
- Interested → **Cancelled** can reinstate **In-stock = Yes**

## Sprint 3 — Orders workflow (admin) + supplier notification (manual WhatsApp)
**Goal:** Move from “Interested” to “Ordered” and notify suppliers cleanly.

- Convert Interested → **Ordered**
- Order views (admin): list + detail; show current stage + key timestamps
- “Send WhatsApp to supplier” button from order detail
  - Pre-constructed message for that order/item
  - Includes magic links for supplier actions

## Sprint 4 — Supplier portal via magic links (Accept/Ship) + closure
**Goal:** Suppliers update status with minimal friction; Srikar can close the loop.

- Supplier magic-link pages (mobile friendly):
  - **Accept** + expected ship date (required; allow “Unknown, will confirm”)
  - **Mark Shipped**
- Admin sees supplier-updated states reflected: Accepted → Shipped
- Srikar marks **Collected**
- Polish pass on workflow screens (worklist/pipeline UX)

## Note
If needed, Sprints 3 and 4 can be merged for speed, at the cost of more integration/debug in a single cycle.
