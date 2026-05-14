CREATE TABLE IF NOT EXISTS products (
    id         SERIAL PRIMARY KEY,
    name       TEXT           NOT NULL,
    price      NUMERIC(10, 2) NOT NULL,
    created_at TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS inventory (
    id         SERIAL PRIMARY KEY,
    product_id INT NOT NULL UNIQUE REFERENCES products(id),
    quantity   INT NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_inventory_product ON inventory(product_id);
