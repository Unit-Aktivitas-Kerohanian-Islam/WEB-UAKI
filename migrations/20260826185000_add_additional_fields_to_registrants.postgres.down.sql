ALTER TABLE registrants
DROP COLUMN IF EXISTS nickname,
DROP COLUMN IF EXISTS school_origin,
DROP COLUMN IF EXISTS origin_city,
DROP COLUMN IF EXISTS has_rohis_exp,
DROP COLUMN IF EXISTS twibbon_url;
