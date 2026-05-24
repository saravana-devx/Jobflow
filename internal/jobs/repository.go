package jobs

import (
	"context"

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
	err := r.db.WithContext(ctx).Model(&Job{}).Where("id = ?", id).First(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *JobsRepository) GetAllJobs(ctx context.Context, userID string) ([]*Job, error) {
	return nil, nil
}

func (r *JobsRepository) UpdateJob(ctx context.Context, job *Job) (*Job, error) {
	return nil, nil
}

func (r *JobsRepository) DeleteJob(ctx context.Context, id string) error {
	return nil
}
