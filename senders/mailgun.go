package senders

import (
	"context"
	"time"

	"github.com/fiffu/diffwatch/lib/models"
	"github.com/fiffu/diffwatch/senders/email"
	"github.com/mailgun/mailgun-go/v4"
)

type mailgunSender struct {
	base
}

type emailFormatter interface {
	Subject() string
	Body() string
}

func (e *mailgunSender) send(ctx context.Context, email emailFormatter, recipient string) (string, error) {
	mg := mailgun.NewMailgun(e.cfg.Mailgun.Domain, e.cfg.Mailgun.APIKey)
	mg.Client().Transport = e.transport

	// Create message with empty body first, then use SetHtml which assigns the MIME type properly.
	message := mg.NewMessage(e.cfg.Mailgun.SenderFrom, email.Subject(), "", recipient)
	message.SetHtml(email.Body())

	timeout := time.Duration(e.cfg.Mailgun.TimeoutSecs) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	_, id, err := mg.Send(ctx, message)
	return id, err
}

func (e *mailgunSender) SendSnapshot(ctx context.Context, notifier *models.Notifier, sub *models.Subscription, before, after *models.Snapshot) (string, error) {
	formatter := &email.SnapshotEmailFormat{Subscription: sub, Previous: before, Current: after}
	return e.send(ctx, formatter, notifier.PlatformIdentifier)
}

func (e *mailgunSender) SendVerification(ctx context.Context, notifier *models.Notifier, verifyURL string) (string, error) {
	formatter := &email.VerificationEmailFormat{VerifyURL: verifyURL}
	return e.send(ctx, formatter, notifier.PlatformIdentifier)
}
