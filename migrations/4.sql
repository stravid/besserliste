CREATE TABLE products (
  id INTEGER PRIMARY KEY,
  category_id INTEGER NOT NULL,
  name TEXT NOT NULL UNIQUE CHECK(length(name) <= 40),
  dimension TEXT NOT NULL CHECK(dimension IN ('dimensionless', 'mass', 'volume')),
  FOREIGN KEY(category_id) REFERENCES categories(id)
);

CREATE TABLE product_changes (
  id INTEGER PRIMARY KEY,
  product_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  category_id INTEGER NOT NULL,
  name TEXT NOT NULL,
  dimension TEXT NOT NULL,
  recorded_at DATETIME NOT NULL,
  FOREIGN KEY(user_id) REFERENCES users(id),
  FOREIGN KEY(product_id) REFERENCES products(id),
  FOREIGN KEY(category_id) REFERENCES categories(id)
);

CREATE TABLE items (
  id INTEGER PRIMARY KEY,
  product_id INTEGER NOT NULL,
  quantity INTEGER NOT NULL CHECK(quantity > 0 AND quantity <= 10000),
  state TEXT NOT NULL CHECK(state IN ('added', 'gathered', 'removed')),
  FOREIGN KEY(product_id) REFERENCES products(id)
);

CREATE TABLE item_changes (
  id INTEGER PRIMARY KEY,
  item_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  quantity INTEGER NOT NULL,
  state TEXT NOT NULL,
  recorded_at DATETIME NOT NULL,
  FOREIGN KEY(item_id) REFERENCES items(id),
  FOREIGN KEY(user_id) REFERENCES users(id)
);

CREATE TABLE idempotency_keys (
  key TEXT PRIMARY KEY CHECK(length(key) = 32),
  processed_at DATETIME NOT NULL
);
