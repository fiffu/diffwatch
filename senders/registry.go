package senders

import (
	"context"
	"net/http"

	"github.com/fiffu/diffwatch/config"
	"github.com/fiffu/diffwatch/lib/models"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Sender interface {
	SendSnapshot(ctx context.Context, notifier *models.Notifier, sub *models.Subscription, before, after *models.Snapshot) (string, error)
	SendVerification(ctx context.Context, notifier *models.Notifier, verifyURL string) (string, error)
}

type Registry map[string]Sender

func NewSenderRegistry(lc fx.Lifecycle, log *zap.Logger, cfg *config.Config, transport http.RoundTripper) Registry {
	base := base{log, cfg, transport}
	return map[string]Sender{
		"email": &mailgunSender{base},
	}
}

type base struct {
	log       *zap.Logger
	cfg       *config.Config
	transport http.RoundTripper
}
