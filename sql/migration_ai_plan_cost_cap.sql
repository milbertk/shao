-- =====================================================================
-- AI assistants: plan feature + monthly cost cap per company (2026-10-22)
--  - Only companies on the plans in AI_PLANS (companies.plan) can use the
--    assistants (internal Aleph and the store assistant).
--  - Each company has a monthly cost cap in US$ (default AI_DEFAULT_MONTHLY_CAP_USD).
--    Cost = tokens used this month (tbl_ai_message) x prices per million tokens.
--  - tbl_company_ai_settings lets you change the cap of ONE company, or turn
--    the assistant on/off for it regardless of the plan (set by hand until
--    the super-admin screen exists).
-- Run ONCE before starting the new shao version. Safe to run twice.
-- =====================================================================

CREATE TABLE IF NOT EXISTS public.tbl_company_ai_settings (
    company_ai_settings_id UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id             VARCHAR(100)  NOT NULL UNIQUE,
    monthly_cap_usd        NUMERIC(10,2),          -- NULL = use the default cap
    enabled_override       BOOLEAN,                -- NULL = follow the plan; TRUE/FALSE = force
    notes                  TEXT,
    created_at             TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ   NOT NULL DEFAULT now()
);

-- Faster monthly sums.
CREATE INDEX IF NOT EXISTS idx_ai_message_company_month
    ON public.tbl_ai_message (company_id, created_at);

-- Settings (edit the values in tbl_configuration when prices change).
INSERT INTO public.tbl_configuration (parameter, value)
SELECT 'AI_PLANS', 'PRO,ENTERPRISE'
WHERE NOT EXISTS (SELECT 1 FROM public.tbl_configuration WHERE parameter = 'AI_PLANS');

INSERT INTO public.tbl_configuration (parameter, value)
SELECT 'AI_DEFAULT_MONTHLY_CAP_USD', '5'
WHERE NOT EXISTS (SELECT 1 FROM public.tbl_configuration WHERE parameter = 'AI_DEFAULT_MONTHLY_CAP_USD');

-- Prices of the model used by the assistants, in US$ per 1,000,000 tokens.
-- CHECK them on the Anthropic pricing page and update if needed.
INSERT INTO public.tbl_configuration (parameter, value)
SELECT 'AI_PRICE_INPUT_PER_MTOK', '1'
WHERE NOT EXISTS (SELECT 1 FROM public.tbl_configuration WHERE parameter = 'AI_PRICE_INPUT_PER_MTOK');

INSERT INTO public.tbl_configuration (parameter, value)
SELECT 'AI_PRICE_OUTPUT_PER_MTOK', '5'
WHERE NOT EXISTS (SELECT 1 FROM public.tbl_configuration WHERE parameter = 'AI_PRICE_OUTPUT_PER_MTOK');

-- Examples:
--   Give one company a US$20 cap:
--   INSERT INTO public.tbl_company_ai_settings (company_id, monthly_cap_usd) VALUES ('<companyId>', 20)
--   ON CONFLICT (company_id) DO UPDATE SET monthly_cap_usd = EXCLUDED.monthly_cap_usd, updated_at = now();
--   Turn the assistant ON for a FREE company (pilot):
--   INSERT INTO public.tbl_company_ai_settings (company_id, enabled_override) VALUES ('<companyId>', TRUE)
--   ON CONFLICT (company_id) DO UPDATE SET enabled_override = EXCLUDED.enabled_override, updated_at = now();
--   Cost of this month per company:
--   SELECT company_id, SUM(input_tokens) AS input_tokens, SUM(output_tokens) AS output_tokens
--   FROM public.tbl_ai_message WHERE created_at >= date_trunc('month', now()) GROUP BY company_id;

-- ---------------------------------------------------------------------
-- REVERT (only if you go back to the previous shao version):
-- DROP TABLE IF EXISTS public.tbl_company_ai_settings;
-- DROP INDEX IF EXISTS public.idx_ai_message_company_month;
-- DELETE FROM public.tbl_configuration WHERE parameter IN
--   ('AI_PLANS','AI_DEFAULT_MONTHLY_CAP_USD','AI_PRICE_INPUT_PER_MTOK','AI_PRICE_OUTPUT_PER_MTOK');
-- ---------------------------------------------------------------------
