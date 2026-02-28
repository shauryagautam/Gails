package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/shaurya/gails/config"
	"github.com/shaurya/gails/framework"
	"go.uber.org/zap"
)

// Worker manages asynq worker server.
type Worker struct {
	Server *asynq.Server
	Mux    *asynq.ServeMux
}

// NewWorker creates a new background job Worker.
func NewWorker(redisCfg config.RedisConfig, queueCfg config.QueueConfig) *Worker {
	concurrency := queueCfg.Concurrency
	if concurrency <= 0 {
		concurrency = 10
	}

	queues := map[string]int{
		"default": 3,
	}
	for _, q := range queueCfg.Queues {
		queues[q.Name] = q.Weight
	}

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: extractRedisAddr(redisCfg.URL)},
		asynq.Config{
			Concurrency: concurrency,
			Queues:      queues,
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				if framework.Log != nil {
					framework.Log.Error("Job failed",
						zap.String("type", task.Type()),
						zap.Error(err),
					)
				}
				framework.RecordJobProcessed(task.Type(), "failure")
			}),
		},
	)

	return &Worker{
		Server: srv,
		Mux:    asynq.NewServeMux(),
	}
}

// Handle registers a job handler.
func (w *Worker) Handle(pattern string, handler asynq.Handler) {
	w.Mux.Handle(pattern, loggingHandler(pattern, handler))
}

// HandleFunc registers a job handler function.
func (w *Worker) HandleFunc(pattern string, handler func(context.Context, *asynq.Task) error) {
	w.Mux.HandleFunc(pattern, loggingHandlerFunc(pattern, handler))
}

// Run starts the worker — this blocks until shutdown signal.
func (w *Worker) Run() {
	if framework.Log != nil {
		framework.Log.Info("Starting Gails worker...")
	}
	if err := w.Server.Run(w.Mux); err != nil {
		if framework.Log != nil {
			framework.Log.Fatal("Worker failed", zap.Error(err))
		}
	}
}

// loggingHandler wraps a handler with start/completion/failure logging + metrics.
func loggingHandler(jobType string, next asynq.Handler) asynq.Handler {
	return asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) error {
		start := time.Now()
		if framework.Log != nil {
			framework.Log.Info("Job started",
				zap.String("type", jobType),
				zap.String("id", task.ResultWriter().TaskID()),
			)
		}

		err := next.ProcessTask(ctx, task)
		duration := time.Since(start)

		if err != nil {
			if framework.Log != nil {
				framework.Log.Error("Job failed",
					zap.String("type", jobType),
					zap.Duration("duration", duration),
					zap.Error(err),
				)
			}
			framework.RecordJobProcessed(jobType, "failure")
			return err
		}

		if framework.Log != nil {
			framework.Log.Info("Job completed",
				zap.String("type", jobType),
				zap.Duration("duration", duration),
			)
		}
		framework.RecordJobProcessed(jobType, "success")
		return nil
	})
}

func loggingHandlerFunc(jobType string, fn func(context.Context, *asynq.Task) error) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, task *asynq.Task) error {
		start := time.Now()
		if framework.Log != nil {
			framework.Log.Info("Job started", zap.String("type", jobType))
		}

		err := fn(ctx, task)
		duration := time.Since(start)

		if err != nil {
			if framework.Log != nil {
				framework.Log.Error("Job failed",
					zap.String("type", jobType),
					zap.Duration("duration", duration),
					zap.Error(err),
				)
			}
			framework.RecordJobProcessed(jobType, "failure")
			return err
		}

		if framework.Log != nil {
			framework.Log.Info("Job completed",
				zap.String("type", jobType),
				zap.Duration("duration", duration),
			)
		}
		framework.RecordJobProcessed(jobType, "success")
		return nil
	}
}

func extractRedisAddr(url string) string {
	// Simple extraction from redis://host:port format
	url = fmt.Sprintf("%s", url)
	if len(url) > 8 && url[:8] == "redis://" {
		return url[8:]
	}
	return "localhost:6379"
}
