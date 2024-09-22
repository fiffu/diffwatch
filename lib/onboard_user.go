package lib

import (
	"context"
	"fmt"
	"time"

	"github.com/fiffu/diffwatch/config"
	"github.com/fiffu/diffwatch/lib/models"
	"github.com/fiffu/diffwatch/senders"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type onboardUser struct {
	cfg     *config.Config
	log     *zap.Logger
	db      *gorm.DB
	senders senders.Registry
}

func (svc *onboardUser) OnboardUser(ctx context.Context, email string, password string) (*models.User, error) {
	user, confirmation, err := svc.createUserAndNotifier(email, password)
	if err != nil {
		return nil, err
	}
	if err = svc.sendVerificationEmail(ctx, confirmation); err != nil {
		return nil, err
	}
	svc.log.Sugar().Infof("Created user %v (%s), confimation nonce: %s", user.ID, email, confirmation.Nonce)
	return user, nil
}

func (svc *onboardUser) createUserAndNotifier(email string, password string) (*models.User, *models.NotifierConfirmation, error) {
	user := models.User{
		Username: email,
		Password: password,
	}
	tx := svc.db.Clauses(clause.Returning{}).Create(&user)
	if err := tx.Error; err != nil {
		return nil, nil, err
	}

	notif := models.Notifier{Platform: "email", PlatformIdentifier: email, UserID: user.ID}
	tx = svc.db.Clauses(clause.Returning{}).Create(&notif)
	if err := tx.Error; err != nil {
		return nil, nil, err
	}

	nonce := svc.generateNonce()
	notifConfirm := models.NotifierConfirmation{
		NotifierID: notif.ID,
		Nonce:      nonce,
		Expiry:     time.Now().UTC().Add(3 * 24 * time.Hour),
	}
	tx = svc.db.Create(&notifConfirm)
	if err := tx.Error; err != nil {
		return nil, nil, err
	}
	notifConfirm.Notifier = notif

	return &user, &notifConfirm, nil
}

func (svc *onboardUser) sendVerificationEmail(ctx context.Context, verification *models.NotifierConfirmation) error {
	verifyURL := fmt.Sprintf("%s/verify/%s", svc.cfg.ServerDNS, verification.Nonce)

	sender := svc.senders["email"]
	id, err := sender.SendVerification(
		ctx,
		&verification.Notifier,
		verifyURL,
	)
	if err != nil {
		svc.log.Sugar().Infow("Failed to send verification email", "err", err)
	} else {
		identifier := verification.Notifier.PlatformIdentifier
		svc.log.Sugar().Infow("Sent verification to "+identifier, "message_id", id)
	}
	return err
}

func (svc *onboardUser) generateNonce() string {
	u, _ := uuid.NewUUID()
	return u.String()
}
