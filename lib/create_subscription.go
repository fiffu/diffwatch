package lib

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fiffu/diffwatch/config"
	"github.com/fiffu/diffwatch/lib/models"
	"github.com/fiffu/diffwatch/lib/snapshotter"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type subscribe struct {
	cfg   *config.Config
	log   *zap.Logger
	db    *gorm.DB
	snaps *snapshotter.Snapshotter
}

func (svc *subscribe) CreateSubscription(ctx context.Context, userID uint, endpoint, xpath string) (*models.Snapshot, *models.Subscription, error) {
	notifier := models.Notifier{}
	tx := svc.db.Where("user_id = ?", userID).First(&notifier)
	if err := tx.Error; err != nil {
		return nil, nil, err
	}

	if !notifier.Verified {
		return nil, nil, errors.New("unable to find verified notifier")
	}

	sub, content, err := svc.subscribeIfValidEndpoint(ctx, userID, notifier.ID, endpoint, xpath)
	if err != nil {
		return nil, nil, err
	}

	snap := models.Snapshot{
		Timestamp:      time.Now().UTC(),
		UserID:         userID,
		SubscriptionID: sub.ID,
		Content:        content.Text,
	}
	tx = svc.db.Create(&snap)
	if err := tx.Error; err != nil {
		return nil, nil, err
	}
	svc.log.Sugar().Infof("Created subscription id:%v and snapshot:%v", sub.ID, snap.ContentDigest)
	return &snap, sub, nil
}

func (svc *subscribe) subscribeIfValidEndpoint(ctx context.Context, userID, notifierID uint, endpoint, xpath string) (*models.Subscription, *models.EndpointContent, error) {
	content, err := svc.snaps.GetEndpointContent(ctx, endpoint, xpath)
	if err != nil {
		return nil, nil, err
	}
	if content.Text == "" {
		return nil, nil, fmt.Errorf("no result extracted from %s using xpath: %s", endpoint, xpath)
	}

	sub := &models.Subscription{
		UserID:     userID,
		NotifierID: notifierID,
		Endpoint:   endpoint,
		XPath:      xpath,
		Title:      content.Title,
		ImageURL:   content.ImageURL,
	}
	tx := svc.db.Clauses(clause.OnConflict{DoNothing: true}).Create(sub)
	if err := tx.Error; err != nil {
		return nil, nil, err
	}
	return sub, content, nil
}
