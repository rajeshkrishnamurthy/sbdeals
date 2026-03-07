# Sprint 1 — Patch: Bundles image + listing fields refinement

## Problem
After real data entry, two blockers were identified in Bundles:

1) There is no obvious bundle image capture flow, and bundle image does not appear in bundle listing.
2) Bundle listing should show **discount %**; **condition(s)** are not needed in listing.

## Goal
Tighten Bundles admin UX so list pages are merchandising-friendly and operationally useful:

- Every bundle can have a visible image.
- Bundles list shows key commerce signal (discount %) instead of low-value clutter (conditions).

## Scope (this patch)

### A) Bundle image support (manual upload)
1. Add bundle image input in **Add Bundle** and **View/Edit Bundle**.
2. Show bundle image thumbnail in **Bundles list**.
3. Keep image handling simple (single image per bundle, no auto-generation).

### B) Bundles list column refinement
1. Add **Discount %** column in Bundles list.
2. Remove **Allowed condition(s)** column from Bundles list.
3. Keep conditions in Add/Edit flow (needed for filtering and validation), just not in list view.

## Detailed requirements

## 1) Image capture + display

### Form behavior (Add/Edit Bundle)
- Add file input: **Bundle image**.
- On create:
  - Image is required for new bundle creation (recommended default for consistency with Books).
- On edit:
  - Existing image preview is visible.
  - Re-upload replaces current image.
  - If no new file uploaded, keep existing image.

### List behavior (Bundles list)
- Add left-side thumbnail column similar to Books list visual style:
  - Fixed-size thumbnail box (tight row height, consistent look).
  - Preserve aspect ratio (contain/fit; no crop/stretch).
  - Neutral placeholder if image missing (for legacy records, if any).

### Data/storage expectations
- Store bundle image bytes + mime type (parallel to existing book-cover handling pattern).
- Expose a simple image endpoint for bundle thumbnail rendering in list/detail.

## 2) Discount % in list
- Add computed **Discount %** in Bundles list.
- Formula:
  - discount% = round((Bundle MRP - Bundle Price) / Bundle MRP * 100)
- If Bundle MRP is zero/invalid:
  - Display `—` (avoid divide-by-zero and misleading values).

## 3) Remove condition(s) from list
- Remove **Allowed condition(s)** column from Bundles list table.
- No change to bundle condition logic in create/edit validation flow.

## Validation & error feedback
- Must follow admin standards: validation failures show toast.
- Bundle image validation error (if required on create): clear toast message.

## Out of scope
- Bundle image auto-generation/collage logic.
- Public catalog rendering changes.
- Any redesign of the bundle form flow beyond adding image field and preview.

## Acceptance criteria
1. Admin can upload a bundle image during create and update it in edit.
2. Bundles list shows a thumbnail image per row (or placeholder when absent).
3. Bundles list shows **Discount %** per row.
4. Bundles list no longer shows **Allowed condition(s)**.
5. Existing condition-based eligibility/filtering in Add/Edit remains intact.
6. Validation errors surface via toast.

## Notes for implementation prompt
- Reuse existing Books image upload/display approach where possible (storage + endpoint + thumbnail rendering) to minimize implementation risk.
- Keep table compact; avoid introducing page width jump regressions.