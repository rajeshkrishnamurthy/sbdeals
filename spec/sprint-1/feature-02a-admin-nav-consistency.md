# Sprint 1 — Fix: Admin top navigation consistency

## Problem
Admin pages currently have inconsistent top navigation across features (e.g., Suppliers vs Books).
This creates a fragmented admin experience.

## Goal
Unify the admin top navigation so all admin pages share the same nav layout and links.

## Requirements (MVP)
- All `/admin/*` pages must render a consistent top navigation.
- Nav must include links to:
  - **Suppliers** → `/admin/suppliers`
  - **Books** → `/admin/books`
- The current section should appear active/highlighted if feasible.
- Keep styling minimal and consistent; no redesign beyond unification.

## Out of scope
- Adding new sections (Bundles, Orders, etc.) unless already present.
- Advanced responsive nav work.

## Acceptance criteria
- Visiting `/admin/suppliers`, `/admin/suppliers/new`, `/admin/suppliers/:id` shows the same nav as `/admin/books` pages.
- Visiting `/admin/books`, `/admin/books/new`, `/admin/books/:id` shows the same nav as suppliers pages.
