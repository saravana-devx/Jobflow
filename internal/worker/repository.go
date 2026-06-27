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

// ClaimJob marks a job 'running' for this worker, but only if it's actually
// claimable: still 'pending', or 'running' but stale (started before staleBefore,
// so its old worker is presumably dead). the conditional WHERE means only one
// worker can win the update, and an already-done job matches nothing.
// returns false when nothing was claimed — caller should just ack and skip.
func (r *WorkerRepository) ClaimJob(ctx context.Context, jobID, workerID string, staleBefore time.Time) (bool, error) {
	res := r.db.WithContext(ctx).
		Model(&jobs.Job{}).
		Where("id = ? AND (status = ? OR (status = ? AND started_at < ?))",
			jobID, jobs.JobStatusPending, jobs.JobStatusRunning, staleBefore).
		Updates(map[string]any{
			"status":     jobs.JobStatusRunning,
			"started_at": time.Now(),
			"worker_id":  workerID,
			"attempts":   gorm.Expr("attempts + 1"),
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
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
