package config

import (
	"github.com/caarlos0/env/v11"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Config struct {
	Env        string `env:"ENVIRONMENT"`
	ServerPort int    `env:"SERVER_PORT"`
	ServerDNS  string `env:"SERVER_DNS"` // Used in verification email when adding a new notifier
	Mailgun    struct {
		APIKey      string `env:"MAILGUN_API_KEY"`
		Domain      string `env:"MAILGUN_DOMAIN"`
		SenderFrom  string `env:"MAILGUN_SENDER_FROM"`
		TimeoutSecs int    `env:"MAILGUN_TIMEOUT_SECS"`
	}

	log   *zap.Logger
	creds map[string]string
}

func NewConfig(lc fx.Lifecycle, log *zap.Logger) *Config {
	cfg := &Config{log: log}
	env.Parse(cfg)

	return cfg
}

func (cfg *Config) GetCreds() map[string]string {
	return cfg.creds
}
