package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/antchfx/htmlquery"
	"github.com/carlmjohnson/requests"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var mu sync.Mutex

func NewSnapshotter(lc fx.Lifecycle, db *gorm.DB, log *zap.Logger, transport http.RoundTripper) *Snapshotter {
	wakeupInterval := 5 * time.Second
	subscriptionPollAge := 1 * time.Hour // poll each subscription every hour
	noContentTTL := 7 * 24 * time.Hour   // stop polling subscription if no data is returned for the past week
	snapshotTTL := 14 * 24 * time.Hour   // each snapshot is only preserved for 1 week

	concurrency := 5
	var mu sync.Mutex

	snapshotter := Snapshotter{
		db, log, transport,
		&mu, concurrency, nil,
		wakeupInterval, subscriptionPollAge, noContentTTL, snapshotTTL,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go snapshotter.Start()
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

	mu          *sync.Mutex
	concurrency int
	cancel      context.CancelFunc

	wakeupInterval           time.Duration // Interval to check for pollable subscriptions
	subscriptionPollInterval time.Duration // We only poll this subscription when the last poll this long ago
	noContentTTL             time.Duration // Purge subscription if it has no content for this duration
	snapshotTTL              time.Duration // Purge snapshots older than this
}

func (s *Snapshotter) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	ticker := time.NewTicker(s.wakeupInterval)

	for {
		select {
		case <-ctx.Done():
			// Locking here to wait for in-flight requests to finish
			mu.Lock()

			s.log.Sugar().Info("Snapshotter stopped")
			return

		case wakeupTime := <-ticker.C:
			s.collectSnapshots(ctx, wakeupTime)
		}
	}
}

func (s *Snapshotter) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *Snapshotter) collectSnapshots(ctx context.Context, wakeupTime time.Time) {
	mu.Lock()
	defer mu.Unlock()

	s.log.Sugar().Info("Snapshotter is waking up")

	m := s.findSubscriptionsForPoll(ctx, wakeupTime, s.collectBatch)

	if m.totalSelected == 0 {
		s.log.Sugar().Infof("No subscriptions processed")
	} else {
		s.log.Sugar().Infow(
			fmt.Sprintf("Processed %d subscriptions", m.totalSelected),
			"errored", m.errored, "updated", m.updated, "unchanged", m.unchanged, "empty", m.empty,
		)
	}

	s.purgeOldSnapshots(ctx, wakeupTime)

	elapsed := time.Now().UTC().Sub(wakeupTime)
	s.log.Sugar().Infof("Snapshotter finished after %d seconds", int(elapsed.Seconds()))
}

type snapshotMetrics struct {
	totalSelected int
	updated       int
	unchanged     int
	empty         int
	errored       int
}

func (m *snapshotMetrics) Add(other *snapshotMetrics) {
	m.totalSelected += other.totalSelected
	m.updated += other.updated
	m.unchanged += other.unchanged
	m.empty += other.empty
}

func (s *Snapshotter) findSubscriptionsForPoll(
	ctx context.Context, now time.Time,
	callbackPerBatch func(context.Context, Subscriptions) (*snapshotMetrics, []error),
) *snapshotMetrics {
	lastPollCutoff := now.Add(-s.subscriptionPollInterval)
	noContentCutoff := now.Add(-s.noContentTTL)

	var subs Subscriptions
	var metrics = &snapshotMetrics{}
	s.db.
		Where("no_content_since > ? AND last_poll_time <= ?", noContentCutoff, lastPollCutoff).
		Or("last_poll_time IS NULL").
		FindInBatches(&subs, s.concurrency, func(tx *gorm.DB, batch int) error {
			batchMetrics, errs := callbackPerBatch(ctx, subs)
			if len(errs) > 0 {
				s.log.Sugar().Errorf("snapshot: batch %d errors: %+v", batch, errs)
			}

			metrics.totalSelected += len(subs)
			metrics.Add(batchMetrics)

			return nil
		})

	return metrics
}

func (s *Snapshotter) collectBatch(ctx context.Context, batch Subscriptions) (*snapshotMetrics, []error) {
	var wg sync.WaitGroup
	var metrics = &snapshotMetrics{}

	errs := make([]error, 0)
	for _, sub := range batch {
		wg.Add(1)

		go func() {
			defer wg.Done()
			m, err := s.collectRecurringSnapshot(ctx, &sub)
			if err != nil {
				errs = append(errs, err)
			}
			metrics.Add(m)
		}()
	}

	wg.Wait()
	return metrics, errs
}

func (s *Snapshotter) collectRecurringSnapshot(ctx context.Context, sub *Subscription) (*snapshotMetrics, error) {
	var m = &snapshotMetrics{}
	content, err := s.GetEndpointContent(ctx, sub.Endpoint, sub.XPath)
	if err != nil {
		m.errored += 1
		return m, err
	}

	requestedAt := time.Now().UTC()

	if content != "" {
		updated, err := s.handleContent(ctx, sub, requestedAt, content)
		switch {
		case err != nil:
			m.errored += 1

		case updated:
			m.updated += 1

		case !updated:
			m.unchanged += 1
		}
		return m, err

	} else {
		m.empty += 1
		return m, s.handleEmptyContent(ctx, sub, requestedAt)
	}
}

func (s *Snapshotter) handleContent(ctx context.Context, sub *Subscription, timestamp time.Time, content string) (bool, error) {
	digest := DigestContent(content)

	var count int64
	var snap Snapshot
	tx := s.db.Model(&snap).Where("subscription_id = ? AND content_digest = ?", sub.ID, digest).Count(&count)
	if err := tx.Error; err != nil {
		return false, tx.Error
	}

	var changed bool
	var tx2 *gorm.DB
	if count == 1 {
		changed = false
		tx2 = s.db.Model(&snap).
			Where("content_digest = ?", digest).
			Update("timestamp", timestamp)
	} else {
		changed = true

		snap.Timestamp = timestamp
		snap.UserID = sub.UserID
		snap.SubscriptionID = sub.ID
		snap.Content = content
		snap.ContentDigest = digest
		tx2 = s.db.Create(&snap)
	}
	return changed, tx2.Error
}

func (s *Snapshotter) handleEmptyContent(ctx context.Context, sub *Subscription, timestamp time.Time) error {
	if sub.NoContentSince.Valid {
		// Don't do anything if we already observed empty content on this subscription
		return nil

	} else {
		tx := s.db.Model(sub).Update("no_content_since", timestamp)
		return tx.Error
	}
}

func (s *Snapshotter) purgeOldSnapshots(ctx context.Context, timestamp time.Time) {
	retentionCutoff := timestamp.Add(-s.snapshotTTL)

	tx := s.db.Delete(&Snapshot{}, "timestamp < ?", retentionCutoff)
	if err := tx.Error; err != nil {
		s.log.Sugar().Errorf("purgeOldSnapshots error: %+v", err)
	}
	s.log.Sugar().Infof("Purged %d old snapshots", tx.RowsAffected)
	return
}

func (s *Snapshotter) GetEndpointContent(ctx context.Context, endpoint, xpath string) (string, error) {
	var html string
	err := requests.URL(endpoint).
		Transport(s.transport).
		ToString(&html).
		Fetch(ctx)
	doc, err := htmlquery.Parse(strings.NewReader(html))
	if err != nil {
		return "", err
	}

	node := htmlquery.FindOne(doc, xpath)
	content := collectText(node)
	return content, nil
}
