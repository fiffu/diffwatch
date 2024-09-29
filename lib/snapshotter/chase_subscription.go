package snapshotter

import (
	"context"
	"sync"
	"time"

	"github.com/fiffu/diffwatch/lib/models"
	"gorm.io/gorm"
)

func (s *Snapshotter) chaseSubscriptions(ctx context.Context, timestamp time.Time) {
	var chasers models.Chasers
	tx := s.db.
		Where("not_before < ?", timestamp).
		InnerJoins("Subscription").
		InnerJoins("Notifier").
		FindInBatches(&chasers, s.concurrency, func(tx *gorm.DB, batch int) error {
			errs := s.chaseBatch(ctx, timestamp, chasers)
			if len(errs) > 0 {
				s.log.Sugar().Warnf("snapshot: batch errors: %+v", errs)
			}
			return nil
		})
	if err := tx.Error; err != nil {
		s.log.Sugar().Infof("Failed to fetch chased subscriptions, err: %v", err)
		return
	}
}

func (s *Snapshotter) chaseBatch(ctx context.Context, chaseTime time.Time, batch []*models.Chaser) []error {
	var wg sync.WaitGroup

	errs := make([]error, 0)
	for _, chaser := range batch {
		wg.Add(1)

		go func() {
			defer wg.Done()
			if err := s.chase(ctx, chaser); err != nil {
				errs = append(errs, err)
			}
		}()
	}

	tx := s.db.Delete(&models.Chaser{}).Where("not_before < ?", chaseTime)
	if err := tx.Error; err != nil {
		errs = append(errs, err)
	}
	return errs
}

func (s *Snapshotter) chase(ctx context.Context, chaser *models.Chaser) error {
	m, err := s.snapshotAndNotify(ctx, &chaser.Subscription)
	if err != nil {
		return err
	}

	snapshotTime := time.Now().UTC()
	tx := s.db.Model(&models.Subscription{}).Where("id = ?", chaser.SubscriptionID).Update("last_poll_time", snapshotTime)
	if err := tx.Error; err != nil {
		return err
	}

	if m.updated > 0 {
		nextChaserAt := snapshotTime.Add(s.chaseInterval)
		s.log.Sugar().Infof("Subscription id:%v had update; scheduling next chaser at %s", chaser.SubscriptionID, nextChaserAt)
	}

	return nil
}
