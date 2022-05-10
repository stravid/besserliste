CREATE TABLE categories_products (
  category_id INTEGER NOT NULL,
  product_id INTEGER NOT NULL,
  FOREIGN KEY(category_id) REFERENCES categories(id),
  FOREIGN KEY(product_id) REFERENCES products(id)
);

CREATE UNIQUE INDEX idx_categories_products ON categories_products(category_id, product_id);

INSERT INTO categories_products (category_id, product_id) SELECT category_id, id FROM products;

PRAGMA foreign_keys=OFF;
CREATE TABLE new_products (
  id INTEGER PRIMARY KEY,
  name_singular TEXT NOT NULL CHECK(length(name_singular) <= 40) COLLATE de_AT,
  name_plural TEXT NOT NULL CHECK(length(name_plural) <= 40) COLLATE de_AT
);
INSERT INTO new_products (id, name_singular, name_plural) SELECT id, name_singular, name_plural FROM products;
DROP TABLE products;
ALTER TABLE new_products RENAME TO products;
CREATE UNIQUE INDEX idx_products_name_singular ON products(lower(name_singular, 'de_AT'));
CREATE UNIQUE INDEX idx_products_name_plural ON products(lower(name_plural, 'de_AT'));
PRAGMA foreign_key_check;
PRAGMA foreign_keys=ON;

PRAGMA foreign_keys=OFF;
CREATE TABLE new_product_changes (
  id INTEGER PRIMARY KEY,
  product_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  name_singular TEXT NOT NULL,
  name_plural TEXT NOT NULL,
  recorded_at DATETIME NOT NULL,
  FOREIGN KEY(user_id) REFERENCES users(id),
  FOREIGN KEY(product_id) REFERENCES products(id)
);
INSERT INTO new_product_changes (id, product_id, user_id, name_singular, name_plural, recorded_at) SELECT id, product_id, user_id, name_singular, name_plural, recorded_at FROM product_changes;
DROP TABLE product_changes;
ALTER TABLE new_product_changes RENAME TO product_changes;
PRAGMA foreign_key_check;
PRAGMA foreign_keys=ON;
