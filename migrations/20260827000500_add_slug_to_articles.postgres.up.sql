ALTER TABLE articles ADD COLUMN IF NOT EXISTS slug TEXT UNIQUE;

-- Backfill slug untuk artikel yang sudah ada di database
UPDATE articles 
SET slug = LOWER(REGEXP_REPLACE(REGEXP_REPLACE(TRIM(title), '[^a-zA-Z0-9\s-]', '', 'g'), '\s+', '-', 'g'))
WHERE slug IS NULL OR slug = '';
