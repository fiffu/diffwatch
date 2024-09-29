package models

import "time"

type Chaser struct {
	SubscriptionID uint `gorm:"index:idx_subscriptionid_notbefore"`
	NotifierID     uint
	NotBefore      time.Time `gorm:"index:idx_subscriptionid_notbefore;notNull"`

	Subscription Subscription
	Notifier     Notifier
}

type Chasers []*Chaser
