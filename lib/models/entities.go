package models

import (
	"crypto/sha1"
	"database/sql"
	"fmt"
	"time"

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

type Snapshot struct {
	Timestamp      time.Time
	UserID         uint `gorm:"index:idx_user_subscription"`
	SubscriptionID uint `gorm:"index:idx_user_subscription"`
	Content        string
	ContentDigest  string
}

type Snapshots []Snapshot

func (s *Snapshot) BeforeCreate(tx *gorm.DB) error {
	s.ContentDigest = DigestContent(s.Content)
	return nil
}

func DigestContent(content string) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(content)))
}
