package utils

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	// ErrWorkerPoolFull is returned when the worker pool queue is full
	ErrWorkerPoolFull = errors.New("worker pool queue is full")
)

// Job represents a job to be processed
type Job struct {
	ID       string
	Data     interface{}
	Priority int // Higher number = higher priority
}

// JobResult represents the result of a job
type JobResult struct {
	JobID  string
	Result interface{}
	Error  error
}

// JobProcessor defines the interface for processing jobs
type JobProcessor interface {
	Process(ctx context.Context, job *Job) (interface{}, error)
}

// WorkerPool manages a pool of workers for concurrent job processing
type WorkerPool struct {
	workers    int
	jobQueue   chan *Job
	resultChan chan *JobResult
	processor  JobProcessor
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	metrics    *WorkerPoolMetrics
}

// WorkerPoolMetrics holds metrics for the worker pool
type WorkerPoolMetrics struct {
	JobsProcessed  int64
	JobsFailed     int64
	JobsInQueue    int64
	ActiveWorkers  int64
	ProcessingTime time.Duration
	mu             sync.RWMutex
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(workers int, queueSize int, processor JobProcessor) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		workers:    workers,
		jobQueue:   make(chan *Job, queueSize),
		resultChan: make(chan *JobResult, queueSize),
		processor:  processor,
		ctx:        ctx,
		cancel:     cancel,
		metrics:    &WorkerPoolMetrics{},
	}
}

// Start starts the worker pool
func (wp *WorkerPool) Start() {
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

// Stop stops the worker pool gracefully
func (wp *WorkerPool) Stop() {
	wp.cancel()
	close(wp.jobQueue)
	wp.wg.Wait()
	close(wp.resultChan)
}

// Submit submits a job to the worker pool
func (wp *WorkerPool) Submit(job *Job) error {
	select {
	case wp.jobQueue <- job:
		wp.metrics.mu.Lock()
		wp.metrics.JobsInQueue++
		wp.metrics.mu.Unlock()
		return nil
	case <-wp.ctx.Done():
		return wp.ctx.Err()
	default:
		return ErrWorkerPoolFull
	}
}

// SubmitBatch submits multiple jobs to the worker pool
func (wp *WorkerPool) SubmitBatch(jobs []*Job) error {
	for _, job := range jobs {
		if err := wp.Submit(job); err != nil {
			return err
		}
	}
	return nil
}

// Results returns the channel for job results
func (wp *WorkerPool) Results() <-chan *JobResult {
	return wp.resultChan
}

// worker is the main worker function
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	wp.metrics.mu.Lock()
	wp.metrics.ActiveWorkers++
	wp.metrics.mu.Unlock()

	defer func() {
		wp.metrics.mu.Lock()
		wp.metrics.ActiveWorkers--
		wp.metrics.mu.Unlock()
	}()

	for {
		select {
		case job, ok := <-wp.jobQueue:
			if !ok {
				return
			}

			wp.metrics.mu.Lock()
			wp.metrics.JobsInQueue--
			wp.metrics.mu.Unlock()

			start := time.Now()
			result, err := wp.processor.Process(wp.ctx, job)
			duration := time.Since(start)

			wp.metrics.mu.Lock()
			wp.metrics.JobsProcessed++
			wp.metrics.ProcessingTime += duration
			if err != nil {
				wp.metrics.JobsFailed++
			}
			wp.metrics.mu.Unlock()

			select {
			case wp.resultChan <- &JobResult{
				JobID:  job.ID,
				Result: result,
				Error:  err,
			}:
			case <-wp.ctx.Done():
				return
			}

		case <-wp.ctx.Done():
			return
		}
	}
}

// GetMetrics returns current worker pool metrics
func (wp *WorkerPool) GetMetrics() *WorkerPoolMetrics {
	wp.metrics.mu.RLock()
	defer wp.metrics.mu.RUnlock()

	return &WorkerPoolMetrics{
		JobsProcessed:  wp.metrics.JobsProcessed,
		JobsFailed:     wp.metrics.JobsFailed,
		JobsInQueue:    wp.metrics.JobsInQueue,
		ActiveWorkers:  wp.metrics.ActiveWorkers,
		ProcessingTime: wp.metrics.ProcessingTime,
	}
}

// GetQueueSize returns the current queue size
func (wp *WorkerPool) GetQueueSize() int {
	return len(wp.jobQueue)
}

// GetActiveWorkers returns the number of active workers
func (wp *WorkerPool) GetActiveWorkers() int64 {
	wp.metrics.mu.RLock()
	defer wp.metrics.mu.RUnlock()
	return wp.metrics.ActiveWorkers
}

// ProcessJobsWithTimeout processes jobs with a timeout
func (wp *WorkerPool) ProcessJobsWithTimeout(jobs []*Job, timeout time.Duration) ([]*JobResult, error) {
	ctx, cancel := context.WithTimeout(wp.ctx, timeout)
	defer cancel()

	// Submit all jobs
	if err := wp.SubmitBatch(jobs); err != nil {
		return nil, err
	}

	var results []*JobResult
	expectedResults := len(jobs)

	for i := 0; i < expectedResults; i++ {
		select {
		case result := <-wp.Results():
			results = append(results, result)
		case <-ctx.Done():
			return results, ctx.Err()
		}
	}

	return results, nil
}

// PriorityJobProcessor implements priority-based job processing
type PriorityJobProcessor struct {
	processor JobProcessor
}

// NewPriorityJobProcessor creates a new priority job processor
func NewPriorityJobProcessor(processor JobProcessor) *PriorityJobProcessor {
	return &PriorityJobProcessor{
		processor: processor,
	}
}

// Process processes a job with priority handling
func (pjp *PriorityJobProcessor) Process(ctx context.Context, job *Job) (interface{}, error) {
	// Add priority-based delay for lower priority jobs
	if job.Priority < 5 {
		delay := time.Duration(5-job.Priority) * time.Millisecond
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return pjp.processor.Process(ctx, job)
}

// BatchProcessor implements batch job processing
type BatchProcessor struct {
	processor JobProcessor
	batchSize int
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(processor JobProcessor, batchSize int) *BatchProcessor {
	return &BatchProcessor{
		processor: processor,
		batchSize: batchSize,
	}
}

// ProcessBatch processes a batch of jobs
func (bp *BatchProcessor) ProcessBatch(ctx context.Context, jobs []*Job) ([]*JobResult, error) {
	var results []*JobResult

	for i := 0; i < len(jobs); i += bp.batchSize {
		end := i + bp.batchSize
		if end > len(jobs) {
			end = len(jobs)
		}

		batch := jobs[i:end]
		for _, job := range batch {
			result, err := bp.processor.Process(ctx, job)
			results = append(results, &JobResult{
				JobID:  job.ID,
				Result: result,
				Error:  err,
			})

			if err != nil {
				// Continue processing other jobs in batch
				continue
			}
		}
	}

	return results, nil
}
