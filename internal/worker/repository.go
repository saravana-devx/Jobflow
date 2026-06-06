package worker

import (
	"context"
	"time"

	"gorm.io/gorm"

	"jobflow/internal/jobs"
)

type WorkerRepository struct {
	db *gorm.DB
}

func NewWorkerRepository(db *gorm.DB) *WorkerRepository {
	return &WorkerRepository{db: db}
}

func (r *WorkerRepository) MarkRunning(ctx context.Context, jobID string, workerID string) error {
	return r.db.WithContext(ctx).
		Model(&jobs.Job{}).
		Where("id = ?", jobID).
		Updates(map[string]any{
			"status":     jobs.JobStatusRunning,
			"started_at": time.Now(),
			"worker_id":  workerID,
			"attempts":   gorm.Expr("attempts + 1"),
		}).Error
}

func (r *WorkerRepository) MarkCompleted(ctx context.Context, jobID string) error {
	return r.db.WithContext(ctx).
		Model(&jobs.Job{}).
		Where("id = ?", jobID).
		Updates(map[string]any{
			"status":       jobs.JobStatusCompleted,
			"completed_at": time.Now(),
		}).Error
}

func (r *WorkerRepository) MarkFailed(ctx context.Context, jobID string, errMsg string) error {
	return r.db.WithContext(ctx).
		Model(&jobs.Job{}).
		Where("id = ?", jobID).
		Updates(map[string]any{
			"status":       jobs.JobStatusFailed,
			"error_msg":    errMsg,
			"completed_at": time.Now(),
		}).Error
}
