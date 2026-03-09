ALTER TABLE enquiries
    ADD COLUMN IF NOT EXISTS order_amount BIGINT NULL;

ALTER TABLE enquiries
    DROP CONSTRAINT IF EXISTS enquiries_status_check;

ALTER TABLE enquiries
    DROP CONSTRAINT IF EXISTS chk_enquiries_status;

ALTER TABLE enquiries
    ADD CONSTRAINT chk_enquiries_status
    CHECK (status IN ('clicked', 'interested', 'ordered'));

ALTER TABLE enquiries
    DROP CONSTRAINT IF EXISTS chk_enquiries_interested_customer_required;

ALTER TABLE enquiries
    DROP CONSTRAINT IF EXISTS chk_enquiries_customer_required;

ALTER TABLE enquiries
    ADD CONSTRAINT chk_enquiries_customer_required
    CHECK (status NOT IN ('interested', 'ordered') OR customer_id IS NOT NULL);

ALTER TABLE enquiries
    DROP CONSTRAINT IF EXISTS chk_enquiries_order_amount_required;

ALTER TABLE enquiries
    ADD CONSTRAINT chk_enquiries_order_amount_required
    CHECK (status <> 'ordered' OR (order_amount IS NOT NULL AND order_amount > 0));
