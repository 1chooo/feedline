package service

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/1chooo/ad-service/internal/model"
)

type EventStore interface {
	InsertAdEvents(ctx context.Context, events []model.AdEvent) error
}

type EventSink interface {
	Track(event model.AdEvent)
}

type noOpEventSink struct{}

func (noOpEventSink) Track(model.AdEvent) {}

// BufferedEventSink keeps delivery reads independent from analytics writes.
// It stores only ad IDs, event types, and timestamps; no user profile data is retained.
type BufferedEventSink struct {
	store      EventStore
	events     chan model.AdEvent
	done       chan struct{}
	stopped    chan struct{}
	flushEvery time.Duration
	dropped    atomic.Int64
	closeOnce  sync.Once
}

func NewBufferedEventSink(store EventStore, capacity int, flushEvery time.Duration) *BufferedEventSink {
	if capacity < 1 {
		capacity = 1
	}
	if flushEvery <= 0 {
		flushEvery = time.Second
	}
	sink := &BufferedEventSink{
		store:      store,
		events:     make(chan model.AdEvent, capacity),
		done:       make(chan struct{}),
		stopped:    make(chan struct{}),
		flushEvery: flushEvery,
	}
	go sink.run()
	return sink
}

func (s *BufferedEventSink) Track(event model.AdEvent) {
	select {
	case s.events <- event:
	default:
		s.dropped.Add(1)
	}
}

func (s *BufferedEventSink) Dropped() int64 {
	return s.dropped.Load()
}

func (s *BufferedEventSink) Close() {
	s.closeOnce.Do(func() {
		close(s.done)
		<-s.stopped
	})
}

func (s *BufferedEventSink) run() {
	defer close(s.stopped)
	ticker := time.NewTicker(s.flushEvery)
	defer ticker.Stop()

	batch := make([]model.AdEvent, 0, 256)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := s.store.InsertAdEvents(ctx, batch); err != nil {
			s.dropped.Add(int64(len(batch)))
		}
		cancel()
		batch = batch[:0]
	}

	for {
		select {
		case event := <-s.events:
			batch = append(batch, event)
			if len(batch) == cap(batch) {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-s.done:
			for {
				select {
				case event := <-s.events:
					batch = append(batch, event)
				default:
					flush()
					return
				}
			}
		}
	}
}
