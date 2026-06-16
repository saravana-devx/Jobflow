package jobs

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type JobsRepository struct {
	db *gorm.DB
}

func NewJobRepository(db *gorm.DB) *JobsRepository {
	return &JobsRepository{db: db}
}

func (r *JobsRepository) CreateJob(ctx context.Context, job *Job) (*Job, error) {
	err := r.db.WithContext(ctx).Create(job).Error
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (r *JobsRepository) CreateJobs(ctx context.Context, job []*Job) ([]*Job, error) {
	err := r.db.WithContext(ctx).Create(job).Error
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (r *JobsRepository) GetJobByID(ctx context.Context, id string) (*Job, error) {
	var job Job
	err := r.db.WithContext(ctx).Model(&Job{}).Where("id = ?", id).Take(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *JobsRepository) GetAllJobs(ctx context.Context, userID string) ([]*Job, error) {
	var jobs []*Job
	err := r.db.WithContext(ctx).Model(&Job{}).Where("user_id = ? ", userID).Find(&jobs).Error
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

func (r *JobsRepository) UpdateJob(ctx context.Context, id string, job *Job) (*Job, error) {
	result := r.db.WithContext(ctx).Model(&Job{}).Where("id = ?", id).Updates(&job)
	if result.Error != nil {
		return nil, result.Error
	}
	return job, nil
}

func (r *JobsRepository) DeleteJob(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&Job{}).Error
}

func (r *JobsRepository) GetUnqueuedPendingJobs(ctx context.Context, before time.Time) ([]*Job, error) {
	var jobs []*Job
	err := r.db.WithContext(ctx).
		Where("status = ? AND queued_at IS NULL AND scheduled_at <= ?", JobStatusPending, before).
		Find(&jobs).Error
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

func (r *JobsRepository) MarkQueued(ctx context.Context, id string, queuedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&Job{}).Where("id = ?", id).Update("queued_at", queuedAt).Error
}
