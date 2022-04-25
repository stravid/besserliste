INSERT INTO users (
  name,
  email
) VALUES
  ('Hannah', 'hannah@longhail.com'),
  ('David', 'david@strauss.io')
;

INSERT INTO categories (
  name,
  ordering
) VALUES
  ('Obst & Gemüse', 1),
  ('Kühlregal', 2),
  ('Theke', 3),
  ('Verpackt', 4),
  ('Getränke', 5),
  ('Tiefkühlregal', 6),
  ('Haushalt', 7),
  ('Sonstiges', 8)
;

INSERT INTO dimensions (
  id,
  name,
  ordering
) VALUES
  (1, 'Stück', 1),
  (2, 'Gewicht', 2),
  (3, 'Volumen', 3),
  (4, 'Dosen', 4),
  (5, 'Flaschen', 5),
  (6, 'Packung', 6)
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
  (2, 'deg', 'deg', 10.0, 0.1, 2),
  (2, 'kg', 'kg', 1000.0, 0.001, 3),
  (3, 'ml', 'ml', 1.0, 1.0, 1),
  (3, 'l', 'l', 1000.0, 0.001, 2),
  (4, 'Dose', 'Dosen', 1.0, 1.0, 1),
  (5, 'Flasche', 'Flaschen', 1.0, 1.0, 1),
  (6, 'Packung', 'Packungen', 1.0, 1.0, 1)
;
