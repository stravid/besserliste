SELECT
      item_id AS id,
      product_name_singular AS name_singular,
      product_name_plural AS name_plural,
      item_quantity AS quantity,
      product_id,
      json_object(
        'id', dimension_id,
        'name', dimension_name,
        'units', json(units)
      ) AS dimension
    FROM (
      SELECT
        item_id,
        item_quantity,
        product_id,
        product_name_singular,
        product_name_plural,
        dimension_id,
        dimension_name,
        json_group_array(json(unit)) AS units
      FROM (
        SELECT
          items.id AS item_id,
          items.quantity AS item_quantity,
          products.id AS product_id,
          products.name_singular AS product_name_singular,
          products.name_plural AS product_name_plural,
          dimensions.id AS dimension_id,
          dimensions.name AS dimension_name,
          json_object(
            'id', units.id,
            'name_singular', units.name_singular,
            'name_plural', units.name_plural,
            'conversion_to_base', units.conversion_to_base,
            'conversion_from_base', units.conversion_from_base
          ) AS unit
        FROM items
        INNER JOIN products ON items.product_id = products.id
        INNER JOIN dimensions ON items.dimension_id = dimensions.id
        INNER JOIN units ON dimensions.id = units.dimension_id
        WHERE items.product_id = ? AND items.dimension_id = ? AND items.state = 'added'
        ORDER BY dimensions.ordering, units.ordering ASC
      )
      GROUP BY item_id, item_quantity, product_id, product_name_singular, product_name_plural, dimension_id, dimension_name
    );
