# AlephK / Service Hub — Deploy guide (database scripts, environment, install)

Updated: 2026-10-24 (after shao #15, usersAppAK #4, p3 #21, showcase #15).
This file lives in the `shao` folder, but it covers the 4 projects:

| Project | Folder | What it is |
|---|---|---|
| shao | `~/projects/go/src/github.com/milbertk/shao` | Go API of Service Hub (port 8081) — database **shao** |
| usersAppAK | `~/projects/go/src/github.com/milbertk/usersAppAK` | Go API of users / companies / AI agent / support (port 8080) — database **usersAK** |
| p3 | `~/projects/react/AlephK/p3` | React management app (Service Hub) |
| showcase | `~/projects/react/AlephK/showcase` | React public store |

---

## 0. Before you start

1. **Back up both databases** (one command each, keep the files):
   ```bash
   pg_dump -h <host> -U <user> -Fc -f shao_$(date +%F).dump    <shao_database>
   pg_dump -h <host> -U <user> -Fc -f usersak_$(date +%F).dump <usersak_database>
   ```
2. Stop the two Go services while you run the scripts (a few seconds of downtime).
3. Run the scripts with `psql -v ON_ERROR_STOP=1 -f <file>` so it stops at the first error.
4. **Order matters.** Every script is "safe to run twice", but the **plan scripts (steps 9–11) overwrite plan limits**
   (`UPDATE tbl_plans ...`). Never run them again after you change limits by hand, and never run step 10 after step 11
   (step 10 would put back the old storage values 50 / 20 / 20 / 10 GB).

### Which scripts did I already run?

Paste this in the **shao** database. Every `false` is a script that is still missing:

```sql
SELECT 'tbl_page_configuration.sql'                     AS script, to_regclass('public.tbl_page_configuration') IS NOT NULL AS done
UNION ALL SELECT 'migration_page_configuration_logo_header', EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='tbl_page_configuration' AND column_name='logo_url')
UNION ALL SELECT 'tbl_ai_assistant.sql',                   to_regclass('public.tbl_ai_message') IS NOT NULL
UNION ALL SELECT 'migration_product_currency_symbol',      EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='tbl_products' AND column_name='currency_symbol')
UNION ALL SELECT 'migration_notifications_and_self_service', to_regclass('public.tbl_notification_log') IS NOT NULL
UNION ALL SELECT 'migration_store_slug',                   EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='tbl_page_configuration' AND column_name='slug')
UNION ALL SELECT 'migration_store_currency_payment',       EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='tbl_page_configuration' AND column_name='payment_instructions')
UNION ALL SELECT 'migration_company_verification',         to_regclass('public.tbl_company_verification') IS NOT NULL
UNION ALL SELECT 'migration_ai_plan_cost_cap',             to_regclass('public.tbl_company_ai_settings') IS NOT NULL
UNION ALL SELECT 'migration_company_plans',                to_regclass('public.tbl_plans') IS NOT NULL
UNION ALL SELECT 'migration_plans_support_storage_price',  EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='tbl_plans' AND column_name='storage_gb')
UNION ALL SELECT 'migration_plans_entrepreneur_special',   to_regclass('public.tbl_company_storage_file') IS NOT NULL
UNION ALL SELECT 'migration_company_legal_texts',          EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='tbl_page_configuration' AND column_name='terms_text')
UNION ALL SELECT 'migration_company_time_zone (NEW)',      EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='tbl_page_configuration' AND column_name='time_zone');
```

And this one in the **usersAK** database:

```sql
SELECT 'aiConversations.sql' AS script, to_regclass('public.ai_conversations') IS NOT NULL AS done
UNION ALL SELECT 'supportRequests.sql (NEW)', to_regclass('public.tbl_support_request') IS NOT NULL;
```

---

## 1. SQL scripts — shao database (folder `shao/sql/`)

Run only the ones that are still missing, **in this order**:

| # | File | What it does | Notes |
|---|---|---|---|
| 1 | `tbl_page_configuration.sql` | Company page table | Only on a new database. |
| 2 | `migration_page_configuration_logo_header.sql` | logo + header images | Only on old databases that had `image_url`. **Drops** the old cover image columns. |
| 3 | `tbl_ai_assistant.sql` | AI conversations / messages (shao) | |
| 4 | `migration_product_currency_symbol.sql` | currency per product | |
| 5 | `migration_notifications_and_self_service.sql` | notifications, company SMTP, "manage my request" links | Then set the store address, see section 4. |
| 6 | `migration_store_slug.sql` | readable store address | |
| 7 | `migration_store_currency_payment.sql` | one currency per company + how to pay | |
| 8 | `migration_company_verification.sql` | "Verified company" badge | Nobody is verified by default. |
| 9 | `migration_ai_plan_cost_cap.sql` | AI monthly cost cap | Needs step 3. Check the AI prices inside. |
| 10 | `migration_company_plans.sql` | plans, company plan, plan requests | |
| 11 | `migration_plans_support_storage_price.sql` | support hours, storage, agreed price | Needs step 10. **Only once, before step 12.** |
| 12 | `migration_plans_entrepreneur_special.sql` | ENTREPRENEUR, SPECIAL, no free plan, storage table | Needs step 11. Moves FREE companies to ENTREPRENEUR. |
| 13 | `migration_company_legal_texts.sql` (2026-10-24) | `terms_text`, `privacy_text` in the Company page | Needed by shao #15. |
| 14 | **`migration_security_cc1.sql`** (NEW, 2026-10-25) | AI questions per plan (max 10% of price), store form / email limits, Firebase bucket, indexes | Needed by shao #16. Check the AI questions of each plan with query (a) inside. |
| 15 | **`migration_company_time_zone.sql`** (NEW, 2026-11-04) | `time_zone` (default `America/Costa_Rica`) and `time_zone_label` in the Company page | Needed by shao #32. **Run it before starting shao #32.** Also put the server in UTC (section 1b). |

**Do not run** `migration_company_plans_usersak.sql` (old, not needed anymore — you can delete it).

```bash
cd ~/projects/go/src/github.com/milbertk/shao
psql -h <host> -U <user> -d <shao_database> -v ON_ERROR_STOP=1 -f sql/migration_company_legal_texts.sql
psql -h <host> -U <user> -d <shao_database> -v ON_ERROR_STOP=1 -f sql/migration_security_cc1.sql
psql -h <host> -U <user> -d <shao_database> -v ON_ERROR_STOP=1 -f sql/migration_company_time_zone.sql
```

## 1b. Server clock and database in UTC (2026-11-04, shao #32)

The server and PostgreSQL work in **UTC (GMT)**. Each company has its own time zone in the Company page (default
Costa Rica, GMT−6); shao uses it for every "now" of the company. Saved appointment hours never change. Once, as root:

```bash
timedatectl set-timezone Etc/UTC
sudo -u postgres psql -c "ALTER SYSTEM SET timezone = 'UTC';"
sudo -u postgres psql -c "ALTER SYSTEM SET log_timezone = 'UTC';"
systemctl reload postgresql
timedatectl | grep "Time zone"          # Etc/UTC (UTC, +0000)
sudo -u postgres psql -c "SHOW timezone;" # UTC
```

Cron times are then in UTC: **08:00 UTC = 02:00 Costa Rica** (no summer time in Costa Rica).

## 2. SQL scripts — usersAK database (folder `usersAppAK/sql/`)

| # | File | What it does |
|---|---|---|
| 1 | `aiConversations.sql` | AI agent conversations (usersAppAK) |
| 2 | **`supportRequests.sql`** (NEW, 2026-10-24) | general support table `tbl_support_request` (all users, all companies) |

```bash
cd ~/projects/go/src/github.com/milbertk/usersAppAK
psql -h <host> -U <user> -d <usersak_database> -v ON_ERROR_STOP=1 -f sql/supportRequests.sql
```

All tables use `gen_random_uuid()` → PostgreSQL 13 or newer (or `CREATE EXTENSION IF NOT EXISTS pgcrypto;` on older versions).

---

## 3. Environment variables

Each project has a `.env.example`. Copy it and fill it in. **Never commit `.env`.**

### shao (`shao/.env.example`)

| Variable | Required | Meaning |
|---|---|---|
| `PORT` | no (8081) | API port |
| `APP_ENV` | **yes in production** (`production`) | `production` = only the listed origins (CORS_ALLOWED_ORIGINS + `cors_allowed_origins.txt`), `ALLOW_ANY_ORIGIN` ignored; `test` = any origin (never on the internet); not set = as before (local list). |
| `ALLOWED_DOMAINS` | **yes in production** | `aleph-k.com,aleph-k.online` — each domain **and all its subdomains**, http and https, may call the API (2026-10-31). One place to add/remove domains; restart, no rebuild. |
| `CORS_ALLOWED_ORIGINS` | optional | extra exact origins outside those domains (or lines in `cors_allowed_origins.txt`; a line without `http(s)://` is a domain). |
| `SHOWCASE_BASE_URL` | **yes in production** | store address for email and share links, e.g. `https://store.aleph-k.com` (wins over `tbl_configuration`). |
| `ALEPHK_LOGO_URL` | optional | public **https** link of the AlephK logo for the footer of emails sent with the AlephK general email. |
| `SMTP_HOST` … `SMTP_FROM_NAME` | optional | AlephK general email (only for companies that checked it in Company page). |
| `TRUSTED_PROXY_HOPS` | **yes behind a proxy** | how many of YOUR proxies add `X-Forwarded-For` (1 = Nginx/Caddy, 2 = Google LB + Cloud Run). Used for the per-IP limits; empty = use the TCP address. |
| `FIREBASE_STORAGE_BUCKET` | recommended | bucket accepted for uploaded file URLs (also in `tbl_configuration`). |
| `ANTHROPIC_API_KEY` | for Aleph | AI assistant key |
| `SMTP_ENCRYPTION_KEY` | **yes** | Encrypts the companies' SMTP passwords. **Set once, never change** (if it changes, companies must type their SMTP password again). Falls back to `JWT_SECRET`. |
| `ALLOW_ANY_ORIGIN=1` | dev only | Accept any origin. Ignored when `APP_ENV=production`. |
| Database | — | `connection.json` (shaohost, shaoport, shaouser, shaopass, shaodatabase) |

shao loads `.env` from its folder by itself (since 2026-10-30); real environment variables (systemd `Environment=`,
Docker `env_file`) win over the file.

### usersAppAK (`usersAppAK/.env.example`) — loads `.env` by itself

| Variable | Required | Meaning |
|---|---|---|
| `PORT` | no (8080) | API port |
| `APP_ENV` | **yes in production** (`production`) | same as shao (origins file: `cors_allowed_origins.txt` next to usersAppAK) |
| `ALLOWED_DOMAINS` | **yes in production** | same as shao; also chooses the login cookie domain |
| `CORS_ALLOWED_ORIGINS` | optional | same as shao |
| `APP_PUBLIC_URL` | **yes** | p3 address for invitation links, e.g. `https://app.aleph-k.com` |
| `AUTH_COOKIE_DOMAIN` | optional | overrides `ALLOWED_DOMAINS` for the cookie (list allowed: `.aleph-k.com,.aleph-k.online`) |
| `AUTH_COOKIE_SECURE` | optional | `true` = always Secure (recommended once everything is https); default auto = Secure on https |
| `ANTHROPIC_API_KEY` (+ optional `ANTHROPIC_MODEL`, `ANTHROPIC_MAX_TOKENS`) | for the AI agent | |
| `FIREBASE_CREDENTIALS_FILE` | for change email / password | path of the Firebase admin json (keep it out of git) |
| `SUPPORT_EMAIL_TO` | **yes** for support | where support requests are emailed (you) |
| `SMTP_HOST`, `SMTP_PORT` (587), `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM` | **yes** for support | mail server to send them. If missing, the request is still saved (`email_sent = false`). |
| Database | — | `connection.json` / `databaseConnection.json` |

> ✅ **Login cookie (fixed 2026-10-31).** Login and logout give the `access_token` cookie the domain of
> `ALLOWED_DOMAINS` that contains the API address (`auth.aleph-k.com` → `aleph-k.com`), so `app.aleph-k.com`,
> `api.aleph-k.com` (shao) and `auth.aleph-k.com` (usersAppAK) share the session. Secure on https.
> **Rule:** a web page must call the APIs on **its own domain** — browsers never send the cookie from
> `aleph-k.com` to `aleph-k.online`. A page on `*.aleph-k.online` uses `api.aleph-k.online` / `auth.aleph-k.online`
> (same servers, extra DNS names + certificates).

### p3 and showcase (React) — `.env.production`, read at **build** time

| Variable | p3 | showcase | Example |
|---|---|---|---|
| `REACT_APP_API_USERS` | ✔ | ✔ | `https://auth.aleph-k.com` |
| `REACT_APP_API_SHAO` | ✔ | ✔ | `https://api.aleph-k.com` |
| `REACT_APP_SHOWCASE_LINK` | ✔ | ✔ | `https://store.aleph-k.com` |
| `REACT_APP_WHATSAPP` | | ✔ | `+50600000000` |

Without these variables the apps use the old LAN addresses (`192.168.100.110`), so development keeps working.
After changing them you must **build again** (they are copied into the build).

---

### CAPTCHA — Cloudflare Turnstile (2026-10-25)

| Where | Variable | Your computer / LAN | Production |
|---|---|---|---|
| shao + usersAppAK | `TURNSTILE_SECRET_KEY` | **empty** (CAPTCHA off) or test secret `1x0000000000000000000000000000000AA` | the widget **Secret key** |
| showcase + p3 (build) | `REACT_APP_TURNSTILE_SITE_KEY` | not needed (localhost/LAN use the test key automatically) | the widget **Site key** |

- Protected: store orders and bookings (shao), plan request (shao), support request (usersAppAK). Token header: `X-Captcha-Token`.
- In Cloudflare, the widget **hostnames** must include your domains (`aleph-k.com`, `aleph-k.online`; Turnstile accepts subdomains of a listed domain).
- Both sides must match: real site key ↔ real secret, test site key ↔ test secret (or empty secret).
- Cloudflare down: requests are refused (safer). `CAPTCHA_FAIL_OPEN=1` lets them pass instead.
- If production shows "Please complete the security check" but no box appears: `REACT_APP_TURNSTILE_SITE_KEY` was missing at build time — build again.

---

### Domains checklist — aleph-k.com and aleph-k.online (2026-10-31)

Suggested names (change freely; only `.env` files change, no code):

| Part | aleph-k.com | aleph-k.online (if used) |
|---|---|---|
| p3 (AlephTelos admin) | `app.aleph-k.com` (later `telos.aleph-k.com`, `voting.aleph-k.com`…) | — |
| showcase (stores) | `store.aleph-k.com` | `store.aleph-k.online` or `aleph-k.online` |
| shao API | `api.aleph-k.com` | `api.aleph-k.online` (same server) |
| usersAppAK API | `auth.aleph-k.com` | `auth.aleph-k.online` (same server) |

1. **DNS**: an A/CNAME record for every name above.
2. **HTTPS certificates** for every name (Let's Encrypt / Caddy / Cloudflare). Pages on https must call APIs on https
   (browsers block http calls from an https page = "mixed content").
3. **Backends `.env`**: `APP_ENV=production`, `ALLOWED_DOMAINS=aleph-k.com,aleph-k.online`, shao `SHOWCASE_BASE_URL`,
   usersAppAK `APP_PUBLIC_URL`; behind Nginx/Caddy also `TRUSTED_PROXY_HOPS=1` and pass `X-Forwarded-Proto`.
4. **Frontends `.env.production`** (then build again): API addresses on the **same domain** as the page.
5. **Firebase console → Authentication → Settings → Authorized domains**: add `app.aleph-k.com`,
   `store.aleph-k.com` (and the .online names). Without it Google / email login fails on the new domains.
6. **Cloudflare Turnstile** widget hostnames: `aleph-k.com`, `aleph-k.online`.
7. **Database**: `tbl_configuration` `SHOWCASE_BASE_URL` can stay; the `.env` value wins.
8. Optional: **Firebase Storage CORS** only if the apps download files with `fetch` (images in `<img>` need nothing).
9. New subdomain later (e.g. `voting.aleph-k.com`): only DNS + certificate + Firebase authorized domain. The
   backends already accept every subdomain of `ALLOWED_DOMAINS`.

---

## 4. One-time data settings (shao database)

```sql
-- Address of the public store used in the links of the emails:
UPDATE public.tbl_configuration SET value = 'https://store.yourdomain.com'
 WHERE parameter = 'SHOWCASE_BASE_URL';
```

Firebase console: add the production domains in **Authentication → Settings → Authorized domains**
(password reset links and Google sign-in fail on unknown domains).

**Storage rules (security S13, 2026-10-25):** publish `p3/storage.rules` (Firebase console → Storage → Rules, paste
and Publish; or `firebase deploy --only storage`). They allow only signed-in users, only inside `companies/...`,
only images (and PDF / Word / Excel for order attachments), with size limits. **Test an upload of each kind after
publishing** (product image, Company page logo/header, order attachment, tracking attachment, support screenshot).

---

## 5. Libraries and install

| Project | Needs | Install / build |
|---|---|---|
| shao | Go 1.23+ | `go mod download` → `go build -o shao .` |
| usersAppAK | Go 1.23+ | `go mod download` → `go build -o usersAppAK .` (uses `github.com/joho/godotenv`, already in go.mod) |
| p3 | Node 18+ | `npm install` → `npm run build` (no new packages; `qrcode` is already in package.json) |
| showcase | Node 18+ | `npm install` → `npm run build` |

Optional, once: `go mod tidy` in usersAppAK marks `godotenv` as a direct dependency (it works without it).

---

## 6. Clean up by hand (I cannot delete files on your computer)

- shao: delete the binaries `main` and `shao` (48 MB each). They are now in `.gitignore`. If they were committed before:
  `git rm --cached main shao`.
- usersAppAK: delete `handlers/tetHandler.go` and `handlers/HuserLogin/HLogout.go` (dead code, nothing uses them).
- usersAppAK: the Firebase admin json (`kdauth-…-firebase-adminsdk-….json`) is now in `.gitignore`. If it was ever pushed
  to GitHub, **create a new key in Firebase and delete the old one**: `git rm --cached kdauth-*.json`.
- shao: `sql/migration_company_plans_usersak.sql` can be deleted (not needed).

---

## 7. Start order and quick test

1. Run the SQL (sections 1–2) → 2. start usersAppAK → 3. start shao → 4. publish the p3 and showcase builds.
5. Test: login + "Forgot your password?", open Company page and save the terms text, open the store and see the
   "Terms and conditions" link, send a support request (you receive the email), save an order, approve a web order.
6. A user without the right permission now gets **403** on write actions: give the permissions in
   "Company users / permissions" before you invite people (owners and managers always have access).
