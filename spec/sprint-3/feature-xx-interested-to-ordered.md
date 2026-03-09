# Sprint 3 / Feature XX — Interested → Ordered (WhatsApp order flow)

## 1) Scope
Define and implement admin conversion of an enquiry from `interested` to `ordered`.

This feature assumes Customer Master exists and is already integrated into enquiry flow.

### In scope
- Exact transition rule for `interested -> ordered`
- Required capture at conversion time
- Customer/address validation and inline enforcement
- Idempotency + duplicate-action behavior
- Admin UX placement, modal behavior, success/error handling

### Out of scope (DO NOT IMPLEMENT)
- Direct `clicked -> ordered` transition
- Payment lifecycle (paid/unpaid states)
- Shipping/fulfillment lifecycle
- Cancel/reopen restore logic
- Customer lifecycle states (prospect/buyer/repeater/loyal)
- Deep audit/history timeline beyond basic modified metadata

---

## 2) Canonical State Rule
Allowed path is strictly:
- `clicked -> interested -> ordered`

Not allowed:
- `clicked -> ordered`

Rationale: stock/publish side-effects are intentionally isolated at Interested stage and must not be coupled into Ordered conversion.

---

## 3) Customer Link Discipline (Retrofit Baseline)
For this feature and forward:
- Enquiry identity is by `customer_id` only.
- Random/free-form name/mobile combinations must not drive conversion logic.
- Customer cannot be changed during `interested -> ordered` conversion.

### Dev retrofit decision
- Existing legacy orphan rows can be backfilled to `customer_id = 1` in dev.
- Post-retrofit, conversions must follow strict customer-link discipline.

---

## 4) Data Capture at Interested → Ordered
On clicking Convert to Ordered, open modal and capture:
1. `order_amount` (required)
2. `note` (optional)

Customer is pre-linked from the enquiry and shown as read-only context in the modal.

### 4.1 Validation
- `order_amount`:
  - required
  - integer only
  - must be `> 0`
- `note`:
  - optional
  - store null if empty-after-trim

---

## 5) Address Enforcement at Conversion
Before allowing conversion to `ordered`, system must ensure linked customer has address.

If customer address is missing:
- enforce capture inline in the same conversion modal
- save address to customer profile
- then allow conversion submission

If customer address already exists:
- no extra address input required

---

## 6) Status Transition + Write Behavior
On successful conversion:
- `status` changes from `interested` to `ordered`
- write `order_amount`
- write optional `note`
- update generic modified metadata (`last_modified_by`, `l_m_at`)

No separate `orders` table in this feature.

---

## 7) Idempotency + Duplicate Action Behavior
If convert action is attempted on already `ordered` record:
- block action
- show explicit error: **"Already ordered"**
- do not mutate order fields

UI should hide/disable Convert action for non-`interested` rows to reduce accidental attempts.

---

## 8) Admin UX
### 8.1 Action placement
- Show **Convert to Ordered** only in row actions for `interested` rows.

### 8.2 Modal
- Fields:
  - Order Amount (required)
  - Note (optional)
  - Address (conditionally required only if customer currently lacks one)
- Customer shown as fixed/read-only context (not editable)

### 8.3 Feedback
- Validation errors shown inline per field
- Success toast/message on conversion completion
- Blocking error for already-ordered scenario

---

## 9) API / Consistency Expectations
- No explicit idempotency-key requirement in MVP.
- Consistency from:
  - transition guard (`interested` only)
  - server-side validation of amount and customer address requirement
  - transactional update for status + order fields + modified metadata (+ customer address write when needed)

---

## 10) Acceptance Criteria
1. Convert action is available only for `interested` rows.
2. Direct `clicked -> ordered` is impossible via UI and blocked server-side.
3. Conversion requires `order_amount` integer `> 0`.
4. Note is optional.
5. Customer cannot be changed during conversion.
6. If customer has no address, modal enforces address entry and persists it to customer before conversion succeeds.
7. If customer already has address, conversion proceeds without mandatory address input.
8. Successful conversion updates status to `ordered` and writes amount/note + `last_modified_by`/`l_m_at`.
9. Re-convert attempt on ordered row returns explicit "Already ordered" error and performs no mutation.
10. No payment/shipping/cancel-reopen behavior is introduced in this feature.

---

## 11) Dependencies / Notes for Implementation Plan
- Depends on Feature XX-A (Customer Master) being available.
- If current Feature-02 still allows Interested without customer, update it to enforce customer linkage at `clicked -> interested` (as retrofit-aligned behavior).
- Keep schema and API contracts customer-reference-centric (`customer_id` as source of truth).
