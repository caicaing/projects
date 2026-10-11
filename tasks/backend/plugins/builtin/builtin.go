// Package builtin provides optional HTTP, logging and Webhook plugins.
package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/caicaing/projects/tasks/framework"
)

func Register(r *taskflow.Registry) error {
	for _, constructor := range []func() taskflow.TaskHandler{
		func() taskflow.TaskHandler { return LogTask{} },
		func() taskflow.TaskHandler { return HTTPTask{} },
	} {
		if err := r.Tasks.Register(constructor); err != nil {
			return err
		}
	}
	for _, constructor := range []func() taskflow.Notifier{
		func() taskflow.Notifier { return LogNotifier{} },
		func() taskflow.Notifier { return WebhookNotifier{} },
	} {
		if err := r.Notifiers.Register(constructor); err != nil {
			return err
		}
	}
	return nil
}

type LogTask struct{}

type logConfig struct {
	Message string `json:"message"`
	Delay   string `json:"delay,omitempty"`
}

func (LogTask) Spec() taskflow.Spec {
	return taskflow.Spec{
		Type: "log", Name: "日志任务",
		Description: "输出一条日志，可用于验证调度或演示长任务。",
		Example: json.RawMessage(`{"message":"Hello, Taskflow!",` +
			`"delay":"0s"}`),
	}
}

func (LogTask) Validate(raw json.RawMessage) error {
	var c logConfig
	if err := taskflow.DecodeConfig(raw, &c); err != nil {
		return err
	}
	if strings.TrimSpace(c.Message) == "" {
		return fmt.Errorf("message 不能为空")
	}
	if c.Delay != "" {
		_, err := validDuration(c.Delay, 0, time.Hour)
		return err
	}
	return nil
}

func (LogTask) Run(ctx context.Context, raw json.RawMessage) (string, error) {
	var c logConfig
	if err := taskflow.DecodeConfig(raw, &c); err != nil {
		return "", err
	}
	delay, _ := time.ParseDuration(c.Delay)
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-timer.C:
		slog.Info("日志任务", "message", c.Message)
		return c.Message, nil
	}
}

type HTTPTask struct{}

type requestConfig struct {
	URL            string            `json:"url"`
	Method         string            `json:"method,omitempty"`
	Headers        map[string]string `json:"headers,omitempty"`
	Body           string            `json:"body,omitempty"`
	ExpectedStatus int               `json:"expected_status,omitempty"`
}

func (HTTPTask) Spec() taskflow.Spec {
	return taskflow.Spec{
		Type: "http", Name: "HTTP 请求",
		Description: "调用 HTTP 接口；默认将 2xx 响应视为成功。",
		Example: json.RawMessage(`{"url":"https://example.com",` +
			`"method":"GET","headers":{},"body":""}`),
	}
}

func (HTTPTask) Validate(raw json.RawMessage) error {
	var c requestConfig
	if err := taskflow.DecodeConfig(raw, &c); err != nil {
		return err
	}
	if err := validateURL(c.URL); err != nil {
		return err
	}
	if c.Method == "" {
		c.Method = "GET"
	}
	allowed := " GET POST PUT PATCH DELETE HEAD OPTIONS "
	if !strings.Contains(allowed, " "+c.Method+" ") {
		return fmt.Errorf("不支持的 HTTP method")
	}
	if c.ExpectedStatus != 0 &&
		(c.ExpectedStatus < 100 || c.ExpectedStatus > 599) {
		return fmt.Errorf("expected_status 须为有效状态码")
	}
	_, err := makeRequest(context.Background(), c, []byte(c.Body))
	return err
}

func (HTTPTask) Run(ctx context.Context, raw json.RawMessage) (string, error) {
	var c requestConfig
	if err := taskflow.DecodeConfig(raw, &c); err != nil {
		return "", err
	}
	if c.Method == "" {
		c.Method = "GET"
	}
	return doRequest(ctx, c, []byte(c.Body))
}

type LogNotifier struct{}

func (LogNotifier) Spec() taskflow.Spec {
	return taskflow.Spec{
		Type: "log", Name: "控制台通知",
		Description: "将通知写入服务端日志，无需额外配置。",
		Example:     json.RawMessage(`{}`),
	}
}

func (LogNotifier) Validate(raw json.RawMessage) error {
	return taskflow.DecodeConfig(raw, &struct{}{})
}

func (LogNotifier) Send(ctx context.Context, _ json.RawMessage,
	n taskflow.Notification) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	slog.Info("任务通知", "title", n.Title, "message", n.Message)
	return nil
}

type WebhookNotifier struct{}

type webhookConfig struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

func (WebhookNotifier) Spec() taskflow.Spec {
	return taskflow.Spec{
		Type: "webhook", Name: "Webhook",
		Description: "以 JSON POST 发送通知，支持自定义请求头。",
		Example: json.RawMessage(`{"url":"https://example.com/webhook",` +
			`"headers":{}}`),
	}
}

func (WebhookNotifier) Validate(raw json.RawMessage) error {
	var c webhookConfig
	if err := taskflow.DecodeConfig(raw, &c); err != nil {
		return err
	}
	if err := validateURL(c.URL); err != nil {
		return err
	}
	r := requestConfig{URL: c.URL, Method: "POST", Headers: c.Headers}
	_, err := makeRequest(context.Background(), r, nil)
	return err
}

func (WebhookNotifier) Send(ctx context.Context, raw json.RawMessage,
	n taskflow.Notification) error {
	var c webhookConfig
	if err := taskflow.DecodeConfig(raw, &c); err != nil {
		return err
	}
	body, err := json.Marshal(n)
	if err != nil {
		return err
	}
	r := requestConfig{URL: c.URL, Method: "POST", Headers: c.Headers}
	_, err = doRequest(ctx, r, body)
	return err
}

func validateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil ||
		(u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("url 必须是无用户凭证的 http 或 https 地址")
	}
	return nil
}

func makeRequest(ctx context.Context, c requestConfig,
	body []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, c.Method, c.URL,
		bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range c.Headers {
		if strings.ContainsAny(key+value, "\r\n") ||
			strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("请求头格式不正确")
		}
		req.Header.Set(key, value)
	}
	return req, nil
}

var requestClient = &http.Client{
	Timeout: 24 * time.Hour,
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

func doRequest(ctx context.Context, c requestConfig,
	body []byte) (string, error) {
	req, err := makeRequest(ctx, c, body)
	if err != nil {
		return "", err
	}
	resp, err := requestClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 65537))
	if err != nil {
		return "", err
	}
	output := fmt.Sprintf("HTTP %d\n%s", resp.StatusCode, b)
	ok := resp.StatusCode >= 200 && resp.StatusCode < 300
	if c.ExpectedStatus != 0 {
		ok = resp.StatusCode == c.ExpectedStatus
	}
	if !ok {
		return output, fmt.Errorf("HTTP 响应状态 %d", resp.StatusCode)
	}
	return truncate(output, 65536), nil
}

func validDuration(raw string, min, max time.Duration) (time.Duration, error) {
	d, err := time.ParseDuration(raw)
	if err != nil || d < min || d > max {
		return 0, fmt.Errorf("时长须介于 %s 和 %s", min, max)
	}
	return d, nil
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "\n…输出已截断"
}
