package senders

import (
	"context"

	"github.com/fiffu/diffwatch/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Sender interface {
	Send(ctx context.Context, subject, body, recipient string) (string, error)
}

type SenderRegistry map[string]Sender

func NewSenderRegistry(lc fx.Lifecycle, log *zap.Logger, cfg *config.Config) SenderRegistry {
	base := base{log, cfg}
	return map[string]Sender{
		"email": &mailgunSender{base},
	}
}

type base struct {
	log *zap.Logger
	cfg *config.Config
}
