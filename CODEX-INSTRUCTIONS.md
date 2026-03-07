# CODEX-INSTRUCTIONS (SBD)

## Purpose
This file defines **how Codex should work on Srikar Book Deals (SBD)**.

The *what* (requirements) for any task must come from the **current feature spec** that Rajesh provides (one feature at a time).

---

## Always-on discipline (use for every SBD task)
1) **One feature at a time**
   - Implement only the functionality explicitly requested in the current feature spec.

2) **Spec-first confirmation**
   - Before coding, summarize the feature in 5–10 lines.
   - List a small set of implementation chunks (testable steps).
   - If anything is ambiguous, STOP and ask.

3) **No scope creep**
   - Do not add endpoints, fields, states, pages, or behaviors not specified.
   - Avoid “while we’re here” changes.

4) **Keep it MVP-simple**
   - Prioritize functional completeness + sleek UI.
   - Do not introduce non-functional work (accessibility/deep security hardening) unless explicitly requested.

5) **Avoid drift**
   - If the feature spec conflicts with the canonical MVP spec (`spec/MVP-SPEC.md`), STOP and ask.

6) **Output expectations**
   - After implementing: list files changed + how to run/test + open questions.

---

## How to use MVP rules (do NOT paste the whole MVP every time)
- The canonical MVP perspective lives in: `spec/MVP-SPEC.md`.
- For each task, include **only the MVP constraints relevant to that feature** (typically 3–8 bullets) in the prompt.

Example prompt structure:
- Global `AGENTS.md` policies
- This file (`CODEX-INSTRUCTIONS.md`)
- The current feature spec (e.g. `spec/sprint-1/feature-01-admin-suppliers.md`)
- A short “Relevant MVP constraints” bullet list (only what matters for this feature)

---

## Optional reference (only when needed)
- `spec/MVP-SPEC.md` — canonical MVP business rules and flows
- `spec/SPRINTS.md` — sprint split
- `spec/SPRINT-1-PLAN.md` — sprint 1 feature order
