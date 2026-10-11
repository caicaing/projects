package api

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"git.woa.com/mooyliu/mooy_test/framework"
)

type server struct {
	engine *taskflow.Engine
	token  string
}

func New(engine *taskflow.Engine, token string) http.Handler {
	s := &server{engine: engine, token: token}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/state", s.state)
	mux.HandleFunc("GET /api/catalog", s.catalog)
	mux.HandleFunc("POST /api/tasks", s.saveTask)
	mux.HandleFunc("PUT /api/tasks/{id}", s.saveTask)
	mux.HandleFunc("DELETE /api/tasks/{id}", s.deleteTask)
	mux.HandleFunc("POST /api/tasks/{id}/run", s.trigger)
	mux.HandleFunc("POST /api/runs/{id}/cancel", s.cancelRun)
	mux.HandleFunc("POST /api/channels", s.saveChannel)
	mux.HandleFunc("PUT /api/channels/{id}", s.saveChannel)
	mux.HandleFunc("DELETE /api/channels/{id}", s.deleteChannel)
	mux.HandleFunc("POST /api/channels/{id}/test", s.testChannel)
	mux.HandleFunc("POST /api/schedules/preview", s.previewSchedule)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter,
		_ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	return s.guard(mux)
}

func (s *server) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		policy := "default-src 'self'; script-src 'self'; " +
			"style-src 'self'; img-src 'self' data:; " +
			"connect-src 'self'; frame-ancestors 'none'; base-uri 'none'"
		w.Header().Set("Content-Security-Policy", policy)
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
			if s.token != "" && !validToken(r, s.token) {
				writeJSON(w, 401, map[string]string{"error": "请输入管理令牌"})
				return
			}
			if r.Method != http.MethodGet && !sameOrigin(r) {
				writeJSON(w, 403, map[string]string{"error": "禁止跨站操作"})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func validToken(r *http.Request, token string) bool {
	want := sha256.Sum256([]byte("Bearer " + token))
	got := sha256.Sum256([]byte(r.Header.Get("Authorization")))
	return subtle.ConstantTimeCompare(want[:], got[:]) == 1
}

func sameOrigin(r *http.Request) bool {
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		return false
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	return err == nil && u.Host == r.Host &&
		(u.Scheme == "https" || u.Scheme == "http")
}

func decode(w http.ResponseWriter, r *http.Request, value any) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeJSON(w, 415, map[string]string{"error": "请使用 application/json"})
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(value); err != nil {
		message := "JSON 参数错误: " + err.Error()
		writeJSON(w, 400, map[string]string{"error": message})
		return false
	}
	if err := d.Decode(new(any)); err != io.EOF {
		writeJSON(w, 400, map[string]string{"error": "只能提交一个 JSON 对象"})
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func result(w http.ResponseWriter, status int, value any, err error) {
	if err == nil {
		writeJSON(w, status, value)
		return
	}
	status = http.StatusBadRequest
	switch {
	case errors.Is(err, taskflow.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, taskflow.ErrNotStarted),
		errors.Is(err, taskflow.ErrClosed):
		status = http.StatusServiceUnavailable
	case errors.Is(err, taskflow.ErrBusy):
		status = http.StatusConflict
	case errors.Is(err, taskflow.ErrStorage):
		status = http.StatusInternalServerError
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
