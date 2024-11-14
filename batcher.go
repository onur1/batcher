package batcher

import (
	"context"
	"time"
)

// Batcher collects values and periodically flushes them based on a specified batch size
// or a time interval. It emits Counts indicating how many values have been collected within
// each interval.
type Batcher[A any] struct {
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

	var ok bool // Tracks whether the input channel is closed

LOOP:
	for {
		select {
		case <-ctx.Done():
			// Exit the loop if context is done
			break LOOP
		case _, ok = <-input:
			if !ok {
				// If input is closed, set to nil to avoid further reads
				input = nil
				break
			}
			// Update state based on input
			t.Value++
			t.Time = time.Now()
		case <-ticker.C:
			// Update state on ticker
			v.prev = v.now
			v.now = b.nextState(t, v.now)
			if v.now.pending == 0 && v.prev.cursor < v.now.cursor {
				// Send count difference if there's a cursor change
				output <- Count{
					Value: v.now.cursor - v.prev.cursor,
					Time:  v.now.seen,
				}
			}
			// Exit if input is closed and all items are flushed
			if input == nil && t.Value == v.now.cursor {
				break LOOP
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
