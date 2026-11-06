-- =====================================================================
-- Company plans (2026-10-22)
--  - tbl_plans: the plans (FREE, BASIC, PRO, ENTERPRISE, TRIAL) with their
--    monthly price in US$ and their limits. Change prices/limits HERE; the
--    app reads them (no code change needed).
--      max_users / max_products: NULL = unlimited
--      all_reports  = FALSE -> only "Orders by status"
--      web_bookings = FALSE -> bookings only from the manager (not the store)
--      ai_assistant = TRUE  -> Aleph (app) + store assistant
--      trial_days   -> TRIAL length; afterwards the company works as FREE
--  - tbl_plan_request: requests sent from the "Your plan" window.
--  - tbl_company_plan: the shao plan of each company (one row per company;
--    no row = FREE), with started_at and ends_at (TRIAL end). The plan lives
--    in shao, not in usersAppAK's companies table: one company can have
--    different plans in different platforms.
--  - PLAN_REQUEST_EMAIL_TO: where the requests are emailed (uses the
--    platform SMTP: SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASSWORD, SMTP_FROM).
-- Run this file in the SHAO database (everything is in shao).
-- Run ONCE before starting the new shao version. Safe to run twice.
-- =====================================================================

CREATE TABLE IF NOT EXISTS public.tbl_plans (
    plan_code          VARCHAR(30)   PRIMARY KEY,           -- FREE, BASIC, PRO, ENTERPRISE, TRIAL
    name               VARCHAR(60)   NOT NULL,
    description        TEXT,
    monthly_price_usd  NUMERIC(10,2),                       -- NULL = "ask for the price"
    max_users          INTEGER,                             -- NULL = unlimited
    max_products       INTEGER,                             -- NULL = unlimited
    all_reports        BOOLEAN       NOT NULL DEFAULT TRUE,
    web_bookings       BOOLEAN       NOT NULL DEFAULT TRUE,
    ai_assistant       BOOLEAN       NOT NULL DEFAULT FALSE,
    trial_days         INTEGER,                             -- only for TRIAL
    sort_order         INTEGER       NOT NULL DEFAULT 0,
    requestable        BOOLEAN       NOT NULL DEFAULT TRUE, -- shown in the request form
    active             BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at         TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ   NOT NULL DEFAULT now()
);

INSERT INTO public.tbl_plans
    (plan_code, name, description, monthly_price_usd, max_users, max_products,
     all_reports, web_bookings, ai_assistant, trial_days, sort_order, requestable)
VALUES
    ('FREE',       'Free',       'To start: 1 user, 10 products, bookings from the manager, Orders by status report.',
                   0,    1,   10,   FALSE, FALSE, FALSE, NULL, 10, FALSE),
    ('BASIC',      'Basic',      'All options without the AI assistant, up to 10 users.',
                   NULL, 10,  NULL, TRUE,  TRUE,  FALSE, NULL, 20, TRUE),
    ('PRO',        'Pro',        'All options plus the AI assistant (Aleph), up to 20 users.',
                   NULL, 20,  NULL, TRUE,  TRUE,  TRUE,  NULL, 30, TRUE),
    ('ENTERPRISE', 'Enterprise', 'All options plus the AI assistant (Aleph), up to 100 users.',
                   NULL, 100, NULL, TRUE,  TRUE,  TRUE,  NULL, 40, TRUE),
    ('TRIAL',      'Trial',      'Try everything (Pro options and Aleph) for 30 days with up to 10 users.',
                   0,    10,  NULL, TRUE,  TRUE,  TRUE,  30,   5,  TRUE)
ON CONFLICT (plan_code) DO NOTHING;

-- Shao plan of each company (no row = FREE).
CREATE TABLE IF NOT EXISTS public.tbl_company_plan (
    company_plan_id  UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id       VARCHAR(100)  NOT NULL UNIQUE,
    plan_code        VARCHAR(30)   NOT NULL REFERENCES public.tbl_plans (plan_code),
    started_at       TIMESTAMPTZ,
    ends_at          TIMESTAMPTZ,                           -- TRIAL end; NULL = no end
    notes            TEXT,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.tbl_plan_request (
    plan_request_id      UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id           VARCHAR(100)  NOT NULL,
    company_name         VARCHAR(200),
    current_plan         VARCHAR(30),
    requested_plan       VARCHAR(30)   NOT NULL,
    contact_name         VARCHAR(150)  NOT NULL,
    contact_email        VARCHAR(200)  NOT NULL,
    contact_phone        VARCHAR(40),
    message              TEXT,
    language             VARCHAR(5),
    status               VARCHAR(20)   NOT NULL DEFAULT 'PENDING',  -- PENDING, CONTACTED, APPROVED, REJECTED
    requested_by_user_id VARCHAR(128),
    email_sent           BOOLEAN       NOT NULL DEFAULT FALSE,
    handled_notes        TEXT,
    handled_at           TIMESTAMPTZ,
    created_at           TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_plan_request_company
    ON public.tbl_plan_request (company_id, created_at);
CREATE INDEX IF NOT EXISTS idx_plan_request_status
    ON public.tbl_plan_request (status, created_at);

-- Where plan requests are emailed (your sales / support address).
INSERT INTO public.tbl_configuration (parameter, value)
SELECT 'PLAN_REQUEST_EMAIL_TO', ''
WHERE NOT EXISTS (SELECT 1 FROM public.tbl_configuration WHERE parameter = 'PLAN_REQUEST_EMAIL_TO');

-- The AI assistant now follows tbl_plans.ai_assistant (AI_PLANS is only a
-- fallback when tbl_plans cannot be read). Keep it in line:
UPDATE public.tbl_configuration SET value = 'PRO,ENTERPRISE,TRIAL'
WHERE parameter = 'AI_PLANS' AND value = 'PRO,ENTERPRISE';

-- =====================================================================
-- HOW TO USE (copy, change the values, run)
-- =====================================================================
-- 1) Set the email that receives the requests:
--    UPDATE public.tbl_configuration SET value = 'sales@yourdomain.com'
--    WHERE parameter = 'PLAN_REQUEST_EMAIL_TO';
--    (The platform SMTP must be set: SMTP_HOST, SMTP_PORT, SMTP_USER,
--     SMTP_PASSWORD, SMTP_FROM in tbl_configuration.)
--
-- 2) Prices (US$ per month; NULL shows "Ask for the price"):
--    UPDATE public.tbl_plans SET monthly_price_usd = 15, updated_at = now() WHERE plan_code = 'BASIC';
--    UPDATE public.tbl_plans SET monthly_price_usd = 35, updated_at = now() WHERE plan_code = 'PRO';
--    UPDATE public.tbl_plans SET monthly_price_usd = 90, updated_at = now() WHERE plan_code = 'ENTERPRISE';
--
-- 3) See pending requests:
--    SELECT created_at, company_name, current_plan, requested_plan, contact_name,
--           contact_email, contact_phone, message, status
--    FROM public.tbl_plan_request WHERE status = 'PENDING' ORDER BY created_at;
--
-- 4) Approve a request = set the company plan + close the request:
--    INSERT INTO public.tbl_company_plan (company_id, plan_code, started_at, notes)
--    VALUES ('<companyId>', 'PRO', now(), 'Paid by SINPE')
--    ON CONFLICT (company_id) DO UPDATE
--      SET plan_code = EXCLUDED.plan_code, started_at = now(), ends_at = NULL,
--          notes = EXCLUDED.notes, updated_at = now();
--    UPDATE public.tbl_plan_request
--       SET status = 'APPROVED', handled_at = now(), updated_at = now(), handled_notes = 'Paid by SINPE'
--     WHERE plan_request_id = '<requestId>';
--
-- 5) Start a 30-day TRIAL:
--    INSERT INTO public.tbl_company_plan (company_id, plan_code, started_at, ends_at)
--    VALUES ('<companyId>', 'TRIAL', now(), now() + interval '30 days')
--    ON CONFLICT (company_id) DO UPDATE
--      SET plan_code = 'TRIAL', started_at = now(), ends_at = now() + interval '30 days', updated_at = now();
--    (If ends_at is left empty, shao sets it to start + trial_days the first
--     time it reads the plan. When it ends, the company works as FREE.)
--
-- 6) Back to FREE: DELETE FROM public.tbl_company_plan WHERE company_id = '<companyId>';
--
-- 7) Companies by plan (companies without a row are FREE):
--    SELECT plan_code, COUNT(*) FROM public.tbl_company_plan GROUP BY plan_code;

-- ---------------------------------------------------------------------
-- REVERT (only if you go back to the previous shao version):
-- DROP TABLE IF EXISTS public.tbl_plan_request;
-- DROP TABLE IF EXISTS public.tbl_company_plan;
-- DROP TABLE IF EXISTS public.tbl_plans;
-- DELETE FROM public.tbl_configuration WHERE parameter = 'PLAN_REQUEST_EMAIL_TO';
-- UPDATE public.tbl_configuration SET value = 'PRO,ENTERPRISE' WHERE parameter = 'AI_PLANS';
-- ---------------------------------------------------------------------
