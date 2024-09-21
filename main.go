package main

import (
	"net/http"

	"github.com/fiffu/diffwatch/app"
	"github.com/fiffu/diffwatch/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	fx.New(
		fx.Provide(zap.NewProduction),
		fx.Provide(config.NewConfig),
		fx.Provide(app.NewDatabase),
		fx.Provide(app.NewTransport),
		fx.Provide(app.NewSnapshotter),
		fx.Provide(app.NewService),
		fx.Provide(app.NewHTTPServer),

		fx.Invoke(func(*http.Server) {}),
	).Run()
}
