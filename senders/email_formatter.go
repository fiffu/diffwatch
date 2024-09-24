package senders

import (
	"fmt"

	"github.com/fiffu/diffwatch/lib/models"
)

type snapshotEmailFormat struct {
	*models.Subscription
	*models.Snapshot
}

func (ef *snapshotEmailFormat) Subject() string {
	return fmt.Sprintf("Diffwatch: new update on %s", ef.Subscription.Endpoint)
}

func (ef *snapshotEmailFormat) Body() string {
	title := ef.Subscription.Title
	if title == "" {
		title = ef.Subscription.Endpoint
	}
	img := ""
	if ef.Subscription.ImageURL != "" {
		img = fmt.Sprintf(`<br><img src="%s" width="40%"`, ef.Subscription.ImageURL)
	}
	return fmt.Sprintf(
		`
			<h3>New changes on <a href="%s">%s</a>:</h3>
			<br>
			<pre>%s</pre>
			%s
			<br><hr>
			<span style="font-size: 0.7em; color: #555555;">Fingerprint: %s</span>
		`,
		ef.Subscription.Endpoint, title,
		ef.Snapshot.Content,
		img,
		ef.Snapshot.ContentDigest,
	)
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
