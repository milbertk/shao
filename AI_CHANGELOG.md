# AI Change Log — shao (AlephK Go backend)

Keeps the latest 15 changes made with Claude (oldest are removed). Newest first.
Earlier shao changes are listed in Claude's change log (memory) and in `.ai_backups/` READMEs.

---

## 33. 2026-11-04 — AI knowledge: clock in the top bar

**Backups:** `.ai_backups/2026-11-04_before_header_clock/`.

**Description:** works with p3 #44 and showcase #25. `blassistant/bAssistantPrompt.go`: new bullet about the clock in
the top bar (company time in its zone; "Your computer" time when the computer is in another zone; the store shows the
company time and "= 18:28 your time"). Version "2026-11-04 (shao #33)". No endpoint, database or tracking change
(the clock reads the existing page configuration). Tests pass. AI knowledge: updated.

**Modified:** bussunessLedger/blassistant/bAssistantPrompt.go

**Revert:** copy the `.bak` file back ("__" = "/"), rebuild and restart shao.

---

## 32. 2026-11-04 — Time zone per company (server in UTC) + fix of #31 (logger package)

**Backups:** `.ai_backups/2026-11-04_before_company_time_zone/`.

**Description:**
- **Run first:** `sql/migration_company_time_zone.sql` (adds `time_zone` VARCHAR(64) NOT NULL DEFAULT
  'America/Costa_Rica' and `time_zone_label` VARCHAR(60) to `tbl_page_configuration`; safe to run twice; REVERT inside).
- The server and the database work in **UTC**. Each company chooses its zone in the Company page (IANA name, e.g.
  `America/New_York`; summer time is automatic). Default: Costa Rica (GMT−6). Optional text for customers (max 60).
- New package `bussunessLedger/bltimezone` (+ tests): `NormalizeTimeZone`, `NormalizeLabel`, `CompanyNow`,
  `CompanyToday`, `CompanyLocation`, `ZoneText` ("Hora de Costa Rica, GMT−6"), a 5-minute cache per company
  (`Forget` after saving the Company page). `main.go` embeds `time/tzdata` (no system zone files needed).
- Uses of the company zone: free slots of the store (past hours of today), "change until 2 hours before" and the
  self-service windows (appointment start built in the company zone), deleting manual slots "from today", the TODAY line
  of Aleph and of the store assistant, the times in the customer and company emails ("10:00–11:00 · Hora de Costa
  Rica, GMT−6"; references, subjects and WhatsApp texts unchanged), the date of a cancel request, plan request dates.
- Saved appointments and slots **do not change** (date + hour, 14:00 stays 14:00); only "now" uses the zone.
- Company page API (`PUT /companies/{id}/page-configuration`): new fields `timeZone`, `timeZoneLabel` (invalid zone =
  400). Public page configuration, day slots and self-service view return `timeZone` (+ label).
- **Fix of #31:** `shared/logger.go` and `logger_test.go` said `package shared` but the folder is `emailnotifier`,
  so shao did not compile with #31. Now `package emailnotifier`; `main.go` calls `emailnotifier.SetupLogger()`.
- Tracking (`lt`) unchanged: no new endpoint. Tests: bltimezone, self-service in New York, email times (summer and
  winter), and a PostgreSQL test of the migration + save. AI knowledge: updated (version "2026-11-04 (shao #32)").

**Created:** sql/migration_company_time_zone.sql, bussunessLedger/bltimezone/bCompanyTime.go (+ _test),
bussunessLedger/bPublicShowcase/bSelfServiceTimeZone_test.go, bussunessLedger/blnotifications/bNotifyTimeZone_test.go
**Modified:** main.go, shared/logger.go, shared/logger_test.go, dataBasesLedger/pageconfiguration/page_configuration.go,
dataBasesLedger/booking/booking_manual_slot_delete_future.go, HandlerLedgers/pageconfigurationinshao/pageConfigurationHandlerCommon.go,
HandlerLedgers/hPublicShowCase/HPublicPageConfiguration.go, bussunessLedger/blpageconfiguration/bPageConfiguration.go,
bussunessLedger/bPublicShowcase/bBookingSlots.go, bSelfService.go, bussunessLedger/bbooking/bSaveBooking.go,
bussunessLedger/blnotifications/bNotifyCommon.go, bNotifyEvents.go, bNotifyUpdates.go, bNotifyCancelRequest.go,
bussunessLedger/blassistant/bAskAssistant.go, bAskShowcaseAssistant.go, bAssistantPrompt.go, bShowcasePrompt.go,
bussunessLedger/blplan/bPlanRequest.go, DEPLOY_README.md

**Server (once):** `timedatectl set-timezone Etc/UTC`; PostgreSQL `timezone` and `log_timezone` = 'UTC' (commands in the
migration and DEPLOY_README); backup cron times in UTC (08:00 UTC = 02:00 Costa Rica).

**Revert:** copy the `.bak` files back ("__" = "/"), delete the created files, run the REVERT lines of the migration,
rebuild and restart shao. (Keep the logger fix: without it shao does not compile.)

---

## 31. 2026-11-01 — Request logs: request id, JSON logs with levels, panic -> clean 500

**Backups:** `.ai_backups/2026-11-01_before_request_logs/`.

**Description:**
- New `shared/logger.go` (+ test): `SetupLogger()` (called in `main.go` after `.env`): every log line, including the
  existing `log.Printf` lines, goes through `log/slog` with a level. `LOG_FORMAT=json` (default when
  `APP_ENV=production`) writes one JSON line with `severity` / `message` (Google Cloud Logging keys); `text` otherwise.
- New `middlewares/RequestLog.go` (+ test), registered with `r.Use` in `router/router.go`: one line per API call with
  `request_id`, `method`, route TEMPLATE (never the real path or query: some paths carry private customer tokens),
  `status`, `duration_ms`, `bytes`, `company_id`; 5xx = ERROR, 4xx = WARN; successful `/health` and `/` not logged.
  Every response has the header `X-Request-ID`. A panic is logged with its stack and answered with a JSON 500 that
  includes `requestId`. Never logged: bodies, cookies, Authorization (emails were already masked).
- `env.example.NEW.txt`: `# LOG_FORMAT=` and the existing (undocumented) `# PLAN_REQUEST_EMAIL_TO=`.
- No database change, no endpoint change (tracking unchanged). AI knowledge: not needed.

**Created:** shared/logger.go, shared/logger_test.go, middlewares/RequestLog.go, middlewares/RequestLog_test.go
**Modified:** main.go, router/router.go, env.example.NEW.txt

**Revert:** copy the `.bak` files back ("__" = "/"), delete the created files, rebuild and restart shao.

---

## 30. 2026-11-01 — AI assistant knowledge updated (new features), model from .env, path test

**Backups:** `.ai_backups/2026-11-01_before_ai_knowledge_update/`.

**Description:**
- `blassistant/bAssistantPrompt.go`: `assistantKnowledge` updated with everything since 24 Oct: AlephK general email
  option (the old text said "without SMTP no emails are sent"), customer service email (blind copy), "[Copy]" emails,
  WhatsApp + social network link for the store "Contact us", "This is an appointment" switch and no appointments in
  orders, "Notify the customer (email)" per step and the order status emails, "New customer" in New order, store link
  in "Showcase link" (no preview card claim), Microsoft sign-in active (Facebook / Apple soon).
- New `assistantKnowledgeVersion` ("2026-11-01 (shao #30)"), sent to Claude as "Knowledge version: ...".
- New `assistantScreenPaths` (must match p3 `assistantRoutes.js`) + `bAssistantPrompt_test.go`:
  `TestAssistantPromptPaths` fails when the knowledge uses a path that is not a real screen;
  `TestAssistantKnowledgeVersion`; `TestAssistantModelFromEnv`.
- Model: `assistantModel()` (`bClaudeClient.go`) = `ANTHROPIC_MODEL` from `.env`, else `AssistantModel`
  (claude-haiku-5-5). Used by the AlephTelos and the store assistants (`bAskAssistant.go`, `bAskShowcaseAssistant.go`).
- `env.example.NEW.txt`: `# ANTHROPIC_MODEL=` documented.
- **Rule from now on:** every change that adds, renames or removes a screen, button, field or feature updates the AI
  knowledge in the same change (this file + usersAppAK `bAISystemKnowledge.go` + p3 `assistantRoutes.js` when paths
  change) and the version line. Each changelog entry says "AI knowledge: updated / not needed".
- No endpoint, database or tracking change. AI knowledge: updated.

**Created:** bussunessLedger/blassistant/bAssistantPrompt_test.go
**Modified:** bussunessLedger/blassistant/bAssistantPrompt.go, bClaudeClient.go, bAskAssistant.go,
bAskShowcaseAssistant.go, env.example.NEW.txt

**Revert:** copy the `.bak` files back ("__" = "/"), delete the test file, rebuild and restart shao.

---

## 29. 2026-10-31 — Appointments never in orders; "Notify the customer" per tracking step

**Backups:** `.ai_backups/2026-10-31_before_step_notify/`.

**Description:**
- **Appointments:** new `blorder/bOrderAppointments.go` (+ test): `BCheckNoAppointments` refuses an order item whose
  product is an appointment (web_behavior BOOKING) with `ErrAppointmentInOrder` -> 400. Used by `BSaveOrderInTx`
  (manual orders and orders from approved web orders) and by the store order request
  (`bPublicShowcase/bPublicWebOrderRequest.go`); handlers `HPublicWebOrderRequest.go` and `HApproveWebOrder.go` map it to 400.
- **Step emails:** migration `sql/migration_step_notify_customer.sql` (run BEFORE restarting):
  `tbl_step_template_step.notify_customer BOOLEAN NOT NULL DEFAULT FALSE` (every step starts unchecked).
  `steptemplate/step_template.go` select / insert / update; `orders/order_item_tracking_query.go` `StepInfo.NotifyCustomer`.
  `blorderitemtracking/bOrderItemTracking.go` `itemStepEmail` (+ test): checked step -> step email (shows the order
  status); otherwise an ORDER status change (placed -> in progress -> partly done -> done) -> status email, always;
  an unchecked step with no status change -> no email. Cancelled orders: nothing. Everything else as before.

**Created:** bussunessLedger/blorder/bOrderAppointments.go, bOrderAppointments_test.go,
bussunessLedger/blorderitemtracking/bOrderItemTracking_test.go, sql/migration_step_notify_customer.sql
**Modified:** dataBasesLedger/steptemplate/step_template.go, dataBasesLedger/orders/order_item_tracking_query.go,
bussunessLedger/blorderitemtracking/bOrderItemTracking.go, bussunessLedger/blorder/bSaveOrder.go,
bussunessLedger/bPublicShowcase/bPublicWebOrderRequest.go, HandlerLedgers/hPublicShowCase/HPublicWebOrderRequest.go,
HandlerLedgers/weborderinshao/HApproveWebOrder.go

**Revert:** copy the `.bak` files back ("__" = "/"), delete the created files, run the migration REVERT line, rebuild shao.

---

## 28. 2026-10-31 — Company contact (WhatsApp + social link) for the store, customer service email (BCC), simpler product kinds

**Backups:** `.ai_backups/2026-10-31_before_company_contact_cs/`.

**Description:**
- **Migration** `sql/migration_company_contact_customer_service.sql` (run BEFORE restarting): `tbl_page_configuration`
  + `social_link`, `social_link_type` (FACEBOOK | INSTAGRAM | TIKTOK | OTHER, icon only), `customer_service_email`
  (all optional). Data: step 0 safety copies (`bak_products_20261031`, `bak_page_configuration_20261031`); A company
  WhatsApp from its products when empty; B products that had the cart get "available for web sales"; C optional;
  D old web behaviors -> MESSENGER_APP. Tested twice on PostgreSQL.
- `page_configuration.go`, handler input, `bPageConfiguration.go` (`normalizeSocialLink`, email check) save the 3 fields.
- Public page configuration (`HPublicPageConfiguration.go`) returns `contactWhatsapp`, `socialLink`, `socialLinkType`.
- Store item kind (`bPublicShowcase.go` `classifyKind`): BOOKING = appointment; PRODUCT (cart) = available for web
  sales; otherwise SERVICE (only "Contact us").
- Emails: `shared/email_notifier.go` `Bcc` (RCPT only, never in headers, `cleanBcc`); every customer email gets a
  blind copy to the customer service email; Reply-To = customer service email (else notification email); log rows
  audience `CUSTOMER_SERVICE`. `bCompanyCopy.go` `companyCopyTargets`: no duplicate "[Copy]" when the customer service
  address already got the blind copy; when the customer email was not sent, customer service gets the copy.
- Tests: `TestCleanBcc`, `TestCompanyCopyTargets`; kind rule checked.

**Created:** sql/migration_company_contact_customer_service.sql
**Modified:** dataBasesLedger/pageconfiguration/page_configuration.go, HandlerLedgers/pageconfigurationinshao/pageConfigurationHandlerCommon.go,
bussunessLedger/blpageconfiguration/bPageConfiguration.go, HandlerLedgers/hPublicShowCase/HPublicPageConfiguration.go,
bussunessLedger/bPublicShowcase/bPublicShowcase.go, shared/email_notifier.go, shared/email_notifier_test.go,
bussunessLedger/blnotifications/bNotifyCommon.go, bCompanyCopy.go, bCompanyCopy_test.go

**Revert:** copy the `.bak` files back ("__" = "/"), run the REVERT lines of the migration, rebuild shao.

---

## 27. 2026-10-31 — Domains from .env (aleph-k.com, aleph-k.online) + HTML emails with logos

**Backups:** `.ai_backups/2026-10-31_before_domains_html_email/`.

**Description:**
- **Domains:** new `shared/allowed_domains.go` (+ test). `ALLOWED_DOMAINS=aleph-k.com,aleph-k.online` in `.env` = each
  domain and all its subdomains, http and https. CORS accepts them in every environment (`router/cors_config.go`,
  `router/router.go`); a line without `http(s)://` in `cors_allowed_origins.txt` / `CORS_ALLOWED_ORIGINS` is a domain.
  CAPTCHA (`blcaptcha`): `TURNSTILE_ALLOWED_HOSTNAMES` accepts `*.domain`; empty + `APP_ENV=production` = only
  `ALLOWED_DOMAINS`. `SHOWCASE_BASE_URL` in `.env` wins over `tbl_configuration` (email links, share page).
- **HTML emails:** `shared/email_notifier.go` sends text + HTML (multipart/alternative, quoted-printable) when
  `HTMLBody` is set. New `blnotifications/bEmailHTML.go` (+ test): company logo on top (https only, else the name),
  body with clickable links (all escaped), and with the AlephK general email a footer "Sent by AlephK on behalf of
  <company>" + AlephK logo (`ALEPHK_LOGO_URL`, empty = text). Sender name "Kire via AlephK" / "Kire vía AlephK".
- Docs: `env.example.NEW.txt`, `cors_allowed_origins.example.txt`, `DEPLOY_README.md` (domains checklist, cookie note).

**Created:** shared/allowed_domains.go, shared/allowed_domains_test.go, bussunessLedger/blnotifications/bEmailHTML.go,
bussunessLedger/blnotifications/bEmailHTML_test.go
**Modified:** shared/email_notifier.go, shared/email_notifier_test.go, router/router.go, router/cors_config.go,
bussunessLedger/blcaptcha/bCaptchaVerify.go, bussunessLedger/blnotifications/bNotifyCommon.go, bNotifyTexts.go,
bussunessLedger/blpageconfiguration/bSharePage.go, env.example.NEW.txt, cors_allowed_origins.example.txt, DEPLOY_README.md

**Revert:** copy the `.bak` files back ("__" = "/"), delete the created files, rebuild shao.

---

## 26. 2026-10-30 — Company copy ("[Copy]") of every customer email

**Backups:** `.ai_backups/2026-10-30_before_company_copy/`.

**Description:** for the company's double check, every customer email (appointments, orders, web orders: created,
approved, rejected, status, modified, updated, tracking steps…) is also sent to the company as "[Copy] <subject>" /
"[Copia] …", same text, with a first line "Sent to the customer: x" or "NOT sent to the customer: reason" (also when
the customer has no email). Reply-To = the customer.
- Where: company's own SMTP → Company page notification email (empty → the company's own sender address);
  AlephK general email → ONLY the notification email, never the AlephK account (SMTP_FROM / SMTP_USER); empty → no copy
  (logged SKIPPED). No usable SMTP → no copy.
- No second copy for events that already send their own company email (appointment requested / customer cancelled /
  reschedule requested, web order created / customer cancelled, cancellation requests).
- Logged in tbl_notification_log with audience COMPANY_COPY; not counted in the customer daily limit.
- Code: new `blnotifications/bCompanyCopy.go` (+ test); `bNotifyCommon.go`: `deliver` → `sendAndLog` + copy;
  `bNotifyTexts.go`: EN/ES copy texts. No database change.

**Modified:** bussunessLedger/blnotifications/bNotifyCommon.go, bNotifyTexts.go
**Created:** bussunessLedger/blnotifications/bCompanyCopy.go, bCompanyCopy_test.go

**Revert:** copy the `.bak` files back, delete bCompanyCopy.go and its test, rebuild shao.

---

## 25. 2026-10-30 — New orders start as PLACED (fix "violates check constraint ck_order_status_valid")

**Backups:** `.ai_backups/2026-10-30_before_order_status_placed/`.

**Description:** creating an order failed with `pq: new row for relation "tbl_order" violates check constraint
"ck_order_status_valid"`: the order screen and the web order approval sent `SCHEDULED`, which is an APPOINTMENT status.
Order statuses are PLACED -> IN_PROGRESS -> PARTIAL_DONE -> DONE, or CANCELLED. Now:
- `blorder/bSaveOrder.go`: `OrderStatusPlaced` + `resolveOrderStatus`: a new order is always saved as PLACED (whatever
  the screen sends); an update keeps the current status (status changes come from the item tracking). Notification
  default status PLACED.
- `blweborder/bApproveWebOrder.go`: the order created from an approved web order is PLACED (was SCHEDULED, same error).
- `orders/order_insert.go`, `order_update.go`: SQL default for an empty status 'SCHEDULED' -> 'PLACED'.
- New `blorder/bSaveOrder_status_test.go`. No database change.

**Modified:** bussunessLedger/blorder/bSaveOrder.go, bussunessLedger/blweborder/bApproveWebOrder.go,
dataBasesLedger/orders/order_insert.go, dataBasesLedger/orders/order_update.go
**Created:** bussunessLedger/blorder/bSaveOrder_status_test.go

**Revert:** copy the `.bak` files back, delete the test file, rebuild shao.

---

## 24. 2026-10-30 — AlephK general email only when the company opts in + customer email lookup

**Backups:** `.ai_backups/2026-10-30_before_default_smtp_checkbox/`.

**Description:**
- **Opt-in:** new column `tbl_page_configuration.use_default_smtp` (**run `sql/migration_default_smtp_option.sql`
  before starting shao**). Company page saves it (`useDefaultSmtp`). Rule: company SMTP if it has one; otherwise the
  AlephK default SMTP (.env `SMTP_*`) **only when `use_default_smtp` is TRUE**; otherwise nothing is sent (SKIPPED:
  "the AlephK general email is not enabled in Company page").
- **Customer email lookup** (`blnotifications/bRecipients.go`, `dataBasesLedger/notifications/recipient_query.go`):
  appointment: typed → linked customer (`tbl_customers`); order: typed → linked customer → its store request →
  registered user of that request (usersAK `usfirebasedata`) → customer id as registered user; web order: typed →
  registered user. No valid email = the customer email is skipped (one SKIPPED row, no mail server used). Used by every
  customer email, including cancellation requests.
- Tests: `bRecipients_test.go`; integration run on PostgreSQL (shao + usersAK): opt-in on/off, all lookups, migration
  twice, Company page saves the flag.

**Modified:** dataBasesLedger/pageconfiguration/page_configuration.go,
HandlerLedgers/pageconfigurationinshao/pageConfigurationHandlerCommon.go,
bussunessLedger/blpageconfiguration/bPageConfiguration.go, bussunessLedger/blnotifications/bNotifyCommon.go,
bNotifyEvents.go, bNotifyUpdates.go, bNotifyCancelRequest.go, env.example.NEW.txt

**New:** sql/migration_default_smtp_option.sql, dataBasesLedger/notifications/recipient_query.go,
bussunessLedger/blnotifications/bRecipients.go, bRecipients_test.go

**Revert:** copy the `.bak` files back, delete the new Go files, run the REVERT line of the migration, rebuild and
restart shao (and revert p3 #34).

---

## 23. 2026-10-30 — Customer emails for every change + AlephK default SMTP when the company has none

**Backups:** `.ai_backups/2026-10-30_before_notifications_all_events/`.

**Description:**
- **Default SMTP:** `loadCompany` uses the company SMTP (tbl_company_smtp); when it is not configured, disabled or
  cannot be read, it uses the **AlephK default SMTP** from the environment: `SMTP_HOST`, `SMTP_PORT` (587),
  `SMTP_SECURITY` (STARTTLS / SSL, 465 = SSL), `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM` (default SMTP_USER),
  `SMTP_FROM_NAME` (default AlephK; sender shown as "Company (AlephK)", reply-to = company notification email).
  Without both, the email is SKIPPED with the reason. `main.go` now loads `.env` when present (never overwrites real
  environment variables). `env.example.NEW.txt` documents the variables.
- **New emails:** appointment **service / date / time changed** (`MODIFIED`, shows the previous one) and a status
  changed from the edit form (Done / Cancelled / Scheduled again / Approved / Rejected); order **details changed**
  (`UPDATED`: items, total, delivery date, status; only when they really change and the status did not);
  order item moved to another **tracking step** (`ITEM_STEP`: product, step name, step status, order status — one
  email instead of a separate status email); **new tracking stage** of an order (`TRACKING_UPDATE`).
  Existing emails unchanged (requests, approvals, rejections, created, status changes, customer cancellations).
- An email that cannot be prepared (record not found) is now also written to `tbl_notification_log` as FAILED.
- New `blnotifications/bNotifyUpdates.go` (+ test), EN/ES texts (usted). Tests: change detection, update event
  choice, default SMTP from env. No DB change.

**Modified:** main.go, env.example.NEW.txt, bussunessLedger/blnotifications/bNotifyCommon.go, bNotifyEvents.go,
bNotifyTexts.go, bussunessLedger/blappontment/bSaveAppointment.go, bussunessLedger/blorder/bSaveOrder.go,
bussunessLedger/blorderitemtracking/bOrderItemTracking.go, bussunessLedger/blordertracking/bOrderTracking.go

**New:** bussunessLedger/blnotifications/bNotifyUpdates.go, bNotifyUpdates_test.go,
bussunessLedger/blappontment/bSaveAppointment_test.go

**Revert:** copy the `.bak` files back, delete the new files, rebuild and restart shao.

---

## 22. 2026-10-30 — Processes: delete when unused, disable / enable when used

**Backups:** `.ai_backups/2026-10-30_before_step_template_delete_disable/`.

**Description:**
- New `DELETE /companies/{companyId}/step-templates/{stepTemplateId}` (`HDeleteStepTemplate`): deletes the process and its
  steps in one transaction **only when no product and no order item uses it**; otherwise 409 "The process is used by
  products or orders… Disable it instead". Tracking `DeleteStepTemplate` / `DeleteStepTemplateBlocked`.
- New `PUT /companies/{companyId}/step-templates/{stepTemplateId}/active` body `{ "active": true|false }`
  (`HSetStepTemplateActive`). Tracking `DisableStepTemplate` / `EnableStepTemplate`. A disabled process is not listed
  for products; products and orders that already use it keep working.
- `GET …/step-templates/{id}` now returns `usage: { products, orderItems }` (products with it; order items whose
  current step or history is one of its steps).
- `GET …/step-templates/search?includeInactive=true` also lists disabled processes (maintenance screen); without it,
  only active ones (product form), as before.
- Saving keeps the process enabled / disabled (a new one is enabled); only the Enable / Disable call changes it.
- Same permission (`NAV_TRACKING_MAINTENANCE_ACCESS`) and plan rule as saving. Integration-tested on PostgreSQL
  (usage counts, delete refused when used by a product or an order, delete of an unused one, disabled hidden from the
  product list, save keeps it disabled, enable). No DB change.

**Modified:** router/router.go, dataBasesLedger/steptemplate/step_template.go (SearchFiltered, SelectUsageWith,
SetActiveWith, DeleteTemplateWith), bussunessLedger/blsteeptemplate/bStepTemplate.go,
HandlerLedgers/steptemplateinshao/HSearchStepTemplates.go, stepTemplateHandlerCommon.go

**New:** HandlerLedgers/steptemplateinshao/HDeleteStepTemplate.go, HSetStepTemplateActive.go

**Revert:** copy the `.bak` files back, delete the 2 new handlers, rebuild and restart shao.

---

## 21. 2026-10-30 — Processes (step templates): stay active, save in one transaction, keep step ids

**Backups:** `.ai_backups/2026-10-30_before_step_template_save/`.

**Description:** QA: saving a process with several steps "saved nothing". Causes found:
1. `BSaveStepTemplate` updated an existing process (same internal code, or opened from the search) with
   `active = false` (the form does not send `active`, so Go used false). The process vanished from the search and
   from the product's Process list, and every later save with that code kept it hidden.
2. Steps were deleted and re-inserted **without a transaction**: any failed insert left the process without steps.
3. Re-inserting created **new step ids**; `tbl_order_item.current_step_id` and `tbl_order_item_step_history`
   reference them with `ON DELETE SET NULL`, so every save erased the current step of existing orders.
Now: a saved process is always active; header + steps run in ONE transaction (`dbManager.WithTx`); existing steps are
updated in place by position (same ids), new ones inserted, leftovers deleted only when no order uses them, otherwise
409 "The step … is used by existing orders…" and nothing changes; the done flag is cleared first so moving it never
hits `step_template_one_done_uq`; at least one step is required (400). Tests: `bStepTemplate_test.go` (cleanSteps,
merge plan) + an integration run on a real PostgreSQL (1 → 6 → 3 steps, order keeps its step, refused delete rolls
back). No DB change. Existing hidden processes: `UPDATE public.tbl_step_template SET active = TRUE WHERE active = FALSE;`

**Modified:** dataBasesLedger/steptemplate/step_template.go (…With(Executor) variants, UpdateWith, ClearDoneStepWith,
StepInUseWith, DeleteStepWith), bussunessLedger/blsteeptemplate/bStepTemplate.go,
HandlerLedgers/steptemplateinshao/stepTemplateHandlerCommon.go (409 for a step in use)

**New:** bussunessLedger/blsteeptemplate/bStepTemplate_test.go

**Revert:** copy the `.bak` files back, delete the test file, rebuild and restart shao.

---

## 20. 2026-10-29 — Basic company without a plan (name, logo, header) + required before a plan request

**Backups:** `.ai_backups/2026-10-29_before_basic_company/`.

**Description:** a new company had no plan, so it was read-only and could not even save its Company page, and a plan
was needed to set it up (circular). Now:
- `PUT /companies/{companyId}/page-configuration` (Company page save) works **without** a plan. The company SMTP
  (`PUT .../smtp-configuration`) and every other write stay read-only (403 `PLAN_REQUIRED`).
- Logo and header uploads: `POST .../storage/check` accepts an optional `kind`; with `PAGE_IMAGE` a company without
  a plan may upload images of at most 5 MB, total storage at most 20 MB. `POST .../storage/files` no longer has the
  `RequireActivePlan` middleware: `BRegisterFile` checks the plan itself and lets only `PAGE_IMAGE` files inside
  `companies/{id}/page-configuration/` through without a plan (product images still get `PLAN_REQUIRED`).
- New `bussunessLedger/blplan/bBasicCompany.go`: the basic company = name + logo + header in
  `tbl_page_configuration`. `POST .../plan-requests` answers **409** with `result: { code: "BASIC_COMPANY_REQUIRED",
  missing: ["logo", ...] }` until the three are set (tracking `CreatePlanRequestBasicCompanyMissing`).
  `GET .../plan` returns `basicCompanyMissing` (tracked in `SelectCompanyPlan`). Test `bBasicCompany_test.go`.
No database change.

**Modified:** router/router.go, bussunessLedger/blplan/bPlanRequest.go, bussunessLedger/blstorage/bStorage.go,
HandlerLedgers/planinshao/HCreatePlanRequest.go, HandlerLedgers/planinshao/HSelectCompanyPlan.go,
HandlerLedgers/storageinshao/HCheckStorage.go

**New:** bussunessLedger/blplan/bBasicCompany.go, bussunessLedger/blplan/bBasicCompany_test.go

**Revert:** copy the `.bak` files back (README lists the paths), delete the 2 new files, rebuild and restart shao.

---

## 19. 2026-10-28 — Database password removed from the startup log

**Backups:** `.ai_backups/2026-10-28_before_no_password_log/`.

**Description:** `dataBasesLedger/dbManager/connection.go` printed host, port, user, **password** and database when
connecting; on a server this goes to the system logs. The password was removed from that line (the rest is unchanged).

**Modified:** dataBasesLedger/dbManager/connection.go

**Revert:** copy the `.bak` file back and rebuild shao.

---
