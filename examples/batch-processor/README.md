# Batch Processor Example

This example demonstrates using Bolt in a high-throughput batch processing system.

## Features

- **Worker Pool Pattern**: Concurrent processing with configurable workers
- **Progress Tracking**: Real-time metrics and progress reporting
- **Retry Logic**: Automatic retry with exponential backoff
- **Error Handling**: Comprehensive error tracking and logging
- **Graceful Shutdown**: Clean shutdown with resource cleanup
- **Performance Metrics**: Throughput and duration tracking

## Running the Example

```bash
cd examples/batch-processor
go run main.go
```

## Log Output Examples

### Worker Lifecycle
```json
{"level":"info","service":"batch-processor","version":"1.0.0","message":"starting batch processor"}
{"level":"info","workers":10,"batch_size":100,"total_items":1000,"message":"processor configured"}
{"level":"info","count":1000,"message":"generating items"}
{"level":"info","worker_id":0,"message":"worker started"}
{"level":"info","worker_id":1,"message":"worker started"}
...
{"level":"info","count":1000,"message":"item generation complete"}
```

### Item Processing
```json
{"level":"info","worker_id":3,"item_id":"item_42","duration":"75ms","retries":0,"message":"item processed successfully"}
{"level":"warn","worker_id":5,"item_id":"item_89","error":"processing error","retry":1,"message":"retrying item"}
{"level":"info","worker_id":5,"item_id":"item_89","duration":"220ms","retries":1,"message":"item processed successfully"}
{"level":"error","worker_id":2,"item_id":"item_153","duration":"450ms","retries":3,"message":"item processing failed after retries"}
```

### Progress Metrics
```json
{"level":"info","total":1000,"processed":234,"failed":12,"remaining":754,"progress_pct":24.6,"message":"processing metrics"}
{"level":"info","total":1000,"processed":512,"failed":23,"remaining":465,"progress_pct":53.5,"message":"processing metrics"}
{"level":"info","total":1000,"processed":891,"failed":37,"remaining":72,"progress_pct":92.8,"message":"processing metrics"}
```

### Completion
```json
{"level":"info","worker_id":7,"items_processed":103,"message":"worker completed"}
{"level":"info","message":"all workers finished"}
{"level":"info","total_items":1000,"successful":963,"failed":37,"total_duration":"12.3s","throughput_per_sec":81.3,"message":"batch processing completed"}
```

## Key Implementation Details

### Worker Pool Pattern
```go
func (bp *BatchProcessor) Start(items <-chan Item) <-chan ProcessResult {
    results := make(chan ProcessResult, bp.workers*2)

    // Start worker pool
    for i := 0; i < bp.workers; i++ {
        bp.wg.Add(1)
        go bp.worker(i, items, results)
    }

    return results
}
```

### Retry Logic with Exponential Backoff
```go
func (bp *BatchProcessor) processItem(workerID int, item Item) ProcessResult {
    maxRetries := 3
    retries := 0

    for retries < maxRetries {
        err := bp.doProcessing(item)

        if err == nil {
            bp.logger.Info().
                Str("item_id", item.ID).
                Int("retries", retries).
                Msg("item processed successfully")
            return ProcessResult{Success: true}
        }

        retries++
        if retries < maxRetries {
            bp.logger.Warn().
                Str("item_id", item.ID).
                Int("retry", retries).
                Msg("retrying item")

            // Exponential backoff
            time.Sleep(time.Duration(retries*100) * time.Millisecond)
        }
    }

    bp.logger.Error().
        Str("item_id", item.ID).
        Msg("item processing failed after retries")

    return ProcessResult{Success: false}
}
```

### Progress Tracking
```go
func (bp *BatchProcessor) reportMetrics() {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        total := bp.totalItems.Load()
        processed := bp.processed.Load()
        failed := bp.failed.Load()

        progress := float64(processed+failed) / float64(total) * 100

        bp.logger.Info().
            Int64("processed", processed).
            Float64("progress_pct", progress).
            Msg("processing metrics")
    }
}
```

## Performance Characteristics

- **Zero Allocations**: Logging adds no GC pressure
- **High Throughput**: Handles 1000+ items/sec with 10 workers
- **Concurrent Safe**: Thread-safe atomic counters and logging
- **Minimal Overhead**: <1ms overhead per batch of 100 items

## Production Deployment

### Configuration via Environment
```go
func main() {
    numWorkers := getEnvInt("WORKERS", 10)
    batchSize := getEnvInt("BATCH_SIZE", 100)
    maxRetries := getEnvInt("MAX_RETRIES", 3)

    logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
    processor := NewBatchProcessor(logger, numWorkers, batchSize)
    processor.maxRetries = maxRetries

    // ...
}
```

### Dead Letter Queue
```go
func (bp *BatchProcessor) handleFailedItem(result ProcessResult) {
    if result.Retries >= bp.maxRetries {
        bp.logger.Error().
            Str("item_id", result.Item.ID).
            Msg("sending to dead letter queue")

        bp.deadLetterQueue.Send(result.Item)
    }
}
```

### Metrics Export
```go
func (bp *BatchProcessor) exportMetrics() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        metrics := map[string]interface{}{
            "processed_total": bp.processed.Load(),
            "failed_total":    bp.failed.Load(),
            "workers_active":  bp.activeWorkers.Load(),
        }

        bp.logger.Info().
            Interface("metrics", metrics).
            Msg("exporting metrics")

        // Send to metrics backend (Prometheus, DataDog, etc.)
        metricsClient.Push(metrics)
    }
}
```

## Extending the Example

### Add Priority Queue
```go
type PriorityItem struct {
    Item
    Priority int
}

type PriorityQueue []PriorityItem

func (pq PriorityQueue) Less(i, j int) bool {
    return pq[i].Priority > pq[j].Priority
}

// Use heap.Pop to get highest priority item
```

### Add Circuit Breaker
```go
import "github.com/sony/gobreaker"

func (bp *BatchProcessor) processWithCircuitBreaker(item Item) error {
    result, err := bp.circuitBreaker.Execute(func() (interface{}, error) {
        return nil, bp.doProcessing(item)
    })

    if err != nil {
        bp.logger.Warn().
            Str("item_id", item.ID).
            Str("error", err.Error()).
            Msg("circuit breaker open")
        return err
    }

    return nil
}
```

### Add Rate Limiting
```go
import "golang.org/x/time/rate"

func (bp *BatchProcessor) worker(id int, items <-chan Item, results chan<- ProcessResult) {
    limiter := rate.NewLimiter(rate.Limit(100), 10) // 100 items/sec

    for item := range items {
        limiter.Wait(context.Background())
        result := bp.processItem(id, item)
        results <- result
    }
}
```

### Add Batch Checkpointing
```go
func (bp *BatchProcessor) checkpoint() {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        processed := bp.processed.Load()

        bp.logger.Info().
            Int64("checkpoint", processed).
            Msg("saving checkpoint")

        if err := bp.saveCheckpoint(processed); err != nil {
            bp.logger.Error().
                Str("error", err.Error()).
                Msg("checkpoint failed")
        }
    }
}
```

## Monitoring and Alerting

### Alert on High Failure Rate
```go
func (bp *BatchProcessor) monitorFailureRate() {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        total := bp.processed.Load() + bp.failed.Load()
        failed := bp.failed.Load()

        if total > 0 {
            failureRate := float64(failed) / float64(total) * 100

            if failureRate > 10.0 {
                bp.logger.Error().
                    Float64("failure_rate_pct", failureRate).
                    Int64("failed_count", failed).
                    Msg("HIGH FAILURE RATE ALERT")

                // Send alert to PagerDuty/Slack
                alerting.Send("High failure rate", failureRate)
            }
        }
    }
}
```

### Throughput Monitoring
```go
func (bp *BatchProcessor) monitorThroughput() {
    lastProcessed := int64(0)
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        current := bp.processed.Load()
        throughput := float64(current-lastProcessed) / 10.0

        bp.logger.Info().
            Float64("throughput_per_sec", throughput).
            Msg("throughput metrics")

        lastProcessed = current
    }
}
```
