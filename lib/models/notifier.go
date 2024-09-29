package models

import (
	"time"

	"gorm.io/gorm"
)

type Notifier struct {
	gorm.Model
	UserID             uint
	Verified           bool
	Platform           string
	PlatformIdentifier string
}

type NotifierConfirmation struct {
	NotifierID uint
	Nonce      string `gorm:"unique_index"`
	Expiry     time.Time

	Notifier Notifier
}
