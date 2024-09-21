package senders

import (
	"context"
	"time"

	"github.com/mailgun/mailgun-go/v4"
)

type mailgunSender struct {
	base
}

func (e *mailgunSender) Send(ctx context.Context, subject, body, recipient string) (string, error) {
	client := mailgun.NewMailgun(e.cfg.Mailgun.Domain, e.cfg.Mailgun.APIKey)

	message := client.NewMessage(e.cfg.Mailgun.SenderFrom, subject, body, recipient)

	timeout := time.Duration(e.cfg.Mailgun.TimeoutSecs) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	_, id, err := client.Send(ctx, message)
	return id, err
}
