SELECT
      id,
      name,
      json_group_array(json(dimension)) AS dimensions
    FROM (
      SELECT
        product_id AS id,
        product_name AS name,
        json_object(
          'id', dimension_id,
          'name', dimension_name,
          'units', json(units)
        ) AS dimension
      FROM (
        SELECT
          product_id,
          product_name,
          dimension_id,
          dimension_name,
          json_group_array(json(unit)) AS units
        FROM (
          SELECT
            products.id AS product_id,
            products.name_plural AS product_name,
            dimensions.id AS dimension_id,
            dimensions.name AS dimension_name,
            json_object(
              'id', units.id,
              'name_singular', units.name_singular,
              'name_plural', units.name_plural,
              'conversion_to_base', units.conversion_to_base,
              'conversion_from_base', units.conversion_from_base
            ) AS unit
          FROM products
          INNER JOIN dimensions_products ON products.id = dimensions_products.product_id
          INNER JOIN dimensions ON dimensions_products.dimension_id = dimensions.id
          INNER JOIN units ON dimensions.id = units.dimension_id
          WHERE products.id = ?
          ORDER BY dimensions.ordering, units.ordering ASC
        )
        GROUP BY product_id, product_name, dimension_id, dimension_name
      )
    )
    GROUP BY id, name;
