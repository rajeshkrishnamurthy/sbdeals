# MVP-RULES-REFERENCE (SBD)

Use this as a **reference**, not an always-on prompt.
Only paste the specific bullets relevant to the feature being implemented.

Canonical source of truth: `spec/MVP-SPEC.md`.

## Core MVP rules (summary)
- Public catalog shows **only In-stock** books and bundles.
- Book listings use an **In-stock boolean** (not a numeric count) and default to **In-stock = Yes** when created.
- Stock is reserved when an enquiry becomes **Interested** (In-stock → No); **Interested → Cancelled** can reinstate to **In-stock = Yes**.
- Bundles: multi-book, no bundles-in-bundles, single-supplier only, explicit price.
- Bundle becomes Invalid if any included book is **not In-stock**; invalid bundles hidden from catalog and shown to admin.
- Lifecycle: Clicked → Interested → (Cancelled) → Ordered → Accepted → Shipped → Collected.
- One item per enquiry/order (no cart).
- Customer WhatsApp CTA includes item link + short item code.
- Non-WhatsApp fallback collects name + phone and creates Interested linked to item.
- Supplier notifications are manual WhatsApp from Srikar.
- Supplier actions are mobile web via magic links (no login/OTP); links never expire in MVP.
- No online payments.
- Accessibility + deep security hardening deferred.
