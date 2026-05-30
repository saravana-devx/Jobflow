package jobs

import (
	"time"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type JobStatus string

type JobType string

const (
	JobEmail          JobType = "sendEmail"
	JobGenerateReport JobType = "generateReport"
	JobResizeImage    JobType = "resizeImage"
	JobExportCSV      JobType = "exportCSV"
)

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusError     JobStatus = "error"
)

type Job struct {
	ID          string         `gorm:"type:uuid;primaryKey;not null;default:uuid_generate_v7()"`
	UserID      string         `gorm:"type:uuid;not null;index"`
	Type        JobType        `gorm:"not null"`
	Payload     datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	Status      JobStatus      `gorm:"type:job_status;not null;default:'pending'"`
	Priority    int            `gorm:"not null;default:5"`
	WorkerID    *string
	Attempts    int            `gorm:"not null;default:0"`
	MaxRetries  int            `gorm:"not null;default:3"`
	ErrorMsg    *string
	ScheduledAt time.Time      `gorm:"not null;default:now()"`
	StartedAt   *time.Time
	CompletedAt *time.Time
	CreatedAt   time.Time      `gorm:"not null;default:now()"`
	UpdatedAt   time.Time      `gorm:"not null;default:now()"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (Job) TableName() string {
	return "jobs"
}
