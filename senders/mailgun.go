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
	mg := mailgun.NewMailgun(e.cfg.Mailgun.Domain, e.cfg.Mailgun.APIKey)
	mg.Client().Transport = e.transport

	// Create message with empty body first.
	message := mg.NewMessage(e.cfg.Mailgun.SenderFrom, subject, "", recipient)
	// SetHtml with the payload proper. This will assign the MIME type properly.
	message.SetHtml(body)

	timeout := time.Duration(e.cfg.Mailgun.TimeoutSecs) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	_, id, err := mg.Send(ctx, message)
	return id, err
}
