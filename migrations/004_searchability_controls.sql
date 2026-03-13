ALTER TABLE memory_items ADD COLUMN searchable INTEGER NOT NULL DEFAULT 1;
ALTER TABLE memory_items ADD COLUMN exclusion_reason TEXT NOT NULL DEFAULT '';

ALTER TABLE handoffs ADD COLUMN searchable INTEGER NOT NULL DEFAULT 1;
ALTER TABLE handoffs ADD COLUMN exclusion_reason TEXT NOT NULL DEFAULT '';

DELETE FROM memory_items_fts;

INSERT INTO memory_items_fts(rowid, title, content)
SELECT rowid, title, content
FROM memory_items
WHERE searchable = 1;

DROP TRIGGER IF EXISTS memory_items_ai;
DROP TRIGGER IF EXISTS memory_items_ad;
DROP TRIGGER IF EXISTS memory_items_au;

CREATE TRIGGER IF NOT EXISTS memory_items_ai AFTER INSERT ON memory_items
WHEN new.searchable = 1
BEGIN
    INSERT INTO memory_items_fts(rowid, title, content)
    VALUES (new.rowid, new.title, new.content);
END;

CREATE TRIGGER IF NOT EXISTS memory_items_ad AFTER DELETE ON memory_items BEGIN
    INSERT INTO memory_items_fts(memory_items_fts, rowid, title, content)
    VALUES ('delete', old.rowid, old.title, old.content);
END;

CREATE TRIGGER IF NOT EXISTS memory_items_au AFTER UPDATE ON memory_items BEGIN
    INSERT INTO memory_items_fts(memory_items_fts, rowid, title, content)
    VALUES ('delete', old.rowid, old.title, old.content);
    INSERT INTO memory_items_fts(rowid, title, content)
    SELECT new.rowid, new.title, new.content
    WHERE new.searchable = 1;
END;
