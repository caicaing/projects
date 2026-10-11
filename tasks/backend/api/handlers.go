package api

import (
	"net/http"
	"time"

	"github.com/caicaing/projects/tasks/framework"
)

func (s *server) previewSchedule(w http.ResponseWriter, r *http.Request) {
	var schedule taskflow.Schedule
	if !decode(w, r, &schedule) {
		return
	}
	times, err := taskflow.PreviewSchedule(schedule, time.Now())
	result(w, 200, map[string]any{"times": times}, err)
}

func (s *server) state(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, s.engine.Snapshot())
}

func (s *server) catalog(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, s.engine.Catalog())
}

func (s *server) saveTask(w http.ResponseWriter, r *http.Request) {
	var t taskflow.Task
	if !decode(w, r, &t) {
		return
	}
	id := r.PathValue("id")
	t, err := s.engine.SaveTask(r.Context(), id, t)
	status := 200
	if id == "" {
		status = 201
	}
	result(w, status, t, err)
}

func (s *server) deleteTask(w http.ResponseWriter, r *http.Request) {
	err := s.engine.DeleteTask(r.Context(), r.PathValue("id"))
	result(w, 200, map[string]bool{"ok": true}, err)
}

func (s *server) trigger(w http.ResponseWriter, r *http.Request) {
	run, err := s.engine.Trigger(r.Context(), r.PathValue("id"))
	result(w, 202, run, err)
}

func (s *server) cancelRun(w http.ResponseWriter, r *http.Request) {
	err := s.engine.CancelRun(r.PathValue("id"))
	result(w, 200, map[string]bool{"ok": true}, err)
}

func (s *server) saveChannel(w http.ResponseWriter, r *http.Request) {
	var c taskflow.Channel
	if !decode(w, r, &c) {
		return
	}
	id := r.PathValue("id")
	c, err := s.engine.SaveChannel(r.Context(), id, c)
	status := 200
	if id == "" {
		status = 201
	}
	result(w, status, c, err)
}

func (s *server) deleteChannel(w http.ResponseWriter, r *http.Request) {
	err := s.engine.DeleteChannel(r.Context(), r.PathValue("id"))
	result(w, 200, map[string]bool{"ok": true}, err)
}

func (s *server) testChannel(w http.ResponseWriter, r *http.Request) {
	err := s.engine.TestChannel(r.Context(), r.PathValue("id"))
	result(w, 200, map[string]bool{"ok": true}, err)
}
