// Batch Processor Example
//
// This example demonstrates using Bolt in a high-throughput batch processing system with:
// - Worker pool pattern for concurrent processing
// - Progress tracking and metrics
// - Error handling and retry logic
// - Batch completion notifications
// - Resource cleanup and graceful shutdown
package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/felixgeelhaar/bolt/v3"
)

// BatchProcessor handles concurrent batch processing
type BatchProcessor struct {
	logger     *bolt.Logger
	workers    int
	batchSize  int
	totalItems atomic.Int64
	processed  atomic.Int64
	failed     atomic.Int64
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

// Item represents a work item to process
type Item struct {
	ID        string
	Data      string
	Timestamp time.Time
}

// ProcessResult represents the result of processing an item
type ProcessResult struct {
	Item     Item
	Success  bool
	Error    error
	Duration time.Duration
	Retries  int
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(logger *bolt.Logger, workers, batchSize int) *BatchProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	return &BatchProcessor{
		logger:    logger,
		workers:   workers,
		batchSize: batchSize,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start begins processing items from the input channel
func (bp *BatchProcessor) Start(items <-chan Item) <-chan ProcessResult {
	results := make(chan ProcessResult, bp.workers*2)

	// Start worker pool
	for i := 0; i < bp.workers; i++ {
		bp.wg.Add(1)
		go bp.worker(i, items, results)
	}

	// Start metrics reporter
	go bp.reportMetrics()

	// Close results channel when all workers finish
	go func() {
		bp.wg.Wait()
		close(results)
		bp.logger.Info().Msg("all workers finished")
	}()

	return results
}

// worker processes items from the channel
func (bp *BatchProcessor) worker(id int, items <-chan Item, results chan<- ProcessResult) {
	defer bp.wg.Done()

	bp.logger.Info().
		Int("worker_id", id).
		Msg("worker started")

	processed := 0
	for {
		select {
		case <-bp.ctx.Done():
			bp.logger.Info().
				Int("worker_id", id).
				Int("items_processed", processed).
				Msg("worker stopping")
			return

		case item, ok := <-items:
			if !ok {
				bp.logger.Info().
					Int("worker_id", id).
					Int("items_processed", processed).
					Msg("worker completed")
				return
			}

			result := bp.processItem(id, item)
			processed++

			if result.Success {
				bp.processed.Add(1)
			} else {
				bp.failed.Add(1)
			}

			results <- result
		}
	}
}

// processItem processes a single item with retry logic
func (bp *BatchProcessor) processItem(workerID int, item Item) ProcessResult {
	start := time.Now()
	maxRetries := 3
	retries := 0

	for retries < maxRetries {
		// Simulate processing
		err := bp.doProcessing(item)

		if err == nil {
			duration := time.Since(start)
			bp.logger.Info().
				Int("worker_id", workerID).
				Str("item_id", item.ID).
				Dur("duration", duration).
				Int("retries", retries).
				Msg("item processed successfully")

			return ProcessResult{
				Item:     item,
				Success:  true,
				Duration: duration,
				Retries:  retries,
			}
		}

		retries++
		if retries < maxRetries {
			bp.logger.Warn().
				Int("worker_id", workerID).
				Str("item_id", item.ID).
				Str("error", err.Error()).
				Int("retry", retries).
				Msg("retrying item")

			// Exponential backoff
			time.Sleep(time.Duration(retries*100) * time.Millisecond)
		}
	}

	// Final failure
	duration := time.Since(start)
	bp.logger.Error().
		Int("worker_id", workerID).
		Str("item_id", item.ID).
		Dur("duration", duration).
		Int("retries", retries).
		Msg("item processing failed after retries")

	return ProcessResult{
		Item:     item,
		Success:  false,
		Error:    fmt.Errorf("max retries exceeded"),
		Duration: duration,
		Retries:  retries,
	}
}

// doProcessing simulates actual processing work
func (bp *BatchProcessor) doProcessing(item Item) error {
	// Simulate variable processing time
	processingTime := time.Duration(50+rand.Intn(100)) * time.Millisecond
	time.Sleep(processingTime)

	// Simulate random failures (10% failure rate)
	if rand.Float32() < 0.1 {
		return fmt.Errorf("processing error")
	}

	return nil
}

// reportMetrics logs processing metrics periodically
func (bp *BatchProcessor) reportMetrics() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-bp.ctx.Done():
			return
		case <-ticker.C:
			total := bp.totalItems.Load()
			processed := bp.processed.Load()
			failed := bp.failed.Load()
			remaining := total - processed - failed

			var progress float64
			if total > 0 {
				progress = float64(processed+failed) / float64(total) * 100
			}

			bp.logger.Info().
				Int64("total", total).
				Int64("processed", processed).
				Int64("failed", failed).
				Int64("remaining", remaining).
				Float64("progress_pct", progress).
				Msg("processing metrics")
		}
	}
}

// Stop gracefully stops the batch processor
func (bp *BatchProcessor) Stop() {
	bp.logger.Info().Msg("stopping batch processor")
	bp.cancel()
}

// generateItems creates a stream of items to process
func generateItems(count int, logger *bolt.Logger) <-chan Item {
	items := make(chan Item, 100)

	go func() {
		defer close(items)

		logger.Info().
			Int("count", count).
			Msg("generating items")

		for i := 0; i < count; i++ {
			items <- Item{
				ID:        fmt.Sprintf("item_%d", i),
				Data:      fmt.Sprintf("data_%d", i),
				Timestamp: time.Now(),
			}
		}

		logger.Info().
			Int("count", count).
			Msg("item generation complete")
	}()

	return items
}

func main() {
	// Initialize logger
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().
		Str("service", "batch-processor").
		Str("version", "1.0.0").
		Msg("starting batch processor")

	// Configuration
	const (
		numWorkers = 10
		batchSize  = 100
		numItems   = 1000
	)

	// Create processor
	processor := NewBatchProcessor(logger, numWorkers, batchSize)
	processor.totalItems.Store(int64(numItems))

	logger.Info().
		Int("workers", numWorkers).
		Int("batch_size", batchSize).
		Int("total_items", numItems).
		Msg("processor configured")

	// Generate items
	items := generateItems(numItems, logger)

	// Start processing
	startTime := time.Now()
	results := processor.Start(items)

	// Collect results
	go func() {
		successCount := 0
		failureCount := 0

		for result := range results {
			if result.Success {
				successCount++
			} else {
				failureCount++
			}
		}

		duration := time.Since(startTime)
		throughput := float64(numItems) / duration.Seconds()

		logger.Info().
			Int("total_items", numItems).
			Int("successful", successCount).
			Int("failed", failureCount).
			Dur("total_duration", duration).
			Float64("throughput_per_sec", throughput).
			Msg("batch processing completed")
	}()

	// Wait for interrupt or completion
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		logger.Info().Msg("received shutdown signal")
		processor.Stop()
	case <-time.After(2 * time.Minute):
		// Timeout for example
		logger.Warn().Msg("processing timeout reached")
		processor.Stop()
	}

	// Wait a bit for cleanup
	time.Sleep(2 * time.Second)

	// Final statistics
	logger.Info().
		Int64("final_processed", processor.processed.Load()).
		Int64("final_failed", processor.failed.Load()).
		Msg("batch processor stopped")
}
