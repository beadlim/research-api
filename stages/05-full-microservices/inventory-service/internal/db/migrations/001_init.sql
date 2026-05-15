CREATE SCHEMA IF NOT EXISTS inventory_schema;
SET search_path = inventory_schema;

CREATE TABLE IF NOT EXISTS inventory (
    id         SERIAL PRIMARY KEY,
    product_id INT NOT NULL UNIQUE,
    quantity   INT NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_inventory_product ON inventory(product_id);
