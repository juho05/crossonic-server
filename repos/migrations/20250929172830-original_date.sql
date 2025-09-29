-- +migrate Up
ALTER TABLE albums ADD COLUMN release_date text;
ALTER TABLE albums ADD COLUMN original_date text;
ALTER TABLE albums ADD COLUMN version text;
UPDATE albums SET release_date = year, original_date = year;
ALTER TABLE albums DROP COLUMN year;

ALTER TABLE songs ADD COLUMN release_date text;
ALTER TABLE songs ADD COLUMN original_date text;
UPDATE songs SET release_date = year, original_date = year;
ALTER TABLE songs DROP COLUMN year;

INSERT INTO system (key, value) VALUES ('needs-full-scan', '1') ON CONFLICT (key) DO NOTHING;

-- +migrate Down
ALTER TABLE albums ADD COLUMN year int;
UPDATE albums SET year =
    CASE
        WHEN original_date IS NULL AND release_date IS NULL THEN NULL
        WHEN original_date IS NOT NULL THEN CAST(SPLIT_PART(original_date, '-', 1) AS int)
        ELSE CAST(SPLIT_PART(release_date, '-', 1) AS int)
    END;
ALTER TABLE albums DROP COLUMN release_date;
ALTER TABLE albums DROP COLUMN original_date;
ALTER TABLE albums DROP COLUMN version;

ALTER TABLE songs ADD COLUMN year int;
UPDATE songs SET year =
                      CASE
                          WHEN original_date IS NULL AND release_date IS NULL THEN NULL
                          WHEN original_date IS NOT NULL THEN CAST(SPLIT_PART(original_date, '-', 1) AS int)
                          ELSE CAST(SPLIT_PART(release_date, '-', 1) AS int)
                          END;
ALTER TABLE songs DROP COLUMN release_date;
ALTER TABLE songs DROP COLUMN original_date;
