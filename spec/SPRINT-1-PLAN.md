# SBD — Sprint 1 Feature Sequence (locked)

## Goal
Deliver a usable public catalog (in-stock only) backed by admin management for suppliers, books, and bundles.

## Feature order (implement in this sequence)
1) **Admin: Suppliers (minimal)**
   - Create/edit supplier (name + WhatsApp/phone + notes)

2) **Admin: Book listing create/edit (In-stock boolean; defaults to Yes)**
   - Minimal required fields (title, supplier, pricing)
   - Cover image mandatory (per Feature 02 spec)
   - MVP rule: **In-stock = Yes** by default on create; admin can toggle Yes/No

3) **Public: Catalog list (in-stock only)** — **DEFERRED** (positioning shift: curated deals approach; customer-facing catalog to a future sprint)

4) **Public: Book detail page** — **DEFERRED** (to be tackled alongside curated deals catalog work)

5) **Admin: Bundle create/edit**
   - Enforce single-supplier bundles (only books from the chosen supplier)
   - Explicit bundle price with default suggestion

6) **Bundle validity + Admin: Invalid Bundles section (basic)**
   - Bundle becomes invalid if any included book is out of stock
   - Invalid bundles not shown in public catalog
   - Admin list to review invalid bundles

## Notes
- Keep spec/business-level focus; Codex will be fed one feature at a time.
