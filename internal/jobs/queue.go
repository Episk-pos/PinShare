package jobs

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// Job represents a generic job interface
type Job interface {
	ID() string
	Execute(ctx context.Context) error
}

// Queue manages a queue of jobs with worker pool
type Queue struct {
	ctx           context.Context
	cancel        context.CancelFunc
	jobs          chan Job
	workers       int
	wg            sync.WaitGroup
	mu            sync.RWMutex
	activeJobs    map[string]Job
	errorHandler  func(job Job, err error)
}

// NewQueue creates a new job queue with the specified number of workers
func NewQueue(ctx context.Context, workers int) *Queue {
	ctx, cancel := context.WithCancel(ctx)

	q := &Queue{
		ctx:        ctx,
		cancel:     cancel,
		jobs:       make(chan Job, 100), // Buffer for 100 jobs
		workers:    workers,
		activeJobs: make(map[string]Job),
		errorHandler: func(job Job, err error) {
			log.Printf("[ERROR] Job %s failed: %v", job.ID(), err)
		},
	}

	return q
}

// Start starts the worker pool
func (q *Queue) Start() {
	for i := 0; i < q.workers; i++ {
		q.wg.Add(1)
		go q.worker(i)
	}
	log.Printf("[INFO] Started job queue with %d workers", q.workers)
}

// worker processes jobs from the queue
func (q *Queue) worker(id int) {
	defer q.wg.Done()

	for {
		select {
		case <-q.ctx.Done():
			log.Printf("[INFO] Worker %d shutting down", id)
			return

		case job, ok := <-q.jobs:
			if !ok {
				log.Printf("[INFO] Worker %d: job channel closed", id)
				return
			}

			log.Printf("[INFO] Worker %d processing job %s", id, job.ID())

			// Track active job
			q.mu.Lock()
			q.activeJobs[job.ID()] = job
			q.mu.Unlock()

			// Execute job
			if err := job.Execute(q.ctx); err != nil {
				if q.errorHandler != nil {
					q.errorHandler(job, err)
				}
			} else {
				log.Printf("[INFO] Worker %d completed job %s", id, job.ID())
			}

			// Remove from active jobs
			q.mu.Lock()
			delete(q.activeJobs, job.ID())
			q.mu.Unlock()
		}
	}
}

// Enqueue adds a job to the queue
func (q *Queue) Enqueue(job Job) error {
	select {
	case <-q.ctx.Done():
		return fmt.Errorf("queue is shut down")
	case q.jobs <- job:
		log.Printf("[INFO] Enqueued job %s", job.ID())
		return nil
	}
}

// Stop stops the queue and waits for all workers to finish
func (q *Queue) Stop() {
	log.Println("[INFO] Stopping job queue...")
	q.cancel()
	close(q.jobs)
	q.wg.Wait()
	log.Println("[INFO] Job queue stopped")
}

// IsActive checks if a job is currently being processed
func (q *Queue) IsActive(jobID string) bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	_, exists := q.activeJobs[jobID]
	return exists
}

// GetActiveJobs returns a list of currently active job IDs
func (q *Queue) GetActiveJobs() []string {
	q.mu.RLock()
	defer q.mu.RUnlock()

	ids := make([]string, 0, len(q.activeJobs))
	for id := range q.activeJobs {
		ids = append(ids, id)
	}
	return ids
}

// SetErrorHandler sets a custom error handler for failed jobs
func (q *Queue) SetErrorHandler(handler func(job Job, err error)) {
	q.errorHandler = handler
}
