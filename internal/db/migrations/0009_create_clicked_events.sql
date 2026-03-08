CREATE TABLE clicked_events (
    id BIGSERIAL PRIMARY KEY,
    item_id BIGINT NOT NULL,
    item_type TEXT NOT NULL CHECK (item_type IN ('BOOK', 'BUNDLE')),
    item_title TEXT NOT NULL,
    source_page TEXT NOT NULL,
    source_rail_id BIGINT NOT NULL DEFAULT 0,
    source_rail_title TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_clicked_events_created_at ON clicked_events(created_at DESC);
CREATE INDEX idx_clicked_events_item_type_item_id ON clicked_events(item_type, item_id);
