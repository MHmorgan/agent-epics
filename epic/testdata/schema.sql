PRAGMA foreign_keys = ON;

-- task: one row per task node in the epic tree
CREATE TABLE IF NOT EXISTS task (
    id         TEXT PRIMARY KEY,
    parent_id  TEXT,
    title      TEXT,
    body       TEXT,
    context    TEXT,
    status     TEXT,
    position   INTEGER,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (parent_id) REFERENCES task(id)
);

-- attribute: epic-level key-value pairs
CREATE TABLE IF NOT EXISTS attribute (
    attribute TEXT PRIMARY KEY,
    value     TEXT
);

-- record: append-only journal of task activity
CREATE TABLE IF NOT EXISTS record (
    id     INTEGER PRIMARY KEY AUTOINCREMENT,
    task   TEXT NOT NULL,
    ts     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    source TEXT NOT NULL,
    text   TEXT NOT NULL
);

-- dep: directed dependency edges between tasks
CREATE TABLE IF NOT EXISTS dep (
    task_id  TEXT NOT NULL,
    after_id TEXT NOT NULL,
    PRIMARY KEY (task_id, after_id),
    FOREIGN KEY (task_id)  REFERENCES task(id),
    FOREIGN KEY (after_id) REFERENCES task(id)
);

CREATE INDEX IF NOT EXISTS idx_task_parent  ON task(parent_id);
CREATE INDEX IF NOT EXISTS idx_task_status  ON task(status);
CREATE INDEX IF NOT EXISTS idx_record_task  ON record(task);
CREATE INDEX IF NOT EXISTS idx_dep_after    ON dep(after_id);
