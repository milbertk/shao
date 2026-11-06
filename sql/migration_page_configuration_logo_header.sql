-- Replaces the single cover image of tbl_page_configuration with two images:
-- a square logo and a rectangular header. Run once on databases created with
-- the previous image_url / image_path columns.
-- Existing cover images are dropped: their storage paths do not match the new
-- logo/ and header/ folders, so companies must upload them again.
ALTER TABLE public.tbl_page_configuration
    DROP COLUMN IF EXISTS image_url,
    DROP COLUMN IF EXISTS image_path,
    ADD COLUMN IF NOT EXISTS logo_url    TEXT,
    ADD COLUMN IF NOT EXISTS logo_path   TEXT,
    ADD COLUMN IF NOT EXISTS header_url  TEXT,
    ADD COLUMN IF NOT EXISTS header_path TEXT;
