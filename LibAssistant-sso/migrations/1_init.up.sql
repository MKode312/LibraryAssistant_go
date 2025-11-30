CREATE TABLE IF NOT EXISTS users
(
    id        BIGINT PRIMARY KEY,
    email     TEXT NOT NULL UNIQUE,
    pass_hash BYTEA NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_email ON users (email);