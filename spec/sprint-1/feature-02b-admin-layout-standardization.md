# Sprint 1 — Fix: Admin layout standardization (prevent page width “jump”)

## Problem
Admin pages (e.g., Suppliers vs Books) render content containers/tables at different widths, causing a visible “jump” when switching between sections.

## Goal
Standardize the admin page layout so all admin list pages share consistent horizontal layout and do not visually jump.

## Requirements (MVP)
- All `/admin/*` pages should use a consistent page container width.
  - Recommend: a single shared layout wrapper with a `max-width` and centered content.
- Tables in list views should not shrink to content width.
  - Recommend: tables use `width: 100%` within the container.
- Keep existing typography/spacing style; this is a standardization pass, not a redesign.

## Suggested acceptance check
- Toggling between `/admin/suppliers` and `/admin/books` should not change the left/right margins of the table block.
- Headings and primary action buttons should align consistently across pages.

## Out of scope
- Responsive/mobile perfection
- Visual redesign of tables
