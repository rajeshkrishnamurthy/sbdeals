# SBD — Sprint 1 Feature Sequence (locked)

## Goal
Finish Sprint 1 as **admin foundations + merchandising controls** (no customer catalog UI):
- Admin CRUD for suppliers/books/bundles
- **Publish/unpublish** controls
- **Rail curation** (which items appear in which rail + ordering)

## Feature order (implement in this sequence)
1) **Admin: Suppliers (minimal)**
   - Create/edit supplier (name + WhatsApp/phone + notes)

2) **Admin: Book listing create/edit (In-stock boolean; defaults to Yes)**
   - Minimal required fields (title, supplier, pricing)
   - Cover image mandatory (per Feature 02 spec)
   - MVP rule: **In-stock = Yes** by default on create; admin can toggle Yes/No

3) **Admin: Bundle create/edit**
   - Enforce single-supplier bundles (only books from the chosen supplier)
   - Explicit bundle price with default suggestion

4) **Admin: Publish/unpublish**
   - Bundles: Draft/Published (or `isPublished`) toggle
   - Optional: Books Draft/Published toggle (only if needed for mixed rails)

5) **Admin: Rails (curation + ordering)**
   - Create/edit rails (title + order)
   - Assign bundles/books into rails + order items within rail

## Explicitly out of scope (moved to Sprint 1a)
- Public catalog list/grid/rails pages
- Book detail / bundle detail pages
- Hover/zoom card interactions
- Any customer-facing presentation/SEO

## Notes
- Keep spec/business-level focus; Codex will be fed one feature at a time.
- Bundle invalidation + Invalid Bundles admin is moved out of Sprint 1 and tracked in `spec/BACKLOG.md` for later stock-management phases.
