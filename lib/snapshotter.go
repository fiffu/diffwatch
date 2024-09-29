package lib

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
	"gorm.io/gorm/clause"
)

var mu sync.Mutex

func NewSnapshotter(lc fx.Lifecycle, db *gorm.DB, log *zap.Logger, transport http.RoundTripper, senders senders.Registry) *Snapshotter {
	wakeupInterval := 1 * time.Hour
	subscriptionPollInterval := 1 * time.Hour // poll each subscription every hour
	noContentTTL := 7 * 24 * time.Hour        // stop polling subscription if no data is returned for the past week
	snapshotTTL := 14 * 24 * time.Hour        // each snapshot is only preserved for 1 week

	concurrency := 5
	var mu sync.Mutex

	snapshotter := Snapshotter{
		db, log, transport, senders,
		&mu, concurrency, nil,
		wakeupInterval, subscriptionPollInterval, noContentTTL, snapshotTTL,
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
	senders   senders.Registry

	mu          *sync.Mutex
	concurrency int
	cancel      context.CancelFunc

	wakeupInterval           time.Duration // Interval to check for pollable subscriptions
	subscriptionPollInterval time.Duration // We only poll this subscription when the last poll this long ago
	noContentTTL             time.Duration // Purge subscription if it has no content for this duration
	snapshotTTL              time.Duration // Purge snapshots older than this
}

type snapshotMetrics struct {
	totalSelected int
	updated       int
	unchanged     int
	errored       int
}

func (s *Snapshotter) tickerWithImmediateTick(interval time.Duration) *time.Ticker {
	withImmediateTick := make(chan time.Time, 1)

	ticker := time.NewTicker(interval)
	tickerC := ticker.C
	go func() {
		withImmediateTick <- time.Now()
		for c := range tickerC {
			withImmediateTick <- c
		}
	}()

	ticker.C = withImmediateTick
	return ticker
}

func (s *Snapshotter) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	ticker := s.tickerWithImmediateTick(s.wakeupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Locking here to wait for in-flight requests to finish
			mu.Lock()

			s.log.Sugar().Info("Snapshotter stopped")
			return

		case batchStartTime := <-ticker.C:
			s.collectSnapshots(ctx, batchStartTime.UTC())
		}
	}
}

func (s *Snapshotter) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *Snapshotter) collectSnapshots(ctx context.Context, batchStartTIme time.Time) {
	mu.Lock()
	defer mu.Unlock()

	m := s.findSubscriptionsForPoll(ctx, batchStartTIme, s.collectBatch)

	if m.totalSelected > 0 {
		args := make([]any, 0)
		if m.errored != 0 {
			args = append(args, "errored", m.errored)
		}
		if m.updated != 0 {
			args = append(args, "updated", m.updated)
		}
		if m.unchanged != 0 {
			args = append(args, "unchanged", m.unchanged)
		}

		s.log.Sugar().Infow(
			fmt.Sprintf("Processed %d subscriptions", m.totalSelected),
			args...,
		)
	}

	s.purgeOldSnapshots(ctx, batchStartTIme)

	elapsed := time.Now().UTC().Sub(batchStartTIme)
	s.log.Sugar().Infow("Snapshotter completed", "elapsed_msecs", int(elapsed.Milliseconds()))
}

func (m *snapshotMetrics) Add(other *snapshotMetrics) {
	m.totalSelected += other.totalSelected
	m.updated += other.updated
	m.unchanged += other.unchanged
}

func (s *Snapshotter) findSubscriptionsForPoll(
	ctx context.Context,
	batchStartTime time.Time,
	callbackPerBatch func(context.Context, models.Subscriptions, time.Time) (*snapshotMetrics, []error),
) *snapshotMetrics {
	lastPollCutoff := batchStartTime.Add(-s.subscriptionPollInterval)
	noContentCutoff := batchStartTime.Add(-s.noContentTTL)

	var subs models.Subscriptions
	var metrics = &snapshotMetrics{}
	s.db.
		Where("no_content_since IS NULL OR no_content_since > ?", noContentCutoff).
		Where("last_poll_time IS NULL OR last_poll_time <= ?", lastPollCutoff).
		InnerJoins("Notifier").
		FindInBatches(&subs, s.concurrency, func(tx *gorm.DB, batch int) error {
			batchMetrics, errs := callbackPerBatch(ctx, subs, batchStartTime)
			if len(errs) > 0 {
				s.log.Sugar().Warnf("snapshot: batch errors: %+v", errs)
			}

			metrics.totalSelected += len(subs)
			metrics.Add(batchMetrics)

			return nil
		})

	return metrics
}

func (s *Snapshotter) collectBatch(ctx context.Context, batch models.Subscriptions, batchStartTime time.Time) (*snapshotMetrics, []error) {
	var wg sync.WaitGroup
	var metrics = &snapshotMetrics{}

	errs := make([]error, 0)
	for _, sub := range batch {
		wg.Add(1)

		go func() {
			defer wg.Done()
			m, err := s.collectRecurringSnapshot(ctx, sub)
			if err != nil {
				errs = append(errs, err)
			}
			metrics.Add(m)
		}()
	}

	tx := s.db.Model(&batch).Update("last_poll_time", batchStartTime)
	if err := tx.Error; err != nil {
		errs = append(errs, err)
	}

	wg.Wait()
	return metrics, errs
}

func (s *Snapshotter) collectRecurringSnapshot(ctx context.Context, sub *models.Subscription) (*snapshotMetrics, error) {
	var m = &snapshotMetrics{}
	var errMetric = &snapshotMetrics{errored: 1}

	content, err := s.GetEndpointContent(ctx, sub.Endpoint, sub.XPath)
	if err != nil {
		s.log.Sugar().Errorw("error collecting snapshot", "err", err)
		return errMetric, err
	}

	requestedAt := time.Now().UTC()

	isChanged, err := s.handleContent(ctx, sub, requestedAt, content)
	switch {
	case err != nil:
		s.log.Sugar().Errorw("error handling snapshot content", "err", err)
		return errMetric, err
	case isChanged:
		m.updated += 1
	default:
		m.unchanged += 1
	}

	if content.Text != "" {
		if err := s.handleEmptyContent(ctx, sub, requestedAt); err != nil {
			return errMetric, err
		}
	}
	return m, err
}

func (s *Snapshotter) handleContent(ctx context.Context, sub *models.Subscription, timestamp time.Time, content *models.EndpointContent) (isChanged bool, err error) {
	currDigest := models.DigestContent(content.Text)

	var firstSeen bool
	var prevSnap models.Snapshot
	tx := s.db.Where("subscription_id = ?", sub.ID).Order("timestamp desc").First(&prevSnap)

	switch tx.Error {
	case gorm.ErrRecordNotFound:
		// First time we see this, so this is considered a change
		firstSeen = true
	case nil:
		if prevSnap.ContentDigest == currDigest {
			// Not changed
			tx := s.db.Model(&prevSnap).Where("timestamp = ?", prevSnap.Timestamp).Update("timestamp", timestamp)
			err = tx.Error
			return
		}
	default:
		// There is an error
		err = tx.Error
		return
	}

	isChanged = true
	newSnap := models.Snapshot{
		Timestamp:      timestamp,
		UserID:         sub.UserID,
		SubscriptionID: sub.ID,
		Content:        content.Text,
		ContentDigest:  currDigest,
	}

	tx2 := s.db.Clauses(clause.Returning{}).Create(&newSnap)
	if err = tx2.Error; err != nil {
		return
	} else {
		p := &prevSnap
		if firstSeen {
			p = nil
		}
		err = s.SendSnapshot(ctx, sub, p, &newSnap)
		if err != nil {
			s.log.Sugar().Errorw("Failed to send snapshot", "err", err)
		}
		return
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

func (s *Snapshotter) handleEmptyContent(ctx context.Context, sub *models.Subscription, timestamp time.Time) error {
	if sub.NoContentSince.Valid {
		// Don't do anything if we already observed empty content on this subscription
		return nil

	} else {
		tx := s.db.Model(sub).Update("no_content_since", timestamp)
		return tx.Error
	}
}

func (s *Snapshotter) purgeOldSnapshots(ctx context.Context, batchStartTime time.Time) {
	retentionCutoff := batchStartTime.Add(-s.snapshotTTL)

	tx := s.db.Delete(&models.Snapshot{}, "timestamp < ?", retentionCutoff)
	if err := tx.Error; err != nil {
		s.log.Sugar().Errorf("purgeOldSnapshots error: %+v", err)
	}
	if tx.RowsAffected > 0 {
		s.log.Sugar().Infof("Purged %d old snapshots", tx.RowsAffected)
	}
	return
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
