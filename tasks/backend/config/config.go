// Package config loads application settings. The framework does not read files.
package config

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"git.woa.com/mooyliu/mooy_test/framework"
	"go.yaml.in/yaml/v3"
)

type Config struct {
	Backend   Backend   `yaml:"backend"`
	Scheduler Scheduler `yaml:"scheduler"`
}

type Backend struct {
	Addr     string `yaml:"addr"`
	Database string `yaml:"database"`
	Token    string `yaml:"token"`
}

type Scheduler struct {
	Workers             int           `yaml:"workers"`
	HistoryLimit        int           `yaml:"history_limit"`
	TaskTimeout         time.Duration `yaml:"task_timeout"`
	NotificationTimeout time.Duration `yaml:"notification_timeout"`
}

func (c Config) EngineOptions(store taskflow.Store) taskflow.Options {
	return taskflow.Options{
		Store: store, Workers: c.Scheduler.Workers,
		HistoryLimit:        c.Scheduler.HistoryLimit,
		DefaultTaskTimeout:  c.Scheduler.TaskTimeout,
		NotificationTimeout: c.Scheduler.NotificationTimeout,
	}
}

// Load resolves database paths relative to the configuration file.
// Values are used literally; configuration does not read the environment.
func Load(path string) (Config, error) {
	var c Config
	file, err := os.Open(path)
	if err != nil {
		return c, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, (1<<20)+1))
	if err != nil {
		return c, err
	}
	if len(data) > 1<<20 {
		return c, fmt.Errorf("conf.yaml 不能超过 1 MiB")
	}
	d := yaml.NewDecoder(bytes.NewReader(data))
	var node yaml.Node
	if err := d.Decode(&node); err != nil {
		return c, fmt.Errorf("读取配置失败: %w", err)
	}
	if err := d.Decode(new(yaml.Node)); err != io.EOF {
		return c, fmt.Errorf("conf.yaml 只能包含一个配置文档")
	}
	if err := validateNode(&node); err != nil {
		return c, err
	}
	data, err = yaml.Marshal(&node)
	if err != nil {
		return c, err
	}
	d = yaml.NewDecoder(bytes.NewReader(data))
	d.KnownFields(true)
	if err := d.Decode(&c); err != nil {
		return c, fmt.Errorf("配置字段错误: %w", err)
	}
	if _, _, err := net.SplitHostPort(c.Backend.Addr); err != nil {
		return c, fmt.Errorf("backend.addr 无效: %w", err)
	}
	if c.Backend.Database == "" {
		return c, fmt.Errorf("backend.database 不能为空")
	}
	if len(c.Backend.Token) < 16 || len(c.Backend.Token) > 4096 {
		return c, fmt.Errorf("backend.token 需要 16–4096 字符")
	}
	if !filepath.IsAbs(c.Backend.Database) {
		c.Backend.Database = filepath.Join(filepath.Dir(path),
			c.Backend.Database)
	}
	c.Backend.Database, err = filepath.Abs(c.Backend.Database)
	return c, err
}

func validateNode(node *yaml.Node) error {
	if node.Kind == yaml.AliasNode || node.Anchor != "" {
		return fmt.Errorf("配置不支持 YAML 锚点或别名")
	}
	for _, child := range node.Content {
		if err := validateNode(child); err != nil {
			return err
		}
	}
	return nil
}
