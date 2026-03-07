ALTER TABLE books
    ADD COLUMN is_published BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN published_at TIMESTAMPTZ NULL,
    ADD COLUMN unpublished_at TIMESTAMPTZ NULL;

UPDATE books
SET unpublished_at = created_at
WHERE unpublished_at IS NULL;

ALTER TABLE bundles
    ADD COLUMN is_published BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN published_at TIMESTAMPTZ NULL,
    ADD COLUMN unpublished_at TIMESTAMPTZ NULL;

UPDATE bundles
SET unpublished_at = created_at
WHERE unpublished_at IS NULL;
