-- PostgreSQL schema (applied automatically on start by pgstore.go).
CREATE TABLE IF NOT EXISTS products (
    id          INTEGER PRIMARY KEY,
    slug        TEXT    NOT NULL UNIQUE,
    name        TEXT    NOT NULL,
    platform    TEXT    NOT NULL,
    region      TEXT    NOT NULL,
    type        TEXT    NOT NULL,
    genre       TEXT    NOT NULL,
    year        INTEGER NOT NULL,
    msrp_cents  BIGINT  NOT NULL,
    colors      JSONB   NOT NULL DEFAULT '[]',
    rating      DOUBLE PRECISION NOT NULL DEFAULT 0,
    sold        INTEGER NOT NULL DEFAULT 0,
    description TEXT    NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS offers (
    product_id  INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    seller      TEXT    NOT NULL,
    rating      DOUBLE PRECISION NOT NULL,
    reviews     INTEGER NOT NULL,
    price_cents BIGINT  NOT NULL CHECK (price_cents > 0),
    stock       INTEGER NOT NULL CHECK (stock >= 0),
    delivery    TEXT    NOT NULL DEFAULT 'Instant',
    PRIMARY KEY (product_id, seller)
);

CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    name          TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS orders (
    id          TEXT PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL,
    subtotal_cents BIGINT NOT NULL,
    fee_cents   BIGINT NOT NULL,
    total_cents BIGINT NOT NULL,
    payment_id  TEXT NOT NULL,
    items       JSONB NOT NULL -- snapshot of lines incl. issued keys
);
CREATE INDEX IF NOT EXISTS orders_user_idx ON orders (user_id, created_at DESC);
