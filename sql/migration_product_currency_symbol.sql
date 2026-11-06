-- Currency symbol per product (2026-10-21).
-- Shown with the price in the public showcase and the AI assistant.
-- Existing products get '$' automatically. Max 5 characters (e.g. $, ₡, €, S/, R$).
-- Run ONCE before starting the new shao version.
ALTER TABLE public.tbl_products
    ADD COLUMN IF NOT EXISTS currency_symbol VARCHAR(5) NOT NULL DEFAULT '$';

-- Check:
-- SELECT currency_symbol, COUNT(*) FROM public.tbl_products GROUP BY 1;

-- Revert (only after reverting the Go code):
-- ALTER TABLE public.tbl_products DROP COLUMN IF EXISTS currency_symbol;
