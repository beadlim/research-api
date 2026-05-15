CREATE TABLE IF NOT EXISTS orders (
    id         SERIAL PRIMARY KEY,
    user_id    INT            NOT NULL,
    total      NUMERIC(10, 2) NOT NULL DEFAULT 0,
    status     TEXT           NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS order_items (
    id         SERIAL PRIMARY KEY,
    order_id   INT            NOT NULL REFERENCES orders(id),
    product_id INT            NOT NULL,
    quantity   INT            NOT NULL CHECK (quantity > 0),
    price      NUMERIC(10, 2) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_orders_user_id    ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_order_items_order ON order_items(order_id);
