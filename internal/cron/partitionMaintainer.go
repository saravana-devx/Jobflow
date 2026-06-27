package cron

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	redisx "jobflow/internal/redis"
)

const (
	partitionCheckInterval = 24 * time.Hour
	// partitionFutureMonths is how many months ahead to pre-create partitions
	// so inserts never fall into jobs_default around a month boundary.
	partitionFutureMonths = 1
	// CREATE IF NOT EXISTS is already idempotent; the lease just avoids redundant runs
	partitionLeaseTTL = 23 * time.Hour
)

// JobPartitionMaintainer keeps the monthly partitions of the range-partitioned
// jobs table provisioned ahead of time. Partitions are created on demand here
// rather than hardcoded in the init SQL, so the table never runs out of
// partitions as time advances.
type JobPartitionMaintainer struct {
	db  *gorm.DB
	rdb *redisx.Redis
}

func NewJobPartitionMaintainer(db *gorm.DB, rdb *redisx.Redis) *JobPartitionMaintainer {
	return &JobPartitionMaintainer{db: db, rdb: rdb}
}

func (c *JobPartitionMaintainer) Start(ctx context.Context) {
	// Provision partitions immediately on boot, then re-check daily.
	runIfLeader(ctx, c.rdb, "cron:partition-maintainer", partitionLeaseTTL, func() {
		c.ensure(ctx)
	})

	ticker := time.NewTicker(partitionCheckInterval)
	defer ticker.Stop()
	log.Println("JobPartitionMaintainer started")
	for {
		select {
		case <-ticker.C:
			runIfLeader(ctx, c.rdb, "cron:partition-maintainer", partitionLeaseTTL, func() {
				c.ensure(ctx)
			})
		case <-ctx.Done():
			log.Println("JobPartitionMaintainer stopped")
			return
		}
	}
}

// ensure creates the partition for the current month and the next
// partitionFutureMonths months. CREATE ... IF NOT EXISTS makes it idempotent,
// so re-running every day is cheap and safe.
func (c *JobPartitionMaintainer) ensure(ctx context.Context) {
	now := time.Now().UTC()
	month := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i <= partitionFutureMonths; i++ {
		start := month.AddDate(0, i, 0)
		if err := c.ensureMonth(ctx, start); err != nil {
			log.Printf("JobPartitionMaintainer: ensure %s failed: %v", start.Format("2006_01"), err)
		}
	}
}

func (c *JobPartitionMaintainer) ensureMonth(ctx context.Context, start time.Time) error {
	end := start.AddDate(0, 1, 0)
	name := fmt.Sprintf("jobs_%s", start.Format("2006_01"))

	// Partition and bound names are derived from time, never user input, so
	// formatting them into the DDL is safe.
	sql := fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS %s PARTITION OF jobs FOR VALUES FROM ('%s') TO ('%s');`,
		name, start.Format("2006-01-02"), end.Format("2006-01-02"),
	)
	if err := c.db.WithContext(ctx).Exec(sql).Error; err != nil {
		return fmt.Errorf("create partition %s: %w", name, err)
	}
	return nil
}
