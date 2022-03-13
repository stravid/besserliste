CREATE TABLE  users (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL CHECK(length(name) <= 256) COLLATE de_AT,
  email TEXT NOT NULL CHECK(length(email) <= 32) COLLATE NOCASE
);

CREATE UNIQUE INDEX idx_users_name ON users(lower(name, 'de_AT'));
CREATE UNIQUE INDEX idx_users_email ON users(email);

CREATE TABLE categories (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL CHECK(length(name) <= 20) COLLATE de_AT,
  ordering INTEGER NOT NULL
);

CREATE UNIQUE INDEX idx_categories_name ON categories(lower(name, 'de_AT'));
CREATE UNIQUE INDEX idx_categories_email ON categories(ordering);

CREATE TABLE dimensions (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL CHECK(length(name) <= 20) COLLATE de_AT,
  ordering INTEGER NOT NULL
);

CREATE UNIQUE INDEX idx_dimensions_name ON dimensions(lower(name, 'de_AT'));
CREATE UNIQUE INDEX idx_dimensions_email ON dimensions(ordering);

CREATE TABLE units (
  id INTEGER PRIMARY KEY,
  dimension_id INTEGER NOT NULL,
  name_singular TEXT NOT NULL CHECK(length(name_singular) <= 20) COLLATE de_AT,
  name_plural TEXT NOT NULL CHECK(length(name_plural) <= 20) COLLATE de_AT,
  conversion_to_base DECIMAL NOT NULL CHECK(conversion_to_base > 0),
  conversion_from_base DECIMAL NOT NULL CHECK(conversion_from_base > 0),
  ordering INTEGER NOT NULL,
  FOREIGN KEY(dimension_id) REFERENCES dimensions(id)
);

CREATE UNIQUE INDEX idx_units_conversion_to_base ON units(dimension_id, conversion_to_base);
CREATE UNIQUE INDEX idx_units_conversion_from_base ON units(dimension_id, conversion_from_base);
CREATE UNIQUE INDEX idx_units_ordering ON units(ordering, dimension_id);
CREATE UNIQUE INDEX idx_units_name_singular ON units(lower(name_singular, 'de_AT'));
CREATE UNIQUE INDEX idx_units_name_plural ON units(lower(name_plural, 'de_AT'));

CREATE TABLE products (
  id INTEGER PRIMARY KEY,
  category_id INTEGER NOT NULL,
  name_singular TEXT NOT NULL CHECK(length(name_singular) <= 40) COLLATE de_AT,
  name_plural TEXT NOT NULL CHECK(length(name_plural) <= 40) COLLATE de_AT,
  FOREIGN KEY(category_id) REFERENCES categories(id)
);

CREATE UNIQUE INDEX idx_products_name_singular ON products(lower(name_singular, 'de_AT'));
CREATE UNIQUE INDEX idx_products_name_plural ON products(lower(name_plural, 'de_AT'));

CREATE TABLE product_changes (
  id INTEGER PRIMARY KEY,
  product_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  category_id INTEGER NOT NULL,
  name_singular TEXT NOT NULL,
  name_plural TEXT NOT NULL,
  recorded_at DATETIME NOT NULL,
  FOREIGN KEY(user_id) REFERENCES users(id),
  FOREIGN KEY(product_id) REFERENCES products(id),
  FOREIGN KEY(category_id) REFERENCES categories(id)
);

CREATE TABLE items (
  id INTEGER PRIMARY KEY,
  product_id INTEGER NOT NULL,
  dimension_id INTEGER NOT NULL,
  quantity INTEGER NOT NULL CHECK(quantity > 0 AND quantity <= 10000),
  state TEXT NOT NULL CHECK(state IN ('added', 'gathered', 'removed')),
  FOREIGN KEY(product_id) REFERENCES products(id),
  FOREIGN KEY(dimension_id) REFERENCES dimensions(id)
);

CREATE UNIQUE INDEX idx_items_added ON items(state, product_id, dimension_id) WHERE state = 'added';

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

CREATE TABLE dimensions_products (
  dimension_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  FOREIGN KEY(dimension_id) REFERENCES dimensions(id),
  FOREIGN KEY(product_id) REFERENCES products(id)
);

CREATE UNIQUE INDEX idx_dimensions_products ON dimensions_products(dimension_id, product_id);

CREATE TABLE idempotency_keys (
  key TEXT PRIMARY KEY CHECK(length(key) = 32),
  processed_at DATETIME NOT NULL
);
