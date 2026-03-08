# Sprint 3 / Feature 01 — Reserve CTA wiring + Clicked event

## Goal
Replace placeholder Reserve CTA behavior with real WhatsApp handoff and deterministic Clicked intent capture, while keeping customer friction near-zero.

---

## Decisions Locked (Stage-3)
1. **Customer CTA wording + affordance:** locked
   - CTA label: **"I’m interested"**
   - Show **WhatsApp icon** adjacent to the CTA label to make channel intent explicit.

2. **WhatsApp prefill content:** Option A (minimal)
   - Prefill message starts with a simple interest line and item context.
   - Preferred pattern:
     - Book: `Hi Srikar, I'm interested in this book: {BOOK_TITLE}.`
     - Bundle: `Hi Srikar, I'm interested in this bundle containing: {BOOK_1}, {BOOK_2}, {BOOK_3}.`
   - Rationale: Srikar will manually engage prospect before conversion; detailed disambiguation can happen in admin via title search.

3. **Clicked linkage model:** Option A
   - Create Clicked on CTA tap.
   - No confirmation flow for whether user actually sent the WhatsApp message.

4. **Clicked API failure behavior:** Option C
   - Still proceed with WhatsApp handoff.
   - No user-facing failure interruption for MVP.
   - Backend diagnostics/retry can be handled later.

5. **Duplicate-tap handling:** Option B
   - Add simple client-side debounce (short window) to suppress accidental duplicate taps.
   - Keep implementation lightweight (no heavy idempotency system in MVP).

6. **User feedback on tap:** Option B
   - Show non-intrusive toast before handoff:
   - **"Connecting to WhatsApp..."**
   - Toast pattern should remain reusable for later messaging (e.g., WhatsApp unavailable guidance).

---

## In Scope
- Wire Reserve CTA from catalog cards to real WhatsApp deep-link behavior.
- Create Clicked record on CTA tap.
- Show "Connecting to WhatsApp..." toast on trigger.
- Add short debounce to CTA tap handler.
- Keep existing catalog card/body behavior unchanged (no PDP navigation added here).

## Out of Scope
- WhatsApp Business API / automated send.
- Confirmation UX asking whether WhatsApp message was sent.
- Rich prefill payload (IDs/links/pricing) for this feature.
- Fallback form flow (separate feature).

---

## Functional Requirements

### 1) CTA Trigger Behavior
On tapping **I’m interested** (with WhatsApp icon):
1. Show toast: **"Connecting to WhatsApp..."**
2. Attempt to create Clicked record in backend.
3. Trigger WhatsApp deep-link with minimal prefilled text.

### 2) Clicked Record Creation
Clicked record must capture at least:
- item reference (book/bundle id)
- item title snapshot
- item type (book/bundle)
- click timestamp
- source context (catalog surface metadata available at client)

### 3) Failure Handling
- If Clicked create fails, do **not** block WhatsApp handoff.
- No disruptive user error in MVP path.
- Failure should be observable via logs for later diagnostics.

### 4) Debounce / Duplicate Protection
- Prevent repeated CTA taps on same card within a short debounce window (implementation-chosen short interval).
- User should still be able to attempt again after debounce window.

### 5) WhatsApp Prefill
- Use a minimal message template (human-readable, concise).
- Must include item title context.
- Keep template centralized for easy future upgrade.

---

## Acceptance Criteria
1. Primary CTA is now **"I’m interested"** with WhatsApp icon (replacing placeholder behavior).
2. On CTA tap, toast appears: **"Connecting to WhatsApp..."**.
3. WhatsApp handoff is attempted from catalog card CTA.
4. Clicked record is created on CTA tap path (best effort; non-blocking on failure).
5. Rapid double-taps do not create obvious duplicate click spam from accidental taps.
6. No new PDP/detail navigation is introduced by this feature.

---

## QA Checklist
- Tap Reserve on Book card: toast + WhatsApp handoff + Clicked created.
- Tap Reserve on Bundle card: same as above.
- Rapid double tap on same CTA: debounce suppresses accidental duplicate triggers.
- Simulated Clicked API failure: WhatsApp handoff still proceeds.
- Verify no regression to card/body interaction behavior.

---

## Implementation Notes for Codex
- Keep CTA handler deterministic and side-effect ordering explicit.
- Keep debounce local/simple; avoid introducing global interaction lock.
- Keep prefill template and toast copy configurable constants (for later Feature revisions).
- Do not pull in fallback flow logic in this feature.