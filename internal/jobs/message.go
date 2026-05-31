package jobs

import (
	"encoding/json"
	"time"

	"gorm.io/datatypes"
)

// JobMessage is the envelope published to the queue and consumed by the worker.
type JobMessage struct {
	ID          string         `json:"id"`
	UserID      string         `json:"userId"`
	Type        JobType        `json:"type"`
	Payload     datatypes.JSON `json:"payload"`
	Priority    int            `json:"priority"`
	MaxRetries  int            `json:"maxRetries"`
	ScheduledAt time.Time      `json:"scheduledAt"`
}

func (j *JobMessage) Unmarshal(data []byte) error {
	return json.Unmarshal(data, j)
}
