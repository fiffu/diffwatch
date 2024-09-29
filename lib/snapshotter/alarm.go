package snapshotter

import (
	"context"
	"time"
)

type Event interface {
	Timestamp() time.Time
}

type event struct{ timestamp time.Time }

func (e event) Timestamp() time.Time { return e.timestamp }

type pollWakeupEvent struct {
	event
}

type chaseWakeupEvent struct {
	event
}

type alarmClock struct {
	cancel      func()
	wakeupTimer *time.Ticker
	chaseTimer  *time.Ticker
	C           chan Event
}

func NewAlarmClock(wakeupInterval time.Duration) *alarmClock {
	return &alarmClock{
		wakeupTimer: time.NewTicker(wakeupInterval),
		chaseTimer:  time.NewTicker(5 * time.Minute),
		C:           make(chan Event),
	}
}

func (a *alarmClock) Start(ctx context.Context) <-chan Event {
	ctx, cancel := context.WithCancel(ctx)
	a.cancel = cancel

	go func() {
		immediateWakeupEvent := pollWakeupEvent{event{time.Now()}}
		a.C <- immediateWakeupEvent

		select {
		case t := <-a.wakeupTimer.C:
			a.C <- pollWakeupEvent{event{t}}

		case t := <-a.chaseTimer.C:
			a.C <- chaseWakeupEvent{event{t}}

		case <-ctx.Done():
			return
		}
	}()

	return a.C
}

func (a *alarmClock) Stop() {
	a.cancel()
	a.wakeupTimer.Stop()
	a.chaseTimer.Stop()
	close(a.C)
}
