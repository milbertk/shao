-- =====================================================================
-- AlephK — "Notify the customer" per tracking step (2026-10-31).
-- Run in the SHAO database BEFORE starting the new shao version.
-- Safe to run twice.
--
-- tbl_step_template_step.notify_customer: when an order item moves to this
-- step, the customer receives an email ONLY if TRUE. Every step starts
-- UNCHECKED (FALSE): check the ones the customer should know about in
-- "Processes" (step tracking). Order status changes (placed, in progress,
-- partly done, done) are always emailed, as before.
-- =====================================================================
ALTER TABLE public.tbl_step_template_step
    ADD COLUMN IF NOT EXISTS notify_customer BOOLEAN NOT NULL DEFAULT FALSE;

-- Check:
--   SELECT t.name AS process, s.step_order, s.name AS step, s.notify_customer
--   FROM public.tbl_step_template_step s
--   JOIN public.tbl_step_template t ON t.step_template_id = s.step_template_id
--   ORDER BY t.name, s.step_order;
-- Check one step by hand (normally done in the Processes screen):
--   UPDATE public.tbl_step_template_step SET notify_customer = TRUE WHERE step_template_step_id = '<id>';
-- REVERT (also restore the previous shao / p3 versions):
--   ALTER TABLE public.tbl_step_template_step DROP COLUMN IF EXISTS notify_customer;
