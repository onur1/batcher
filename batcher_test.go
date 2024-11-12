package batcher

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestBatcherBatchFlush tests that Batcher flushes when the batch size is reached.
func TestBatcherBatchFlush(t *testing.T) {
	batchSize := 5
	flushInterval := 2 // in seconds
	batcher := NewBatcher[int](flushInterval, batchSize)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	input := make(chan int)
	output := make(chan Count)
	go batcher.CountLoop(ctx, input, output)

	// Send items equal to batch size
	for i := 0; i < batchSize; i++ {
		input <- i
	}

	// Wait for the batcher to emit a batch
	select {
	case count := <-output:
		if count.Value != batchSize {
			t.Errorf("Expected batch size %d, got %d", batchSize, count.Value)
		}
	case <-time.After(time.Second * 3):
		t.Error("Expected a batch flush, but timed out waiting for it")
	}
}

// TestBatcherIntervalFlush tests that Batcher flushes after the flush interval.
func TestBatcherIntervalFlush(t *testing.T) {
	batchSize := 5
	flushInterval := 2 // 2 second interval
	batcher := NewBatcher[int](flushInterval, batchSize)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	input := make(chan int)
	output := make(chan Count)
	go batcher.CountLoop(ctx, input, output)

	// Send fewer items than the batch size
	for i := 0; i < batchSize-2; i++ {
		input <- i
	}

	// Wait for the batcher to emit due to interval flush
	select {
	case count := <-output:
		if count.Value != batchSize-2 {
			t.Errorf("Expected interval flush of %d items, got %d", batchSize-2, count.Value)
		}
	case <-time.After(time.Second * 4):
		t.Error("Expected an interval flush, but timed out waiting for it")
	}
}

// TestBatcherNoFlushWithoutData verifies no flush occurs if there are no items in the batch.
func TestBatcherNoFlushWithoutData(t *testing.T) {
	batchSize := 5
	flushInterval := 1 // in seconds
	batcher := NewBatcher[int](flushInterval, batchSize)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	input := make(chan int)
	output := make(chan Count)
	go batcher.CountLoop(ctx, input, output)

	// Wait for a longer interval than the flush interval
	select {
	case <-output:
		t.Error("Expected no flush without data, but received a flush")
	case <-time.After(time.Second * 2):
		// Test passes if we reach here without a flush
	}
}

func TestBatcherState(t *testing.T) {
	bc := func(nsecs int64, pending, cursor int) batcherState {
		return batcherState{
			seen:    time.Unix(0, nsecs).UTC(),
			pending: pending,
			cursor:  cursor,
		}

	}
	var (
		c = NewBatcher[int](2, 3)
		d = time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
		s batcherState
	)

	t.Run("start pending if the count is less than the limit", func(t *testing.T) {
		s = c.nextState(Count{Value: 1, Time: d}, batcherState{})
		assert.Equal(t, bc(1258490098651387237, 2, 0), s)
	})

	t.Run("decrease pending if the state has not changed", func(t *testing.T) {
		s = c.nextState(Count{Value: 1, Time: d}, s)
		assert.Equal(t, bc(1258490098651387237, 1, 0), s)
	})

	t.Run("emit new count once pending has reached 0", func(t *testing.T) {
		s = c.nextState(Count{Value: 1, Time: d}, s)
		assert.Equal(t, bc(1258490098651387237, 0, 1), s)
		// noop
		s = c.nextState(Count{Value: 1, Time: d}, s)
		assert.Equal(t, bc(1258490098651387237, 0, 1), s)
	})

	t.Run("once again start pending and emit after 2 attempts", func(t *testing.T) {
		d = d.Add(time.Minute)

		s = c.nextState(Count{Value: 3, Time: d}, s)
		assert.Equal(t, bc(1258490158651387237, 2, 1), s)

		s = c.nextState(Count{Value: 3, Time: d}, s)
		assert.Equal(t, bc(1258490158651387237, 1, 1), s)

		s = c.nextState(Count{Value: 3, Time: d}, s)
		assert.Equal(t, bc(1258490158651387237, 0, 3), s)
	})

	t.Run("emit right-away because the Value 8(-3) is gt the buffer size ", func(t *testing.T) {
		d = d.Add(time.Minute)

		s = c.nextState(Count{Value: 8, Time: d}, s)
		assert.Equal(t, bc(1258490218651387237, 0, 8), s)
	})

	t.Run("emit the last received Value", func(t *testing.T) {
		d = d.Add(time.Minute)

		s = c.nextState(Count{Value: 9, Time: d}, s)
		assert.Equal(t, bc(1258490278651387237, 2, 8), s)

		d = d.Add(time.Minute)

		s = c.nextState(Count{Value: 10, Time: d}, s)
		assert.Equal(t, bc(1258490338651387237, 1, 8), s)

		d = d.Add(time.Minute)

		s = c.nextState(Count{Value: 14, Time: d}, s)
		assert.Equal(t, bc(1258490398651387237, 0, 14), s)

		// noop

		s = c.nextState(Count{Value: 14, Time: d}, s)
		assert.Equal(t, bc(1258490398651387237, 0, 14), s)
	})

	t.Run("emit while pending if the max limit is reached", func(t *testing.T) {
		d = d.Add(time.Minute)

		c.flushInterval = 4

		s = c.nextState(Count{Value: 15, Time: d}, s)
		assert.Equal(t, bc(1258490458651387237, 4, 14), s)

		s = c.nextState(Count{Value: 15, Time: d}, s)
		assert.Equal(t, bc(1258490458651387237, 3, 14), s)

		d = d.Add(time.Minute)

		s = c.nextState(Count{Value: 16, Time: d}, s)
		assert.Equal(t, bc(1258490518651387237, 2, 14), s)

		d = d.Add(time.Minute)

		s = c.nextState(Count{Value: 18, Time: d}, s)
		assert.Equal(t, bc(1258490578651387237, 0, 18), s)
	})
}
