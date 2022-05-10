INSERT INTO dimensions (id, name, ordering) VALUES (7, 'Glas', 7);
INSERT INTO dimensions (id, name, ordering) VALUES (8, 'Becher', 8);

INSERT INTO units (
  dimension_id,
  name_singular,
  name_plural,
  conversion_to_base,
  conversion_from_base,
  ordering
) VALUES
  (7, 'Glas', 'Gläser', 1.0, 1.0, 1),
  (8, 'Becher', 'Becher', 1.0, 1.0, 1)
;

INSERT INTO dimensions_products (dimension_id, product_id)
SELECT dimensions.id, products.id FROM products, dimensions
WHERE products.name_singular = 'Joghurt' AND dimensions.name IN ('Glas', 'Becher')
ON CONFLICT (dimensions_products.dimension_id, dimensions_products.product_id) DO NOTHING;

INSERT INTO dimensions_products (dimension_id, product_id)
SELECT dimensions.id, products.id FROM products, dimensions
WHERE products.name_singular = 'Sahne' AND dimensions.name IN ('Becher')
ON CONFLICT (dimensions_products.dimension_id, dimensions_products.product_id) DO NOTHING;
