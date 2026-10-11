package model

import (
	"encoding/json"
	"time"
)

type Spec struct {
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Example     json.RawMessage `json:"example"`
}

type Schedule struct {
	Kind       string `json:"kind"`
	Expression string `json:"expression"`
	Timezone   string `json:"timezone"`
	Time       string `json:"time,omitempty"`
	Minute     int    `json:"minute,omitempty"`
	Weekdays   []int  `json:"weekdays,omitempty"`
	Day        int    `json:"day,omitempty"`
	Every      int    `json:"every,omitempty"`
	Unit       string `json:"unit,omitempty"`
}

type Task struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Type        string          `json:"type"`
	Config      json.RawMessage `json:"config"`
	Schedule    Schedule        `json:"schedule"`
	Enabled     bool            `json:"enabled"`
	Timeout     string          `json:"timeout"`
	Retries     int             `json:"retries"`
	RetryDelay  string          `json:"retry_delay"`
	NotifyOn    string          `json:"notify_on"`
	NotifierIDs []string        `json:"notifier_ids"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	NextRun     *time.Time      `json:"next_run"`
}

type Channel struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Type      string          `json:"type"`
	Config    json.RawMessage `json:"config"`
	Enabled   bool            `json:"enabled"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type Delivery struct {
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	Status      string `json:"status"`
	Error       string `json:"error,omitempty"`
}

type Run struct {
	ID         string     `json:"id"`
	TaskID     string     `json:"task_id"`
	TaskName   string     `json:"task_name"`
	Trigger    string     `json:"trigger"`
	Status     string     `json:"status"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
	DurationMS int64      `json:"duration_ms"`
	Attempts   int        `json:"attempts"`
	Output     string     `json:"output"`
	Error      string     `json:"error,omitempty"`
	Deliveries []Delivery `json:"deliveries"`
}

type Notification struct {
	Title   string `json:"title"`
	Message string `json:"message"`
	Run     Run    `json:"run"`
}

type State struct {
	Version  int       `json:"version"`
	Tasks    []Task    `json:"tasks"`
	Channels []Channel `json:"channels"`
	Runs     []Run     `json:"runs"`
}

type Catalog struct {
	Tasks     []Spec `json:"tasks"`
	Notifiers []Spec `json:"notifiers"`
}
