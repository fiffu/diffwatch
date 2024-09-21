package main

import (
	"crypto/sha1"
	"database/sql"
	"fmt"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func NewDatabase(lc fx.Lifecycle, log *zap.Logger) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("diffwatch.sqlite"), &gorm.Config{})
	if err != nil {
		log.Sugar().Panic("failed to connect database", "err", err)
	}
	log.Info("Database started")

	log.Info("Starting migrations")
	db.AutoMigrate(
		&User{},
		&Notifier{},
		&NotifierConfirmation{},
		&Subscription{},
		&User{},
		&Snapshot{},
	)
	return db
}

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
}

type Subscription struct {
	gorm.Model
	UserID     uint
	NotifierID uint
	Endpoint   string `gorm:"index:idx_endpoint_xpath"` // Composite index on endpoint & xpath
	XPath      string `gorm:"index:idx_endpoint_xpath"`

	Notifier Notifier
}

type Snapshot struct {
	Timestamp      time.Time
	UserID         uint
	SubscriptionID uint
	Content        string
	ContentDigest  string

	Subscription Subscription
}

func (s *Snapshot) BeforeCreate(tx *gorm.DB) error {
	digest := fmt.Sprintf("%x", sha1.Sum([]byte(s.Content)))
	s.ContentDigest = digest
	return nil
}
