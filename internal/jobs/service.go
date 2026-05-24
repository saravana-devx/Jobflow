package jobs

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

type Service struct {
	repo *JobsRepository
}

func NewService(repo *JobsRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateJobService(ctx context.Context, req *CreateJobRequest) (*CreateJobResult, error) {

	if req.MaxRetries != nil && *req.MaxRetries > 10 {
		return nil, fmt.Errorf("%w: max retries must be 10 or less", ErrInvalidJobInput)
	}

	job := &Job{
		UserID:  req.UserID,
		Type:    req.Type,
		Payload: req.Payload,
	}

	// safe assign optional fields
	if req.MaxRetries != nil {
		job.MaxRetries = *req.MaxRetries
	}

	if req.ScheduledAt != nil {
		job.ScheduledAt = *req.ScheduledAt
	}

	if req.Priority != nil {
		job.Priority = *req.Priority
	}

	createdJob, err := s.repo.CreateJob(ctx, job)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrToCreateJob, err)
	}

	return &CreateJobResult{Job: createdJob}, nil
}

func (s *Service) CreateJobsService(ctx context.Context, req *[]CreateJobRequest, userID string) (*[]CreateJobResult, error) {
	for _, r := range *req {
		if r.MaxRetries != nil && *r.MaxRetries > 10 {
			return nil, fmt.Errorf("%w: max retries must be 10 or less", ErrInvalidJobInput)
		}
	}

	jobs := make([]*Job, 0, len(*req))
	for _, r := range *req {
		job := &Job{
			UserID:  userID,
			Type:    r.Type,
			Payload: r.Payload,
		}

		if r.MaxRetries != nil {
			job.MaxRetries = *r.MaxRetries
		}
		if r.ScheduledAt != nil {
			job.ScheduledAt = *r.ScheduledAt
		}

		if r.Priority != nil {
			job.Priority = *r.Priority
		}

		jobs = append(jobs, job)
	}

	createdJobs, err := s.repo.CreateJobs(ctx, jobs)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrToCreateJobs, err)
	}

	result := make([]CreateJobResult, 0, len(createdJobs))
	for _, j := range createdJobs {
		result = append(result, CreateJobResult{
			Job: j,
		})
	}

	return &result, nil
}

func (s *Service) GetJobByIdService(ctx context.Context, id string) (*GetJobByIdResult, error) {
	result, err := s.repo.GetJobByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJobNotFound
		}
		return nil, fmt.Errorf("%w: %v", ErrToGetJob, err)
	}
	return &GetJobByIdResult{Job: result}, nil
}

func (s *Service) GetAllJobsService(ctx context.Context, userID string) (*GetAllJobsResult, error) {
	return nil, nil
}

func (s *Service) UpdateJobService(ctx context.Context, id string, req *UpdateJobRequest) (*Job, error) {
	return nil, nil
}

func (s *Service) DeleteJobService(ctx context.Context, id string) error {
	return nil
}
