
# batcher

**batcher** is a Go library that buffers and batches data items based on configurable thresholds of count or time interval. It enables efficient data handling by processing items in batches either when a specified number of items is collected or when a time interval has passed since the last batch.

## Features

- **Batch by Count**: Collects and processes data once a specified item count is reached.
- **Batch by Time**: Processes data at regular intervals if the item count is not yet reached.

## Installation

To install the `batcher` library, use the following `go get` command:

```sh
go get github.com/onur1/batcher
```

## Usage

The following example sets up a `Batcher` that flushes either when 5 items are collected or 4 seconds have elapsed since the last flush:

```go
package main

import (
    "context"
    "fmt"
    "time"
    "github.com/onur1/batcher"
)

func main() {
    // Initialize the batcher with a 4-second flush interval and a batch size of 5
    batchSize := 5
    flushInterval := 4 // in seconds
    b := batcher.NewBatcher[int](flushInterval, batchSize)

    // Channels for input and batched output
    input := make(chan int)
    output := make(chan batcher.Count)

    // Start the batcher's processing loop
    go b.CountLoop(context.TODO(), input, output)

    // Simulate sending items to the batcher
    go func() {
        for i := 1; i <= 10; i++ {
            input <- i
            time.Sleep(500 * time.Millisecond) // Adjust to control item arrival rate
        }
        close(input) // Close input channel to end batching after all items are sent
    }()

    // Process batches from the batcher's output channel
    for count := range output {
        fmt.Printf("Batch of %d items received at %v\n", count.Value, count.Time)
    }
}

// Output:
// Batch of 5 items received after 0.50 seconds
// Batch of 2 items received after 4.50 seconds
```

### Explanation

- **NewBatcher**: Creates a new `Batcher` instance with the specified flush interval (in seconds) and batch size.
- **CountLoop**: Starts the main loop that listens to incoming items on the `input` channel and flushes batches based on count or time interval. Results are sent to the `output` channel provided by the caller.
- **output Channel**: An output channel where `Count` structs are sent, representing each batch with the item count and the timestamp of the batch.

## API

### `NewBatcher`

```go
func NewBatcher[A any](flushInterval, batchSize int) *Batcher[A]
```

Creates a new `Batcher` instance.

- `flushInterval`: Time interval in seconds for flushing the batch if the count threshold hasn’t been met.
- `batchSize`: Number of items to collect before the batch is flushed.

### `CountLoop`

```go
func (b *Batcher[A]) CountLoop(ctx context.Context, input <-chan A, output chan Count)
```

Begins the batch processing loop. It listens for items on the `input` channel and emits batches based on count or time interval to the `output` channel. The caller is responsible for providing this output channel.

**Count**: Represents a single batch, including the number of items in the batch and the timestamp at which it was created.

## License

MIT License. See [LICENSE](LICENSE) for details.
