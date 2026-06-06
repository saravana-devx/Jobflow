package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"jobflow/internal/jobs"
)

type Job = jobs.JobMessage

type jobStatusEvent struct {
	JobID  string `json:"jobID"`
	Status string `json:"status"`
	UserID string `json:"userID"`
}

func (w *Worker) handleJob(body []byte) error {
	var job Job
	if err := job.Unmarshal(body); err != nil {
		return fmt.Errorf("unmarshal failed: %w", err)
	}

	log.Printf("received job id=%s type=%s", job.ID, job.Type)

	switch job.Type {
	case jobs.JobEmail:
		return w.handleSendEmail(&job)
	case jobs.JobGenerateReport:
		return w.handleReportGeneration(&job)
	case jobs.JobResizeImage:
		return w.handleResizeImage(&job)
	case jobs.JobExportCSV:
		return w.handleExportCSV(&job)
	default:
		log.Printf("unknown job type: %s", job.Type)
	}

	return nil
}

func (w *Worker) publishStatusEvent(ctx context.Context, job *Job, status jobs.JobStatus) {
	event, err := json.Marshal(jobStatusEvent{JobID: job.ID, Status: string(status), UserID: job.UserID})
	if err != nil {
		log.Printf("job=%s: failed to marshal status event: %v", job.ID, err)
		return
	}
	if err := w.rdb.Publish(ctx, "user:"+job.UserID+":jobs", event); err != nil {
		log.Printf("job=%s: failed to publish status event: %v", job.ID, err)
	}
}

func (w *Worker) handleSendEmail(job *Job) error {
	ctx := context.Background()
	if err := w.repo.MarkRunning(ctx, job.ID, w.workerID); err != nil {
		return fmt.Errorf("sendEmail job=%s: failed to mark running: %w", job.ID, err)
	}
	log.Printf("sendEmail job=%s worker=%s queue_wait=%s", job.ID, w.workerID, time.Since(job.ScheduledAt))
	time.Sleep(2 * time.Second)
	if err := w.repo.MarkCompleted(ctx, job.ID); err != nil {
		return fmt.Errorf("sendEmail job=%s: failed to mark completed: %w", job.ID, err)
	}
	w.publishStatusEvent(ctx, job, jobs.JobStatusCompleted)
	log.Printf("sendEmail job=%s completed", job.ID)
	return nil
}

func (w *Worker) handleReportGeneration(job *Job) error {
	ctx := context.Background()
	if err := w.repo.MarkRunning(ctx, job.ID, w.workerID); err != nil {
		return fmt.Errorf("reportGeneration job=%s: failed to mark running: %w", job.ID, err)
	}
	log.Printf("reportGeneration job=%s worker=%s queue_wait=%s", job.ID, w.workerID, time.Since(job.ScheduledAt))
	time.Sleep(3 * time.Second)
	if err := w.repo.MarkCompleted(ctx, job.ID); err != nil {
		return fmt.Errorf("reportGeneration job=%s: failed to mark completed: %w", job.ID, err)
	}
	w.publishStatusEvent(ctx, job, jobs.JobStatusCompleted)
	log.Printf("reportGeneration job=%s completed", job.ID)
	return nil
}

func (w *Worker) handleResizeImage(job *Job) error {
	ctx := context.Background()
	if err := w.repo.MarkRunning(ctx, job.ID, w.workerID); err != nil {
		return fmt.Errorf("resizeImage job=%s: failed to mark running: %w", job.ID, err)
	}
	log.Printf("resizeImage job=%s worker=%s queue_wait=%s", job.ID, w.workerID, time.Since(job.ScheduledAt))
	time.Sleep(2 * time.Second)
	if err := w.repo.MarkCompleted(ctx, job.ID); err != nil {
		return fmt.Errorf("resizeImage job=%s: failed to mark completed: %w", job.ID, err)
	}
	w.publishStatusEvent(ctx, job, jobs.JobStatusCompleted)
	log.Printf("resizeImage job=%s completed", job.ID)
	return nil
}

func (w *Worker) handleExportCSV(job *Job) error {
	ctx := context.Background()
	if err := w.repo.MarkRunning(ctx, job.ID, w.workerID); err != nil {
		return fmt.Errorf("exportCSV job=%s: failed to mark running: %w", job.ID, err)
	}
	log.Printf("exportCSV job=%s worker=%s queue_wait=%s", job.ID, w.workerID, time.Since(job.ScheduledAt))
	time.Sleep(2 * time.Second)
	if err := w.repo.MarkFailed(ctx, job.ID, "Failed to export CSV"); err != nil {
		return fmt.Errorf("exportCSV job=%s: failed to mark failed: %w", job.ID, err)
	}
	w.publishStatusEvent(ctx, job, jobs.JobStatusFailed)
	log.Printf("exportCSV job=%s failed", job.ID)
	return nil
}
