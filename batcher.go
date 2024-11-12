package batcher

import (
	"context"
	"time"
)

// Batcher collects values and periodically flushes them based on a specified batch size
// or a time interval. It emits Counts indicating how many values have been collected within
// each interval.
type Batcher[A any] struct {
	// C             chan Count // Channel through which Counts are emitted
	flushInterval int // Flush interval in seconds
	batchSize     int // Number of items to collect before flushing
}

// NewBatcher initializes and returns a new Batcher with the provided
// flush interval and batch size.
func NewBatcher[A any](flushInterval, batchSize int) *Batcher[A] {
	if flushInterval < 1 {
		flushInterval = 1
	}
	return &Batcher[A]{
		// C:             make(chan Count),
		flushInterval: flushInterval,
		batchSize:     batchSize,
	}
}

// Count represents a batch of values collected by the batcher, along with the timestamp
// when it was created.
type Count struct {
	Value int       // The number of items in the batch
	Time  time.Time // The time at which the count was taken
}

type prevCur[A any] struct {
	now  A
	prev A
}

// CountLoop listens for incoming items on a provided channel and flushes the counts
// either when the batch size is reached or the flush interval has passed.
func (b *Batcher[A]) CountLoop(ctx context.Context, input <-chan A, output chan Count) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	t, v := Count{}, prevCur[batcherState]{now: batcherState{}}

	cctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var ok bool

LOOP:
	for {
		select {
		case <-cctx.Done():
			break LOOP
		case _, ok = <-input:
			if !ok {
				input = nil
				break
			}
			t.Value = t.Value + 1
			t.Time = time.Now()
		case <-ticker.C:
			v.prev = v.now
			v.now = b.nextState(t, v.now)
			if v.now.pending == 0 && (v.prev.cursor < v.now.cursor) {
				output <- Count{
					Value: v.now.cursor - v.prev.cursor,
					Time:  v.now.seen,
				}
				if input == nil {
					cancel()
				}
			}
		}
	}

	close(output)
}

type batcherState struct {
	seen    time.Time
	pending int
	cursor  int
}

func (b *Batcher[A]) nextState(last Count, s batcherState) batcherState {
	cursor := last.Value - s.cursor
	n := batcherState{
		cursor:  s.cursor,
		pending: s.pending,
	}
	if s.pending > 0 {
		if cursor >= b.batchSize || s.pending == 1 {
			n.pending = 0
			n.cursor = last.Value
		} else {
			n.pending -= 1
		}
	} else if s.seen.Before(last.Time) {
		if cursor >= b.batchSize {
			n.cursor = last.Value
		} else {
			n.pending = b.flushInterval
		}
	}
	n.seen = last.Time
	return n
}
