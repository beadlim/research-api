CREATE SCHEMA IF NOT EXISTS users_schema;
SET search_path = users_schema;

CREATE TABLE IF NOT EXISTS users (
    id         SERIAL PRIMARY KEY,
    name       TEXT        NOT NULL,
    email      TEXT        NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
