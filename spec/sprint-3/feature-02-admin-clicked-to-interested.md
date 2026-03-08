# Sprint 3 / Feature 02 — Admin Clicked → Interested conversion

## Goal
Enable admins to process inbound Clicked enquiries into Interested using a deterministic lifecycle transition, with clean auditability and future-ready status modeling.

---

## Decisions Locked (Stage-3)
1. **Required fields at conversion:** Option A
   - Required: **buyer name, phone**
   - Optional: **note**

2. **Workflow model:** Single enquiry lifecycle
   - Use one enquiry record that transitions status (not disconnected parallel records).
   - For this feature, transition supported: **`clicked -> interested`**.

3. **Admin list behavior:** Option C
   - Provide **status filter tabs**.
   - Default tab should be **Clicked**.
   - Model must be lifecycle-ready for future states.

4. **Feature boundary with stock effects:** Option B
   - This feature handles only status transition + buyer data capture.
   - Stock/catalog side effects are deferred to **Feature 03**.

5. **Idempotency on conversion submit:** Option A
   - Repeated conversion attempts must not create duplicate transition effects.
   - If already converted, return/notify: **Already converted**.

6. **Audit attribution:** Option A
   - Capture **converted_by** (admin user id) and **converted_at** timestamp.

7. **Buyer data prefill:** Locked as manual entry
   - Do not prefill from Clicked (no buyer identity exists there in current design).
   - No customer master lookup/select in this feature.

8. **Phone validation/localization:** Option A with India assumption
   - Country fixed to **+91**.
   - Admin enters local mobile number only (no editable country-code control).
   - Persist normalized phone in canonical format with +91 prefix.

9. **Duplicate handling scope:** No duplicate prevention in this feature
   - Same prospect can exist across multiple books/bundles.
   - Duplicate suppression/dedupe rules are deferred until customer-master design.

10. **Post-success UX:** Option A
   - Stay on same list/tab.
   - Show success toast.
   - Converted row should disappear from Clicked view.

11. **Status set implemented in Feature 02:** Option A
   - Implement only **clicked** and **interested** now.
   - Keep structure extensible for future statuses (ordered/accepted/shipped/etc.).

---

## In Scope
- Admin-facing list of enquiries with status tabs (default Clicked).
- Convert action from Clicked to Interested.
- Conversion form: name + phone required, note optional.
- India phone handling (+91 fixed country behavior).
- Idempotent conversion behavior.
- Audit fields on conversion.
- UI feedback for success/already-converted.

## Out of Scope
- Stock or catalog visibility side effects (Feature 03).
- Customer master integration / existing-customer search.
- Cross-item/person dedupe strategy.
- Full downstream pipeline actions beyond Interested.

---

## Functional Requirements

### 1) Data / State Model
- Enquiry has lifecycle status.
- Supported statuses in this feature: `clicked`, `interested`.
- Status architecture should be enum/constant-based and extensible.

### 2) Admin List + Filters
- Show enquiries in admin panel with status tabs.
- Default selected tab: **Clicked**.
- Clicked list is the active processing queue for conversion.

### 3) Convert Action (clicked -> interested)
- Available only for records currently in `clicked` status.
- Opens conversion input (inline or modal, implementation choice).
- Required fields: buyer name, local phone number.
- Optional field: note.

### 4) Phone Rules (India)
- Country code fixed to +91 (non-editable by admin).
- Accept local mobile entry and normalize before persistence.
- Basic validation only (required + sanity/length).

### 5) Submit / Transition Semantics
- On valid submit, transition status from `clicked` to `interested`.
- Persist buyer fields on the same enquiry workflow record.
- Persist audit fields: `converted_by`, `converted_at`.

### 6) Idempotency
- If the record is already `interested`, repeated convert attempts must be no-op.
- Return clear user feedback (e.g., toast/message): **Already converted**.

### 7) Post-Submit UX
- Keep admin on the same list/tab.
- Show success toast on successful conversion.
- Converted record must no longer appear in Clicked tab; it should appear under Interested tab.

### 8) Side-Effect Guardrail
- This feature must not perform stock reservation or catalog hide/show changes.
- Those are implemented only in Feature 03.

---

## Acceptance Criteria
1. Admin sees status tabs with default **Clicked**.
2. Admin can convert a clicked enquiry to interested by entering required fields.
3. Phone entry follows India rules (+91 fixed; local input normalized).
4. Successful conversion updates status, captures buyer data, and stores `converted_by` + `converted_at`.
5. Converted enquiry exits Clicked view and is visible under Interested.
6. Repeated conversion calls are idempotent and do not duplicate effects.
7. No stock/catalog visibility side effects occur in this feature.

---

## QA Checklist
- Open admin list: Clicked tab selected by default.
- Convert valid clicked record: success toast, row removed from Clicked.
- Verify same record appears under Interested with buyer details.
- Attempt reconvert on same record (or replay request): gets Already converted behavior.
- Invalid phone input rejected by basic validation.
- Confirm persisted phone format includes +91 normalization.
- Confirm `converted_by` and `converted_at` are set.
- Confirm item stock and catalog visibility are unchanged by this feature.

---

## Implementation Notes for Codex
- Keep status transition logic centralized (service/domain layer), not scattered across handlers.
- Keep transition API idempotent at server side (do not rely only on UI disable states).
- Use constants/enums for statuses; avoid free-text status writes.
- Keep filters generic so future statuses can be added with minimal UI/backend diff.
- Do not couple conversion endpoint with inventory/catalog services in this feature.
