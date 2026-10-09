CREATE TABLE books (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    title        TEXT    NOT NULL,
    author       TEXT    NOT NULL,
    category     TEXT    NOT NULL DEFAULT '',
    description  TEXT    NOT NULL DEFAULT '',
    total_pages  INTEGER NOT NULL CHECK (total_pages > 0),
    pages_read   INTEGER NOT NULL DEFAULT 0 CHECK (pages_read >= 0),
    status       TEXT    NOT NULL DEFAULT 'want_to_read'
                 CHECK (status IN ('want_to_read','reading','completed','abandoned')),
    created_at   TEXT    NOT NULL,
    updated_at   TEXT    NOT NULL,
    CHECK (pages_read <= total_pages)
);

CREATE INDEX idx_books_status   ON books(status);
CREATE INDEX idx_books_category ON books(category);
CREATE INDEX idx_books_author   ON books(author);