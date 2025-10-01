-- migrations/000000_create_users_table.up.sql
CREATE TABLE usuarios (
    id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    nome TEXT,
    email TEXT
);