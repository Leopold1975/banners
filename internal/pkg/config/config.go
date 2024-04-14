package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Server     Server     `yaml:"server"`
	Logger     Logger     `yaml:"logger"`
	PostgresDB PostgresDB `yaml:"db"`
	Auth       Auth       `yaml:"auth"`
	RedisCache RedisCache `yaml:"rdb"`
}

type Server struct {
	Addr         string        `yaml:"addr"`
	ReadTimeout  time.Duration `yaml:"readTimeout"`
	IdleTimeout  time.Duration `yaml:"idleTimeout"`
	WriteTimeout time.Duration `yaml:"writeTimeout"`
}

type Logger struct {
	Level     string   `yaml:"level"`
	Output    []string `yaml:"output"`
	ErrOutput []string `yaml:"errOutput"`
}

type PostgresDB struct {
	Addr     string `yaml:"addr"`
	Username string `env:"POSTGRES_USER"     env-required:"true" yaml:"username"`
	Password string `env:"POSTGRES_PASSWORD" yaml:"password"`
	DB       string `env:"POSTGRES_DB"       env-required:"true" yaml:"db"`
	SSLmode  string `yaml:"sslmode"`
	MaxConns string `yaml:"maxConns"`
	Reload   bool   `yaml:"reload"`
	Version  int    `yaml:"version"`
}

type Auth struct {
	TTL    time.Duration `yaml:"ttl"`
	Secret string        `env:"SECRET" env-required:"true" yaml:"secret"`
}

type RedisCache struct {
	Addr     string        `yaml:"addr"`
	Password string        `yaml:"password"`
	DB       int           `yaml:"db"`
	ExpTime  time.Duration `yaml:"exp"`
}

func New(configPath string) (Config, error) {
	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		return Config{}, fmt.Errorf("read config error: %w", err)
	}

	return cfg, nil
}
