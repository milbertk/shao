-- Customer-facing page configuration: exactly one row per company.
-- company_id uses VARCHAR so it accepts the same string ids the other
-- tables receive from the API; add a FK to your companies table if desired.
CREATE TABLE IF NOT EXISTS public.tbl_page_configuration (
    page_configuration_id UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id            VARCHAR(100) NOT NULL UNIQUE,
    display_name          VARCHAR(120) NOT NULL,
    font_family           VARCHAR(60),
    logo_url              TEXT,
    logo_path             TEXT,
    header_url            TEXT,
    header_path           TEXT,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT now()
);
