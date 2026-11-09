# AI Change Log — shao (AlephK Go backend)

Keeps the latest 15 changes made with Claude (oldest are removed). Newest first.
Earlier shao changes are listed in Claude's change log (memory) and in `.ai_backups/` READMEs.

---

## 43. 2026-11-08 — Customer service email in the store (for the "pending approval" pop-up)

**Backups:** `.ai_backups/2026-11-08_before_request_sent_popup/`.

**Description:** works with showcase #30.
- `GET /public/companies/{companyId}/page-configuration` returns `contactEmail` = the customer service email of the
  Company page, only when it is a valid address (new `blpageconfiguration.PublicContactEmail`, tested). It is already
  the Reply-To of every customer email.
- The manage page data (`bSelfService.go`) also returns `contactEmail`, `socialLink` and `socialLinkType`
  (`companyPublicInfo` now returns a small struct).
- AI knowledge: Aleph (`bAssistantPrompt.go`, version "2026-11-08 (shao #43)") and the store assistant
  (`bShowcasePrompt.go`: requests must be approved, check your email, contact the company).
Tests: build, vet, unit tests, PostgreSQL integration (valid / invalid / empty email; manage page contact).
No database script, no new endpoint (same tracking). AI knowledge: updated.

**New:** bussunessLedger/blpageconfiguration/bPublicContact.go, bPublicContact_test.go

**Modified:** HandlerLedgers/hPublicShowCase/HPublicPageConfiguration.go, bussunessLedger/bPublicShowcase/bSelfService.go,
bussunessLedger/blassistant/bAssistantPrompt.go, bShowcasePrompt.go

**Undo:** restore the `.bak` files from the backup folder, delete the two new files and rebuild.

---

## 42. 2026-11-08 — AI knowledge: store booking window, contact fields at the end

**Backups:** `.ai_backups/2026-11-08_before_booking_info_last/`.

**Description:** works with showcase #29. The store booking window line now says: date, times, then "Your choice"
and "Your information" at the end of the form, then Send. Version "2026-11-08 (shao #42)". No code or database
change. AI knowledge: updated.

**Modified:** bussunessLedger/blassistant/bAssistantPrompt.go

**Undo:** restore the `.bak` file from the backup folder.

---

## 41. 2026-11-08 — Change the location of a booking request or an appointment

**Backups:** `.ai_backups/2026-11-08_before_appointment_location/`.

**Description:** works with p3 #51. New endpoint
`PUT /companies/{companyId}/appointments/{appointmentId}/location`, body `{ "locationId": "<id>" }` (`""` = no location).
Same permissions as the appointment screens (NAV_APPOINTMENTS_ACCESS, NAV_APPROVAL_B_ACCESS, NAV_SELF_BOOKING), active
plan, tracking `SetAppointmentLocation`.
- `BSetAppointmentLocation`: a cancelled or rejected appointment can't change (400); the location must belong to the
  company and be active (a disabled location is accepted only if it's the current one); unknown location 400; not
  found 404.
- When "one location at a time" is on, the answer has `otherLocationCount` (appointments at that hour in another
  location) so the screen can warn. The save is not blocked.
- A SCHEDULED appointment whose location changed sends the customer the "appointment modified" email.
Tests: build, vet, unit tests, PostgreSQL integration `TestSetAppointmentLocation` (set, other-location count,
disabled, unknown, remove, not found, cancelled, HTTP 200/400/405). No database script.
AI knowledge: updated ("2026-11-08 (shao #41)").

**New:** dataBasesLedger/appointments/appointment_location_update.go,
bussunessLedger/blappontment/bAppointmentLocation.go, HandlerLedgers/appointmentInShao/HAppointmentLocation.go

**Modified:** router/router.go, bussunessLedger/blassistant/bAssistantPrompt.go

**Undo:** restore the `.bak` files from the backup folder, delete the three new files and rebuild.

---

## 40. 2026-11-08 — Past dates are history: never inserted again (fix of the false "same location" error)

**Backups:** `.ai_backups/2026-11-08_before_past_dates_preview/`.

**Description:** works with p3 #50. Saving a calendar schedule that had a past date (e.g. November 1) sent that date
again; shao keeps past dates (it only replaces dates from today on), so the database refused the duplicate and the
screen showed "two ranges… in the same location" although the day had one range. New `DropPastDates` in
`bScheduleRules.go` (called in `BSaveBooking` with the company's today): dates before today are ignored when saving
(they stay in the database as history). Tests `TestDropPastDates` and PostgreSQL integration (past date sent again ->
saved, not duplicated). No database script. AI knowledge: updated ("2026-11-08 (shao #40)").

**Modified:** bussunessLedger/bbooking/bScheduleRules.go, bScheduleRules_test.go, bSaveBooking.go,
bussunessLedger/blassistant/bAssistantPrompt.go

**Undo:** restore the `.bak` files from the backup folder; rebuild.

---

## 39. 2026-11-08 — Same hours in two locations (database rule), one mode per schedule, no partial saves

**Backups:** `.ai_backups/2026-11-08_before_schedule_steps/`.

**Description:** works with p3 #49. **Run `sql/migration_booking_overlap_by_location.sql` first** on every shao
database (your computer and the server).
- The script rebuilds the exclusion constraints of `tbl_booking_manual_slot` and `tbl_booking_recurring_range`
  (e.g. `ex_manual_no_overlap`, made before locations) with the location first: two DIFFERENT locations may overlap
  (the "same time" switch decides), same location / both without a fixed location still refused. The rest of each
  rule is kept; it prints the old rules (undo) and is safe to run twice. Tested on PostgreSQL 16.
- New `bussunessLedger/bbooking/bScheduleRules.go`: `KeepActiveMode` (a schedule keeps only the hours of its mode;
  saving deletes the other mode's hours — future dates; past dates stay as history; appointments untouched),
  `CheckSameLocationOverlaps` (same location at the same time refused BEFORE anything is deleted, so a refused save
  never loses the saved hours), `RangesOverlapError` (friendly text, 409) and `overlapOr` (a database exclusion error
  becomes the same friendly text instead of "pq: conflicting key value…").
- `DEPLOY_README.md`: row 18 + check query. Tests `TestKeepActiveMode`, `TestCheckSameLocationOverlaps`; PostgreSQL
  integration `TestScheduleRules` (two locations saved, same location refused with the saved dates kept, switch to
  weekly deletes the dates). `lt` tracking unchanged. AI knowledge: updated ("2026-11-08 (shao #39)").

**Created:** sql/migration_booking_overlap_by_location.sql, bussunessLedger/bbooking/bScheduleRules.go,
bussunessLedger/bbooking/bScheduleRules_test.go
**Modified:** bussunessLedger/bbooking/bSaveBooking.go, bussunessLedger/blassistant/bAssistantPrompt.go, DEPLOY_README.md

**Undo:** restore the `.bak` files, delete the created files; database only if needed: re-create the old rules
printed by step 0 of the script.

---

## 38. 2026-11-08 — "Customer chooses": places per location when "same time" is ON

**Backups:** `.ai_backups/2026-11-08_before_places_per_location/`.

**Description:** works with p3 #48. `bBookingSlots.go`: new `slotCopies()` counts the taken places of every slot.
A "Customer chooses" slot is offered in every active location: with "We can attend in more than one location at
the same time" ON, each location has the block's own places (e.g. 3 in Heredia and 3 in San Pedro, counted per
location); with it OFF the places are shared (the person can only be in one place). Fixed-location and no-location
slots work as before. `GetAvailableSlots` reads the booking settings once. Test `TestSlotCopiesPlaces`; PostgreSQL
integration test (switch ON -> 10:00 back in Heredia only; OFF -> shared again) passes. No database or endpoint
change; `lt` tracking unchanged. AI knowledge: updated ("2026-11-08 (shao #38)").

**Modified:** bussunessLedger/bPublicShowcase/bBookingSlots.go, bussunessLedger/bPublicShowcase/bSlotValidation_test.go,
bussunessLedger/blassistant/bAssistantPrompt.go

**Undo:** restore the `.bak` files from the backup folder; rebuild.

---

## 37. 2026-11-08 — AI knowledge: Booking schedules "Create your locations first" box

**Backups:** `.ai_backups/2026-11-08_before_locations_hint/`.

**Description:** works with p3 #47. `bAssistantPrompt.go`: Booking schedules explains the box shown while "We use
locations" is off (or no active location) with the button "Go to Locations"; version "2026-11-08 (shao #37)".
AI knowledge: updated. No other change.

**Modified:** bussunessLedger/blassistant/bAssistantPrompt.go

**Undo:** restore the `.bak` file from the backup folder.

---

## 36. 2026-11-08 — devtools/alephk-dev.sh: start everything for development with one command

**Backups:** `.ai_backups/2026-11-08_before_dev_start_script/`.

**Description:** new `devtools/alephk-dev.sh` (+ `devtools/README.md`) for your computer only: opens shao, usersAppAK,
p3 and showcase in VS Code (`code`, or `--one-window`), starts usersAppAK (8080) and shao (8081) with `go run .` in
their own terminal windows, waits until both answer (90 s), then starts p3 (3000) and showcase (3001) with `npm start`
in their own windows. `PORT=` in a project's `.env` wins (only that line is read). Commands `back`, `front`, `code`,
`status`, `stop`, options `--no-code`, `--dry-run`; a running program is not started twice; warns when PostgreSQL is
down; finds gnome-terminal / konsole / xfce4-terminal / mate-terminal / tilix / kitty / alacritty / xterm. Tested with
shellcheck and fake programs (start, skip running ones, status, stop). No code, database or server change.
AI knowledge: not needed (no screen or feature change).

**Created:** devtools/alephk-dev.sh, devtools/README.md

**Undo:** delete `devtools/`; restore AI_CHANGELOG.md from the backup.

---

## 35. 2026-11-08 — Locations options: "We use locations" switch + "Customer chooses"

**Backups:** `.ai_backups/2026-11-08_before_locations_options/`.

**Description:** works with p3 #46, showcase #27, usersAppAK #19. Run `sql/migration_company_locations_options.sql`
first (after `migration_company_locations.sql`) on every shao database (your computer and the server).
- **"We use locations"** (company switch, `tbl_company_booking_settings.locations_enabled`, default OFF; the
  migration turns it ON once for companies that already have a location). OFF = AlephK works as before locations:
  free times carry no location, disabled locations don't hide hours, the "one location at a time" rule is off,
  appointment capacity is not split by location, emails / self-service / My bookings show no location, a
  reschedule keeps the saved location, a store request with a `locationId` is accepted and the id ignored.
  `PUT /companies/{companyId}/booking-settings` now takes `{ locationsEnabled?, locationsSameTime? }` (a field not
  sent keeps its value); turning locations ON (or "same time" OFF) is refused with 409 `LOCATION_OVERLAP` and the
  list while hours clash in two locations.
- **"Customer chooses"** (`customer_chooses_location` on weekly ranges and date slots; location stays NULL, a check
  constraint keeps them apart): the free times are repeated in every active location (sorted by name), the places
  are shared (taken = all locations), the customer picks one and the appointment saves that location. In the
  "one location at a time" rule such a block clashes with any located or "customer chooses" block at the same hours.
- `sql/data_assign_locations.sql`: STEP 1/2 also turn "We use locations" ON; STEP 3 lists only companies with it ON.
  `DEPLOY_README.md`: new check-query row and start-order note.
- Tests: `TestClashCustomerChooses`, `TestMergeSettings`, `TestMatchSlotLocationsOff`, `TestExpandSlotCustomerChooses`;
  PostgreSQL 16 integration (off behaviour, refusal with a clash, expansion and shared places) and HTTP tests pass.
  `lt` tracking: `HBookingSettings` details `locationsEnabled`, `locationsSameTime`.
  AI knowledge: updated (`bAssistantPrompt.go` version "2026-11-08 (shao #35)", `bShowcasePrompt.go`).

**Created:** sql/migration_company_locations_options.sql
**Modified:** dataBasesLedger/locations/booking_settings.go, dataBasesLedger/booking/ (booking_recurring_range_insert/
select/update.go, booking_manual_slot_insert/select/update.go, booking_slots_query.go, booking_location_times.go),
bussunessLedger/bllocations/ (bOverlap.go, bSettingsAndMaps.go, bLocations_test.go), bussunessLedger/bPublicShowcase/
(bBookingSlots.go, bSlotValidation.go, bSlotValidation_test.go, bSelfService.go, bMyRequests.go),
bussunessLedger/bbooking/bSaveBooking.go, bussunessLedger/blappontment/bSaveAppointment.go,
bussunessLedger/blnotifications/ (bNotifyEvents.go, bNotifyUpdates.go, bNotifyCancelRequest.go),
bussunessLedger/blassistant/ (bAssistantPrompt.go, bShowcasePrompt.go), HandlerLedgers/locationsinshao/
(HBookingSettings.go, locationHandlerCommon.go), sql/data_assign_locations.sql, DEPLOY_README.md

**Undo:** restore the `.bak` files from the backup folder; the UNDO lines are at the end of the migration (only
needed if you go back to shao #34).

---

## 34. 2026-11-07 — Locations ("Sedes") for appointments

**Backups:** `.ai_backups/2026-11-07_before_company_locations/`.

**Description:** works with p3 #45, showcase #26 and usersAppAK #18.
- **Run first:** `sql/migration_company_locations.sql` (new `tbl_company_location` with name unique per company,
  address, details, map point, active; optional `location_id` (UUID, FK) on `tbl_booking_recurring_range`,
  `tbl_booking_manual_slot` and `tbl_appointment`; new `tbl_company_booking_settings.locations_same_time`, default
  FALSE). Safe to run twice; changes no current data; REVERT inside. Optional `sql/data_assign_locations.sql` gives a
  location to the current hours and appointments of ONE company (one location, or by day of the week).
- New layers: `dataBasesLedger/locations`, `bussunessLedger/bllocations` (+ tests), `HandlerLedgers/locationsinshao`
  with `lt` tracking. Routes: `GET/POST /companies/{id}/locations`, `GET/DELETE .../locations/{locationId}`,
  `PUT .../locations/{locationId}/active`, `GET/PUT /companies/{id}/booking-settings`. Reading: any member (the forms
  need the list). Changes: `NAV_LOCATIONS_ACCESS` or `NAV_BOOKING_ACCESS` (OWNER / MANAGER always) + active plan.
- A used location is never deleted: DELETE disables it. Disabling with future appointments is allowed (they keep
  their location); the answer says how many there are. Enabling again checks the clashes.
- **One location at a time** (switch OFF): saving a schedule is refused (409 `LOCATION_OVERLAP` + list) when its
  hours overlap hours of another location in ANY active schedule of the company (weekly vs weekly, date vs date, date
  vs its weekday). Turning the switch OFF is refused while such clashes exist. Hours with no location are not checked.
- Free slots carry the location (name, address, details, point, `mapsUrl`); hours of a disabled location are not
  offered. Capacity is counted per location (appointments with no location count everywhere, to be safe).
- Store request / self-service reschedule: `locationId` must match the chosen time; without it (older pages) the
  first free slot at that time is taken and its location is stored. The appointment keeps its location.
- Internal appointments: optional `locationId` (must be in the company; a disabled one only if it already had it;
  not sent = keep, "" = remove); new warning `otherLocationCount` (same time in another location, switch OFF).
  Search filter `locationId`. "My bookings" shows the location.
- Emails: "📍 Location: name, address" + "Open in Maps" link; a new location sends the "changed" email.
- Assistants: knowledge "2026-11-07 (shao #34)", new path `/shao/locations`; slots and appointments of the tools
  include the location.
- Tests: unit (overlaps, map links, validation, slot matching, email text) + PostgreSQL (UUID and text company ids)
  + HTTP handlers. `go build`, `go vet`, `go test` pass. AI knowledge: updated.

**Created:** sql/migration_company_locations.sql, sql/data_assign_locations.sql, dataBasesLedger/locations/ (6 files),
dataBasesLedger/booking/booking_location_times.go, bussunessLedger/bllocations/ (bLocations.go, bOverlap.go,
bSettingsAndMaps.go, bLocations_test.go), HandlerLedgers/locationsinshao/ (7 files)
**Modified:** router/router.go, dataBasesLedger/booking/ (recurring range insert/select/update, manual slot
insert/select/update, booking_slots_query.go), dataBasesLedger/appointments/ (insert, select, update, search,
public_token, customer_query), bussunessLedger/bbooking/bSaveBooking.go, blappontment/bSaveAppointment.go,
bPublicShowcase/ (bBookingSlots, bSlotValidation (+ test), bPublicAppointmentRequest, bPublicFormGuard, bSelfService,
bMyRequests), blnotifications/ (bNotifyEvents, bNotifyUpdates (+ test), bNotifyTexts, bNotifyCancelRequest),
blassistant/ (bAssistantPrompt, bAssistantTools, bAssistantInsightTools, bShowcasePrompt),
HandlerLedgers/hbooking/Hsavebooking.go, HandlerLedgers/appointmentInShao/HSearchAppointments.go, DEPLOY_README.md

**Revert:** copy the `.bak` files back ("__" = "/"), delete the created files and folders, rebuild; then (only if
needed) run the UNDO lines at the end of the migration.

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
