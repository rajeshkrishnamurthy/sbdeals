CREATE TABLE books (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    cover_image BYTEA NOT NULL,
    cover_mime_type TEXT NOT NULL,
    supplier_id BIGINT NOT NULL REFERENCES suppliers(id),
    category TEXT NOT NULL,
    format TEXT NOT NULL,
    condition TEXT NOT NULL,
    mrp NUMERIC(12, 2) NOT NULL CHECK (mrp >= 0),
    my_price NUMERIC(12, 2) NOT NULL CHECK (my_price >= 0),
    bundle_price NUMERIC(12, 2) NULL CHECK (bundle_price IS NULL OR bundle_price >= 0),
    author TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    in_stock BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_books_supplier_id ON books(supplier_id);
CREATE INDEX idx_books_title ON books(title);
