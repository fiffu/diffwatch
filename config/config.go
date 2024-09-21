package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/caarlos0/env"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Config struct {
	Env            string `env:"ENVIRONMENT"`
	BasicAuthCreds string `env:"BASIC_AUTH_CREDS"`
	Mailgun        struct {
		Domain string `env:"MAILGUN_DOMAIN"`
		APIKey string `env:"MAILGUN_API_KEY"`
	}

	log   *zap.Logger
	creds map[string]string
}

func NewConfig(lc fx.Lifecycle, log *zap.Logger) *Config {
	cfg := &Config{log: log}
	env.Parse(cfg)

	creds, err := cfg.parseCreds()
	if err != nil {
		if cfg.Env != "development" {
			cfg.log.Sugar().Infof("%s (credentials will be set to default in development env)", err)
			creds = map[string]string{"admin": "password"}
		} else {
			cfg.log.Sugar().Panic(err)
		}
	}
	cfg.creds = creds

	return cfg
}

func (cfg *Config) GetCreds() map[string]string {
	return cfg.creds
}

func (cfg *Config) parseCreds() (map[string]string, error) {
	if cfg.BasicAuthCreds == "" {
		return nil, errors.New("BASIC_AUTH_CREDS envvar must be populated")
	}

	creds := strings.Split(cfg.BasicAuthCreds, ",")
	if len(creds) == 0 {
		return nil, errors.New("BASIC_AUTH_CREDS envvar should be filled with comma-separated values -- user1:pass1,user2:pass2")
	}

	result := make(map[string]string)
	for _, cred := range creds {
		userPass := strings.Split(cred, ":")
		if len(userPass) != 2 {
			return nil, fmt.Errorf("failed to parse '%s', each credential should be delimited by a colon -- user1:pass1,user2:pass2", cred)
		}

		user, pass := userPass[0], userPass[1]
		result[strings.Trim(user, " ")] = strings.Trim(pass, " ")
	}

	return result, nil
}
