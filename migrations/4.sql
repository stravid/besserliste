CREATE TABLE dimensions (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL UNIQUE CHECK(length(name) <= 20),
  ordering INTEGER NOT NULL
);

CREATE TABLE units (
  id INTEGER PRIMARY KEY,
  dimension_id INTEGER NOT NULL,
  name_singular TEXT NOT NULL UNIQUE CHECK(length(name_singular) <= 20),
  name_plural TEXT NOT NULL UNIQUE CHECK(length(name_plural) <= 20),
  conversion_to_base DECIMAL NOT NULL CHECK(conversion_to_base > 0),
  conversion_from_base DECIMAL NOT NULL CHECK(conversion_from_base > 0),
  ordering INTEGER NOT NULL,
  FOREIGN KEY(dimension_id) REFERENCES dimensions(id)
);

CREATE UNIQUE INDEX idx_units_to_base ON units(dimension_id, conversion_to_base);
CREATE UNIQUE INDEX idx_units_from_base ON units(dimension_id, conversion_from_base);
CREATE UNIQUE INDEX idx_ordering_units ON units(ordering, dimension_id);

CREATE TABLE products (
  id INTEGER PRIMARY KEY,
  category_id INTEGER NOT NULL,
  name_singular TEXT NOT NULL UNIQUE CHECK(length(name_singular) <= 40),
  name_plural TEXT NOT NULL UNIQUE CHECK(length(name_plural) <= 40),
  FOREIGN KEY(category_id) REFERENCES categories(id)
);

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

CREATE UNIQUE INDEX idx_added_items ON items(state, product_id, dimension_id) WHERE state = 'added';

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

INSERT INTO dimensions (
  id,
  name,
  ordering
) VALUES
  (1, 'Stück', 1),
  (2, 'Gewicht', 2),
  (3, 'Volumen', 3),
  (4, 'Dosen', 4),
  (5, 'Flaschen', 5)
;

INSERT INTO units (
  dimension_id,
  name_singular,
  name_plural,
  conversion_to_base,
  conversion_from_base,
  ordering
) VALUES
  (1, 'Stück', 'Stück', 1.0, 1.0, 1),
  (2, 'g', 'g', 1.0, 1.0, 1),
  (2, 'kg', 'kg', 1000.0, 0.001, 2),
  (3, 'ml', 'ml', 1.0, 1.0, 1),
  (3, 'l', 'l', 1000.0, 0.001, 2),
  (4, 'Dose', 'Dosen', 1.0, 1.0, 1),
  (5, 'Flasche', 'Flaschen', 1.0, 1.0, 1)
;
