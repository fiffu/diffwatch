package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/fiffu/diffwatch/config"
	"github.com/fiffu/diffwatch/lib"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewAPI(lc fx.Lifecycle, cfg *config.Config, log *zap.Logger, svc *lib.Service) *http.Server {
	addr := fmt.Sprintf(":%d", cfg.ServerPort)
	srv := &http.Server{Addr: addr, Handler: router(cfg, log, svc)}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go srv.ListenAndServe()
			return nil
		},
		OnStop: srv.Shutdown,
	})

	return srv
}

func router(cfg *config.Config, log *zap.Logger, svc *lib.Service) http.Handler {
	ctrl := &controller{log, svc}

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	r.Route("/api", func(r chi.Router) {
		if creds := cfg.GetCreds(); len(creds) > 0 {
			r.Use(middleware.BasicAuth("diffwatch", creds))
		} else {
			log.Sugar().Info("Auth is disabled since no credentials are defined")
		}

		r.Route("/users", func(r chi.Router) {
			r.Post("/", ctrl.onboardUser)
			r.Post("/{user_id}/subscription", ctrl.subscribe)
			r.Get("/{user_id}/subscription/{subscription_id}/latest", ctrl.viewSnapshot)
		})
	})
	r.Get("/verify/{nonce}", ctrl.verifyNotifier)

	return r
}

type controller struct {
	log *zap.Logger
	svc *lib.Service
}

func (ctrl *controller) reject(w http.ResponseWriter, status int, err error) {
	if err != nil {
		http.Error(w, err.Error(), status)
	} else {
		w.WriteHeader(status)
	}
}

func (ctrl *controller) resolve(w http.ResponseWriter, status int, body any) {
	if b, err := json.Marshal(body); err != nil {
		ctrl.reject(w, http.StatusInternalServerError, err)
		ctrl.log.Sugar().Error("Request failed", "error", err)
		return
	} else {
		w.WriteHeader(status)
		if b != nil {
			w.Write(b)
		}
	}
}

func (ctrl *controller) onboardUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	email := r.FormValue("email")
	password := r.FormValue("password")

	if email == "" {
		ctrl.reject(w, 400, errors.New("Email is required"))
		return
	}
	if password == "" {
		ctrl.reject(w, 400, errors.New("Password is required"))
		return
	}

	user, err := ctrl.svc.OnboardUser(ctx, email, password)
	if err != nil {
		ctrl.reject(w, 500, err)
		return
	}
	ctrl.resolve(w, http.StatusAccepted, user)
}

func (ctrl *controller) subscribe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := chi.URLParam(r, "user_id")
	endpoint := r.FormValue("endpoint")
	xpath := r.FormValue("xpath")

	snap, err := ctrl.svc.Subscribe(ctx, parseInt(userID), endpoint, xpath)
	if err != nil {
		ctrl.reject(w, 500, err)
		return
	}
	ctrl.resolve(w, http.StatusOK, map[string]any{
		"subscription_id": snap.SubscriptionID,
		"content":         snap.Content,
	})
}

func (ctrl *controller) viewSnapshot(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := chi.URLParam(r, "user_id")
	snapshotID := chi.URLParam(r, "subscription_id")

	snap, err := ctrl.svc.FindSnapshot(ctx, parseInt(userID), parseInt(snapshotID))
	if err != nil {
		ctrl.reject(w, 500, err)
		return
	}
	ctrl.resolve(w, 200, snap.Content)
}

func (ctrl *controller) verifyNotifier(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	nonce := chi.URLParam(r, "nonce")

	ok, err := ctrl.svc.VerifyNotifier(ctx, nonce)
	if err != nil {
		ctrl.reject(w, 500, err)
		return
	}
	ctrl.resolve(w, http.StatusOK, map[string]any{"verified": ok})
}

func parseInt(s string) uint {
	u, _ := strconv.ParseUint(s, 10, 64)
	return uint(u)
}
