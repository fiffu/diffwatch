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

type alarmStartupEvent struct {
	event
}

type alarmShutdownEvent struct {
	event
}

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

type IntervalsConfig struct {
	Wakeup time.Duration
	Chase  time.Duration
}

func NewAlarmClock(intervals IntervalsConfig) *alarmClock {
	return &alarmClock{
		wakeupTimer: time.NewTicker(intervals.Wakeup),
		chaseTimer:  time.NewTicker(intervals.Chase),
		cancel:      nil,
		C:           make(chan Event),
	}
}

func (a *alarmClock) newEvent() event {
	return event{time.Now().UTC()}
}

func (a *alarmClock) buildEvent(t time.Time) event {
	return event{t}
}

func (a *alarmClock) Start() <-chan Event {
	alarmCtx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel

	go func() {
		a.C <- alarmStartupEvent{a.newEvent()}

		for {
			select {
			case t := <-a.wakeupTimer.C:
				a.C <- pollWakeupEvent{a.buildEvent(t)}

			case t := <-a.chaseTimer.C:
				a.C <- chaseWakeupEvent{a.buildEvent(t)}

			case <-alarmCtx.Done():
				a.C <- alarmShutdownEvent{a.newEvent()}
				close(a.C)
				return
			}
		}
	}()

	return a.C
}

func (a *alarmClock) Stop() {
	a.cancel()
	a.wakeupTimer.Stop()
	a.chaseTimer.Stop()
}
