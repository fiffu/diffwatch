package app

import (
	"database/sql"
	"time"

	"github.com/fiffu/diffwatch/lib/models"
)

type SubscriptionView struct {
	ID             uint         `json:"id"`
	UserID         uint         `json:"user_id"`
	Notifier       NotifierView `json:"notifier"`
	Endpoint       string       `json:"endpoint"`
	XPath          string       `json:"xpath"`
	Title          string       `json:"title"`
	ImageURL       string       `json:"image_url"`
	LastPollTime   *string      `json:"last_poll_time"`
	NoContentSince *string      `json:"no_content_since"`
}

type NotifierView struct {
	Platform   string `json:"platform"`
	Identifier string `json:"identifier"`
	Verified   bool   `json:"verified"`
}

func (view NotifierView) From(entity *models.Notifier) NotifierView {
	return NotifierView{
		Platform:   entity.Platform,
		Identifier: entity.PlatformIdentifier,
		Verified:   entity.Verified,
	}
}

func (view SubscriptionView) From(entity *models.Subscription) SubscriptionView {
	return SubscriptionView{
		ID:             entity.ID,
		UserID:         entity.UserID,
		Notifier:       NotifierView{}.From(&entity.Notifier),
		Endpoint:       entity.Endpoint,
		XPath:          entity.XPath,
		Title:          entity.Title,
		ImageURL:       entity.ImageURL,
		LastPollTime:   isoformat(entity.LastPollTime),
		NoContentSince: isoformat(entity.NoContentSince),
	}
}

type Fromable[Entity any, Repr any] interface {
	From(Entity) Repr
}

func FromMany[T any, U Fromable[T, U]](elems []T) []U {
	out := make([]U, len(elems))
	for i, t := range elems {
		var u U
		out[i] = u.From(t)
	}
	return out
}

func isoformat(t sql.NullTime) *string {
	if t.Valid {
		t.Time.UTC().Format(time.RFC3339)
	}
	return nil
}
