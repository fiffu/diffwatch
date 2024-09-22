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
	return fmt.Sprintf(
		`
			<h3>New changes on <a href="%s">%s</a>:</h1>
			<br>
			<pre>%s</pre>
			<br>
			Fingerprint: %s
		`,
		ef.Subscription.Endpoint, ef.Subscription.Endpoint,
		ef.Snapshot.Content, ef.Snapshot.ContentDigest,
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
