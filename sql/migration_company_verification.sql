-- =====================================================================
-- Company verification (2026-10-22)
-- The public store shows the "Verified company" badge ONLY when the
-- company has a row here with is_verified = TRUE. Nobody is verified by
-- default. There is no screen yet: a super-admin (SA) option will manage
-- it later. Until then, set it by hand (example at the end).
-- Run ONCE before starting the new shao version. Safe to run twice.
-- =====================================================================

CREATE TABLE IF NOT EXISTS public.tbl_company_verification (
    company_verification_id UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id              VARCHAR(100) NOT NULL UNIQUE,
    is_verified             BOOLEAN      NOT NULL DEFAULT FALSE,
    verified_at             TIMESTAMPTZ,
    verified_by             VARCHAR(150),   -- who checked it (SA user / name)
    notes                   TEXT,           -- what was checked (documents, etc.)
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- Example: verify one company by hand.
-- INSERT INTO public.tbl_company_verification (company_id, is_verified, verified_at, verified_by, notes)
-- VALUES ('<companyId>', TRUE, now(), 'Milbert', 'Legal ID checked')
-- ON CONFLICT (company_id) DO UPDATE
--   SET is_verified = EXCLUDED.is_verified, verified_at = EXCLUDED.verified_at,
--       verified_by = EXCLUDED.verified_by, notes = EXCLUDED.notes, updated_at = now();

-- Example: remove the badge.
-- UPDATE public.tbl_company_verification SET is_verified = FALSE, updated_at = now() WHERE company_id = '<companyId>';

-- ---------------------------------------------------------------------
-- REVERT (only if you go back to the previous shao version):
-- DROP TABLE IF EXISTS public.tbl_company_verification;
-- ---------------------------------------------------------------------
