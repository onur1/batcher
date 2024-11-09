package batcher_test

import (
	"context"
	"fmt"
	"time"

	"github.com/onur1/batcher"
)

func ExampleBatcher() {
	// Initialize the batcher with a 4-second flush interval and a batch size of 5
	batchSize := 5
	flushInterval := 4 // in seconds
	b := batcher.NewBatcher[int](flushInterval, batchSize)

	// Channel for sending items to the batcher
	input := make(chan int)
	dst := make(chan batcher.Count)

	// Start the batcher's processing loop
	go b.CountLoop(context.TODO(), input, dst)

	go func() {
		// Simulate sending items to the batcher
		for i := 1; i <= 7; i++ {
			input <- i
			if i >= 4 {
				time.Sleep(500 * time.Millisecond)
			}
		}
		close(input)
	}()

	// Receive and process batches from the batcher's output channel
	for count := range dst {
		fmt.Printf("Batch of %d items received after %.2f seconds\n", count.Value, time.Since(count.Time).Seconds())
	}

	// Output:
	// Batch of 5 items received after 0.50 seconds
	// Batch of 2 items received after 4.50 seconds
}
