package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	taskflow "github.com/caicaing/projects/tasks/framework"
)

func storedTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339Nano)
}

func stateRecords(s taskflow.State) (map[string]map[string][]any, error) {
	result := make(map[string]map[string][]any, len(storeTables))
	for _, table := range storeTables {
		result[table.name] = make(map[string][]any)
	}
	add := func(table string, values ...any) {
		key := values[0].(string)
		if table == "task_channels" || table == "deliveries" {
			key += "\x00" + values[1].(string)
		}
		result[table][key] = values
	}
	for i, t := range s.Tasks {
		config, err := json.Marshal(t.Config)
		if err != nil {
			return nil, err
		}
		schedule, err := json.Marshal(t.Schedule)
		if err != nil {
			return nil, err
		}
		add("tasks", t.ID, i, t.Name, t.Description, t.Type,
			string(config), string(schedule),
			t.Enabled, t.Timeout, t.Retries, t.RetryDelay, t.NotifyOn,
			t.CreatedAt.Format(time.RFC3339Nano),
			t.UpdatedAt.Format(time.RFC3339Nano), storedTime(t.NextRun))
		for j, id := range t.NotifierIDs {
			add("task_channels", t.ID, id, j)
		}
	}
	for i, ch := range s.Channels {
		config, err := json.Marshal(ch.Config)
		if err != nil {
			return nil, err
		}
		add("channels", ch.ID, i, ch.Name, ch.Type, string(config), ch.Enabled,
			ch.CreatedAt.Format(time.RFC3339Nano),
			ch.UpdatedAt.Format(time.RFC3339Nano))
	}
	for i, r := range s.Runs {
		add("runs", r.ID, i, r.TaskID, r.TaskName, r.Trigger, r.Status,
			r.StartedAt.Format(time.RFC3339Nano), storedTime(r.FinishedAt),
			r.DurationMS, r.Attempts, r.Output, r.Error)
		for j, d := range r.Deliveries {
			add("deliveries", r.ID, d.ChannelID, j,
				d.ChannelName, d.Status, d.Error)
		}
	}
	return result, nil
}

func (store *Store) Save(ctx context.Context, s taskflow.State) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.closed {
		return fmt.Errorf("SQLite 存储已关闭")
	}
	if s.Version != 1 {
		return fmt.Errorf("不支持的数据版本: %d", s.Version)
	}
	next, err := stateRecords(s)
	if err != nil {
		return err
	}
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Remove children before parents; insert parents before their associations.
	for i := len(storeTables) - 1; i >= 0; i-- {
		table := storeTables[i]
		conditions := make([]string, table.keys)
		for j, column := range table.columns[:table.keys] {
			conditions[j] = column + " = ?"
		}
		for key, values := range store.current[table.name] {
			if _, exists := next[table.name][key]; !exists {
				query := "DELETE FROM " + table.name + " WHERE " +
					strings.Join(conditions, " AND ")
				if _, err := tx.ExecContext(ctx, query,
					values[:table.keys]...); err != nil {
					return err
				}
			}
		}
	}
	for _, table := range storeTables {
		updates := make([]string, 0, len(table.columns)-table.keys)
		for _, column := range table.columns[table.keys:] {
			updates = append(updates, column+" = excluded."+column)
		}
		columns := strings.Join(table.columns, ",")
		values := strings.TrimSuffix(
			strings.Repeat("?,", len(table.columns)), ",")
		keys := strings.Join(table.columns[:table.keys], ",")
		query := "INSERT INTO " + table.name + " (" + columns +
			") VALUES (" + values + ") ON CONFLICT (" + keys +
			") DO UPDATE SET " + strings.Join(updates, ",")
		for key, values := range next[table.name] {
			if reflect.DeepEqual(values, store.current[table.name][key]) {
				continue
			}
			if _, err := tx.ExecContext(ctx, query, values...); err != nil {
				return err
			}
		}
	}
	if _, err := tx.ExecContext(ctx, "PRAGMA user_version = 1"); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	store.current = next
	return nil
}
