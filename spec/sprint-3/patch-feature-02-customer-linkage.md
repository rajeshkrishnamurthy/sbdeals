# Patch — Feature 02 (Clicked → Interested) Customer Linkage Retrofit

## Purpose
Patch `feature-02-admin-clicked-to-interested.md` to align with Customer Master rollout and the ordered-flow discipline in `feature-xx-interested-to-ordered.md`.

This patch replaces buyer free-text capture in conversion with mandatory customer linkage.

---

## Applies To
- Base doc: `spec/sprint-3/feature-02-admin-clicked-to-interested.md`
- Related doc: `spec/sprint-3/feature-xx-interested-to-ordered.md`

---

## Patch Summary (Normative)

### 1) Conversion payload change (clicked -> interested)
Old:
- Required: buyer name, phone
- Optional: note

New:
- Required: `customer_id`
- Optional: note

`customer_id` must reference an existing customer record from Customer Master.

---

### 2) Conversion UX change
Old:
- Manual buyer name/phone entry modal.

New:
- Customer selection modal (search existing customers).
- Allow quick-create customer from modal if no match exists.
- On successful quick-create, auto-select new `customer_id` and continue conversion.

---

### 3) Data ownership discipline
- Enquiry conversion logic must be customer-reference-centric.
- Free-form buyer name/mobile should not be persisted as conversion source-of-truth.
- `customer_id` is mandatory for `interested` state going forward.

---

### 4) Audit field naming alignment
Replace Feature-02 specific audit semantics:
- from: `converted_by`, `converted_at`
- to: existing generic fields: `last_modified_by`, `l_m_at`

On successful `clicked -> interested`, update generic modified metadata.

---

### 5) Idempotency semantics (unchanged behavior, updated message)
- Repeated conversion attempt on already-`interested` row remains no-op.
- Message text may remain equivalent ("Already converted") or status-specific if standardized later.

---

### 6) Validation updates
- Remove India `+91` phone validation from this conversion feature.
- Phone validation responsibility belongs to Customer Master at create/edit time.
- Conversion endpoint validates only:
  - status guard (`clicked` only)
  - valid `customer_id`
  - optional note constraints

---

### 7) Backfill / retrofit data plan (dev environment)
- For existing legacy rows lacking `customer_id`, backfill to `customer_id = 1` (agreed dev shortcut).
- After backfill, enforce non-null `customer_id` for all new clicked->interested conversions.

---

## Required Edits in Feature-02 Spec
Implement these textual updates in `feature-02-admin-clicked-to-interested.md`:

1. **Decisions Locked**
   - Replace required fields decision with mandatory `customer_id` + optional note.
   - Remove "manual buyer entry" and "no customer master lookup/select" decisions.
   - Update audit attribution decision to `last_modified_by` + `l_m_at`.

2. **In Scope**
   - Add customer select / quick-create integration at conversion.

3. **Out of Scope**
   - Remove statements that customer master integration is out of scope.

4. **Functional Requirements**
   - Replace conversion field requirements from name/phone to customer selection.
   - Remove phone normalization section from conversion requirements.
   - Add customer lookup/quick-create behavior.

5. **Acceptance Criteria**
   - Add criteria asserting conversion requires valid `customer_id`.
   - Add criteria for quick-create flow and successful linkage.
   - Replace converted audit fields with generic modified fields.

6. **QA Checklist**
   - Replace phone validation checks with customer selection / quick-create checks.
   - Verify `last_modified_by` / `l_m_at` on successful conversion.

---

## Compatibility Notes
- This patch is intentionally narrow: it changes Feature-02 conversion input/ownership discipline only.
- Feature-03 stock/publish side-effects remain untouched.
- Ordered conversion behavior remains governed by `feature-xx-interested-to-ordered.md`.
