CREATE TABLE bundles (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL DEFAULT '',
    supplier_id BIGINT NOT NULL REFERENCES suppliers(id),
    category TEXT NOT NULL,
    allowed_conditions TEXT[] NOT NULL,
    bundle_price NUMERIC(12, 2) NOT NULL CHECK (bundle_price >= 0),
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE bundle_books (
    bundle_id BIGINT NOT NULL REFERENCES bundles(id) ON DELETE CASCADE,
    book_id BIGINT NOT NULL REFERENCES books(id),
    position INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (bundle_id, book_id)
);

CREATE INDEX idx_bundles_supplier_id ON bundles(supplier_id);
CREATE INDEX idx_bundles_category ON bundles(category);
CREATE INDEX idx_bundle_books_bundle_id ON bundle_books(bundle_id);
CREATE INDEX idx_bundle_books_book_id ON bundle_books(book_id);
