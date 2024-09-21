package app

import (
	"net/http"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewTransport(lc fx.Lifecycle, log *zap.Logger) http.RoundTripper {
	return http.DefaultTransport
}

type transport struct {
	base http.RoundTripper
	log  *zap.Logger
}

func (tpt *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	return tpt.base.RoundTrip(req)
}
