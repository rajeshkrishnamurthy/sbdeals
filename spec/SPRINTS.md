# SBD MVP — Sprint Split (proposed)

## Sprint 1 — Public Catalog (in-stock only) + Book/Bundle basics
**Goal:** Users can browse a clean catalog; admin can create the inventory that powers it.

- Public pages: Home/Catalog, Item Detail (Book + Bundle)
- Show **only in-stock** items in catalog (per inventory rules)
- Book management (admin): add/edit, set individual + bundle price, attach supplier, initial inventory=1
- Bundle management (admin): create/edit bundle, enforce “no bundle-of-bundles” + **single-supplier**, set explicit price (default suggestion)
- Bundle validity rules wired to stock (bundle becomes invalid if any included book stock=0) + **Invalid Bundles** admin view (basic list is fine)

## Sprint 2 — Customer “Clicked/Interested” pipeline (WhatsApp + fallback form) + stock reservation
**Goal:** Turn browsing into trackable demand; reserve stock at Interested.

- Per-item CTA: **“WhatsApp Srikar”** with prefilled message (includes item link + short code)
- Create **Clicked** record on CTA click; admin list of Clicked (recent)
- Convert Clicked → Interested (capture buyer name + phone + notes)
- Non-WhatsApp fallback form: name + phone → creates **Interested** directly (linked to item)
- Stock reservation rules: when item becomes **Interested**, set stock to 0; catalog hides it
- Interested → **Cancelled** reinstates stock to 1

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
