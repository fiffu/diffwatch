package models

import (
	"database/sql"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username    string `gorm:"unique"`
	Password    string
	LastLoginAt sql.NullTime

	Notifiers     []Notifier
	Subscriptions []Subscription
}
