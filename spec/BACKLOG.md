# SBD — Backlog (spec)

## Captured from Sprint-1 feature discussions

### Bundle invalidation + Invalid Bundles admin (moved out of Sprint 1)
- Move this out of Sprint 1; it depends on later stock-management behavior.
- Scope to revisit later:
  - Bundle becomes invalid if any included book is out of stock.
  - Invalid bundles are hidden from future public catalog rendering.
  - Admin has an Invalid Bundles list/view for remediation.
- Date captured: 2026-03-07

### Publish/Unpublish × Stock interaction
- Decide enforcement behavior when a **Published book** is later marked **Out-of-stock**:
  - Option A: auto-unpublish immediately
  - Option B: block stock change until admin unpublishes first
- Rajesh requested to defer this to the **Out-of-stock feature discussion**.
- Date captured: 2026-03-07
