-- =====================================================================
-- Security fixes after control point CC 1 (2026-10-25).
-- Run in the SHAO database BEFORE starting the new shao version.
-- Safe to run twice.
--
--  S14  tbl_plans.ai_interactions_month: AI assistant questions per month.
--       RULE: the AI of a plan may cost AT MOST 10% of the plan price.
--       NULL (normal) = computed by shao from the price:
--           floor(price x 10% / AI_COST_PER_INTERACTION_USD)
--       A number LOWERS that limit (it can never go above 10% of the price).
--       Free plans (TRIAL, price 0): the number is the fixed limit (50).
--       Plans without price (SPECIAL): the number, NULL = unlimited.
--  S4   Limits of the public store forms and of customer emails.
--  S13  Firebase bucket accepted for file URLs.
-- =====================================================================

-- 1) S14 — AI interactions per plan ---------------------------------------
ALTER TABLE public.tbl_plans ADD COLUMN IF NOT EXISTS ai_interactions_month INTEGER
    CHECK (ai_interactions_month IS NULL OR ai_interactions_month >= 0);

UPDATE public.tbl_plans SET ai_interactions_month = 50, updated_at = now()
 WHERE plan_code = 'TRIAL' AND ai_interactions_month IS NULL;

-- Faster count of the questions of a company this month.
CREATE INDEX IF NOT EXISTS ix_ai_message_company_questions
    ON public.tbl_ai_message (company_id, created_at) WHERE role = 'user';

-- 2) Configuration (change the values here or later with UPDATE) ----------
INSERT INTO public.tbl_configuration (parameter, value)
SELECT v.parameter, v.value
  FROM (VALUES
        ('AI_COST_PER_INTERACTION_USD',     '0.006'),   -- S14 cost of one question (Haiku, rounded up)
        ('PUBLIC_FORM_IP_DAILY_LIMIT',      '20'),      -- S4 orders/bookings per visitor IP, per company, per day
        ('PUBLIC_FORM_COMPANY_DAILY_LIMIT', '300'),     -- S4 store orders (and bookings) per company per day
        ('CUSTOMER_EMAIL_DAILY_LIMIT',      '10'),      -- S4 customer emails one address receives per day
        ('FIREBASE_STORAGE_BUCKET',         'kdauth-9893b.firebasestorage.app')  -- S13 (env FIREBASE_STORAGE_BUCKET wins)
       ) AS v(parameter, value)
 WHERE NOT EXISTS (SELECT 1 FROM public.tbl_configuration c WHERE c.parameter = v.parameter);

-- 3) S4 — faster daily counts ----------------------------------------------
CREATE INDEX IF NOT EXISTS ix_web_order_request_company_created
    ON public.tbl_web_order_request (company_id, created_at);
CREATE INDEX IF NOT EXISTS ix_appointment_public_created
    ON public.tbl_appointment (company_id, created_at) WHERE registered_by = 'showcase-guest';
CREATE INDEX IF NOT EXISTS ix_notification_log_customer_recipient
    ON public.tbl_notification_log (lower(recipient), created_at)
    WHERE audience = 'CUSTOMER' AND status = 'SENT';

-- =====================================================================
-- CHECKS
-- =====================================================================
-- a) AI questions per month of each plan (same rule as shao):
--    SELECT plan_code, monthly_price_usd, ai_assistant, ai_interactions_month,
--           CASE
--             WHEN NOT ai_assistant THEN 0
--             WHEN monthly_price_usd IS NULL THEN ai_interactions_month
--             WHEN monthly_price_usd <= 0 THEN COALESCE(ai_interactions_month, 50)
--             ELSE LEAST(COALESCE(ai_interactions_month, 2147483647),
--                        FLOOR(monthly_price_usd * 0.10 /
--                              (SELECT value::numeric FROM public.tbl_configuration
--                                WHERE parameter = 'AI_COST_PER_INTERACTION_USD')))
--           END AS questions_per_month
--      FROM public.tbl_plans WHERE active ORDER BY sort_order;
--
-- b) Lower the questions of a plan (never above 10% of the price):
--    UPDATE public.tbl_plans SET ai_interactions_month = 300, updated_at = now() WHERE plan_code = 'PRO';
--    UPDATE public.tbl_plans SET ai_interactions_month = NULL, updated_at = now() WHERE plan_code = 'PRO'; -- back to 10%
--
-- c) Questions used this month per company:
--    SELECT company_id, COUNT(*) FROM public.tbl_ai_message
--     WHERE role = 'user' AND created_at >= date_trunc('month', now())
--     GROUP BY company_id ORDER BY 2 DESC;
--
-- d) Blocked store forms (tracking) and skipped emails:
--    SELECT * FROM public.tbl_notification_log
--     WHERE error_message = 'daily email limit reached for this address' ORDER BY created_at DESC;
--
-- ---------------------------------------------------------------------
-- REVERT (only with the previous shao version):
-- DROP INDEX IF EXISTS public.ix_ai_message_company_questions;
-- DROP INDEX IF EXISTS public.ix_web_order_request_company_created;
-- DROP INDEX IF EXISTS public.ix_appointment_public_created;
-- DROP INDEX IF EXISTS public.ix_notification_log_customer_recipient;
-- DELETE FROM public.tbl_configuration WHERE parameter IN ('AI_COST_PER_INTERACTION_USD',
--   'PUBLIC_FORM_IP_DAILY_LIMIT', 'PUBLIC_FORM_COMPANY_DAILY_LIMIT', 'CUSTOMER_EMAIL_DAILY_LIMIT', 'FIREBASE_STORAGE_BUCKET');
-- ALTER TABLE public.tbl_plans DROP COLUMN IF EXISTS ai_interactions_month;
-- ---------------------------------------------------------------------
