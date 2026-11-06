# Report 3 — Security Review

**System reviewed:** `shao`, `usersAppAK`, `p3`, `showcase` · **Date:** 2026-10-21
**Method:** source code review.

**Not reviewed:**

- Your shared library `github.com/milbertk/security`, which does the JWT and company-membership checks. It is outside these folders, so findings that depend on it are marked *(verify)*.
- Firebase console settings and Storage rules.
- Server or hosting configuration.
- The live database.

**Severity:**

- **Critical** = can be abused today with little effort.
- **High** = serious, fix before launch.
- **Medium** = fix soon.
- **Low** = good practice.

---

## What is already done well
- Login uses Firebase ID tokens, verified on the server, and then an **HttpOnly** session cookie.
- Company data is always filtered by `company_id` taken from the **validated context**, not from the request body (for example products, `BSaveProduct`).
- SQL uses **parameters** (`$1, $2…`) everywhere I looked, so there is no obvious SQL injection.
- Public endpoints are separate (`/public/...`) and read only public products (`show_on_web = TRUE AND status = 'ACTIVE'`).
- The public AI assistant has body-size limits and daily limits per visitor, IP and company, and rejects unknown JSON fields.
- Every action is tracked (`handlertracking` / `lt`), which is very useful for investigating incidents.

---

## Findings

### S1 — Permissions are not enforced (Critical · M)
- **Backend:** every company route uses `middlewares.RequireOptionPermission("")`. An empty code **skips the check**.
- **Frontend:** `maintemplateShao.js` forces `hasFullAccess = true`, and the `/my-access` endpoint it calls does not exist.
- **Risk:** any member of a company, even one you meant to limit to "orders only", can call every endpoint: reports, approvals, page configuration, products and so on.
- **Fix:**
  1. Set the real `NAV_*` code per route group.
  2. Add `/my-access`.
  3. Remove the override.
  4. *(verify)* Confirm in the `security` library that `RequireActiveCompanyUser` really restricts the company-user routes to OWNER/MANAGER.

### S2 — Public orders trust prices and products from the browser (Critical · M)
- **Where:** `bPublicWebOrderRequest.go` uses `unitPrice`, `productId` and `productName` sent by the visitor.
- **Risk:** anyone can send an order for ₡1 instead of ₡45,000, or include products from other companies or hidden products. Staff could approve it without noticing.
- **Fix:** look up each product on the server (same company, public, active), use the database price, and reject anything else. Use a transaction.

### S3 — Users can give themselves any plan (High · S)
- **Where:** `usersAppAK HcompanyCreate.go` accepts `plan` and `status` from the request body. The p3 company form shows a plan selector.
- **Risk:** once plans matter, anyone can set ENTERPRISE for free.
- **Fix:** ignore `plan` and `status` from clients. Only your back-office (super admin) can change them.

### S4 — Public forms can be abused for spam and email flooding (High · M)
- **Where:** `POST /public/companies/{id}/web-order-requests` and `/appointment-requests` have **no rate limit, CAPTCHA or validation**. Each request sends **two emails**, one to the business and a confirmation to **any email address the visitor types**.
- **Risk:**
  - A script can create thousands of fake orders and bookings.
  - It can use your mail server to send thousands of emails to strangers. Your domain or SMTP account gets blacklisted.
- **Fix:**
  - Per-IP and per-company rate limits (you already have a limiter for the assistant; reuse it).
  - Cloudflare Turnstile or reCAPTCHA on submit.
  - Validate email and phone formats.
  - A cap on customer confirmation emails per address per day.

### S5 — Secrets stored in plain files inside the project folders (High · S)
- **Where:**
  - `shao/connection.json` and `databaseConnection.json` hold the database host, user and **password** for two databases.
  - `shao/.env` holds the `JWT_SECRET` and `ANTHROPIC_API_KEY`.
  - `usersAppAK/kdauth-…-firebase-adminsdk-….json` is a **Firebase Admin private key**.
  - `usersAppAK` has the same kind of files.
- **Neither Go project has a `.gitignore`.** `shao` has a git remote configured.
- **Risk:**
  - If any of these files is ever committed and pushed, anyone with access to the repository can read your database, sign valid session tokens, or act as admin on your Firebase project.
  - The Firebase key is the most dangerous: it can create and delete users and read Storage.
- **Fix (today):**
  1. Add a `.gitignore` to both Go projects listing `.env`, `*.json` secrets, `*adminsdk*.json`, binaries (`main`, `shao`) and `.ai_backups/`.
  2. Run `git log --all -- connection.json .env '*adminsdk*'` to check whether they were ever committed. If yes, **rotate** the DB password, the JWT secret, the Anthropic key and the Firebase key.
  3. In production, load secrets from environment variables or a secret manager (for example Google Secret Manager, since you plan to use Google Cloud).

### S6 — Database connections without encryption (High · S)
- **Where:** `sslmode=disable` in all connection files.
- **Risk:** acceptable only when the database is on the same machine. Over a network (for example Cloud SQL), passwords and data travel in clear text.
- **Fix:** `sslmode=require` (or `verify-full`) in production. Also give each service its own database user with only the permissions it needs.

### S7 — Session cookie and token handling (High · S)
- **Where:** `usersAppAK/handlers/HuserLogin/HuserLogin.go`.
  - The cookie is `Secure: false`.
  - The **JWT is also returned in the JSON body** (`Token: jwt`), so page JavaScript can read it. That removes the protection HttpOnly gives you.
  - The session lasts **7 days**. Logout only clears the cookie; the token stays valid until it expires.
- **Fix:**
  - `Secure: true` on HTTPS.
  - Stop returning the token in the body.
  - Shorter sessions (for example 1 day with silent refresh).
  - *(verify)* Make `RequireActiveUser` check the user's status in the database, so a disabled user is blocked immediately.

### S8 — No HTTPS and no server hardening (High · M)
- **Where:** both `main.go` use `http.ListenAndServe(":8080/8081", …)`, plain HTTP with **no timeouts**. Only two handlers limit the body size.
- **Risk:** traffic can be read on the network. Slow or huge requests can tie up the server (denial of service).
- **Fix:**
  - Put both APIs behind HTTPS (Cloud Run, a load balancer or Nginx/Caddy).
  - Use `http.Server{ReadHeaderTimeout, ReadTimeout, WriteTimeout, IdleTimeout}`.
  - Use a global `http.MaxBytesReader` (for example 1 MB).
  - Add a panic-recovery middleware in `shao` (usersAppAK already has one).

### S9 — CORS "allow any origin" switch (Medium · S)
- **Where:** both routers reflect **any** origin with `Allow-Credentials: true` when `ALLOW_ANY_ORIGIN=1`.
- **Risk:** if that variable is ever left on in production, any website can call your API **as the logged-in user** (it reads their data, approves orders…).
- **Fix:** remove the switch from production builds. Keep a strict list of your real domains, loaded from configuration.

### S10 — Any user can search the whole user directory (Medium · S)
- **Where:** `GET /searchUser` (`usersAppAK HuserGetBYDesc.go` → `dsearchUsersByEmailNameOrId.go`). Any active user can search **all** users by uid, name or email. An empty search returns up to 100 users with their emails.
- **Risk:** a leak of personal data across companies, and harvesting of emails.
- **Fix:**
  - Require a minimum search length.
  - Return only exact email matches when inviting.
  - Restrict the endpoint to company OWNER/MANAGER.
  - Never return the full list.

### S11 — Tokens and personal data written to logs (Medium · S)
- **Where:**
  - `usersAppAK HuserCreate.go` prints the full **Authorization header (Bearer token)**.
  - Many `fmt.Println` calls print emails, UIDs and company data (`bCreateCompany.go`, `HuserGetBYDesc.go`…).
  - The email fallback writes full email bodies, with customer phone numbers and addresses, to the log when SMTP isn't configured.
- **Fix:** never log tokens. Log only IDs, and use structured logging with levels (debug logs off in production).

### S12 — Customer data sent to your personal inbox (Medium · S)
- **Where:** `shao/shared/email_notifier.go` falls back to a hard-coded personal Gmail for every company's orders and appointment requests.
- **Risk:** privacy (Ley 8968). You receive the personal data of other businesses' customers.
- **Fix:** send to the company's address. If none is configured, don't send, and show a warning in the app.

### S13 — File uploads controlled only by the browser (Medium · M, verify)
- **Where:** product images, page images and order attachments upload directly from the browser to Firebase Storage. The size and type checks (`ProductImageUploadService.js`, 5 MB, images only) run **only in the browser**. The backend stores whatever `url` and `storagePath` the client sends (`HOrderAttachments.go`).
- **Risk:** without strict **Firebase Storage Rules**, any signed-in user could upload big or malicious files, overwrite or delete another company's files, or register attachment URLs pointing anywhere.
- **Fix:**
  - Storage Rules that allow writes only to `companies/{companyId}/...` for members of that company, with a size limit and `image/*` / `application/pdf` only.
  - On the server, accept only URLs from your own bucket and paths that start with that company's folder.

### S14 — Public AI assistant cost exposure (Medium · S)
- **Where:** the limits are kept **in memory** (`bPublicRateLimiter.go`). They reset when the server restarts and don't work across several instances. There is no global monthly spending cap.
- **Fix:** store counters in the database (you already count per company there), and add a global daily and monthly token budget that turns the assistant off when reached.

### S15 — Detailed error messages returned to the client (Low · S)
- **Examples:** `"Error retrieving companies: " + err.Error()`, `"Invalid JSON body: " + err.Error()`, `"Error searching users: " + err.Error()` in usersAppAK.
- **Fix:** return a generic message, and log the detail on the server.

### S16 — Missing browser security headers (Low · S)
- **What's missing:** no `Strict-Transport-Security`, `Content-Security-Policy`, `X-Content-Type-Options`, `Referrer-Policy` or `frame-ancestors`, on the APIs or the static sites.
- **Fix:** add them at the hosting or proxy level (Firebase Hosting / Cloud Run / Nginx).

### S17 — Data integrity (Low/Medium · M)
- **What happens:** no database transactions (see Report 2, Aspect 17), and the public appointment does not verify that the service belongs to the company.
- **Risk:** junk or half-saved data, not a direct breach.

### S18 — Dependencies, backups and monitoring (Medium · S/M)
- **Fix:**
  - Run `govulncheck ./...` on both Go services and `npm audit` on both React apps.
  - Schedule automatic **database backups** and test one restore.
  - Add uptime monitoring and alerts for 5xx errors.
  - Keep the tracking table, but add a retention period.

### S19 — Privacy and compliance basics (Medium · S)
- **What's needed:** you store names, phones, emails, addresses and GPS coordinates of your clients' customers. Costa Rica's Ley 8968 (and PRODHAB) applies.
- **Fix:**
  - Privacy Policy and Terms.
  - A consent checkbox on public forms.
  - A data-deletion process.
  - A data-processing agreement in your terms (the businesses are data owners; you are the processor).

---

## Security roadmap

| Phase | When | Items |
|---|---|---|
| **Now (this week)** | before anything else | S5 `.gitignore` + check git history + rotate if exposed · S11 stop logging tokens · S9 make sure `ALLOW_ANY_ORIGIN` is off |
| **P0 – before the pilot** | 2–3 weeks | S1 permissions · S2 server-side prices · S4 rate limit + CAPTCHA on public forms · S7 cookie and token fixes · S8 HTTPS + timeouts · S12 company email · S13 Storage Rules |
| **P1 – before charging** | 4–8 weeks | S3 plan only from back-office · S6 SSL to DB + least-privilege DB users · S10 user search restriction · S14 AI budget cap · S18 backups, vulnerability scan, monitoring · S19 legal and privacy |
| **P2 – ongoing** | continuous | S15 generic errors · S16 security headers · S17 transactions · periodic review of the `security` library and Firebase rules |

**Note:** I did not see any data stolen or any sign of an attack. These are weaknesses found in the code. Fixing S1, S2, S4 and S5 removes most of the real risk for an MVP.
