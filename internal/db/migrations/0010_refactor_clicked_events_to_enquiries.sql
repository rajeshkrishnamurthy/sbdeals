ALTER TABLE clicked_events RENAME TO enquiries;

ALTER TABLE enquiries
    ADD COLUMN status TEXT NOT NULL DEFAULT 'clicked' CHECK (status IN ('clicked', 'interested')),
    ADD COLUMN buyer_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN buyer_phone TEXT NOT NULL DEFAULT '',
    ADD COLUMN buyer_note TEXT NOT NULL DEFAULT '',
    ADD COLUMN converted_by TEXT NOT NULL DEFAULT '',
    ADD COLUMN converted_at TIMESTAMPTZ NULL;

CREATE INDEX idx_enquiries_status_created_at ON enquiries(status, created_at DESC);
