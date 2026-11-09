-- =====================================================================
-- AlephK — company locations ("Sedes") for appointments (2026-11-07).
-- Run in the SHAO database BEFORE starting the new shao version (shao #34).
-- Safe to run twice (IF NOT EXISTS everywhere). Run it as a script
-- (DBeaver: Alt+X, or psql -v ON_ERROR_STOP=1 -f).
--
-- It changes NO current data: every schedule, slot and appointment keeps
-- "no location" (NULL) and works exactly as today. To give your current
-- data a location, run sql/data_assign_locations.sql AFTER this one.
--
--   1) tbl_company_location         the locations of each company
--   2) location_id (optional)       on recurring ranges, manual slots and
--                                   appointments
--   3) tbl_company_booking_settings "we can attend in more than one
--                                   location at the same time" (off)
-- =====================================================================

-- ---------------------------------------------------------------------
-- 1) Locations of each company (name, address, details, map point).
CREATE TABLE IF NOT EXISTS public.tbl_company_location (
    location_id  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id   VARCHAR(100) NOT NULL,
    name         VARCHAR(120) NOT NULL,
    address      VARCHAR(300),
    details      VARCHAR(500),            -- floor, office number, parking...
    latitude     NUMERIC(9,6),            -- map point (optional)
    longitude    NUMERIC(9,6),
    active       BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT ck_company_location_name CHECK (btrim(name) <> ''),
    CONSTRAINT ck_company_location_lat  CHECK (latitude  IS NULL OR latitude  BETWEEN -90  AND 90),
    CONSTRAINT ck_company_location_lng  CHECK (longitude IS NULL OR longitude BETWEEN -180 AND 180),
    CONSTRAINT ck_company_location_point CHECK ((latitude IS NULL) = (longitude IS NULL))
);

-- The name is unique inside a company (upper/lower case and spaces ignored).
CREATE UNIQUE INDEX IF NOT EXISTS ux_company_location_name
    ON public.tbl_company_location (company_id, lower(btrim(name)));

CREATE INDEX IF NOT EXISTS ix_company_location_company
    ON public.tbl_company_location (company_id, active);

-- ---------------------------------------------------------------------
-- 2) Optional location on schedules, slots and appointments.
--    NULL = no location (today's behaviour).
ALTER TABLE public.tbl_booking_recurring_range ADD COLUMN IF NOT EXISTS location_id UUID;
ALTER TABLE public.tbl_booking_manual_slot     ADD COLUMN IF NOT EXISTS location_id UUID;
ALTER TABLE public.tbl_appointment             ADD COLUMN IF NOT EXISTS location_id UUID;

-- A location that is used can not be deleted by mistake (shao disables it
-- instead). Added only once.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_recurring_range_location') THEN
        ALTER TABLE public.tbl_booking_recurring_range
            ADD CONSTRAINT fk_recurring_range_location
            FOREIGN KEY (location_id) REFERENCES public.tbl_company_location (location_id);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_manual_slot_location') THEN
        ALTER TABLE public.tbl_booking_manual_slot
            ADD CONSTRAINT fk_manual_slot_location
            FOREIGN KEY (location_id) REFERENCES public.tbl_company_location (location_id);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_appointment_location') THEN
        ALTER TABLE public.tbl_appointment
            ADD CONSTRAINT fk_appointment_location
            FOREIGN KEY (location_id) REFERENCES public.tbl_company_location (location_id);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS ix_recurring_range_location
    ON public.tbl_booking_recurring_range (location_id) WHERE location_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS ix_manual_slot_location
    ON public.tbl_booking_manual_slot (location_id) WHERE location_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS ix_appointment_location
    ON public.tbl_appointment (location_id) WHERE location_id IS NOT NULL;

-- ---------------------------------------------------------------------
-- 3) Company booking settings. No row = default (FALSE): the company is in
--    ONE location at a time, so shao refuses schedules with the same hours
--    in two different locations.
CREATE TABLE IF NOT EXISTS public.tbl_company_booking_settings (
    company_id          VARCHAR(100) PRIMARY KEY,
    locations_same_time BOOLEAN      NOT NULL DEFAULT FALSE,
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------------
-- Checks (all must answer TRUE):
--   SELECT to_regclass('public.tbl_company_location') IS NOT NULL AS locations,
--          to_regclass('public.tbl_company_booking_settings') IS NOT NULL AS settings,
--          (SELECT count(*) FROM information_schema.columns
--            WHERE table_schema = 'public' AND column_name = 'location_id'
--              AND table_name IN ('tbl_booking_recurring_range','tbl_booking_manual_slot','tbl_appointment')) = 3 AS columns;
--
-- UNDO (only if you go back to the previous shao version; the locations
-- and the location of every appointment are lost):
--   ALTER TABLE public.tbl_appointment             DROP COLUMN IF EXISTS location_id;
--   ALTER TABLE public.tbl_booking_manual_slot     DROP COLUMN IF EXISTS location_id;
--   ALTER TABLE public.tbl_booking_recurring_range DROP COLUMN IF EXISTS location_id;
--   DROP TABLE IF EXISTS public.tbl_company_booking_settings;
--   DROP TABLE IF EXISTS public.tbl_company_location;
-- =====================================================================
