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

	queuedAt := time.Now()
	if err := s.repo.MarkQueued(ctx, job.ID, queuedAt); err != nil {
		log.Printf("job=%s: published but failed to record queued_at: %v", job.ID, err)
	} else {
		job.QueuedAt = &queuedAt
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
		log.Printf("job=%s: publish failed, will be retried by reconciler: %v", createdJob.ID, err)
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
			log.Printf("job=%s: publish failed, will be retried by reconciler: %v", j.ID, err)
		}
		result = append(result, CreateJobResult{Job: j})
	}

	return &result, nil
}

const queuePublishGrace = 2 * time.Minute

func (s *Service) RepublishStuckJobs(ctx context.Context) error {
	cutoff := time.Now().Add(-queuePublishGrace)
	stuck, err := s.repo.GetUnqueuedPendingJobs(ctx, cutoff)
	if err != nil {
		return fmt.Errorf("load unqueued jobs: %w", err)
	}

	for _, job := range stuck {
		if err := s.publishJob(ctx, job); err != nil {
			log.Printf("job=%s: reconciler republish failed: %v", job.ID, err)
			continue
		}
		log.Printf("job=%s: republished by reconciler", job.ID)
	}

	return nil
}

// how long a job can sit in 'running' before we assume its worker died. must be
// longer than any real job — it's a fixed window, not a heartbeat, so a job that
// genuinely runs longer would get re-published as a duplicate.
const stuckRunningGrace = 5 * time.Minute

// ReapStuckJobs recovers jobs whose worker died mid-run (stuck in 'running').
// each one is re-published to try again, or marked failed once it's out of
// retries. re-publishing is safe because ClaimJob skips jobs that are already
// done, so a duplicate won't run twice.
func (s *Service) ReapStuckJobs(ctx context.Context) error {
	cutoff := time.Now().Add(-stuckRunningGrace)
	stuck, err := s.repo.GetStuckRunningJobs(ctx, cutoff)
	if err != nil {
		return fmt.Errorf("load stuck running jobs: %w", err)
	}

	for _, job := range stuck {
		if job.Attempts >= job.MaxRetries {
			if err := s.repo.MarkFailed(ctx, job.ID, "worker stalled; retries exhausted"); err != nil {
				log.Printf("job=%s: reaper mark-failed: %v", job.ID, err)
				continue
			}
			log.Printf("job=%s: reaper marked failed after stall (attempts=%d/%d)", job.ID, job.Attempts, job.MaxRetries)
			continue
		}
		if err := s.publishJob(ctx, job); err != nil {
			log.Printf("job=%s: reaper republish failed: %v", job.ID, err)
			continue
		}
		log.Printf("job=%s: reaper re-published stalled job (attempts=%d/%d)", job.ID, job.Attempts, job.MaxRetries)
	}

	return nil
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

func (s *Service) DeleteJobService(ctx context.Context, id string, userID string) error {
	job, err := s.repo.GetJobByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrJobNotFound
		}
		return fmt.Errorf("%w: %v", ErrToGetJob, err)
	}

	if job.UserID != userID {
		return ErrUnauthorized
	}

	if err := s.repo.DeleteJob(ctx, id); err != nil {
		return fmt.Errorf("%w: %v", ErrToDeleteJob, err)
	}
	return nil
}
