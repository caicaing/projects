package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	taskflow "github.com/caicaing/projects/tasks/framework"
)

func (store *Store) load(ctx context.Context, s *taskflow.State) error {
	for _, table := range storeTables {
		query := "SELECT " + strings.Join(table.columns, ",") +
			" FROM " + table.name + " ORDER BY position"
		rows, err := store.db.QueryContext(ctx, query)
		if err != nil {
			return err
		}
		for rows.Next() {
			if err := scanRecord(rows, table.name, s); err != nil {
				_ = rows.Close()
				return fmt.Errorf("读取 %s 失败: %w", table.name, err)
			}
		}
		err = rows.Err()
		_ = rows.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func scanRecord(rows *sql.Rows, table string, s *taskflow.State) error {
	var position int
	var created, updated string
	var optional sql.NullString
	switch table {
	case "tasks":
		var t taskflow.Task
		var config, schedule string
		if err := rows.Scan(&t.ID, &position, &t.Name, &t.Description, &t.Type,
			&config, &schedule, &t.Enabled, &t.Timeout,
			&t.Retries, &t.RetryDelay,
			&t.NotifyOn, &created, &updated, &optional); err != nil {
			return err
		}
		t.Config, t.NotifierIDs = json.RawMessage(config), []string{}
		if err := json.Unmarshal([]byte(schedule), &t.Schedule); err != nil {
			return err
		}
		if err := parseStoredTimes(created, updated, optional,
			&t.CreatedAt, &t.UpdatedAt, &t.NextRun); err != nil {
			return err
		}
		s.Tasks = append(s.Tasks, t)
	case "channels":
		var ch taskflow.Channel
		var config string
		if err := rows.Scan(&ch.ID, &position, &ch.Name, &ch.Type, &config,
			&ch.Enabled, &created, &updated); err != nil {
			return err
		}
		ch.Config = json.RawMessage(config)
		var unused *time.Time
		if err := parseStoredTimes(created, updated, optional,
			&ch.CreatedAt, &ch.UpdatedAt, &unused); err != nil {
			return err
		}
		s.Channels = append(s.Channels, ch)
	case "task_channels":
		var taskID, channelID string
		if err := rows.Scan(&taskID, &channelID, &position); err != nil {
			return err
		}
		for i := range s.Tasks {
			if s.Tasks[i].ID == taskID {
				s.Tasks[i].NotifierIDs = append(
					s.Tasks[i].NotifierIDs, channelID)
				break
			}
		}
	case "runs":
		var r taskflow.Run
		if err := rows.Scan(&r.ID, &position, &r.TaskID,
			&r.TaskName, &r.Trigger,
			&r.Status, &created, &optional, &r.DurationMS, &r.Attempts,
			&r.Output, &r.Error); err != nil {
			return err
		}
		var unused time.Time
		if err := parseStoredTimes(created, created, optional,
			&r.StartedAt, &unused, &r.FinishedAt); err != nil {
			return err
		}
		r.Deliveries = []taskflow.Delivery{}
		s.Runs = append(s.Runs, r)
	case "deliveries":
		var runID string
		var d taskflow.Delivery
		if err := rows.Scan(&runID, &d.ChannelID, &position,
			&d.ChannelName, &d.Status, &d.Error); err != nil {
			return err
		}
		for i := range s.Runs {
			if s.Runs[i].ID == runID {
				s.Runs[i].Deliveries = append(s.Runs[i].Deliveries, d)
				break
			}
		}
	}
	return nil
}

func parseStoredTimes(first, second string, optional sql.NullString,
	a, b *time.Time, c **time.Time) error {
	var err error
	if *a, err = time.Parse(time.RFC3339Nano, first); err != nil {
		return err
	}
	if *b, err = time.Parse(time.RFC3339Nano, second); err != nil {
		return err
	}
	if optional.Valid {
		value, err := time.Parse(time.RFC3339Nano, optional.String)
		if err != nil {
			return err
		}
		*c = &value
	}
	return nil
}
