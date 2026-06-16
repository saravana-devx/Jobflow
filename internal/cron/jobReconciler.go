package cron

import (
	"context"
	"log"
	"time"

	"jobflow/internal/jobs"
)

const jobReconcileInterval = 1 * time.Minute

// JobReconciler periodically republishes jobs whose DB row was committed
// but whose queue publish never succeeded, closing the dual-write gap
// between job creation and queue publish.
type JobReconciler struct {
	jobsService *jobs.Service
}

func NewJobReconciler(jobsService *jobs.Service) *JobReconciler {
	return &JobReconciler{jobsService: jobsService}
}

func (c *JobReconciler) Start(ctx context.Context) {
	ticker := time.NewTicker(jobReconcileInterval)
	defer ticker.Stop()
	log.Println("JobReconciler started")
	for {
		select {
		case <-ticker.C:
			if err := c.jobsService.RepublishStuckJobs(ctx); err != nil {
				log.Printf("JobReconciler: sweep failed: %v", err)
			}
		case <-ctx.Done():
			log.Println("JobReconciler stopped")
			return
		}
	}
}
