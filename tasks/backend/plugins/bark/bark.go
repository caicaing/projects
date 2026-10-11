package bark

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/caicaing/projects/tasks/framework"
)

const defaultBarkServer = "https://api.day.app"

// Bark sends notifications through the Bark JSON push API.
type Bark struct{}

// Register makes Bark available in a registry without sending a request.
func Register(r *taskflow.Registry) error {
	return r.Notifiers.Register(func() taskflow.Notifier { return Bark{} })
}

type barkConfig struct {
	ServerURL  string   `json:"server_url,omitempty"`
	DeviceKey  string   `json:"device_key,omitempty"`
	DeviceKeys []string `json:"device_keys,omitempty"`
	Group      string   `json:"group,omitempty"`
	Sound      string   `json:"sound,omitempty"`
}

type barkPush struct {
	DeviceKey string `json:"device_key"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Group     string `json:"group,omitempty"`
	Sound     string `json:"sound,omitempty"`
}

var barkClient = &http.Client{
	Timeout: 10 * time.Second,
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

func (Bark) Spec() taskflow.Spec {
	return taskflow.Spec{
		Type: "bark", Name: "Bark 手机通知",
		Description: "通过 Bark 推送到 iPhone；device_keys 可配置多个设备，" +
			"兼容单个 device_key，支持自建服务器、group 和 sound。",
		Example: json.RawMessage(`{"server_url":"https://api.day.app",` +
			`"device_keys":["DEVICE_KEY_1","DEVICE_KEY_2"],` +
			`"group":"Taskflow","sound":"minuet"}`),
	}
}

func (Bark) Validate(raw json.RawMessage) error {
	_, _, err := decodeBarkConfig(raw)
	return err
}

func decodeBarkConfig(raw json.RawMessage) (barkConfig, string, error) {
	var c barkConfig
	if err := taskflow.DecodeConfig(raw, &c); err != nil {
		return c, "", err
	}
	keys, err := barkDeviceKeys(c)
	if err != nil {
		return c, "", err
	}
	c.DeviceKeys = keys
	if c.ServerURL == "" {
		c.ServerURL = defaultBarkServer
	}
	u, err := url.Parse(c.ServerURL)
	if err != nil || u.Hostname() == "" || u.User != nil ||
		(u.Scheme != "http" && u.Scheme != "https") ||
		u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return c, "", fmt.Errorf("server_url 必须是无凭证、" +
			"查询参数和片段的 http 或 https 服务器地址")
	}
	endpoint, err := url.JoinPath(u.String(), "push")
	if err != nil {
		return c, "", fmt.Errorf("server_url 格式不正确")
	}
	return c, endpoint, nil
}

// barkDeviceKeys normalizes both formats without duplicate deliveries.
func barkDeviceKeys(c barkConfig) ([]string, error) {
	if c.DeviceKey != "" && len(c.DeviceKeys) > 0 {
		return nil, fmt.Errorf("device_key 与 device_keys 只能填写一个")
	}
	keys := c.DeviceKeys
	field := "device_keys"
	if c.DeviceKey != "" {
		keys = []string{c.DeviceKey}
		field = "device_key"
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("请填写 device_key 或非空 device_keys")
	}
	seen := make(map[string]bool, len(keys))
	result := make([]string, 0, len(keys))
	for i, key := range keys {
		if key == "" || strings.ContainsFunc(key, unicode.IsSpace) {
			return nil, fmt.Errorf("%s 第 %d 项不能为空或包含空白字符",
				field, i+1)
		}
		if !seen[key] {
			seen[key] = true
			result = append(result, key)
		}
	}
	return result, nil
}

func (Bark) Send(ctx context.Context, raw json.RawMessage,
	n taskflow.Notification) error {
	c, endpoint, err := decodeBarkConfig(raw)
	if err != nil {
		return err
	}
	if len(c.DeviceKeys) == 1 {
		return sendBarkPush(ctx, endpoint, c.DeviceKeys[0], c, n)
	}
	// All workers share the caller's deadline; one failed device does not
	// stop delivery to the others. Each result slot has exactly one writer.
	results := make([]error, len(c.DeviceKeys))
	workers := min(4, len(c.DeviceKeys))
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func(start int) {
			defer wg.Done()
			for i := start; i < len(c.DeviceKeys); i += workers {
				if err := ctx.Err(); err != nil {
					results[i] = err
					continue
				}
				results[i] = sendBarkPush(ctx, endpoint, c.DeviceKeys[i], c, n)
			}
		}(worker)
	}
	wg.Wait()
	return barkBatchResult(ctx, results)
}

func barkBatchResult(ctx context.Context, results []error) error {
	failed := 0
	var details []error
	for i, err := range results {
		if err == nil {
			continue
		}
		failed++
		// Bound diagnostics and identify devices by ordinal, never by key.
		if len(details) < 10 {
			details = append(details, fmt.Errorf("设备 #%d: %w", i+1, err))
		}
	}
	if failed == 0 {
		return nil
	}
	if failed > 10 {
		details = append(details, fmt.Errorf("另有 %d 个设备失败", failed-10))
	}
	if ctx.Err() != nil {
		details = append(details, ctx.Err())
	}
	return fmt.Errorf("Bark 批量投递失败（成功 %d/%d）: %w",
		len(results)-failed, len(results), errors.Join(details...))
}

func sendBarkPush(ctx context.Context, endpoint, key string,
	c barkConfig, n taskflow.Notification) error {
	body, err := json.Marshal(barkPush{
		DeviceKey: key, Title: n.Title, Body: n.Message,
		Group: c.Group, Sound: c.Sound,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := barkClient.Do(req)
	if err != nil {
		return fmt.Errorf("Bark 请求失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Bark HTTP 响应状态 %d", resp.StatusCode)
	}
	return decodeBarkResponse(resp.Body)
}

func decodeBarkResponse(body io.Reader) error {
	const maxResponse = 64 << 10
	data, err := io.ReadAll(io.LimitReader(body, maxResponse+1))
	if err != nil {
		return fmt.Errorf("读取 Bark 响应失败: %w", err)
	}
	if len(data) > maxResponse {
		return fmt.Errorf("Bark 响应超过 64 KiB")
	}
	var result struct {
		Code    *int   `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("Bark 响应不是有效 JSON")
	}
	if result.Code == nil {
		return fmt.Errorf("Bark 响应缺少 code")
	}
	if *result.Code != 200 {
		return fmt.Errorf("Bark 推送失败（code=%d）: %s",
			*result.Code, result.Message)
	}
	return nil
}
