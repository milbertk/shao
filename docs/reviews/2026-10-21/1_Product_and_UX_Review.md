# Report 1 — Product & User Experience Review (MVP readiness)

**System reviewed:** AlephK / Service Hub — `shao` (Go backend), `usersAppAK` (Go users/companies backend), `p3` (management app, React), `showcase` (public store, React).
**Date:** 2026-10-21 · **Type:** code review (no live database or production environment was available).

---

## Honest overall verdict

**The product is more complete than most MVPs** — in features it is already past a minimum product. You have:

- a catalog, a public store, a cart and web orders
- bookings with real time slots
- approval flows and order tracking with steps
- reports with PDF export, roles, two languages, and two AI assistants

That breadth is a real strength.

**What is missing is not more features. It is "closing the loop" and "trust":**

1. The right person never gets told (notifications go to one global address).
2. The customer never hears back after an approval, rejection or cancellation.
3. Some screens still say "demo".
4. Permissions and plans exist on screen but do nothing yet.
5. A new business owner gets no guided path from sign-up to a shareable store.

**Recommendation:**

- **Do not charge money yet.** Fix the **P0** items below (about 2–3 weeks of focused work).
- Then run a **closed pilot with 3–5 friendly businesses** (Costa Rica, Spanish, mostly mobile).
- Do the **P1** items during the pilot, guided by what those businesses actually use.
- **Cut or hide** features nobody touches, rather than polishing everything.

**Priority legend:**

- **P0** = before any real customer.
- **P1** = during the pilot or before charging.
- **P2** = growth / later.

**Effort legend:** **S** = hours to 1 day, **M** = 2–5 days, **L** = 1–3 weeks.

---

## Aspect 1 — Notifications go to the wrong person (P0 · M)
- **What happens:** every "new web order" and "new appointment request" email goes to one global address (`SMTP_TO_DEFAULT` in `tbl_configuration`). If that isn't set, it goes to your personal Gmail, which is hard-coded in `shao/shared/email_notifier.go`.
- **Why it matters:** a business on your platform never finds out it has a new order. This is the single most important loop in the product.
- **Improve:**
  - Send each notification to the **company's** contact email or owner, read from the company record.
  - Add an "Notification email / WhatsApp" field in Page Configuration.
  - Keep your address only as a BCC while in the pilot.

## Aspect 2 — The customer never hears back (P0 · M)
- **What happens:** the customer gets "we received your request." Nothing is sent when the business approves, rejects, cancels or finishes an order or appointment (`bApproveWebOrder.go`, `bChangeAppointmentStatus.go` send no email).
- **Why it matters:** the customer is left guessing, and the business ends up answering the same question by phone.
- **Improve:**
  - Send a short status email on approve, reject, cancel and done.
  - In each email and confirmation, add a **"Talk to us on WhatsApp"** link prefilled with the reference (`WEB-123`).

## Aspect 3 — Customers can't cancel, reschedule or check status (P1 · M)
- **What happens:** the business can cancel or finish appointments (Appointment Management). The customer has no way to do anything after submitting.
- **Improve:**
  - Each confirmation email gets a private link: `/{company}/r/{reference}?k={secret}`.
  - The link opens a small page showing the status, a **Cancel** button and a **Request another time** button.
  - This removes many WhatsApp messages and no-shows.

## Aspect 4 — Double bookings are possible (P0 · M)
- **What happens:** the public booking only offers free slots in the browser, but the server **does not check** that the slot is still free, is inside the schedule, or is in the future (`bPublicAppointmentRequest.go`). Internal appointments only return a *warning* count of conflicts.
- **Improve:** validate the slot on the server against the same rules as `bBookingSlots.go`. Reject it if it was taken in the meantime, and show "This time was just taken — please choose another."

## Aspect 5 — No onboarding path for a new business (P1 · M)
- **What happens:** after sign-up the owner sees up to 15 menu options and must discover the correct order alone.
- **Improve:** add a first-run **checklist** on the Service Hub home:
  1. Company details.
  2. Logo and header (Page configuration).
  3. First category.
  4. First product with a photo and price.
  5. WhatsApp number.
  6. **Share your store link** (copy, WhatsApp, QR).

  Show progress, for example "3 of 6 done."

## Aspect 6 — Confusing names in menus and URLs (P1 · S)
- **Examples:** "Approval A / Approval B" (`/shao/approvala`, `/shao/aprovalb`), "Tracking maintenance", "Steps", "Self booking".
- **Improve:**
  - Use the business owner's words: **Web orders**, **Booking requests**, **Order stages**, **Book for a customer**.
  - Group the menu: *Sell* (Products, Categories, Web orders, Orders), *Bookings* (Schedules, Requests, Appointments), *Customers*, *My store* (Page, Share link), *Reports*, *Team*.

## Aspect 7 — The public store doesn't look like the business's store (P0 · S)
- **What happens:**
  - The showcase header shows "Showcase / Public showcase" instead of the company's name and logo.
  - Every company shows a **"Verified company"** badge, even though nothing is verified.
  - The sign-in dialog still says **"Demo only. Use sample details; no real account is created."** (`CompanyShowcase.jsx`, `ShowcaseAuthDialog`).
- **Why it matters:** customers lose trust, and it sounds like a test site.
- **Improve:** show the company logo and name in the header, remove the badge (or earn it later through real verification), and remove all demo texts.

## Aspect 8 — Store link is not shareable (P1 · S/M)
- **What happens:** the public URL uses the company's internal UUID (`/showcase/3f2a…`).
- **Improve:**
  - Use a short, readable slug, for example `alephk.com/maderas-finas`.
  - Add a **QR code** and a **Share on WhatsApp** button.
  - Add title, description and image meta tags so the link shows a nice preview in WhatsApp and Facebook.

## Aspect 9 — Customer account on the store gives no benefit (P1 · M)
- **What happens:** a customer can sign in on the showcase, but there is no "My orders" or "My bookings" page.
- **Improve:** either **remove sign-in from the store for the MVP** (guest checkout is enough and is simpler), or add a simple *My orders / My bookings* list.

## Aspect 10 — Browsing by category (P1 · S)
- **What happens:** products appear in one list. Categories were shown as internal IDs (now hidden).
- **Improve:** add category chips with real names at the top of the store ("All · Tables · Boards · Services"). The backend public item would need to include the category name.

## Aspect 11 — How the customer pays (P1 · S)
- **What happens:** orders are requests only, with no payment step. That's fine for a Costa Rica MVP.
- **Improve:**
  - Add a **"Payment instructions"** text per company (for example SINPE Móvil number, bank account), shown after the order and in the email.
  - Optionally let the customer **upload the payment receipt**.
  - Real online payments can wait (P2).

## Aspect 12 — One currency per cart (P2 · S)
- **What happens:** products can now have different currency symbols. A cart that mixes ₡ and $ adds the numbers together into a total that means nothing.
- **Improve:** recommend **one currency per company** (set in Page configuration). Alternatively, block mixing in the cart with a clear message.

## Aspect 13 — Mobile experience of the management app (P1 · M)
- Most small-business owners will manage orders **from their phone**. The showcase is responsive. The management screens (`p3`) are large forms (several files over 50 KB of JSX).
- **Improve:**
  - Test every daily-use screen (Web orders, Booking requests, Appointments) on a phone.
  - Make those three screens mobile-first: list cards and big Approve / Reject buttons.

## Aspect 14 — Leftover and duplicate screens (P1 · S)
- **Found in `p3`:**
  - Old copies of the store (`Modules/showcase`, `Modules/shao/CompanyShowcase`, routes `/showcase/:companySlug` and `/shao/showcase`).
  - `OrderMaintenance2.jsx`.
  - `Modules/Auth.zip` inside `src`.
  - A duplicate folder `Modules/shao/costumers`.
  - `mockAssistant.js`.
- **Improve:** delete or archive them. Fewer screens means fewer bugs and less confusion when you or a teammate work on the code.

## Aspect 15 — Translations (P2 · S)
- Some texts still appear in English inside the Spanish interface (seen in the product form: "Check if this product has no taxes.", "Show price with taxes").
- **Improve:** run one pass that compares `en.json` and `es.json` keys and fills in the missing Spanish texts.

## Aspect 16 — AI assistants: keep, but control them (P2 · S)
- Both assistants (internal Aleph and the store's shopping assistant) are a real differentiator.
- **Improve:**
  - Show the visitor a short notice when the daily limit is reached.
  - Add a **monthly cost cap per company** on your side.
  - Make the assistant a **plan feature** (for example, PRO only).

## Aspect 17 — Features that could wait or be hidden for the MVP (P1 · S)
- **Candidates to hide until a pilot customer asks:**
  - "Inventory product" type (no stock tracking exists).
  - Facebook, Microsoft and Apple login (not working — see Report 2).
  - Store customer sign-in.
  - Detailed per-item tracking stages.
- **Principle:** every visible option that doesn't work perfectly costs more trust than it adds.

---

## What is already good (keep it)
- Clean separation of the backend into layers (`dataBasesLedger` / `bussunessLedger` / `HandlerLedgers`) and consistent action tracking.
- Real time-slot engine for bookings, approval flows, order stages and reports with PDF export.
- Two languages from the start.
- The public store now looks professional (zoom card, cart, currency symbol).

---

## Suggested roadmap (UX / product)

| Phase | When | Items |
|---|---|---|
| **P0 – before the pilot** | ~2–3 weeks | 1 Notifications to the company · 2 Customer status emails · 4 Server-side slot check · 7 Store branding and no demo texts |
| **P1 – during the pilot** | 4–8 weeks | 3 Customer self-service link · 5 Onboarding checklist · 6 Menu names and grouping · 8 Share link / QR / slug · 9 Store sign-in decision · 10 Category chips · 11 Payment instructions · 13 Mobile daily screens · 14 Cleanup · 17 Hide unfinished features |
| **P2 – growth** | after pilot | 12 One currency per company · 15 Translations pass · 16 AI cost caps and plan feature · online payments · e-invoicing (Hacienda) |
