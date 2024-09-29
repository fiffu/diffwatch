package snapshotter

import (
	"context"
	"time"
)

func (s *Snapshotter) chaseSubscription(ctx context.Context, timestamp time.Time, subscriberID uint) {
	// TODO
	// var sub models.Subscription
	// tx := s.db.
	// 	Where("subscriptions.id = ?", subscriberID).
	// 	InnerJoins("Notifier").
	// 	Find(&sub)
	// if err := tx.Error; err != nil {
	// 	s.log.Sugar().Infof("Failed to chase on subscription %v due to err: %v", subscriberID, err)
	// 	return
	// }

	// s.chase(ctx, timestamp, &sub, &sub.Notifier)
}

// func (s *Snapshotter) chase(ctx context.Context, timestamp time.Time, sub *models.Subscription, notif *models.Notifier) error {
// 	content, err := s.GetEndpointContent(ctx, sub.Endpoint, sub.XPath)
// 	if err != nil {
// 		return err
// 	}

// }
