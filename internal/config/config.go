package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App       AppConfig       `yaml:"app"`
	Server    ServerConfig    `yaml:"server"`
	Database  DatabaseConfig  `yaml:"database"`
	Redis     RedisConfig     `yaml:"redis"`
	JWT       JWTConfig       `yaml:"jwt"`
	Email     EmailConfig     `yaml:"email"`
	Log       LogConfig       `yaml:"log"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
}

type AppConfig struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Environment string `yaml:"environment"`
}

type ServerConfig struct {
	Port            int `yaml:"port"`
	ReadTimeout     int `yaml:"read_timeout"`
	WriteTimeout    int `yaml:"write_timeout"`
	IdleTimeout     int `yaml:"idle_timeout"`
	ShutdownTimeout int `yaml:"shutdown_timeout"`
}

type DatabaseConfig struct {
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	User            string `yaml:"user"`
	Password        string `yaml:"password"`
	Database        string `yaml:"database"`
	SSLMode         string `yaml:"ssl_mode"`
	MaxOpenConns    int    `yaml:"max_open_conns"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
	ConnMaxLifetime int    `yaml:"conn_max_lifetime"`
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type JWTConfig struct {
	AccessSecret         string `yaml:"access_secret"`
	RefreshSecret        string `yaml:"refresh_secret"`
	AccessTokenDuration  int    `yaml:"access_token_duration"`
	RefreshTokenDuration int    `yaml:"refresh_token_duration"`
}

type EmailConfig struct {
	SMTPHost     string `yaml:"smtp_host"`
	SMTPPort     int    `yaml:"smtp_port"`
	SMTPUsername string `yaml:"smtp_username"`
	SMTPPassword string `yaml:"smtp_password"`
	FromEmail    string `yaml:"from_email"`
	FromName     string `yaml:"from_name"`
}

type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type RateLimitConfig struct {
	RequestsPerMinute int    `yaml:"requests_per_minute"`
	Burst             int    `yaml:"burst"`
	Enabled           bool   `yaml:"enabled"`
	Window            int    `yaml:"window"` // in seconds
	MetricsNamespace  string `yaml:"metrics_namespace"`
}

func Load(path string) (*Config, error) {
	// Read config file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	// Parse YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Override with environment variables
	overrideWithEnv(&cfg)

	// Validate
	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

func overrideWithEnv(cfg *Config) {
	// App
	if v := os.Getenv("APP_ENVIRONMENT"); v != "" {
		cfg.App.Environment = v
	}

	// Server
	if v := os.Getenv("SERVER_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &cfg.Server.Port)
	}

	// Database
	if v := os.Getenv("DB_HOST"); v != "" {
		cfg.Database.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &cfg.Database.Port)
	}
	if v := os.Getenv("DB_USER"); v != "" {
		cfg.Database.User = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		cfg.Database.Password = v
	}
	if v := os.Getenv("DB_DATABASE"); v != "" {
		cfg.Database.Database = v
	}

	// Redis
	if v := os.Getenv("REDIS_HOST"); v != "" {
		cfg.Redis.Host = v
	}
	if v := os.Getenv("REDIS_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &cfg.Redis.Port)
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}

	// JWT
	if v := os.Getenv("JWT_ACCESS_SECRET"); v != "" {
		cfg.JWT.AccessSecret = v
	}
	if v := os.Getenv("JWT_REFRESH_SECRET"); v != "" {
		cfg.JWT.RefreshSecret = v
	}

	// Email
	if v := os.Getenv("SMTP_HOST"); v != "" {
		cfg.Email.SMTPHost = v
	}
	if v := os.Getenv("SMTP_USERNAME"); v != "" {
		cfg.Email.SMTPUsername = v
	}
	if v := os.Getenv("SMTP_PASSWORD"); v != "" {
		cfg.Email.SMTPPassword = v
	}

	// Rate limit
	if v := os.Getenv("RATE_LIMIT_REQUESTS_PER_MINUTE"); v != "" {
		fmt.Sscanf(v, "%d", &cfg.RateLimit.RequestsPerMinute)
	}
	if v := os.Getenv("RATE_LIMIT_BURST"); v != "" {
		fmt.Sscanf(v, "%d", &cfg.RateLimit.Burst)
	}
	if v := os.Getenv("RATE_LIMIT_ENABLED"); v != "" {
		// accept "1", "true", "TRUE", "True"
		lower := strings.ToLower(v)
		if lower == "1" || lower == "true" || lower == "t" {
			cfg.RateLimit.Enabled = true
		} else {
			cfg.RateLimit.Enabled = false
		}
	}
	if v := os.Getenv("RATE_LIMIT_WINDOW"); v != "" {
		fmt.Sscanf(v, "%d", &cfg.RateLimit.Window)
	}
	if v := os.Getenv("RATE_LIMIT_METRICS_NAMESPACE"); v != "" {
		cfg.RateLimit.MetricsNamespace = v
	}
}

func validate(cfg *Config) error {
	if cfg.Server.Port == 0 {
		return fmt.Errorf("server port is required")
	}
	if cfg.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if cfg.JWT.AccessSecret == "" {
		return fmt.Errorf("JWT access secret is required")
	}
	if !strings.Contains(cfg.App.Environment, "production") &&
		!strings.Contains(cfg.App.Environment, "development") &&
		!strings.Contains(cfg.App.Environment, "local") {
		return fmt.Errorf("invalid environment: %s", cfg.App.Environment)
	}
	return nil
}
