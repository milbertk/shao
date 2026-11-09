-- =====================================================================
-- shao — hours of one schedule may overlap in DIFFERENT locations
-- (2026-11-08, with "We can attend in more than one location at the same time")
--
-- Your database has exclusion constraints that refuse two blocks of hours of
-- the same schedule at overlapping times, for example:
--   ex_manual_no_overlap  ->  "conflicting key value violates exclusion constraint"
-- They were made before locations existed, so they also refuse Heredia
-- 09:00-12:00 + San Pedro 09:00-12:00. This script rebuilds every exclusion
-- constraint of tbl_booking_manual_slot and tbl_booking_recurring_range with
-- one more part: the location. After it:
--   * same location (or both without location / both "Customer chooses") -> still refused;
--   * two DIFFERENT locations -> allowed by the database. AlephK itself
--     decides with the company switch "same time" (OFF = refused, with a clear message).
--
-- The rest of each constraint (columns, time range, WHERE) is kept exactly
-- as it is. Safe to run twice: a constraint that already has the location is
-- skipped. Run on every shao database (your computer and the server), after
-- migration_company_locations.sql.
--
--   psql -h localhost -U <user> -d shao -v ON_ERROR_STOP=1 -f sql/migration_booking_overlap_by_location.sql
-- =====================================================================

-- 0) BEFORE: see the constraints (copy this output somewhere: it is your undo).
SELECT conrelid::regclass AS table_name, conname, pg_get_constraintdef(oid) AS definition
FROM pg_constraint
WHERE contype = 'x'
  AND conrelid IN ('public.tbl_booking_manual_slot'::regclass, 'public.tbl_booking_recurring_range'::regclass)
ORDER BY 1, 2;

BEGIN;

DO $$
DECLARE
  c        record;
  def      text;
  new_def  text;
  head     text;
  changed  int := 0;
BEGIN
  FOR c IN
    SELECT oid, conrelid::regclass AS tbl, conname
    FROM pg_constraint
    WHERE contype = 'x'
      AND conrelid IN ('public.tbl_booking_manual_slot'::regclass, 'public.tbl_booking_recurring_range'::regclass)
  LOOP
    def := pg_get_constraintdef(c.oid);
    IF def ILIKE '%location_id%' THEN
      RAISE NOTICE 'skip %.% (already has the location): %', c.tbl, c.conname, def;
      CONTINUE;
    END IF;
    -- "EXCLUDE USING gist (" + location part + the original elements.
    head := substring(def FROM '^EXCLUDE USING [a-z_]+ \(');
    IF head IS NULL THEN
      RAISE EXCEPTION 'unexpected definition of %.%: %', c.tbl, c.conname, def;
    END IF;
    new_def := head || 'COALESCE((location_id)::text, ''''::text) WITH =, ' || substring(def FROM length(head) + 1);
    EXECUTE format('ALTER TABLE %s DROP CONSTRAINT %I', c.tbl, c.conname);
    EXECUTE format('ALTER TABLE %s ADD CONSTRAINT %I %s', c.tbl, c.conname, new_def);
    RAISE NOTICE 'rebuilt %.%: %', c.tbl, c.conname, new_def;
    changed := changed + 1;
  END LOOP;
  RAISE NOTICE 'constraints rebuilt with the location: %', changed;
END $$;

COMMIT;

-- 1) AFTER: every exclusion constraint now starts with the location.
SELECT conrelid::regclass AS table_name, conname, pg_get_constraintdef(oid) AS definition
FROM pg_constraint
WHERE contype = 'x'
  AND conrelid IN ('public.tbl_booking_manual_slot'::regclass, 'public.tbl_booking_recurring_range'::regclass)
ORDER BY 1, 2;

-- UNDO (only if you go back): drop each constraint and add it again with the
-- definition printed in step 0, e.g.
--   ALTER TABLE public.tbl_booking_manual_slot DROP CONSTRAINT ex_manual_no_overlap;
--   ALTER TABLE public.tbl_booking_manual_slot ADD CONSTRAINT ex_manual_no_overlap <definition from step 0>;
-- (it fails while two rows in different locations overlap: change them first).
