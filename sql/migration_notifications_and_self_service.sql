-- =====================================================================
-- Notifications, company SMTP and customer self-service (2026-10-22)
-- Run ONCE, before starting the new shao version.
-- Safe to run twice (IF NOT EXISTS everywhere).
-- =====================================================================

-- 1) Company page: notification email, WhatsApp and email language.
--    notification_language: 'en' (default) or 'es'.
ALTER TABLE public.tbl_page_configuration
    ADD COLUMN IF NOT EXISTS notification_email    VARCHAR(254),
    ADD COLUMN IF NOT EXISTS contact_whatsapp      VARCHAR(30),
    ADD COLUMN IF NOT EXISTS notification_language VARCHAR(5) NOT NULL DEFAULT 'en';

-- 2) Company's own SMTP (emails are sent with the company's mail server).
--    The password is stored ENCRYPTED (AES-GCM) by shao; never in clear text.
CREATE TABLE IF NOT EXISTS public.tbl_company_smtp (
    company_smtp_id    UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id         VARCHAR(100) NOT NULL UNIQUE,
    host               VARCHAR(255) NOT NULL,
    port               INTEGER      NOT NULL,
    security           VARCHAR(10)  NOT NULL DEFAULT 'STARTTLS', -- STARTTLS | SSL
    username           VARCHAR(255) NOT NULL,
    password_encrypted TEXT         NOT NULL,
    from_email         VARCHAR(254) NOT NULL,
    from_name          VARCHAR(120),
    enabled            BOOLEAN      NOT NULL DEFAULT TRUE,
    last_test_at       TIMESTAMPTZ,
    last_test_ok       BOOLEAN,
    last_test_error    TEXT,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- 3) Optional customer email on internal orders (for status notifications).
ALTER TABLE public.tbl_order
    ADD COLUMN IF NOT EXISTS customer_email VARCHAR(254);

-- 4) Private "manage my request" links (random token per record).
ALTER TABLE public.tbl_appointment       ADD COLUMN IF NOT EXISTS public_token VARCHAR(64);
ALTER TABLE public.tbl_web_order_request ADD COLUMN IF NOT EXISTS public_token VARCHAR(64);
ALTER TABLE public.tbl_order             ADD COLUMN IF NOT EXISTS public_token VARCHAR(64);

CREATE UNIQUE INDEX IF NOT EXISTS ux_appointment_public_token
    ON public.tbl_appointment (public_token) WHERE public_token IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS ux_web_order_public_token
    ON public.tbl_web_order_request (public_token) WHERE public_token IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS ux_order_public_token
    ON public.tbl_order (public_token) WHERE public_token IS NOT NULL;

-- 5) Faster slot checks for bookings.
CREATE INDEX IF NOT EXISTS ix_appointment_slot
    ON public.tbl_appointment (company_id, service_id, appointment_date);

-- 6) Notification log: what was sent, to whom, and whether it failed.
CREATE TABLE IF NOT EXISTS public.tbl_notification_log (
    notification_id UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id      VARCHAR(100) NOT NULL,
    entity_type     VARCHAR(30)  NOT NULL,   -- WEB_ORDER | ORDER | APPOINTMENT | SMTP_TEST
    entity_id       VARCHAR(100) NOT NULL,
    event           VARCHAR(40)  NOT NULL,   -- CREATED | APPROVED | CANCELLED ...
    audience        VARCHAR(20)  NOT NULL,   -- CUSTOMER | COMPANY
    recipient       VARCHAR(254),
    status          VARCHAR(20)  NOT NULL,   -- SENT | FAILED | SKIPPED
    error_message   TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS ix_notification_log_entity
    ON public.tbl_notification_log (company_id, entity_type, entity_id);

-- 7) Base URL of the public store, used for the links inside emails.
--    Change it when you deploy (e.g. https://store.yourdomain.com).
INSERT INTO public.tbl_configuration (parameter, value)
SELECT 'SHOWCASE_BASE_URL', 'http://192.168.100.110:3001'
WHERE NOT EXISTS (
    SELECT 1 FROM public.tbl_configuration WHERE parameter = 'SHOWCASE_BASE_URL'
);

-- ---------------------------------------------------------------------
-- Checks:
--   SELECT company_id, notification_email, notification_language FROM public.tbl_page_configuration;
--   SELECT * FROM public.tbl_notification_log ORDER BY created_at DESC LIMIT 20;
--
-- Revert (only after reverting the Go code):
--   DROP TABLE IF EXISTS public.tbl_notification_log;
--   DROP TABLE IF EXISTS public.tbl_company_smtp;
--   DROP INDEX IF EXISTS public.ix_appointment_slot;
--   DROP INDEX IF EXISTS public.ux_order_public_token;
--   DROP INDEX IF EXISTS public.ux_web_order_public_token;
--   DROP INDEX IF EXISTS public.ux_appointment_public_token;
--   ALTER TABLE public.tbl_order             DROP COLUMN IF EXISTS public_token, DROP COLUMN IF EXISTS customer_email;
--   ALTER TABLE public.tbl_web_order_request DROP COLUMN IF EXISTS public_token;
--   ALTER TABLE public.tbl_appointment       DROP COLUMN IF EXISTS public_token;
--   ALTER TABLE public.tbl_page_configuration
--       DROP COLUMN IF EXISTS notification_email,
--       DROP COLUMN IF EXISTS contact_whatsapp,
--       DROP COLUMN IF EXISTS notification_language;
--   DELETE FROM public.tbl_configuration WHERE parameter = 'SHOWCASE_BASE_URL';
-- =====================================================================
