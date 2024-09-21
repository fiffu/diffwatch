package main

import (
	"net/http"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	fx.New(
		fx.Provide(zap.NewExample),
		fx.Provide(NewConfig),
		fx.Provide(NewDatabase),
		fx.Provide(NewTransport),
		fx.Provide(NewService),
		fx.Provide(NewHTTPServer),

		fx.Invoke(func(*http.Server) {}),
	).Run()
}
