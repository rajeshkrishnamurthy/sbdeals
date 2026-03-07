CREATE TABLE rails (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL UNIQUE,
    rail_type TEXT NOT NULL CHECK (rail_type IN ('BOOK', 'BUNDLE')),
    position INTEGER NOT NULL,
    is_published BOOLEAN NOT NULL DEFAULT FALSE,
    published_at TIMESTAMPTZ NULL,
    unpublished_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE rail_items (
    rail_id BIGINT NOT NULL REFERENCES rails(id) ON DELETE CASCADE,
    item_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (rail_id, item_id)
);

CREATE INDEX idx_rails_position ON rails(position);
CREATE INDEX idx_rail_items_rail_id ON rail_items(rail_id);
