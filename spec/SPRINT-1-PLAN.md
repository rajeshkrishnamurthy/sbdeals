# SBD — Sprint 1 Feature Sequence (locked)

## Goal
Deliver a usable public catalog (in-stock only) backed by admin management for suppliers, books, and bundles.

## Feature order (implement in this sequence)
1) **Admin: Suppliers (minimal)**
   - Create/edit supplier (name + WhatsApp/phone + notes)

2) **Admin: Book listing create/edit (inventory defaults to 1)**
   - Minimal required fields (title, supplier, prices)
   - Optional metadata/photos
   - MVP rule: inventory is **1 by default** on create

3) **Public: Catalog list (in-stock only)**
   - Show only in-stock books

4) **Public: Book detail page**
   - Clean detail view for a book listing

5) **Admin: Bundle create/edit**
   - Enforce single-supplier bundles (only books from the chosen supplier)
   - Explicit bundle price with default suggestion

6) **Bundle validity + Admin: Invalid Bundles section (basic)**
   - Bundle becomes invalid if any included book is out of stock
   - Invalid bundles not shown in public catalog
   - Admin list to review invalid bundles

## Notes
- Keep spec/business-level focus; Codex will be fed one feature at a time.
