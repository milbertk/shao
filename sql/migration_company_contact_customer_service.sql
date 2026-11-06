-- =====================================================================
-- AlephK — company contact + customer service email (2026-10-31).
-- Run in the SHAO database BEFORE starting the new shao version.
-- Safe to run twice. Run it as a script (DBeaver: Alt+X).
--
-- 1) New optional columns in tbl_page_configuration (empty = not used):
--      social_link            one social network link of the company (store "Contact us")
--      social_link_type       FACEBOOK | INSTAGRAM | TIKTOK | OTHER (only for the icon)
--      customer_service_email blind copy (BCC) of every email sent to customers
-- 2) Data so the current store keeps working (see each step).
-- =====================================================================

ALTER TABLE public.tbl_page_configuration
    ADD COLUMN IF NOT EXISTS social_link            TEXT,
    ADD COLUMN IF NOT EXISTS social_link_type       VARCHAR(20),
    ADD COLUMN IF NOT EXISTS customer_service_email VARCHAR(254);

-- ---------------------------------------------------------------------
-- STEP 0 — safety copies of the two tables (only the first time), so the
-- data steps can be undone. Delete them later with DROP TABLE.
CREATE TABLE IF NOT EXISTS public.bak_products_20261031 AS SELECT * FROM public.tbl_products;
CREATE TABLE IF NOT EXISTS public.bak_page_configuration_20261031 AS SELECT * FROM public.tbl_page_configuration;

-- ---------------------------------------------------------------------
-- STEP A — company WhatsApp from the products (the store now uses the
-- company number, not the product number). Only for companies with an
-- EMPTY contact WhatsApp; takes the phone used by most of its products.
-- Preview first:
--   SELECT c.company_id, c.contact_whatsapp,
--          (SELECT p.web_contact_phone FROM public.tbl_products p
--            WHERE p.company_id::text = c.company_id::text AND btrim(COALESCE(p.web_contact_phone, '')) <> ''
--            GROUP BY p.web_contact_phone ORDER BY COUNT(*) DESC LIMIT 1) AS from_products
--   FROM public.tbl_page_configuration c;
UPDATE public.tbl_page_configuration c
SET contact_whatsapp = sub.phone, updated_at = now()
FROM (
    SELECT DISTINCT ON (p.company_id) p.company_id, btrim(p.web_contact_phone) AS phone
    FROM public.tbl_products p
    WHERE btrim(COALESCE(p.web_contact_phone, '')) <> ''
    GROUP BY p.company_id, btrim(p.web_contact_phone)
    ORDER BY p.company_id, COUNT(*) DESC
) sub
WHERE c.company_id::text = sub.company_id::text
  AND btrim(COALESCE(c.contact_whatsapp, '')) = '';

-- ---------------------------------------------------------------------
-- STEP B — keep "Add to cart" on the products that have it today.
-- New rule: Book = appointment; Add to cart = "available for web sales";
-- otherwise only "Contact us". Before, every non-service product had the
-- cart, so those get "available for web sales" = TRUE.
-- Preview first:
--   SELECT company_id, product_code, name FROM public.tbl_products
--   WHERE product_type <> 'SERVICE' AND COALESCE(web_behavior, '') <> 'BOOKING'
--     AND available_for_web_sales = FALSE;
UPDATE public.tbl_products
SET available_for_web_sales = TRUE
WHERE product_type <> 'SERVICE'
  AND COALESCE(web_behavior, '') <> 'BOOKING'
  AND available_for_web_sales = FALSE;

-- ---------------------------------------------------------------------
-- STEP C (OPTIONAL, read before running) — services that are NOT
-- appointments and were checked "available for web sales" would now show
-- "Add to cart" (before: only contact). Run this only if you want them to
-- stay "Contact us" only:
--   SELECT company_id, product_code, name FROM public.tbl_products
--   WHERE product_type = 'SERVICE' AND COALESCE(web_behavior, '') <> 'BOOKING'
--     AND available_for_web_sales = TRUE;
-- UPDATE public.tbl_products SET available_for_web_sales = FALSE
-- WHERE product_type = 'SERVICE' AND COALESCE(web_behavior, '') <> 'BOOKING'
--   AND available_for_web_sales = TRUE;

-- ---------------------------------------------------------------------
-- STEP D — old web behaviors become "not an appointment" (the form now has
-- only the checkbox "This is an appointment"). Harmless.
UPDATE public.tbl_products
SET web_behavior = 'MESSENGER_APP'
WHERE COALESCE(web_behavior, '') NOT IN ('BOOKING', 'MESSENGER_APP');

-- Checks:
--   SELECT company_id, contact_whatsapp, social_link, social_link_type, customer_service_email
--   FROM public.tbl_page_configuration;
-- REVERT (also restore the previous shao / p3 / showcase versions):
--   ALTER TABLE public.tbl_page_configuration
--     DROP COLUMN IF EXISTS social_link, DROP COLUMN IF EXISTS social_link_type,
--     DROP COLUMN IF EXISTS customer_service_email;
--   Data of steps A, B, D (from the STEP 0 copies):
--   UPDATE public.tbl_products p SET available_for_web_sales = b.available_for_web_sales,
--          web_behavior = b.web_behavior
--   FROM public.bak_products_20261031 b WHERE b.product_internal_id = p.product_internal_id;
--   UPDATE public.tbl_page_configuration c SET contact_whatsapp = b.contact_whatsapp
--   FROM public.bak_page_configuration_20261031 b WHERE b.company_id = c.company_id;
