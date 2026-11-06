-- =====================================================================
-- Plans 2026-10-23: no free plan, ENTREPRENEUR, SPECIAL, storage limits,
-- every option in every plan, read-only without an active plan.
-- Run in the SHAO database, AFTER sql/migration_company_plans.sql and
-- sql/migration_plans_support_storage_price.sql, BEFORE starting the new
-- shao version. Safe to run twice.
--
--  - FREE becomes ENTREPRENEUR (US$10/month + taxes, 1 user, 100 products,
--    5 GB). Companies on FREE move to ENTREPRENEUR so they keep working.
--    The FREE row stays (inactive) only for history.
--  - TRIAL is the only free plan (testing). It can always be requested; you
--    approve each one by hand (the request email shows previous trials).
--  - SPECIAL: no limits (NULL = unlimited), AI included, NOT requestable,
--    shown only to the company that has it. Only you assign it (step 3).
--  - Every plan: all reports + store bookings. Plans differ in users,
--    products, storage, AI assistant and human support only.
--  - A company WITHOUT an active plan (no row in tbl_company_plan, or its
--    plan / trial ended) is READ-ONLY until you assign a plan.
--  - tbl_company_storage_file: size of each file a company uploaded
--    (Firebase), to measure storage against tbl_plans.storage_gb.
-- Limits are configurable per plan here (no code change needed):
--    max_users / max_products / storage_gb: NULL = unlimited.
-- =====================================================================

-- 1) Plans --------------------------------------------------------------
INSERT INTO public.tbl_plans
    (plan_code, name, description, monthly_price_usd, max_users, max_products,
     all_reports, web_bookings, ai_assistant, trial_days, sort_order, requestable,
     support_hours_month, storage_gb)
VALUES
    ('ENTREPRENEUR', 'Entrepreneur', 'For one person starting a business: 1 user, 100 products, 5 GB, every option except the AI assistant.',
                     10,   1,    100,  TRUE, TRUE, FALSE, NULL, 10, TRUE,  0, 5),
    ('SPECIAL',      'Special',      'Agreed with AlephK: no limits.',
                     NULL, NULL, NULL, TRUE, TRUE, TRUE,  NULL, 60, FALSE, 0, NULL)
ON CONFLICT (plan_code) DO NOTHING;

UPDATE public.tbl_plans SET max_users = 10,  max_products = NULL, storage_gb = 5,  all_reports = TRUE, web_bookings = TRUE,
       ai_assistant = TRUE,  monthly_price_usd = 0,
       description = 'Free for 30 days to try everything: up to 10 users, 5 GB, AI assistant.', updated_at = now()
 WHERE plan_code = 'TRIAL';
UPDATE public.tbl_plans SET max_users = 10,  max_products = NULL, storage_gb = 10, all_reports = TRUE, web_bookings = TRUE,
       description = 'Every option without the AI assistant, up to 10 users, 10 GB.', updated_at = now()
 WHERE plan_code = 'BASIC';
UPDATE public.tbl_plans SET max_users = 20,  max_products = NULL, storage_gb = 10, all_reports = TRUE, web_bookings = TRUE,
       description = 'Every option plus the AI assistant (Aleph), up to 20 users, 10 GB, 1 hour of support per month.', updated_at = now()
 WHERE plan_code = 'PRO';
UPDATE public.tbl_plans SET max_users = 100, max_products = NULL, storage_gb = 25, all_reports = TRUE, web_bookings = TRUE,
       description = 'Every option plus the AI assistant (Aleph), up to 100 users, 25 GB, 2 hours of support per month.', updated_at = now()
 WHERE plan_code = 'ENTERPRISE';

-- 2) FREE -> ENTREPRENEUR -----------------------------------------------
UPDATE public.tbl_company_plan SET plan_code = 'ENTREPRENEUR', updated_at = now(),
       notes = TRIM(BOTH ' ' FROM COALESCE(notes, '') || ' [moved from FREE 2026-10-23]')
 WHERE plan_code = 'FREE';
UPDATE public.tbl_plans SET active = FALSE, requestable = FALSE, updated_at = now() WHERE plan_code = 'FREE';

-- AI fallback list (only used when tbl_plans cannot be read).
UPDATE public.tbl_configuration SET value = 'PRO,ENTERPRISE,TRIAL,SPECIAL'
 WHERE parameter = 'AI_PLANS';

-- 3) Storage --------------------------------------------------------------
CREATE TABLE IF NOT EXISTS public.tbl_company_storage_file (
    storage_file_id     UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id          VARCHAR(100)  NOT NULL,
    storage_path        VARCHAR(500)  NOT NULL,              -- Firebase path
    size_bytes          BIGINT        NOT NULL CHECK (size_bytes >= 0),
    kind                VARCHAR(30)   NOT NULL,              -- PRODUCT_IMAGE | PAGE_IMAGE | ORDER_ATTACHMENT | TRACKING_ATTACHMENT | OTHER
    created_by_user_id  VARCHAR(128),
    created_at          TIMESTAMPTZ   NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ                          -- not counted anymore
);
CREATE UNIQUE INDEX IF NOT EXISTS ux_company_storage_file_path
    ON public.tbl_company_storage_file (company_id, storage_path);
CREATE INDEX IF NOT EXISTS ix_company_storage_file_used
    ON public.tbl_company_storage_file (company_id) WHERE deleted_at IS NULL;

-- Count the attachments that already exist (they have a size).
INSERT INTO public.tbl_company_storage_file (company_id, storage_path, size_bytes, kind)
SELECT company_id, storage_path, COALESCE(file_size, 0), 'ORDER_ATTACHMENT'
  FROM public.tbl_order_attachment
 WHERE storage_path IS NOT NULL AND storage_path <> ''
ON CONFLICT (company_id, storage_path) DO NOTHING;

INSERT INTO public.tbl_company_storage_file (company_id, storage_path, size_bytes, kind)
SELECT company_id, storage_path, COALESCE(file_size, 0), 'TRACKING_ATTACHMENT'
  FROM public.tbl_order_tracking_attachment
 WHERE storage_path IS NOT NULL AND storage_path <> ''
ON CONFLICT (company_id, storage_path) DO NOTHING;
-- Product and page images uploaded before today have no size: they count
-- from the next time they are replaced.

-- =====================================================================
-- HOW TO USE (copy, change the values, run)
-- =====================================================================
-- a) See the plans and their limits:
--    SELECT plan_code, name, monthly_price_usd, max_users, max_products, storage_gb,
--           ai_assistant, support_hours_month, requestable, active
--    FROM public.tbl_plans ORDER BY sort_order;
--
-- b) Change a limit or a price (NULL = unlimited / "ask for the price"):
--    UPDATE public.tbl_plans SET max_products = 150, updated_at = now() WHERE plan_code = 'ENTREPRENEUR';
--    UPDATE public.tbl_plans SET monthly_price_usd = 25, updated_at = now() WHERE plan_code = 'BASIC';
--
-- c) Approve a request / assign a plan (the company stops being read-only):
--    INSERT INTO public.tbl_company_plan (company_id, plan_code, started_at, notes)
--    VALUES ('<companyId>', 'ENTREPRENEUR', now(), 'Paid by SINPE')
--    ON CONFLICT (company_id) DO UPDATE
--      SET plan_code = EXCLUDED.plan_code, started_at = now(), ends_at = NULL,
--          notes = EXCLUDED.notes, updated_at = now();
--    UPDATE public.tbl_plan_request
--       SET status = 'APPROVED', handled_at = now(), updated_at = now(), handled_notes = 'Paid by SINPE'
--     WHERE plan_request_id = '<requestId>';
--    (For a monthly plan paid until a date, also set ends_at; when it passes
--     the company becomes read-only.)
--
-- d) Approve a 30-day TRIAL:
--    INSERT INTO public.tbl_company_plan (company_id, plan_code, started_at, ends_at)
--    VALUES ('<companyId>', 'TRIAL', now(), now() + interval '30 days')
--    ON CONFLICT (company_id) DO UPDATE
--      SET plan_code = 'TRIAL', started_at = now(), ends_at = now() + interval '30 days', updated_at = now();
--    Previous trial requests of a company:
--    SELECT created_at, status FROM public.tbl_plan_request
--     WHERE company_id = '<companyId>' AND requested_plan = 'TRIAL' ORDER BY created_at DESC;
--
-- e) SPECIAL (no limits; only you assign it):
--    INSERT INTO public.tbl_company_plan (company_id, plan_code, started_at, agreed_price_usd, notes)
--    VALUES ('<companyId>', 'SPECIAL', now(), 120, 'Special agreement 2026-10')
--    ON CONFLICT (company_id) DO UPDATE
--      SET plan_code = 'SPECIAL', started_at = now(), ends_at = NULL, agreed_price_usd = EXCLUDED.agreed_price_usd,
--          notes = EXCLUDED.notes, updated_at = now();
--
-- f) Read-only companies (no active plan):
--    SELECT company_id, plan_code, ends_at FROM public.tbl_company_plan
--     WHERE ends_at IS NOT NULL AND ends_at <= now();
--    (Companies without a row in tbl_company_plan are read-only too.)
--
-- g) Storage used per company (MB):
--    SELECT company_id, ROUND(SUM(size_bytes) / 1024.0 / 1024.0, 1) AS used_mb, COUNT(*) AS files
--    FROM public.tbl_company_storage_file WHERE deleted_at IS NULL
--    GROUP BY company_id ORDER BY used_mb DESC;

-- ---------------------------------------------------------------------
-- REVERT (only with the previous shao version):
-- UPDATE public.tbl_plans SET active = TRUE, updated_at = now() WHERE plan_code = 'FREE';
-- UPDATE public.tbl_company_plan SET plan_code = 'FREE', updated_at = now() WHERE plan_code = 'ENTREPRENEUR';
-- DELETE FROM public.tbl_plans WHERE plan_code IN ('ENTREPRENEUR', 'SPECIAL')
--   AND NOT EXISTS (SELECT 1 FROM public.tbl_company_plan WHERE plan_code IN ('ENTREPRENEUR', 'SPECIAL'));
-- UPDATE public.tbl_plans SET storage_gb = 10 WHERE plan_code = 'TRIAL';
-- UPDATE public.tbl_plans SET storage_gb = 20 WHERE plan_code IN ('BASIC', 'PRO');
-- UPDATE public.tbl_plans SET storage_gb = 50 WHERE plan_code = 'ENTERPRISE';
-- DROP TABLE IF EXISTS public.tbl_company_storage_file;
-- ---------------------------------------------------------------------
