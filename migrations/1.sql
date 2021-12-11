CREATE TABLE  users (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL UNIQUE CHECK(length(name) <= 256),
  email TEXT NOT NULL CHECK(length(email) <= 32),
    UNIQUE (email COLLATE NOCASE)
);
