//go:build integration

// Integration tests for the jobs domain. These talk to a real Postgres and a
// real RabbitMQ, so they are gated behind the `integration` build tag and do
// NOT run during a plain `go test ./...`.
//
// Run them with:
//
//	go test -tags=integration ./internal/jobs/...
//
// They expect the same services docker-compose brings up (Postgres + RabbitMQ)
// and a populated .env in the repo root.
package jobs

import (
	"context"
	"os"
	"testing"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"jobflow/internal/auth"
	"jobflow/internal/config"
	"jobflow/internal/database"
	"jobflow/internal/rabbitmq"
)

var (
	testDB      *gorm.DB
	testMQ      *rabbitmq.RabbitMQ
	testService *Service
)

// TestMain wires up shared dependencies once for the whole package, then tears
// them down. This is the standard Go integration-test setup/teardown hook.
func TestMain(m *testing.M) {
	if err := config.Load(); err != nil {
		panic("integration test: config load failed: " + err.Error())
	}

	db, err := database.ConnectDB()
	if err != nil {
		panic("integration test: db connect failed: " + err.Error())
	}
	testDB = db

	mq := rabbitmq.NewRabbitMQConnection()
	if err := mq.InitializeQueues(rabbitmq.AppQueues); err != nil {
		panic("integration test: queue init failed: " + err.Error())
	}
	testMQ = mq

	testService = NewService(NewJobRepository(testDB), testMQ)

	code := m.Run()
	os.Exit(code)
}

// seedUser inserts a throwaway user so the jobs.user_id foreign key is
// satisfied, and returns its ID. The caller is responsible for cleanup.
func seedUser(t *testing.T) string {
	t.Helper()
	u := &auth.User{Email: "itest+" + time.Now().Format("150405.000000") + "@example.com", PasswordHash: "x"}
	if err := testDB.Create(u).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return u.ID
}

// TestCreateJobService_PersistsAndMarksQueued is the happy path: a created job
// lands in the DB AND gets queued_at stamped, proving the publish + bookkeeping
// path ran end-to-end.
func TestCreateJobService_PersistsAndMarksQueued(t *testing.T) {
	ctx := context.Background()
	userID := seedUser(t)

	res, err := testService.CreateJobService(ctx, &CreateJobRequest{
		UserID:  userID,
		Type:    JobEmail,
		Payload: datatypes.JSON([]byte(`{"to":"a@b.com"}`)),
	})
	if err != nil {
		t.Fatalf("CreateJobService returned error: %v", err)
	}

	// TODO: implement by developer
	// Hint: reload the job by ID and assert it exists with status=pending and a
	// non-nil QueuedAt, proving the dual-write completed.
	_ = res
}

// TestCreateJobsService_BulkAllQueued verifies the bulk path creates every job
// and that one publish failure does not abort the rest (the reconciler backstop
// means partial publishes are recoverable, not fatal).
func TestCreateJobsService_BulkAllQueued(t *testing.T) {
	// TODO: implement by developer
	// Hint: submit a batch via CreateJobsService and assert every returned job
	// has a row in the DB.
	t.Skip("integration test body to be implemented by developer")
}

// TestRepublishStuckJobs_RecoversUnqueued verifies the reconciler picks up a
// job whose queued_at is NULL and republishes it.
func TestRepublishStuckJobs_RecoversUnqueued(t *testing.T) {
	// TODO: implement by developer
	// Hint: insert a pending job with queued_at=NULL and scheduled_at in the
	// past, run RepublishStuckJobs, then assert queued_at is now set.
	t.Skip("integration test body to be implemented by developer")
}
