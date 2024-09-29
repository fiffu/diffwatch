package lib

import (
	"context"
	"errors"

	"github.com/fiffu/diffwatch/config"
	"github.com/fiffu/diffwatch/lib/models"
	"github.com/fiffu/diffwatch/senders"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Service struct {
	cfg     *config.Config
	log     *zap.Logger
	db      *gorm.DB
	senders senders.Registry

	snapshotter *Snapshotter
	*onboardUser
	*subscribe
}

func NewService(lc fx.Lifecycle, cfg *config.Config, log *zap.Logger, db *gorm.DB, snapshotter *Snapshotter, senders senders.Registry) *Service {
	return &Service{
		cfg, log, db, senders,
		snapshotter,
		&onboardUser{cfg, log, db, senders},
		&subscribe{cfg, log, db, snapshotter},
	}
}

func (svc *Service) VerifyNotifier(ctx context.Context, nonce string) (bool, error) {
	confirm := models.NotifierConfirmation{}
	tx := svc.db.Where("nonce = ?", nonce).First(&confirm)
	if err := tx.Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	tx = svc.db.Model(&models.Notifier{}).Where("id = ?", confirm.NotifierID).Update("verified", true)
	if err := tx.Error; err != nil {
		return false, err
	}

	return true, nil
}

func (svc *Service) ListSubscriptions(ctx context.Context, userID uint, limit, offset int) ([]*models.Subscription, error) {
	var subs models.Subscriptions
	tx := svc.db.
		Where("subscriptions.user_id = ?", userID).
		Order("subscriptions.id desc").
		InnerJoins("Notifier").
		Limit(limit).Offset(offset).
		Find(&subs)
	if err := tx.Error; err != nil {
		return nil, err
	}
	return subs, nil
}

func (svc *Service) FindSnapshot(ctx context.Context, userID, subscriptionID uint) (*models.Snapshot, error) {
	snap := &models.Snapshot{}
	tx := svc.db.
		Where("user_id = ?", userID).
		Where("subscription_id = ?", subscriptionID).
		Order("timestamp desc").
		First(&snap)
	if err := tx.Error; err != nil {
		return nil, err
	}
	return snap, nil
}

func (svc *Service) PushSnapshot(ctx context.Context, userID, subscriptionID uint) (*models.Snapshot, *models.Snapshot, error) {
	sub := models.Subscription{}
	tx := svc.db.
		Where("subscriptions.user_id = ?", userID).
		Where("subscriptions.id = ?", subscriptionID).
		InnerJoins("Notifier").
		Find(&sub)
	if err := tx.Error; err != nil {
		return nil, nil, err
	}

	var snaps models.Snapshots
	tx = svc.db.
		Where("user_id = ?", userID).
		Where("subscription_id = ?", subscriptionID).
		Order("timestamp desc").
		Limit(2).
		Find(&snaps)
	if err := tx.Error; err != nil {
		return nil, nil, err
	}

	var previous, current *models.Snapshot
	current = &snaps[0]
	if len(snaps) == 2 {
		previous = &snaps[1]
	}

	err := svc.snapshotter.SendSnapshot(ctx, &sub, previous, current)
	return previous, current, err
}
