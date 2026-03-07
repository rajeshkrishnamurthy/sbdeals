ALTER TABLE bundles
    ADD COLUMN IF NOT EXISTS bundle_image BYTEA,
    ADD COLUMN IF NOT EXISTS bundle_image_mime_type TEXT NOT NULL DEFAULT '';
