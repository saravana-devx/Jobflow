package logger

import (
	"log/slog"
	"os"
)

// Init sets up slog as the global JSON logger and returns it. JSON keeps logs
// queryable by fields like job_id / worker_id. call once at startup; attach
// fields later with slog.With("job_id", id).
func Init(level slog.Level) *slog.Logger {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	l := slog.New(h)
	slog.SetDefault(l)
	return l
}
