-- =====================================================================
-- Time zone per company (2026-11-04)
-- The server and the database work in UTC. Each company chooses its time
-- zone in "Company page"; every "now" of the company (today's past slots,
-- "until 2 hours before", Aleph's TODAY, email times) uses it.
-- Saved appointment and slot hours do NOT change: 14:00 stays 14:00.
--   time_zone       IANA name, e.g. America/Costa_Rica, America/New_York
--                   (handles summer time). Default: America/Costa_Rica (GMT-6).
--   time_zone_label optional text shown to customers, e.g. "Hora de Costa Rica".
-- Run ONCE before starting the new shao version. Safe to run twice.
-- =====================================================================

ALTER TABLE public.tbl_page_configuration
    ADD COLUMN IF NOT EXISTS time_zone VARCHAR(64) NOT NULL DEFAULT 'America/Costa_Rica';

ALTER TABLE public.tbl_page_configuration
    ADD COLUMN IF NOT EXISTS time_zone_label VARCHAR(60);

-- Check:
-- SELECT company_id, display_name, time_zone, time_zone_label FROM public.tbl_page_configuration;

-- Example: set a zone by hand (normally done in "Company page").
-- UPDATE public.tbl_page_configuration
--    SET time_zone = 'America/New_York', updated_at = now()
--  WHERE company_id = '<companyId>';

-- ---------------------------------------------------------------------
-- Server and database in UTC (do this on the production server, once):
--   sudo timedatectl set-timezone Etc/UTC
--   sudo -u postgres psql -c "ALTER SYSTEM SET timezone = 'UTC';"
--   sudo -u postgres psql -c "ALTER SYSTEM SET log_timezone = 'UTC';"
--   sudo systemctl reload postgresql
-- ---------------------------------------------------------------------

-- ---------------------------------------------------------------------
-- REVERT (only if you go back to the previous shao version):
-- ALTER TABLE public.tbl_page_configuration DROP COLUMN IF EXISTS time_zone_label;
-- ALTER TABLE public.tbl_page_configuration DROP COLUMN IF EXISTS time_zone;
-- ---------------------------------------------------------------------
