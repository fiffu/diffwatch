package main

import (
	"net/http"
	"os"
	"time"

	"github.com/fiffu/diffwatch/app"
	"github.com/fiffu/diffwatch/config"
	"github.com/fiffu/diffwatch/senders"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger() (*zap.Logger, error) {
	switch os.Getenv("ENVIRONMENT") {
	default:
		return zap.NewDevelopment()

	case "production":
		logCfg := zap.NewProductionConfig()
		logCfg.EncoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			t = t.UTC()
			zapcore.ISO8601TimeEncoder(t, enc)
		}
		return logCfg.Build()
	}
}

func main() {
	fx.New(
		fx.Provide(config.NewConfig),
		fx.Provide(NewLogger),

		fx.Provide(senders.NewSenderRegistry),

		fx.Provide(app.NewDatabase),
		fx.Provide(app.NewTransport),
		fx.Provide(app.NewSnapshotter),
		fx.Provide(app.NewService),
		fx.Provide(app.NewHTTPServer),

		fx.Invoke(func(*http.Server) {}),
	).Run()
}
