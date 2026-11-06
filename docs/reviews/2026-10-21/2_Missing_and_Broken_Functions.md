# Report 2 — Options That Don't Work or Have No Function Behind Them

**System reviewed:** `shao`, `usersAppAK`, `p3`, `showcase` · **Date:** 2026-10-21
**Method:** reading the code. I could not run your Firebase project, database or servers, so items marked *(verify)* should be confirmed by clicking through once.

**Priority:**

- **P0** = must fix before any real customer.
- **P1** = before charging money.
- **P2** = later.

**Effort:** **S** = hours to 1 day, **M** = 2–5 days, **L** = 1–3 weeks.

---

## Aspect 1 — Facebook, Microsoft and Apple login don't work (P0 to hide · M to fix)
- **Where:** `p3/src/Modules/Auth/Components/Login_Component.js`, and the showcase `showcaseFirebaseAuth.js`.
- **Facebook and Microsoft:** the buttons and code exist (`signInWithPopup` with `facebookProvider` / `microsoftProvider`). They fail because the **providers are not configured in the Firebase console**. That is configuration, not code:
  - **Facebook:** create an app at developers.facebook.com, then copy the App ID and secret into Firebase → Authentication → Sign-in method. Add the OAuth redirect URI that Firebase shows into the Facebook app. The app must be switched to **Live** mode, which needs a Privacy Policy URL.
  - **Microsoft:** register an app in Azure (Entra ID), then copy the client ID and secret into Firebase. Add the Firebase redirect URI.
  - **Both:** add your production domain in Firebase → Authentication → **Authorized domains**.
- **Apple:** only shows "Apple sign-in coming soon." It needs an Apple Developer account (paid yearly), a Service ID, a key, and domain verification.
- **MVP advice:** **show only Google and email/password now.** Hide the other three buttons until each one is configured and tested. A broken login button is the worst first impression.

## Aspect 2 — No "Forgot password" (P0 · S)
- **Where:** the p3 login and register screens. No password-reset call (`sendPasswordResetEmail`) exists anywhere.
- **Effect:** an email/password user who forgets their password is locked out forever.
- **Fix:** add a "Forgot password?" link that calls Firebase `sendPasswordResetEmail(auth, email)`. Do the same in the showcase sign-in.

## Aspect 3 — No real plan or subscription flow (P0 if you will charge · L)
- **What exists:**
  - The **company form lets the user pick their own plan** (FREE / BASIC / PRO / ENTERPRISE) and status (`p3 CompanyForm.jsx`). The backend accepts both from the request body (`usersAppAK HcompanyCreate.go`).
  - A nice "no active plan" screen exists (`p3/src/Modules/shao/joinToPLan/NoActivePlan.jsx`), but **no screen uses it**, and its "send" only writes to the console (mock).
  - The `shao` backend **never checks** the plan.
- **Effect:** today everything is free and unlimited, and anyone can mark their own company as ENTERPRISE.
- **What to build (minimum):**
  1. Remove the plan and status fields from the company form. The backend always creates new companies as `TRIAL` or `FREE` and ignores those fields from the client.
  2. A plans table with the features and limits of each plan (for example products, users, AI messages).
  3. A **"Choose / request a plan"** screen. This can reuse `NoActivePlan.jsx` and connect it to a real endpoint.
  4. A trial end date and a check in `shao` (middleware) that returns "plan required" when expired.
  5. Payment: for the first customers, **manual** payment (SINPE or transfer) that you approve in a back-office. Stripe or PayPal can come later.

## Aspect 4 — No back-office for you, the platform owner (P1 · M)
- **What's missing:** there is no screen for you to see all companies, approve plan requests, extend trials, disable an abusive company, or see usage and AI cost.
- **Fix:** a small "Super admin" area, visible only to your user ID or to a role stored in the database.

## Aspect 5 — Team permissions have no effect (P0 · M)
- **Where:**
  - **Frontend:** the menu calls `GET /companies/{id}/my-access` (`p3 .../mainShao/templates/Menuaccessservice.js`), but **that route doesn't exist in `shao`** (`router/router.go`). On top of that, `maintemplateShao.js` has `const hasFullAccess = true`, which looks like a debugging override, so **every user sees every option**.
  - **Backend:** every route uses `RequireOptionPermission("")`. An empty code means **no permission check at all**.
- **Effect:** the "Company users / permissions" screen (`CompanyUsersAccess`) saves permissions that are never applied.
- **Fix:**
  1. Add the `GET /companies/{companyId}/my-access` endpoint, which returns the role and permission codes.
  2. Remove the `true` override.
  3. Put the real `NAV_*` code on each route group in `router.go`. The codes already match the menu keys.

## Aspect 6 — Notifications don't reach the business, and nothing reaches the customer after a status change (P0 · M)
- Covered in Report 1, Aspects 1–2. Also:
  - Emails are sent **inside** the web request, so the customer waits for the email server.
  - SMTP settings are global, with no company name or logo in the emails.
- **Fix:** company notification email, status-change emails to the customer, and sending emails in the background (a goroutine or a small queue).

## Aspect 7 — Public booking accepts invalid requests (P0 · M)
- **Where:** `shao/bussunessLedger/bPublicShowcase/bPublicAppointmentRequest.go`.
- **Missing checks:**
  - The service belongs to this company, is public and is bookable.
  - The time is inside the schedule and still free.
  - The date is not in the past.
  - Date and time formats are valid.
- **Fix:** reuse the slot logic from `bBookingSlots.go` on the server before inserting.

## Aspect 8 — Public web order trusts the browser (P0 · M)
- **Where:** `shao/bussunessLedger/bPublicShowcase/bPublicWebOrderRequest.go`.
- **Problem:** the **unit price comes from the browser** (`unitPrice` in the showcase payload). The comment "recompute the total on the server" uses those client prices. Product IDs are not checked against the company's public, active catalog.
- **Fix:** load each product from the database (same company, `show_on_web`, `ACTIVE`). Use the **database price** and name, and reject unknown products. Save the header and items in one **database transaction**.

## Aspect 9 — Customer can't cancel or reschedule, and can't see order status (P1 · M)
- See Report 1, Aspect 3. This is the "cancel appointments" example you mentioned: the **business** side exists, the **customer** side does not.

## Aspect 10 — Store sign-in doesn't link orders to the customer (P1 · S, verify)
- **Where:** the public web order handler reads `security.GetClaims(r)`, but the public route has **no authentication middleware**. So the claims are very likely always empty, and every order is stored as `GUEST` even when the customer is signed in.
- **Fix:** an "optional auth" middleware for public routes that reads the cookie when present. Alternatively, decide to drop store sign-in for the MVP.

## Aspect 11 — Demo leftovers in the public store (P0 · S)
- **Where:** `showcase/.../CompanyShowcase.jsx`:
  - The sign-in dialog shows "Demo only… no real account is created."
  - There are unused "Place order (demo)" texts.
  - `MOCK_COMPANIES` and `MOCK_PRODUCTS` are still in the file.
  - The header shows "Showcase / Public showcase."
  - A "Verified company" badge appears for everyone.
- **Fix:** remove them and show the real company identity.

## Aspect 12 — Support request form sends nothing (P1 · S)
- **Where:** `p3/src/Shared/Components/HeaderTools/SupportRequestForm.jsx` uses `MOCK_SEND_DELAY_MS` and `setTimeout`. Nothing is sent.
- **Fix:** add an endpoint that emails you (or stores a ticket), or replace the form with a "Contact support on WhatsApp" link for the MVP.

## Aspect 13 — The app can't be deployed without editing code (P0 · S)
- **Where:**
  - `p3/src/Shared/Utils/globalVar.js` and `showcase/.../globalVar.js` hard-code `http://192.168.100.110:8080/8081/3001`.
  - Both backends hard-code CORS origins to localhost and LAN addresses, and ports in `main.go`.
- **Fix:** read API URLs from `REACT_APP_*` environment variables, and read CORS origins and ports from environment variables in Go.

## Aspect 14 — No account or data deletion (P1 · M)
- **What's missing:** there is no way for a user to delete their account, and no way to delete a company and its data. Products, categories and customers can only be marked `INACTIVE`, which is fine.
- **Why it matters:** Costa Rica's personal data law (Ley 8968), and **Google/Facebook login reviews, expect a deletion option** and a privacy policy.
- **Fix:** a "Delete my account" request flow, even if it is handled manually by you at first.

## Aspect 15 — Legal pages missing (P0 · S)
- **What's missing:** no Terms of Service, Privacy Policy or cookie notice were found in either frontend.
- **Why it matters:** Facebook login (Aspect 1) **requires** a Privacy Policy URL, and you store customer names, phones, emails and addresses.
- **Fix:** add two static pages and link them in the login, the store footer and the order confirmation.

## Aspect 16 — "Inventory product" type without inventory (P2 · M)
- **What happens:** the product type `INVENTORY_PRODUCT` exists, but no stock quantities, stock movements or "out of stock" logic exist.
- **Fix:** hide the type for the MVP, or add a simple stock count that decreases on approval.

## Aspect 17 — No database transactions anywhere (P1 · M)
- **What happens:** no `BEGIN`/`COMMIT` is used in `shao`. Multi-step saves (order plus items, approve web order → create order → update status) can stop halfway and leave half-saved data.
- **Fix:** wrap those flows in `db.BeginTx` … `Commit`.

## Aspect 18 — Database schema not versioned (P1 · S)
- **What happens:** `shao/sql/` only has a few scripts (AI tables, page configuration, currency symbol). The main tables (products, orders, appointments…) have no creation scripts in the repository.
- **Fix:**
  1. Export the current schema once (`pg_dump --schema-only`).
  2. Then use numbered migrations (for example `golang-migrate`).
  3. This is also your disaster-recovery plan.

## Aspect 19 — No automated tests or build pipeline (P1 · M)
- **What's missing:** no `_test.go` files, no frontend tests beyond the default, no CI.
- **Fix:** start small. Add tests for money and time logic (slots, totals, currency formatting), plus a GitHub Action that builds both Go services and both React apps on every push.

## Aspect 20 — Small dead code (P2 · S)
- `usersAppAK/handlers/HuserLogin/HLogout.go` clears a cookie named `auth`, but the real cookie is `access_token`. The router uses the correct logout in `securityGen`, so this file is dead code.
- `usersAppAK/handlers/tetHandler.go` and the `/testwithauth` and `/profile` test routes are still registered.
- Binaries `main` and `shao` (48 MB each) sit in the `shao` folder.

---

## Summary roadmap (missing and broken functions)

| Phase | Items |
|---|---|
| **P0 – before the pilot** | 1 Hide non-working logins · 2 Forgot password · 5 Permissions working · 6 Notifications · 7 Booking validation · 8 Order price validation · 11 Remove demo leftovers · 13 Environment-based config · 15 Legal pages |
| **P1 – before charging** | 3 Plans and subscription · 4 Back-office · 9 Customer self-service · 10 Store sign-in decision · 12 Support form · 14 Account deletion · 17 Transactions · 18 Schema migrations · 19 Tests and CI · (configure Facebook / Microsoft if pilot users ask) |
| **P2 – later** | 16 Inventory · 20 Cleanup · Apple login · online payments · e-invoicing |
