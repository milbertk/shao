-- =====================================================================
-- Company legal texts (2026-10-24) — shao database.
-- Terms and conditions + privacy policy written by each company in the
-- Company page. NULL = the store does not show the link.
-- Run BEFORE starting the new shao version. Safe to run twice.
-- =====================================================================

ALTER TABLE public.tbl_page_configuration
    ADD COLUMN IF NOT EXISTS terms_text   TEXT,
    ADD COLUMN IF NOT EXISTS privacy_text TEXT;

-- Check:
--   SELECT company_id, display_name, terms_text IS NOT NULL AS has_terms, privacy_text IS NOT NULL AS has_privacy
--   FROM public.tbl_page_configuration;
-- Revert (only with the previous shao version):
--   ALTER TABLE public.tbl_page_configuration DROP COLUMN IF EXISTS terms_text, DROP COLUMN IF EXISTS privacy_text;
