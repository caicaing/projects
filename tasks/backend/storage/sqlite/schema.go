package sqlite

import "strings"

type storeTable struct {
	name    string
	columns []string
	keys    int
}

var storeTables = []storeTable{
	{"tasks", strings.Fields("id position name description type config " +
		"schedule enabled timeout retries retry_delay notify_on " +
		"created_at updated_at next_run"), 1},
	{"channels", strings.Fields("id position name type config enabled " +
		"created_at updated_at"), 1},
	{"task_channels", strings.Fields("task_id channel_id position"), 2},
	{"runs", strings.Fields("id position task_id task_name trigger status " +
		"started_at finished_at duration_ms attempts output error"), 1},
	{"deliveries", strings.Fields("run_id channel_id position " +
		"channel_name status error"), 2},
}

const storeSchema = `
CREATE TABLE IF NOT EXISTS tasks (
 id TEXT PRIMARY KEY, position INTEGER NOT NULL, name TEXT NOT NULL,
 description TEXT NOT NULL, type TEXT NOT NULL, config TEXT NOT NULL,
 schedule TEXT NOT NULL, enabled INTEGER NOT NULL, timeout TEXT NOT NULL,
 retries INTEGER NOT NULL, retry_delay TEXT NOT NULL, notify_on TEXT NOT NULL,
 created_at TEXT NOT NULL, updated_at TEXT NOT NULL, next_run TEXT
);
CREATE TABLE IF NOT EXISTS channels (
 id TEXT PRIMARY KEY, position INTEGER NOT NULL, name TEXT NOT NULL,
 type TEXT NOT NULL, config TEXT NOT NULL, enabled INTEGER NOT NULL,
 created_at TEXT NOT NULL, updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS task_channels (
 task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
 channel_id TEXT NOT NULL REFERENCES channels(id), position INTEGER NOT NULL,
 PRIMARY KEY (task_id, channel_id)
);
CREATE TABLE IF NOT EXISTS runs (
 id TEXT PRIMARY KEY, position INTEGER NOT NULL, task_id TEXT NOT NULL,
 task_name TEXT NOT NULL, trigger TEXT NOT NULL, status TEXT NOT NULL,
 started_at TEXT NOT NULL, finished_at TEXT, duration_ms INTEGER NOT NULL,
 attempts INTEGER NOT NULL, output TEXT NOT NULL, error TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS runs_task_id ON runs(task_id);
CREATE TABLE IF NOT EXISTS deliveries (
 run_id TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
 channel_id TEXT NOT NULL, position INTEGER NOT NULL,
 channel_name TEXT NOT NULL,
 status TEXT NOT NULL, error TEXT NOT NULL, PRIMARY KEY (run_id, channel_id)
);`
