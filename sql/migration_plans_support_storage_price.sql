-- =====================================================================
-- Plans: human support hours, storage and negotiated price (2026-10-22)
-- Run in the SHAO database, AFTER sql/migration_company_plans.sql and
-- BEFORE starting the new shao version. Safe to run twice.
--
--  tbl_plans (definition of each plan, change it here):
--    support_hours_month: hours of human support included per month (0 = none)
--    storage_gb: storage included in GB (NULL = not defined). Shown in the
--                plans window; it is not measured or blocked yet.
--  tbl_company_plan (plan of each company):
--    agreed_price_usd: price negotiated with THIS company (US$ per month).
--                      NULL = the normal price of its plan (tbl_plans).
-- =====================================================================

ALTER TABLE public.tbl_plans ADD COLUMN IF NOT EXISTS support_hours_month NUMERIC(5,1) NOT NULL DEFAULT 0;
ALTER TABLE public.tbl_plans ADD COLUMN IF NOT EXISTS storage_gb INTEGER;

UPDATE public.tbl_plans SET support_hours_month = 2, storage_gb = 50, updated_at = now() WHERE plan_code = 'ENTERPRISE';
UPDATE public.tbl_plans SET support_hours_month = 1, storage_gb = 20, updated_at = now() WHERE plan_code = 'PRO';
UPDATE public.tbl_plans SET support_hours_month = 0, storage_gb = 20, updated_at = now() WHERE plan_code = 'BASIC';
UPDATE public.tbl_plans SET support_hours_month = 0, storage_gb = 10, updated_at = now() WHERE plan_code IN ('FREE', 'TRIAL');

ALTER TABLE public.tbl_company_plan ADD COLUMN IF NOT EXISTS agreed_price_usd NUMERIC(10,2);

-- =====================================================================
-- EXAMPLES
-- =====================================================================
-- See the plan definitions:
--   SELECT plan_code, monthly_price_usd, max_users, max_products, support_hours_month, storage_gb,
--          all_reports, web_bookings, ai_assistant
--   FROM public.tbl_plans ORDER BY sort_order;
--
-- Change support hours or storage of a plan:
--   UPDATE public.tbl_plans SET support_hours_month = 6, storage_gb = 100, updated_at = now() WHERE plan_code = 'ENTERPRISE';
--
-- Negotiated price for ONE company (it sees "Your price: US$70 / month (agreed)"):
--   UPDATE public.tbl_company_plan SET agreed_price_usd = 70, notes = 'Agreed 2026-10, 12 months', updated_at = now()
--   WHERE company_id = '<companyId>';
-- Back to the normal plan price:
--   UPDATE public.tbl_company_plan SET agreed_price_usd = NULL, updated_at = now() WHERE company_id = '<companyId>';
--
-- Monthly income (agreed price or plan price) of companies with a paid plan:
--   SELECT cp.company_id, cp.plan_code, COALESCE(cp.agreed_price_usd, p.monthly_price_usd) AS monthly_usd
--   FROM public.tbl_company_plan cp JOIN public.tbl_plans p ON p.plan_code = cp.plan_code
--   WHERE cp.ends_at IS NULL OR cp.ends_at > now()
--   ORDER BY monthly_usd DESC NULLS LAST;

-- ---------------------------------------------------------------------
-- REVERT (only with the previous shao version):
-- ALTER TABLE public.tbl_plans DROP COLUMN IF EXISTS support_hours_month;
-- ALTER TABLE public.tbl_plans DROP COLUMN IF EXISTS storage_gb;
-- ALTER TABLE public.tbl_company_plan DROP COLUMN IF EXISTS agreed_price_usd;
-- ---------------------------------------------------------------------
