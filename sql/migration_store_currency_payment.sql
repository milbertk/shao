-- =====================================================================
-- One currency per company + payment instructions (2026-10-22)
--  - currency_symbol: the currency of ALL prices of the company's store,
--    cart and emails (set in "Company page"). Default "$".
--  - payment_instructions: how customers pay (SINPE Movil, bank account...),
--    shown after an order and in the emails. Optional.
-- Run ONCE before starting the new shao version. Safe to run twice.
-- =====================================================================

ALTER TABLE public.tbl_page_configuration
    ADD COLUMN IF NOT EXISTS currency_symbol      VARCHAR(5) NOT NULL DEFAULT '$',
    ADD COLUMN IF NOT EXISTS payment_instructions TEXT;

-- Start each company with the currency most used by its products (only
-- companies still on the default "$"; products keep their own value).
UPDATE public.tbl_page_configuration pc
SET currency_symbol = top.currency_symbol,
    updated_at      = now()
FROM (
    SELECT DISTINCT ON (company_id::text)
           company_id::text AS company_id,
           currency_symbol
    FROM public.tbl_products
    WHERE COALESCE(TRIM(currency_symbol), '') <> ''
    GROUP BY company_id::text, currency_symbol
    ORDER BY company_id::text, COUNT(*) DESC
) AS top
WHERE pc.company_id::text = top.company_id
  AND pc.currency_symbol = '$'
  AND top.currency_symbol <> '$';

-- ---------------------------------------------------------------------
-- REVERT (only if you go back to the previous shao version):
-- ALTER TABLE public.tbl_page_configuration
--     DROP COLUMN IF EXISTS currency_symbol,
--     DROP COLUMN IF EXISTS payment_instructions;
-- ---------------------------------------------------------------------
