package models

import (
	"time"

	"gorm.io/gorm"
)

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
