package lib

import (
	"context"
	"fmt"
	"time"

	"github.com/fiffu/diffwatch/config"
	"github.com/fiffu/diffwatch/lib/models"
	"github.com/fiffu/diffwatch/senders"
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
	if err = svc.sendVerificationEmail(ctx, email, confirmation.Nonce); err != nil {
		return nil, err
	}
	svc.log.Sugar().Infof("Created user %v (%s), confimation nonce: %s", user.ID, email, confirmation.Nonce)
	return user, nil
}

func (svc *onboardUser) createUserAndNotifier(email string, password string) (*models.User, *models.NotifierConfirmation, error) {
	user := &models.User{
		Username: email,
		Password: password,
	}
	tx := svc.db.
		Clauses(clause.Returning{}).
		Create(user)
	if err := tx.Error; err != nil {
		return nil, nil, err
	}

	notif := &models.Notifier{Platform: "email", PlatformIdentifier: email, UserID: user.ID}
	tx = svc.db.
		Clauses(clause.Returning{}).
		Create(notif)
	if err := tx.Error; err != nil {
		return nil, nil, err
	}

	nonce := svc.generateNonce()
	notifConfirm := &models.NotifierConfirmation{
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

func (svc *onboardUser) sendVerificationEmail(ctx context.Context, email, nonce string) error {
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

func (svc *onboardUser) generateNonce() string {
	// u, _ := uuid.NewUUID()
	// return u.String()
	return "11111111-1111-1111-1111-111111111111"
}
