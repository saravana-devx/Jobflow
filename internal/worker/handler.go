package worker

import (
	"encoding/json"
	"fmt"
	"log"

	"pulseDashboard/internal/jobs"
)

// Job is the message envelope received from the queue.
type Job = jobs.JobMessage

// handleJob unmarshals the raw message and routes it to the correct handler.
func handleJob(body []byte) error {
	var job Job
	if err := job.Unmarshal(body); err != nil {
		return fmt.Errorf("unmarshal failed: %w", err)
	}

	log.Printf("received job id=%s type=%s", job.ID, job.Type)

	switch job.Type {
	case jobs.JobEmail:
		return handleSendEmail(&job)
	case jobs.JobGenerateReport:
		return handleReportGeneration(&job)
	default:
		log.Printf("unknown job type: %s", job.Type)
	}

	return nil
}

// --- individual handlers ---

type sendEmailPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

func handleSendEmail(job *Job) error {
	var p sendEmailPayload
	if err := json.Unmarshal(job.Payload, &p); err != nil {
		return fmt.Errorf("sendEmail job=%s: invalid payload: %w", job.ID, err)
	}

	log.Printf("sendEmail job=%s to=%s subject=%q", job.ID, p.To, p.Subject)
	// TODO: call email service
	return nil
}

type reportGenerationPayload struct {
	ReportType string         `json:"reportType"`
	Format     string         `json:"format"` // "pdf" | "csv" | "xlsx"
	Filters    map[string]any `json:"filters,omitempty"`
}

func handleReportGeneration(job *Job) error {
	var p reportGenerationPayload
	if err := json.Unmarshal(job.Payload, &p); err != nil {
		return fmt.Errorf("reportGeneration job=%s: invalid payload: %w", job.ID, err)
	}

	log.Printf("reportGeneration job=%s type=%s format=%s user=%s", job.ID, p.ReportType, p.Format, job.UserID)
	// TODO: generate report and upload to storage
	return nil
}
