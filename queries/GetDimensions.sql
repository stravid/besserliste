SELECT
      id,
      name,
      json_group_array(json(unit)) AS units
    FROM (
      SELECT
        dimensions.id AS id,
        name,
        json_object(
          'id', units.id,
          'name_singular', units.name_singular,
          'name_plural', units.name_plural,
          'conversion_to_base', units.conversion_to_base,
          'conversion_from_base', units.conversion_from_base
        ) AS unit
      FROM dimensions
      INNER JOIN units ON dimensions.id = units.dimension_id
      ORDER BY dimensions.ordering, units.ordering ASC
    )
    GROUP BY id
    LIMIT 10;
