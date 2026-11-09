-- =====================================================================
-- AlephK — give a location to the CURRENT schedules and appointments
-- (shao database, 2026-11-07). OPTIONAL: run it only for a company that
-- wants locations. Run it AFTER sql/migration_company_locations.sql and
-- sql/migration_company_locations_options.sql (2026-11-08): STEP 1 and
-- STEP 2 also turn on "We use locations" for that company.
--
-- Works in DBeaver (run each step with Ctrl+Enter, or the whole file with
-- Alt+X) and in psql (psql -v ON_ERROR_STOP=1 -f ...). Change the values in
-- the "-- EDIT" lines first, and set v_run := TRUE in the step you want
-- (STEP 1 OR STEP 2): with v_run = FALSE a step changes nothing, so running
-- the whole file by mistake is harmless. Every step only fills EMPTY
-- locations: what already has a location is never changed. Safe to run twice.
--
--   STEP 0  see the companies and what they have
--   STEP 1  ONE location for everything of a company (one office)
--   STEP 2  several locations by day of the week (e.g. Monday/Wednesday in
--           Heredia, Tuesday/Thursday in San Pedro) — run once per location
--   STEP 3  check: hours in two locations at the same time
--   STEP 4  check: what each location has
--   UNDO    remove the locations given by these steps
-- =====================================================================

-- ---------------------------------------------------------------------
-- STEP 0 — the companies and their current schedules / appointments.
SELECT pc.company_id,
       pc.display_name,
       (SELECT count(*) FROM public.tbl_booking_recurring_range r WHERE r.company_id::text = pc.company_id::text) AS weekly_hours,
       (SELECT count(*) FROM public.tbl_booking_manual_slot s     WHERE s.company_id::text = pc.company_id::text) AS date_slots,
       (SELECT count(*) FROM public.tbl_appointment a             WHERE a.company_id::text = pc.company_id::text) AS appointments,
       (SELECT count(*) FROM public.tbl_company_location l        WHERE l.company_id = pc.company_id::text)      AS locations
FROM public.tbl_page_configuration pc
ORDER BY pc.display_name;


-- ---------------------------------------------------------------------
-- STEP 1 — ONE location for everything of ONE company.
-- Creates the location (or uses the one with that name) and puts it on
-- every weekly hour, date slot and appointment of the company that has no
-- location yet.
DO $$
DECLARE
    v_run         boolean := FALSE;                  -- EDIT: TRUE to run this step
    v_company     text    := 'Kire';                 -- EDIT: part of the company name (must match ONE company)
    v_name        text    := 'Oficina principal';    -- EDIT: location name
    v_address     text    := NULL;                   -- EDIT: e.g. '100 m norte del parque, Heredia' (or NULL)
    v_details     text    := NULL;                   -- EDIT: e.g. 'Piso 2, oficina 5' (or NULL)
    v_latitude    numeric := NULL;                   -- EDIT: e.g. 9.998 (or NULL)
    v_longitude   numeric := NULL;                   -- EDIT: e.g. -84.117 (or NULL)
    v_only_future boolean := TRUE;                   -- EDIT: TRUE = appointments from today; FALSE = all (history too)

    v_company_id  text;
    v_matches     int;
    v_zone        text;
    v_today       date;
    v_location_id uuid;
    n_ranges      int;
    n_slots       int;
    n_appts       int;
BEGIN
    IF NOT v_run THEN
        RAISE NOTICE 'Step skipped: set v_run := TRUE to run it.';
        RETURN;
    END IF;
    SELECT count(*), min(company_id::text) INTO v_matches, v_company_id
    FROM public.tbl_page_configuration
    WHERE display_name ILIKE '%' || v_company || '%';
    IF v_matches <> 1 THEN
        RAISE EXCEPTION 'The name "%" matches % companies; write a part that matches exactly one (see STEP 0).', v_company, v_matches;
    END IF;

    -- "Today" of the company (its time zone, default Costa Rica).
    SELECT COALESCE(NULLIF(time_zone, ''), 'America/Costa_Rica') INTO v_zone
    FROM public.tbl_page_configuration WHERE company_id::text = v_company_id;
    v_today := (now() AT TIME ZONE COALESCE(v_zone, 'America/Costa_Rica'))::date;

    SELECT location_id INTO v_location_id
    FROM public.tbl_company_location
    WHERE company_id = v_company_id AND lower(btrim(name)) = lower(btrim(v_name));
    IF v_location_id IS NULL THEN
        INSERT INTO public.tbl_company_location (company_id, name, address, details, latitude, longitude)
        VALUES (v_company_id, btrim(v_name), NULLIF(btrim(v_address), ''), NULLIF(btrim(v_details), ''), v_latitude, v_longitude)
        RETURNING location_id INTO v_location_id;
        RAISE NOTICE 'Location "%" created.', v_name;
    ELSE
        RAISE NOTICE 'Location "%" already exists: it is used as it is.', v_name;
    END IF;

    UPDATE public.tbl_booking_recurring_range
       SET location_id = v_location_id
     WHERE company_id::text = v_company_id AND location_id IS NULL;
    GET DIAGNOSTICS n_ranges = ROW_COUNT;

    UPDATE public.tbl_booking_manual_slot
       SET location_id = v_location_id
     WHERE company_id::text = v_company_id AND location_id IS NULL
       AND (NOT v_only_future OR slot_date >= v_today);
    GET DIAGNOSTICS n_slots = ROW_COUNT;

    UPDATE public.tbl_appointment
       SET location_id = v_location_id
     WHERE company_id::text = v_company_id AND location_id IS NULL
       AND (NOT v_only_future OR appointment_date >= v_today);
    GET DIAGNOSTICS n_appts = ROW_COUNT;

    -- "We use locations" ON for this company (2026-11-08).
    INSERT INTO public.tbl_company_booking_settings (company_id, locations_enabled, updated_at)
    VALUES (v_company_id, TRUE, now())
    ON CONFLICT (company_id) DO UPDATE SET locations_enabled = TRUE, updated_at = now();

    RAISE NOTICE 'Company %: location "%" given to % weekly hours, % date slots and % appointments; "We use locations" is ON.',
        v_company_id, v_name, n_ranges, n_slots, n_appts;
END $$;


-- ---------------------------------------------------------------------
-- STEP 2 — several locations by DAY OF THE WEEK (instead of STEP 1).
-- Run this block once per location, changing the name and the days.
-- Example: first 'Heredia' with MONDAY, WEDNESDAY; then 'San Pedro' with
-- TUESDAY, THURSDAY. Days: MONDAY TUESDAY WEDNESDAY THURSDAY FRIDAY
-- SATURDAY SUNDAY. Only hours / appointments with NO location are changed.
DO $$
DECLARE
    v_run         boolean := FALSE;                           -- EDIT: TRUE to run this step
    v_company     text    := 'Kire';                          -- EDIT: part of the company name
    v_name        text    := 'Heredia';                       -- EDIT: location name
    v_address     text    := NULL;                            -- EDIT: address (or NULL)
    v_days        text[]  := ARRAY['MONDAY', 'WEDNESDAY'];    -- EDIT: days of this location
    v_only_future boolean := TRUE;                            -- EDIT: TRUE = appointments from today; FALSE = all

    v_company_id  text;
    v_matches     int;
    v_zone        text;
    v_today       date;
    v_location_id uuid;
    n_ranges      int;
    n_slots       int;
    n_appts       int;
BEGIN
    IF NOT v_run THEN
        RAISE NOTICE 'Step skipped: set v_run := TRUE to run it.';
        RETURN;
    END IF;
    SELECT count(*), min(company_id::text) INTO v_matches, v_company_id
    FROM public.tbl_page_configuration
    WHERE display_name ILIKE '%' || v_company || '%';
    IF v_matches <> 1 THEN
        RAISE EXCEPTION 'The name "%" matches % companies; write a part that matches exactly one (see STEP 0).', v_company, v_matches;
    END IF;
    SELECT array_agg(upper(btrim(d))) INTO v_days FROM unnest(v_days) AS d;
    IF EXISTS (SELECT 1 FROM unnest(v_days) AS d
               WHERE d NOT IN ('MONDAY','TUESDAY','WEDNESDAY','THURSDAY','FRIDAY','SATURDAY','SUNDAY')) THEN
        RAISE EXCEPTION 'Days must be MONDAY ... SUNDAY: %', v_days;
    END IF;

    SELECT COALESCE(NULLIF(time_zone, ''), 'America/Costa_Rica') INTO v_zone
    FROM public.tbl_page_configuration WHERE company_id::text = v_company_id;
    v_today := (now() AT TIME ZONE COALESCE(v_zone, 'America/Costa_Rica'))::date;

    SELECT location_id INTO v_location_id
    FROM public.tbl_company_location
    WHERE company_id = v_company_id AND lower(btrim(name)) = lower(btrim(v_name));
    IF v_location_id IS NULL THEN
        INSERT INTO public.tbl_company_location (company_id, name, address)
        VALUES (v_company_id, btrim(v_name), NULLIF(btrim(v_address), ''))
        RETURNING location_id INTO v_location_id;
        RAISE NOTICE 'Location "%" created.', v_name;
    END IF;

    UPDATE public.tbl_booking_recurring_range
       SET location_id = v_location_id
     WHERE company_id::text = v_company_id AND location_id IS NULL
       AND upper(day_of_week::text) = ANY (v_days);
    GET DIAGNOSTICS n_ranges = ROW_COUNT;

    -- to_char(..., 'FMDAY') gives MONDAY ... SUNDAY (English).
    UPDATE public.tbl_booking_manual_slot
       SET location_id = v_location_id
     WHERE company_id::text = v_company_id AND location_id IS NULL
       AND to_char(slot_date, 'FMDAY') = ANY (v_days)
       AND (NOT v_only_future OR slot_date >= v_today);
    GET DIAGNOSTICS n_slots = ROW_COUNT;

    UPDATE public.tbl_appointment
       SET location_id = v_location_id
     WHERE company_id::text = v_company_id AND location_id IS NULL
       AND to_char(appointment_date, 'FMDAY') = ANY (v_days)
       AND (NOT v_only_future OR appointment_date >= v_today);
    GET DIAGNOSTICS n_appts = ROW_COUNT;

    -- "We use locations" ON for this company (2026-11-08).
    INSERT INTO public.tbl_company_booking_settings (company_id, locations_enabled, updated_at)
    VALUES (v_company_id, TRUE, now())
    ON CONFLICT (company_id) DO UPDATE SET locations_enabled = TRUE, updated_at = now();

    RAISE NOTICE 'Company %: location "%" (%) given to % weekly hours, % date slots and % appointments; "We use locations" is ON.',
        v_company_id, v_name, array_to_string(v_days, ', '), n_ranges, n_slots, n_appts;
END $$;


-- ---------------------------------------------------------------------
-- STEP 3 — check: the same weekly hours in two DIFFERENT locations.
-- While the company uses locations and the switch "We can attend in more
-- than one location at the same time" is OFF (the default) this must return
-- NO rows. If it returns
-- rows: change those hours in Booking schedules, or turn the switch ON in
-- Locations. (Only active schedules in weekly mode.)
SELECT pc.display_name                                          AS company,
       a.day_of_week,
       to_char(a.start_time, 'HH24:MI') || '-' || to_char(a.end_time, 'HH24:MI') AS hours_1,
       la.name AS location_1, ca.name AS schedule_1,
       to_char(b.start_time, 'HH24:MI') || '-' || to_char(b.end_time, 'HH24:MI') AS hours_2,
       lb.name AS location_2, cb.name AS schedule_2
FROM public.tbl_booking_recurring_range a
JOIN public.tbl_booking_recurring_range b
  ON b.company_id::text = a.company_id::text
 AND b.day_of_week::text = a.day_of_week::text
 AND b.recurring_range_id::text > a.recurring_range_id::text
 AND a.start_time < b.end_time AND b.start_time < a.end_time
 AND a.location_id <> b.location_id
JOIN public.tbl_company_location la ON la.location_id = a.location_id AND la.active
JOIN public.tbl_company_location lb ON lb.location_id = b.location_id AND lb.active
JOIN public.tbl_booking_config ca ON ca.booking_config_id = a.booking_config_id AND ca.active AND upper(ca.availability_mode::text) <> 'MANUAL'
JOIN public.tbl_booking_config cb ON cb.booking_config_id = b.booking_config_id AND cb.active AND upper(cb.availability_mode::text) <> 'MANUAL'
LEFT JOIN public.tbl_page_configuration pc ON pc.company_id::text = a.company_id::text
LEFT JOIN public.tbl_company_booking_settings st ON st.company_id = a.company_id::text
WHERE a.active AND b.active
  AND COALESCE(st.locations_enabled, FALSE) = TRUE
  AND COALESCE(st.locations_same_time, FALSE) = FALSE
ORDER BY 1, 2, 3;


-- ---------------------------------------------------------------------
-- STEP 4 — check: what each location has now.
SELECT pc.display_name AS company,
       l.name          AS location,
       l.active,
       (SELECT count(*) FROM public.tbl_booking_recurring_range r WHERE r.location_id = l.location_id) AS weekly_hours,
       (SELECT count(*) FROM public.tbl_booking_manual_slot s     WHERE s.location_id = l.location_id) AS date_slots,
       (SELECT count(*) FROM public.tbl_appointment a             WHERE a.location_id = l.location_id) AS appointments
FROM public.tbl_company_location l
LEFT JOIN public.tbl_page_configuration pc ON pc.company_id::text = l.company_id
ORDER BY 1, 2;


-- ---------------------------------------------------------------------
-- UNDO (only if needed) — remove the locations of ONE company from its
-- hours and appointments (the locations themselves stay; delete them in
-- the Locations screen). Write the company id from STEP 0 and run:
--   UPDATE public.tbl_booking_recurring_range SET location_id = NULL WHERE company_id::text = 'COMPANY-ID';
--   UPDATE public.tbl_booking_manual_slot     SET location_id = NULL WHERE company_id::text = 'COMPANY-ID';
--   UPDATE public.tbl_appointment             SET location_id = NULL WHERE company_id::text = 'COMPANY-ID';
-- =====================================================================
