package snapshotter

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/antchfx/htmlquery"
	"github.com/carlmjohnson/requests"
	"github.com/fiffu/diffwatch/lib/models"
	"github.com/fiffu/diffwatch/senders"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var mu sync.Mutex

func NewSnapshotter(lc fx.Lifecycle, db *gorm.DB, log *zap.Logger, transport http.RoundTripper, senders senders.Registry) *Snapshotter {
	wakeupInterval := 1 * time.Hour    // interval to check for pollable subscriptions
	pollInterval := 1 * time.Hour      // poll each subscription every hour
	chaseInterval := 10 * time.Minute  // if subscription updated, check again after this duration
	noContentTTL := 7 * 24 * time.Hour // stop polling subscription if no data is returned for the past week
	snapshotTTL := 14 * 24 * time.Hour // each snapshot is only preserved for 1 week

	concurrency := 5

	snapshotter := Snapshotter{
		db, log, transport, senders,
		&mu, concurrency, NewAlarmClock(wakeupInterval),
		pollInterval, chaseInterval, noContentTTL, snapshotTTL,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go snapshotter.Start(ctx)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Sugar().Info("Trying to stop snapshotter")
			snapshotter.Stop()
			return nil
		},
	})

	return &snapshotter
}

type Snapshotter struct {
	db        *gorm.DB
	log       *zap.Logger
	transport http.RoundTripper
	senders   senders.Registry

	mu          *sync.Mutex
	concurrency int
	alarmClock  *alarmClock

	pollInterval  time.Duration // We only poll this subscription when the last poll this long ago
	chaseInterval time.Duration // When a subscription is updated, we'll poll it again after this duration
	noContentTTL  time.Duration // Purge subscription if it has no content for this duration
	snapshotTTL   time.Duration // Purge snapshots older than this
}

func (s *Snapshotter) Start(ctx context.Context) {
	c := s.alarmClock.Start(ctx)

	go func() {
		for evt := range c {
			s.handleEvent(evt)
		}
	}()
}

func (s *Snapshotter) Stop() {
	mu.Lock()
	s.alarmClock.Stop()
	s.log.Sugar().Info("Snapshotter stopped")
}

func (s *Snapshotter) handleEvent(evt Event) {
	mu.Lock()
	defer mu.Unlock()

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	switch evt.(type) {
	case pollWakeupEvent:
		s.pollSnapshots(ctx, evt.Timestamp())
	case chaseWakeupEvent:
		s.chaseSubscriptions(ctx, evt.Timestamp())
	}
}

func (s *Snapshotter) SendSnapshot(ctx context.Context, sub *models.Subscription, before, after *models.Snapshot) error {
	notifier := sub.Notifier

	sender, ok := s.senders[notifier.Platform]
	if !ok {
		return fmt.Errorf("unsupported notifier platform: %s", notifier.Platform)
	}

	_, err := sender.SendSnapshot(ctx, &notifier, sub, before, after)
	if err != nil {
		s.log.Sugar().Infow("Failed to send update", "err", err)
	}
	return err
}

func (s *Snapshotter) GetEndpointContent(ctx context.Context, endpoint, xpath string) (*models.EndpointContent, error) {
	ret := &models.EndpointContent{}

	var html string
	err := requests.URL(endpoint).
		Transport(s.transport).
		ToString(&html).
		Fetch(ctx)
	doc, err := htmlquery.Parse(strings.NewReader(html))
	if err != nil {
		return ret, err
	}

	ret.Text = SelectText(doc, xpath)
	ret.Title = SelectText(doc, "/html/head/title")
	ret.ImageURL = ExtractImageURL(doc)
	return ret, nil
}
