-- =====================================================================
-- Readable store address (slug) per company (2026-10-22)
-- Example: http://<store>/maderas-finas instead of http://<store>/<uuid>.
-- Optional: companies without a slug keep using their internal id.
-- Unique without caring about upper/lower case.
-- Run ONCE before starting the new shao version. Safe to run twice.
-- =====================================================================

ALTER TABLE public.tbl_page_configuration
    ADD COLUMN IF NOT EXISTS slug VARCHAR(60);

CREATE UNIQUE INDEX IF NOT EXISTS ux_page_configuration_slug
    ON public.tbl_page_configuration (lower(slug))
    WHERE slug IS NOT NULL;

-- Example: set a slug by hand (normally done in "Company page").
-- UPDATE public.tbl_page_configuration SET slug = 'maderas-finas', updated_at = now()
--  WHERE company_id = '<companyId>';

-- ---------------------------------------------------------------------
-- REVERT (only if you go back to the previous shao version):
-- DROP INDEX IF EXISTS public.ux_page_configuration_slug;
-- ALTER TABLE public.tbl_page_configuration DROP COLUMN IF EXISTS slug;
-- ---------------------------------------------------------------------
