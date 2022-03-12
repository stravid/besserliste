CREATE TABLE dimensions (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL UNIQUE CHECK(length(name) <= 20),
  ordering INTEGER NOT NULL
);

CREATE TABLE units (
  id INTEGER PRIMARY KEY,
  dimension_id INTEGER NOT NULL,
  singular_name TEXT NOT NULL UNIQUE CHECK(length(singular_name) <= 20),
  plural_name TEXT NOT NULL UNIQUE CHECK(length(plural_name) <= 20),
  is_base_unit BOOLEAN NOT NULL,
  conversion_to_base DECIMAL NOT NULL CHECK(conversion_to_base > 0),
  conversion_from_base DECIMAL NOT NULL CHECK(conversion_from_base > 0),
  ordering INTEGER NOT NULL,
  FOREIGN KEY(dimension_id) REFERENCES dimensions(id)
);

CREATE UNIQUE INDEX idx_base_units ON units(dimension_id) WHERE is_base_unit = TRUE;
CREATE UNIQUE INDEX idx_ordering_units ON units(ordering, dimension_id);

CREATE TABLE products (
  id INTEGER PRIMARY KEY,
  category_id INTEGER NOT NULL,
  name TEXT NOT NULL UNIQUE CHECK(length(name) <= 40),
  FOREIGN KEY(category_id) REFERENCES categories(id)
);

CREATE TABLE product_changes (
  id INTEGER PRIMARY KEY,
  product_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  category_id INTEGER NOT NULL,
  name TEXT NOT NULL,
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
  singular_name,
  plural_name,
  is_base_unit,
  conversion_to_base,
  conversion_from_base,
  ordering
) VALUES
  (1, 'Stück', 'Stück', TRUE, 1.0, 1.0, 1),
  (2, 'g', 'g', TRUE, 1.0, 1.0, 1),
  (2, 'kg', 'kg', FALSE, 1000.0, 0.001, 2),
  (3, 'ml', 'ml', TRUE, 1.0, 1.0, 1),
  (3, 'l', 'l', FALSE, 1000.0, 0.001, 2),
  (4, 'Dose', 'Dosen', TRUE, 1.0, 1.0, 1),
  (5, 'Flasche', 'Flaschen', TRUE, 1.0, 1.0, 1)
;
