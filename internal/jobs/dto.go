package jobs

import (
	"time"

	"gorm.io/datatypes"
)

type CreateJobRequest struct {
	UserID      string         `json:"userId"`
	Type        JobType        `json:"type"`
	Payload     datatypes.JSON `json:"payload"`
	Priority    *int           `json:"priority,omitempty"`
	MaxRetries  *int           `json:"maxRetries,omitempty"`
	ScheduledAt *time.Time     `json:"scheduledAt,omitempty"`
}

type CreateJobResult struct {
	Job *Job
}

type GetJobByIdResult struct {
	Job *Job
}

type GetAllJobsResult struct {
	Jobs []*Job
}

type UpdateJobRequest struct {
	UserID      string         `json:"userId"`
	Type        JobType        `json:"type"`
	Payload     datatypes.JSON `json:"payload"`
	Priority    *int           `json:"priority,omitempty"`
	MaxRetries  *int           `json:"maxRetries,omitempty"`
	ScheduledAt *time.Time     `json:"scheduledAt,omitempty"`
}

type UpdateJobResult struct {
	Job *Job
}