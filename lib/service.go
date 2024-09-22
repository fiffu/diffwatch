package lib

import (
	"context"
	"errors"

	"github.com/fiffu/diffwatch/config"
	"github.com/fiffu/diffwatch/lib/models"
	"github.com/fiffu/diffwatch/lib/snapshotter"
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

	snaps *snapshotter.Snapshotter
	*onboardUser
	*subscribe
}

func NewService(lc fx.Lifecycle, cfg *config.Config, log *zap.Logger, db *gorm.DB, snaps *snapshotter.Snapshotter, senders senders.Registry) *Service {
	return &Service{
		cfg, log, db, senders,
		snaps,
		&onboardUser{cfg, log, db, senders},
		&subscribe{cfg, log, db, snaps},
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
