-- =====================================================================
-- AlephK general email option (2026-10-30). Run in the SHAO database
-- BEFORE starting the new shao version. Safe to run twice.
--
-- tbl_page_configuration.use_default_smtp: the company accepts that, while it
-- has no SMTP of its own, its customer emails are sent with the AlephK general
-- email (SMTP_* in shao's .env). FALSE (default) = without a company SMTP no
-- email is sent at all.
-- =====================================================================
ALTER TABLE public.tbl_page_configuration
    ADD COLUMN IF NOT EXISTS use_default_smtp BOOLEAN NOT NULL DEFAULT FALSE;

-- Check:
--   SELECT company_id, display_name, use_default_smtp FROM public.tbl_page_configuration;
-- Turn it on for one company by hand (normally done in Company page):
--   UPDATE public.tbl_page_configuration SET use_default_smtp = TRUE WHERE company_id = '<companyId>';
-- REVERT (also restore the previous shao version):
--   ALTER TABLE public.tbl_page_configuration DROP COLUMN IF EXISTS use_default_smtp;
