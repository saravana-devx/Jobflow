package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
	"jobflow/internal/rabbitmq"
)

type Service struct {
	repo *JobsRepository
	mq   *rabbitmq.RabbitMQ
}

func NewService(repo *JobsRepository, mq *rabbitmq.RabbitMQ) *Service {
	return &Service{repo: repo, mq: mq}
}

func (s *Service) publishJob(ctx context.Context, job *Job) error {
	msg := JobMessage{
		ID:          job.ID,
		UserID:      job.UserID,
		Type:        job.Type,
		Payload:     job.Payload,
		Priority:    job.Priority,
		MaxRetries:  job.MaxRetries,
		ScheduledAt: job.ScheduledAt,
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal job for publish: %w", err)
	}
	delay := time.Until(job.ScheduledAt)
	if err := s.mq.PublishDelayed(ctx, rabbitmq.QueueJobs, payload, delay); err != nil {
		return err
	}
	log.Printf("published job: id=%s delay=%s", job.ID, delay.Round(time.Second))
	return nil
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

	if err := s.publishJob(ctx, createdJob); err != nil {
		return nil, fmt.Errorf("publish job: %w", err)
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
		if err := s.publishJob(ctx, j); err != nil {
			return nil, fmt.Errorf("publish job %s: %w", j.ID, err)
		}
		result = append(result, CreateJobResult{Job: j})
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
	result, err := s.repo.GetAllJobs(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJobNotFound
		}
		return nil, fmt.Errorf("%w: %v", ErrToGetAllJobs, err)
	}

	return &GetAllJobsResult{Jobs: result}, nil
}

func (s *Service) UpdateJobService(ctx context.Context, id string, userID string, req *UpdateJobRequest) (*Job, error) {

	job, err := s.repo.GetJobByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJobNotFound
		}
		return nil, fmt.Errorf("%w: %v", ErrToGetJob, err)
	}

	if job.UserID != userID {
		return nil, ErrUnauthorized
	}

	if req.MaxRetries != nil && *req.MaxRetries > 10 {
		return nil, fmt.Errorf("%w: max retries must be 10 or less", ErrInvalidJobInput)
	}

	// Merge: only fields present in the request overwrite the loaded job.
	// UserID and all unmanaged fields (Status, Attempts, timestamps, ...)
	// stay as loaded, so a full-row Save in the repo can't wipe them.
	if req.Type != "" {
		job.Type = req.Type
	}
	if req.Payload != nil {
		job.Payload = req.Payload
	}
	if req.MaxRetries != nil {
		job.MaxRetries = *req.MaxRetries
	}
	if req.ScheduledAt != nil {
		job.ScheduledAt = *req.ScheduledAt
	}
	if req.Priority != nil {
		job.Priority = *req.Priority
	}

	result, err := s.repo.UpdateJob(ctx, job.ID, job)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrToUpdateJob, err)
	}

	return result, nil

}

func (s *Service) DeleteJobService(ctx context.Context, id string) error {
	err := s.repo.DeleteJob(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrJobNotFound
		}
		return fmt.Errorf("%w: %v", ErrToDeleteJob, err)
	}
	return nil
}
