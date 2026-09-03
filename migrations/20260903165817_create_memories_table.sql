-- +goose Up
CREATE TABLE IF NOT EXISTS memories (
  id            INTEGER PRIMARY KEY,
  text          TEXT NOT NULL,
  project       TEXT NOT NULL DEFAULT '',
  tags          TEXT NOT NULL DEFAULT '[]',
  source        TEXT NOT NULL DEFAULT '',
  created_at    TEXT NOT NULL,
  updated_at    TEXT NOT NULL,
  expires_at    TEXT    
);

-- таблица fts5 индекса
CREATE VIRTUAL TABLE memories_fts USING fts5(
    text,
    content=memories,
    content_rowid=rowid
);

-- заполнение memories_fts таблицы
CREATE TRIGGER memories_ai AFTER INSERT ON memories BEGIN
    INSERT INTO memories_fts(rowid, text) VALUES (new.rowid, new.text);
END;

CREATE TRIGGER memories_ad AFTER DELETE ON memories BEGIN
    INSERT INTO memories_fts(memories_fts, rowid, text)
        VALUES('delete', old.rowid, old.text);
END;

CREATE TRIGGER memories_au AFTER UPDATE ON memories BEGIN
    INSERT INTO memories_fts(memories_fts, rowid, text)
        VALUES('delete', old.rowid, old.text);
    INSERT INTO memories_fts(rowid, text) VALUES (new.rowid, new.text);
END;

CREATE INDEX idx_memories_project ON memories(project);
CREATE INDEX idx_memories_expires ON memories(expires_at)
    WHERE expires_at IS NOT NULL;
-- +goose Down
DROP INDEX IF EXISTS idx_memories_project;
DROP INDEX IF EXISTS idx_memories_expires;
DROP TRIGGER IF EXISTS memories_au;
DROP TRIGGER IF EXISTS memories_ad;
DROP TRIGGER IF EXISTS memories_ai;
DROP TABLE IF EXISTS memories_fts;
DROP TABLE IF EXISTS memories;