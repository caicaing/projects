package config

// Config 应用配置
type Config struct {
	Server ServerConfig
	Auth   AuthConfig
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port string
}

// AuthConfig 认证配置
type AuthConfig struct {
	Token string
}

// NewConfig 创建默认配置
func NewConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port: ":8801",
		},
		Auth: AuthConfig{
			Token: "123456",
		},
	}
}
