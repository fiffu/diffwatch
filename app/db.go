package app

import (
	"github.com/fiffu/diffwatch/lib/models"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func NewDatabase(lc fx.Lifecycle, log *zap.Logger) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("diffwatch.sqlite"), &gorm.Config{})
	if err != nil {
		log.Sugar().Panic("failed to connect database", "err", err)
	}
	log.Info("Database started")

	log.Info("Starting migrations")
	db.AutoMigrate(
		&models.User{},
		&models.Notifier{},
		&models.NotifierConfirmation{},
		&models.Subscription{},
		&models.User{},
		&models.Snapshot{},
	)
	return db
}
