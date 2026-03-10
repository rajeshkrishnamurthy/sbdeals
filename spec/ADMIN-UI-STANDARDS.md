# SBD — Admin UI Standards (MVP)

Purpose: keep the admin experience sleek and consistent across masters (Suppliers, Books, Bundles) and prevent visual regressions (e.g., page width “jump”).

## Navigation (top nav)
- All `/admin/*` pages must use the same top navigation.
- Nav items (MVP):
  - **Suppliers** → `/admin/suppliers`
  - **Books** → `/admin/books`
  - **Bundles** → `/admin/bundles`
- Current section should be highlighted/active if feasible.

## Layout consistency (no page “jump”)
- All `/admin/*` pages must use a consistent page container/wrapper.
  - Use a shared wrapper with a consistent `max-width` and centered content.
- List tables must not shrink to content width.
  - Tables in admin list views should be `width: 100%` within the container.
- Headings and primary action buttons should align consistently across admin pages.

## Styling scope
- This is standardization, not redesign: keep typography/spacing consistent with existing admin pages.

## Switch toggles (admin lists)
- For list-row boolean actions (for example `Published/Unpublished`, `In-stock`), use a **track-only switch toggle**.
- Do not show `ON/OFF` text labels inside the control.
- The control width must hug the visual switch track (no extra empty right-side capsule space).
- Use color to indicate state:
  - `ON`: green track
  - `OFF`: neutral gray track
- Keep a clear white knob and left/right knob movement to indicate state transition.
- Keep any secondary metadata (for example recency like `(1d)`) outside the switch control.

## Validation & error feedback (admin)
- Validation errors should be shown as a **toast** message (clear, human-readable) at the time of submit/action.
- Avoid silent failures or inline-only errors in MVP; toast should always appear.

## Acceptance check
- Switching between `/admin/suppliers`, `/admin/books`, `/admin/bundles` does not change the left/right margins of the main table block (no visual jump).
- Triggering a validation error shows a toast with an understandable message.
