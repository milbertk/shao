-- =====================================================================
-- AlephK — locations options (2026-11-08, shao #35).
-- Run in the SHAO database AFTER migration_company_locations.sql and BEFORE
-- starting the new shao. Safe to run twice. Run it on every shao database
-- (your computer and the server).
--
--   1) "We use locations" (company switch). OFF = AlephK works as before
--      locations: no location in schedules, forms, the store or emails.
--      Companies that ALREADY have a location are turned ON once (so what
--      you set up keeps working); every other company starts OFF.
--   2) "Customer chooses": a block of hours (weekly or a date) can be
--      offered in ANY active location; the customer picks the place.
-- =====================================================================

-- ---------------------------------------------------------------------
-- 1) "We use locations" (added and filled only the first time).
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'tbl_company_booking_settings'
          AND column_name = 'locations_enabled'
    ) THEN
        ALTER TABLE public.tbl_company_booking_settings
            ADD COLUMN locations_enabled BOOLEAN NOT NULL DEFAULT FALSE;

        INSERT INTO public.tbl_company_booking_settings (company_id, locations_enabled, updated_at)
        SELECT DISTINCT l.company_id, TRUE, now()
        FROM public.tbl_company_location l
        ON CONFLICT (company_id) DO UPDATE
        SET locations_enabled = TRUE, updated_at = now();
    END IF;
END $$;

-- ---------------------------------------------------------------------
-- 2) "Customer chooses" on weekly hours and date slots. When TRUE the block
--    has no location of its own (location_id stays NULL).
ALTER TABLE public.tbl_booking_recurring_range
    ADD COLUMN IF NOT EXISTS customer_chooses_location BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE public.tbl_booking_manual_slot
    ADD COLUMN IF NOT EXISTS customer_chooses_location BOOLEAN NOT NULL DEFAULT FALSE;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'ck_recurring_range_choice') THEN
        ALTER TABLE public.tbl_booking_recurring_range
            ADD CONSTRAINT ck_recurring_range_choice
            CHECK (NOT customer_chooses_location OR location_id IS NULL);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'ck_manual_slot_choice') THEN
        ALTER TABLE public.tbl_booking_manual_slot
            ADD CONSTRAINT ck_manual_slot_choice
            CHECK (NOT customer_chooses_location OR location_id IS NULL);
    END IF;
END $$;

-- ---------------------------------------------------------------------
-- Checks:
--   (a) the three new columns (must answer 3):
--   SELECT count(*) FROM information_schema.columns
--    WHERE table_schema = 'public'
--      AND ((table_name = 'tbl_company_booking_settings' AND column_name = 'locations_enabled')
--        OR (table_name IN ('tbl_booking_recurring_range','tbl_booking_manual_slot')
--            AND column_name = 'customer_chooses_location'));
--   (b) which companies use locations:
--   SELECT pc.display_name, s.locations_enabled, s.locations_same_time
--   FROM public.tbl_company_booking_settings s
--   LEFT JOIN public.tbl_page_configuration pc ON pc.company_id::text = s.company_id;
--
-- Turn a company ON or OFF by hand (the Locations screen does the same):
--   INSERT INTO public.tbl_company_booking_settings (company_id, locations_enabled)
--   VALUES ('COMPANY-ID', TRUE)
--   ON CONFLICT (company_id) DO UPDATE SET locations_enabled = EXCLUDED.locations_enabled, updated_at = now();
--
-- UNDO (only if you go back to shao #34):
--   ALTER TABLE public.tbl_booking_manual_slot     DROP COLUMN IF EXISTS customer_chooses_location;
--   ALTER TABLE public.tbl_booking_recurring_range DROP COLUMN IF EXISTS customer_chooses_location;
--   ALTER TABLE public.tbl_company_booking_settings DROP COLUMN IF EXISTS locations_enabled;
-- =====================================================================
