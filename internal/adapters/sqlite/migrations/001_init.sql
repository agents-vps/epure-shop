PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;

CREATE TABLE IF NOT EXISTS users (
    id          TEXT PRIMARY KEY,
    email       TEXT UNIQUE NOT NULL,
    name        TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role        TEXT NOT NULL DEFAULT 'customer' CHECK (role IN ('customer', 'admin')),
    created_at  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS categories (
    id        TEXT PRIMARY KEY,
    slug      TEXT UNIQUE NOT NULL,
    name      TEXT NOT NULL,
    parent_id TEXT REFERENCES categories(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS discounts (
    id         TEXT PRIMARY KEY,
    code       TEXT UNIQUE NOT NULL,
    percent    INTEGER NOT NULL CHECK (percent >= 0 AND percent <= 100),
    active     INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    expires_at TEXT
);

CREATE TABLE IF NOT EXISTS products (
    id            TEXT PRIMARY KEY,
    slug          TEXT UNIQUE NOT NULL,
    category_id   TEXT NOT NULL REFERENCES categories(id),
    name          TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    price         INTEGER NOT NULL,
    compare_price INTEGER,
    stock         INTEGER NOT NULL DEFAULT 0,
    status        TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published')),
    image_url     TEXT NOT NULL DEFAULT '',
    rating        REAL NOT NULL DEFAULT 0,
    review_count  INTEGER NOT NULL DEFAULT 0,
    created_at    TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS carts (
    id          TEXT PRIMARY KEY,
    user_id     TEXT REFERENCES users(id) ON DELETE CASCADE,
    discount_id TEXT REFERENCES discounts(id) ON DELETE SET NULL,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS cart_items (
    cart_id    TEXT NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
    product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    quantity   INTEGER NOT NULL CHECK (quantity > 0),
    PRIMARY KEY (cart_id, product_id)
);

CREATE TABLE IF NOT EXISTS orders (
    id               TEXT PRIMARY KEY,
    ref              TEXT UNIQUE NOT NULL,
    user_id          TEXT REFERENCES users(id) ON DELETE SET NULL,
    email            TEXT NOT NULL,
    status           TEXT NOT NULL DEFAULT 'pending',
    subtotal         INTEGER NOT NULL,
    shipping_cost    INTEGER NOT NULL DEFAULT 0,
    discount_amount  INTEGER NOT NULL DEFAULT 0,
    total            INTEGER NOT NULL,
    shipping_address TEXT NOT NULL DEFAULT '{}',
    created_at       TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS order_items (
    id         TEXT PRIMARY KEY,
    order_id   TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id TEXT NOT NULL,
    name       TEXT NOT NULL,
    unit_price INTEGER NOT NULL,
    quantity   INTEGER NOT NULL CHECK (quantity > 0),
    image_url  TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS sessions (
    token_hash TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS idempotency (
    key        TEXT PRIMARY KEY,
    order_id   TEXT NOT NULL,
    created_at TEXT NOT NULL
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_products_category   ON products(category_id);
CREATE INDEX IF NOT EXISTS idx_products_slug        ON products(slug);
CREATE INDEX IF NOT EXISTS idx_products_status      ON products(status);
CREATE INDEX IF NOT EXISTS idx_cart_items_cart      ON cart_items(cart_id);
CREATE INDEX IF NOT EXISTS idx_orders_user          ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_ref           ON orders(ref);
CREATE INDEX IF NOT EXISTS idx_order_items_order    ON order_items(order_id);
