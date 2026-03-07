# Srikar Book Deals (SBD) — MVP Feature Spec (Business-level)

> Notes
> - This is intentionally **non-technical** and avoids code-level object structuring.
> - Delivery to Codex will be **one feature at a time**; this doc is a shared MVP perspective for future specs.
> - Priority: **functional completeness + sleek UI**.
> - Accessibility/screen readers + deep security hardening are **explicitly out of scope** for early MVP.

## 1) Core goal
Sell used/gently-used books online via a responsive catalog, using WhatsApp as the primary communication channel, while tracking each buyer journey to closure and coordinating supplier fulfillment.

## 2) Public user experience (Customer)

> MVP UX guidance (agreed recommendations)
> - WhatsApp prefill should include an item link + short item code to reduce ambiguity.
> - Handle stock race: if an item goes out of stock while someone is viewing, show “no longer available”.
> - Cancellation notes-only is acceptable for MVP.
> - Supplier expected ship date is required; allow “Unknown, will confirm” as a valid value.
### 2.1 Catalog browsing
- Users can browse a catalog of:
  - Individual **Books**
  - **Bundles** (a bundle is a curated set of books)
- Only items that are **in stock** are visible in the catalog.

### 2.2 Item detail
- Each Book/Bundle has a detail view with:
  - cover image (book)
  - title
  - price (derived from MRP/discount)
  - key metadata (condition, format, category)

### 2.2.1 Supplier-specific listings (important MVP assumption)
- The public catalog shows supplier-specific listings.
- The “same book” can appear multiple times in the catalog if sourced from multiple suppliers.
- Each catalog listing maps to exactly one supplier (no de-duplication required in MVP).

### 2.3 WhatsApp interest (one item per enquiry)
- Each Book/Bundle has a CTA: **“WhatsApp Srikar”**.
- Clicking the CTA opens WhatsApp with a **pre-filled message** for that specific Book/Bundle.
- **MVP constraint:** one WhatsApp interest/enquiry corresponds to **exactly one item** (one book or one bundle). No cart.

### 2.4 Non-WhatsApp fallback
- If a user does not have WhatsApp, provide an alternate “express interest” path.
- MVP fallback: user can submit **name + phone number** so Srikar can contact them (typically via WhatsApp or call/SMS as appropriate).
- Submitting this form creates an **Interested** record directly (since we have contact details), associated with the specific item (book/bundle).

### 2.5 “Clicked” event (lead stub)
- When the user clicks “WhatsApp Srikar”, the system creates a record in stage **Clicked** containing at minimum:
  - which item was clicked (Book/Bundle)
  - time of click
- Clicked records are kept permanently; admin views default to showing recent Clicked entries.
- This record exists so Srikar can convert it into a real enquiry without having to search for the item.

## 3) Back-office experience (Srikar/Admin)
### 3.1 Admin sections (minimum)
- Dashboard / Worklist
- Books management
- Bundles management
- Clicked list
- Enquiries pipeline
- Orders pipeline
- Suppliers
- Invalid Bundles

### 3.2 Enquiry lifecycle
The canonical lifecycle is:
**Clicked → Interested → (Cancelled) → Ordered → Accepted → Shipped → Collected**

- **Clicked → Interested**
  - Srikar opens a Clicked record and converts it to Interested by adding:
    - buyer name
    - buyer phone number
    - optional notes

- **Interested → Ordered**
  - Srikar manually converts an Interested into an Order (no automation).

- **Collected**
  - Marking Collected is the operational “completed” state.

### 3.3 Stock rules (MVP)
- Use an **In-stock** boolean for book listings and bundles (instead of a numeric stock count).

### Freeze note (for Sprint 1 Books UI)
- Admin Books list uses a small cover thumbnail with fixed box size and **contain/fit** rendering (aspect ratio preserved; no crop/stretch; row height stays tight).
- New book listings default to **In-stock = Yes**.
- Public catalog must show **only In-stock** items.
- Primary reservation trigger remains: when an enquiry becomes **Interested**:
  - Book becomes Interested → set **In-stock = No** (reserved/out-of-catalog)
  - Bundle becomes Interested → set **In-stock = No** for the bundle AND set **In-stock = No** for all books in the bundle
- Srikar can manually switch a book listing back to **In-stock = Yes** at any stage.
- When a book’s stock is set to **No**, every bundle containing that book becomes **Invalid/Inactive** (hidden from catalog) and appears in **Invalid Bundles**.
- When a book is set back to **In-stock = Yes**, bundles containing that book should be re-checked:
  - if all books in the bundle are In-stock, the bundle becomes active/In-stock again
  - otherwise it remains invalid/inactive

### 3.4 Fast ways to add books
MVP should support at least one fast path:
- Quick-add form with the required book listing fields (including cover image).
- Bulk add is deferred unless explicitly requested.

## 4) Bundles
- Bundles contain **multiple books**.
- Bundles cannot contain other bundles.
- **Single-supplier constraint:** books inside a bundle must belong to the **same supplier**.
- Bundle price is set explicitly, with a default suggestion of the sum of included books’ “price when bundled”.

## 5) Suppliers & fulfillment
### 5.1 Supplier importance
Suppliers fulfill orders; each book listing is associated to one supplier.

### 5.2 Supplier notifications (WhatsApp, manual send)
- From the order screen, Srikar can click **“Send WhatsApp to supplier”**.
- WhatsApp opens with a pre-constructed message containing the relevant item(s) (in MVP: one item).
- No automated sending required.

### 5.3 Supplier action portal (mobile-friendly)
- Suppliers use a **mobile-friendly web portal** (not a native app) to update order statuses.
- No login/OTP in MVP; suppliers use **magic links**.
- Magic links **do not expire** in MVP.

### 5.4 Supplier status steps
- Supplier can mark an order **Accepted** and provide an **expected shipping date**.
- Supplier can mark an order **Shipped**.

## 6) Pricing
- A book listing stores:
  - **MRP**
  - **My price** (selling price)
  - **Discount** is **derived** from MRP and My price (not stored as a manual input)
  - optional **My price (in bundle)** (input field; not derived)
- A bundle stores:
  - an explicit bundle price

## 7) Out of scope (explicit)
- Online payments (MVP is offline lifecycle only)
- Accessibility/screen-reader support
- Deep security hardening (beyond basic admin gating)
- Automated WhatsApp bot / WhatsApp Business API integration
