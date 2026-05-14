CREATE TABLE IF NOT EXISTS users (
    id         SERIAL PRIMARY KEY,
    name       TEXT        NOT NULL,
    email      TEXT        NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS products (
    id         SERIAL PRIMARY KEY,
    name       TEXT           NOT NULL,
    price      NUMERIC(10, 2) NOT NULL,
    created_at TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS orders (
    id         SERIAL PRIMARY KEY,
    user_id    INT            NOT NULL REFERENCES users(id),
    total      NUMERIC(10, 2) NOT NULL DEFAULT 0,
    status     TEXT           NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS order_items (
    id         SERIAL PRIMARY KEY,
    order_id   INT            NOT NULL REFERENCES orders(id),
    product_id INT            NOT NULL REFERENCES products(id),
    quantity   INT            NOT NULL CHECK (quantity > 0),
    price      NUMERIC(10, 2) NOT NULL
);

CREATE TABLE IF NOT EXISTS inventory (
    id         SERIAL PRIMARY KEY,
    product_id INT NOT NULL UNIQUE REFERENCES products(id),
    quantity   INT NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_orders_user_id    ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_order_items_order ON order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_inventory_product ON inventory(product_id);
