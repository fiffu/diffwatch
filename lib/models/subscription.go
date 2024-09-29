package models

import (
	"database/sql"

	"gorm.io/gorm"
)

type Subscription struct {
	gorm.Model
	UserID         uint
	NotifierID     uint
	Endpoint       string `gorm:"index:idx_endpoint_xpath"` // Composite index on endpoint & xpath
	XPath          string `gorm:"index:idx_endpoint_xpath"`
	Title          string
	ImageURL       string
	LastPollTime   sql.NullTime
	NoContentSince sql.NullTime

	Notifier Notifier
}

type Subscriptions []Subscription
