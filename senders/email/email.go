package email

import (
	_ "embed"
	"fmt"
	"html/template"
	"strings"

	"github.com/fiffu/diffwatch/lib/models"
)

var (
	//go:embed snapshot.html
	snapshotHTML     string
	snapshotTemplate = template.Must(template.New("snapshot.html").Parse(snapshotHTML))

	//go:embed verify.html
	verifyHTML     string
	verifyTemplate = template.Must(template.New("verify.html").Parse(verifyHTML))
)

func mustFillTemplate(tmpl *template.Template, values any) string {
	buf := new(strings.Builder)
	err := tmpl.Execute(buf, values)
	if err != nil {
		return ""
	}
	return buf.String()
}

type SnapshotEmailFormat struct {
	Subscription      *models.Subscription
	Previous, Current *models.Snapshot
}

func (ef *SnapshotEmailFormat) Subject() string {
	return fmt.Sprintf("Diffwatch: new update on %s", ef.Subscription.Title)
}

func (ef *SnapshotEmailFormat) Body() string {
	return mustFillTemplate(snapshotTemplate, ef)
}

type VerificationEmailFormat struct {
	VerifyURL string
}

func (ef *VerificationEmailFormat) Subject() string {
	return "Diffwatch: Email verification required"
}

func (ef *VerificationEmailFormat) Body() string {
	return mustFillTemplate(verifyTemplate, ef)
}
