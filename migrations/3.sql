CREATE TABLE categories (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL UNIQUE CHECK(length(name) <= 20),
  ordering INTEGER NOT NULL UNIQUE
);

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
