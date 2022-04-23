SELECT
      id,
      name_singular,
      name_plural
    FROM products
    WHERE lower(name_singular, 'de_AT') = lower(?) OR lower(name_plural, 'de_AT') = lower(?)
    LIMIT 1
;
