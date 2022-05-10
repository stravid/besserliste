INSERT INTO categories_products (category_id, product_id)
SELECT categories.id, products.id FROM products, categories
WHERE products.name_singular = 'Faschiertes' AND categories.name IN ('Kühlregal', 'Theke')
ON CONFLICT (categories_products.category_id, categories_products.product_id) DO NOTHING;

INSERT INTO categories_products (category_id, product_id)
SELECT categories.id, products.id FROM products, categories
WHERE products.name_singular = 'Frühstücksspeck' AND categories.name IN ('Kühlregal', 'Theke')
ON CONFLICT (categories_products.category_id, categories_products.product_id) DO NOTHING;

INSERT INTO categories_products (category_id, product_id)
SELECT categories.id, products.id FROM products, categories
WHERE products.name_singular = 'Neuburger' AND categories.name IN ('Kühlregal', 'Theke')
ON CONFLICT (categories_products.category_id, categories_products.product_id) DO NOTHING;

INSERT INTO categories_products (category_id, product_id)
SELECT categories.id, products.id FROM products, categories
WHERE products.name_singular = 'Scheibenkäse' AND categories.name IN ('Kühlregal', 'Theke')
ON CONFLICT (categories_products.category_id, categories_products.product_id) DO NOTHING;
