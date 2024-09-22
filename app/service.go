package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fiffu/diffwatch/config"
	"github.com/fiffu/diffwatch/senders"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Service struct {
	cfg     *config.Config
	log     *zap.Logger
	db      *gorm.DB
	snaps   *Snapshotter
	senders senders.Registry
}

func NewService(lc fx.Lifecycle, cfg *config.Config, log *zap.Logger, db *gorm.DB, snaps *Snapshotter, senders senders.Registry) *Service {
	return &Service{cfg, log, db, snaps, senders}
}

func (svc *Service) OnboardUser(ctx context.Context, email string, password string) (*User, error) {
	user, confirmation, err := svc.createUserAndNotifier(email, password)
	if err != nil {
		return nil, err
	}
	if err = svc.sendVerificationEmail(ctx, email, confirmation.Nonce); err != nil {
		return nil, err
	}
	svc.log.Sugar().Infof("Created user %v (%s), confimation nonce: %s", user.ID, email, confirmation.Nonce)
	return user, nil
}

func (svc *Service) createUserAndNotifier(email string, password string) (*User, *NotifierConfirmation, error) {
	user := &User{
		Username: email,
		Password: password,
	}
	tx := svc.db.
		Clauses(clause.Returning{}).
		Create(user)
	if err := tx.Error; err != nil {
		return nil, nil, err
	}

	notif := &Notifier{Platform: "email", PlatformIdentifier: email, UserID: user.ID}
	tx = svc.db.
		Clauses(clause.Returning{}).
		Create(notif)
	if err := tx.Error; err != nil {
		return nil, nil, err
	}

	nonce := svc.generateNonce()
	notifConfirm := &NotifierConfirmation{
		NotifierID: notif.ID,
		Nonce:      nonce,
		Expiry:     time.Now().UTC().Add(3 * 24 * time.Hour),
	}
	tx = svc.db.Create(notifConfirm)
	if err := tx.Error; err != nil {
		return nil, nil, err
	}

	return user, notifConfirm, nil
}

func (svc *Service) sendVerificationEmail(ctx context.Context, email, nonce string) error {
	url := fmt.Sprintf("%s/verify/%s", svc.cfg.ServerDNS, nonce)

	sender := svc.senders["email"]
	id, err := sender.Send(
		ctx,
		"Diffwatch: Email verification required",
		fmt.Sprintf(`Click here to verify your email: <a href="%s">%s</a>`, url, url),
		email,
	)
	if err != nil {
		svc.log.Sugar().Infow("Failed to send verification email", "err", err)
	} else {
		svc.log.Sugar().Infow("Sent verification to "+email, "message_id", id)
	}
	return err
}

func (svc *Service) generateNonce() string {
	// u, _ := uuid.NewUUID()
	// return u.String()
	return "11111111-1111-1111-1111-111111111111"
}

func (svc *Service) VerifyNotifier(ctx context.Context, nonce string) (bool, error) {
	confirm := NotifierConfirmation{}
	tx := svc.db.Where("nonce = ?", nonce).First(&confirm)
	if err := tx.Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	tx = svc.db.Model(&Notifier{}).Where("id = ?", confirm.NotifierID).Update("verified", true)
	if err := tx.Error; err != nil {
		return false, err
	}

	return true, nil
}

func (svc *Service) Subscribe(ctx context.Context, userID uint, endpoint, xpath string) (*Snapshot, error) {
	notifier := Notifier{}
	tx := svc.db.Where("user_id = ?", userID).First(&notifier)
	if err := tx.Error; err != nil {
		return nil, err
	}

	if !notifier.Verified {
		return nil, errors.New("unable to find verified notifier")
	}

	sub, content, err := svc.subscribeIfValidEndpoint(ctx, userID, notifier.ID, endpoint, xpath)
	if err != nil {
		return nil, err
	}

	snap := Snapshot{
		Timestamp:      time.Now().UTC(),
		UserID:         userID,
		SubscriptionID: sub.ID,
		Content:        content,
	}
	tx = svc.db.Create(&snap)
	if err := tx.Error; err != nil {
		return nil, err
	}
	svc.log.Sugar().Infof("Created subscription id:%v and snapshot:%v", sub.ID, snap.Timestamp)
	return &snap, nil
}

func (svc *Service) subscribeIfValidEndpoint(ctx context.Context, userID, notifierID uint, endpoint, xpath string) (*Subscription, string, error) {
	content, err := svc.snaps.GetEndpointContent(ctx, endpoint, xpath)
	if err != nil {
		return nil, "", err
	}
	if content == "" {
		return nil, "", fmt.Errorf("no result extracted from %s using xpath: %s", endpoint, xpath)
	}

	sub := &Subscription{
		UserID:     userID,
		NotifierID: notifierID,
		Endpoint:   endpoint,
		XPath:      xpath,
	}
	tx := svc.db.Clauses(clause.OnConflict{DoNothing: true}).Create(sub)
	if err := tx.Error; err != nil {
		return nil, "", err
	}
	return sub, content, nil
}

func (svc *Service) FindSnapshot(ctx context.Context, userID, subscriptionID uint) (*Snapshot, error) {
	snap := &Snapshot{}
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
