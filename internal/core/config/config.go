package config

// Config 应用配置
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
}

// ServerConfig HTTP 服务配置
type ServerConfig struct {
	Host string
	Port int
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	DSN string
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// Load 加载配置（stub）
func Load() (*Config, error) {
	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
	}, nil
}
