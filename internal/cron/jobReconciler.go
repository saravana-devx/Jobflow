package cron

import (
	"context"
	"log"
	"time"

	"jobflow/internal/jobs"
	redisx "jobflow/internal/redis"
)

const jobReconcileInterval = 1 * time.Minute

// keep the lease just under the interval so it expires before the next tick
const jobReconcileLeaseTTL = 55 * time.Second

// JobReconciler runs two sweeps every tick:
//   - republish jobs that were saved but never made it onto the queue
//   - reap jobs stuck in 'running' because their worker died
// only the leader replica runs the sweep (redis lease).
type JobReconciler struct {
	jobsService *jobs.Service
	rdb         *redisx.Redis
}

func NewJobReconciler(jobsService *jobs.Service, rdb *redisx.Redis) *JobReconciler {
	return &JobReconciler{jobsService: jobsService, rdb: rdb}
}

func (c *JobReconciler) Start(ctx context.Context) {
	ticker := time.NewTicker(jobReconcileInterval)
	defer ticker.Stop()
	log.Println("JobReconciler started")
	for {
		select {
		case <-ticker.C:
			runIfLeader(ctx, c.rdb, "cron:job-reconciler", jobReconcileLeaseTTL, func() {
				c.sweep(ctx)
			})
		case <-ctx.Done():
			log.Println("JobReconciler stopped")
			return
		}
	}
}

func (c *JobReconciler) sweep(ctx context.Context) {
	if err := c.jobsService.RepublishStuckJobs(ctx); err != nil {
		log.Printf("JobReconciler: republish sweep failed: %v", err)
	}
	if err := c.jobsService.ReapStuckJobs(ctx); err != nil {
		log.Printf("JobReconciler: reap sweep failed: %v", err)
	}
}
