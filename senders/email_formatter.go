package senders

import (
	"fmt"

	"github.com/fiffu/diffwatch/lib/models"
)

type snapshotEmailFormat struct {
	subscription      *models.Subscription
	previous, current *models.Snapshot
}

func (ef *snapshotEmailFormat) Subject() string {
	return fmt.Sprintf("Diffwatch: new update on %s", ef.subscription.Title)
}

func (ef *snapshotEmailFormat) Body() string {
	title := ef.subscription.Title
	if title == "" {
		title = ef.subscription.Endpoint
	}

	body := ""
	body += fmt.Sprintf(`<h3>New changes on <a href="%s">%s</a></h3>`, ef.subscription.Endpoint, ef.subscription.Title)
	if ef.previous != nil {
		body += fmt.Sprintf(`Previous value: <pre style="padding: 15px; background-color: #eeeeee">%s</pre>`, ef.previous.Content)
	}
	body += fmt.Sprintf(`Latest value: <pre style="padding: 15px; background-color: #eeeeee">%s</pre>`, ef.current.Content)
	if ef.subscription.ImageURL != "" {
		body += fmt.Sprintf(`<br><img src="%s" width="40%"`, ef.subscription.ImageURL)
	}
	body += fmt.Sprintf(`<br><hr><span style="font-size: 0.7em; color: #555555;">Fingerprint: %s</span>`, ef.current.ContentDigest)
	return body
}

type verificationEmailFormat struct {
	verifyURL string
}

func (ef *verificationEmailFormat) Subject() string {
	return "Diffwatch: Email verification required"
}

func (ef *verificationEmailFormat) Body() string {
	url := ef.verifyURL
	return fmt.Sprintf(`Click here to verify your email: <a href="%s">%s</a>`, url, url)
}
