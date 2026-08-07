package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type Client struct {
	client *asynq.Client
}

func NewClient(redisAddr string) *Client {
	return &Client{client: asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})}
}

func (c *Client) Close() error {
	return c.client.Close()
}

// Enqueue adds a task with default retry (3) and timeout (5m).
func (c *Client) Enqueue(ctx context.Context, jobType string, payload any, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	task := asynq.NewTask(jobType, mustMarshal(payload), opts...)
	if task == nil {
		return nil, fmt.Errorf("new task: nil task for %s", jobType)
	}
	defaults := []asynq.Option{asynq.MaxRetry(3), asynq.Timeout(5 * time.Minute)}
	return c.client.EnqueueContext(ctx, task, append(defaults, opts...)...)
}

func (c *Client) AsynqClient() *asynq.Client { return c.client }

type Server struct {
	server *asynq.Server
}

func NewServer(redisAddr string, concurrency int, log zerolog.Logger) *Server {
	if concurrency <= 0 {
		concurrency = 10
	}
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{Concurrency: concurrency, Logger: asynqLogger{log: log}},
	)
	return &Server{server: srv}
}

// Register maps job types to handlers, then starts the worker.
func (s *Server) Start(mux *asynq.ServeMux) error {
	if mux == nil {
		mux = asynq.NewServeMux()
	}
	return s.server.Start(mux)
}

func (s *Server) Shutdown(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		s.server.Shutdown()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func NewRedis(addr string) *redis.Client {
	return redis.NewClient(&redis.Options{Addr: addr})
}

type asynqLogger struct {
	log zerolog.Logger
}

func (l asynqLogger) Debug(args ...any)                 { l.log.Debug().Msg(fmt.Sprint(args...)) }
func (l asynqLogger) Info(args ...any)                  { l.log.Info().Msg(fmt.Sprint(args...)) }
func (l asynqLogger) Warn(args ...any)                  { l.log.Warn().Msg(fmt.Sprint(args...)) }
func (l asynqLogger) Error(args ...any)                 { l.log.Error().Msg(fmt.Sprint(args...)) }
func (l asynqLogger) Fatal(args ...any)                 { l.log.Error().Msg("fatal: " + fmt.Sprint(args...)) }
func (l asynqLogger) Debugf(format string, args ...any) { l.log.Debug().Msgf(format, args...) }
func (l asynqLogger) Infof(format string, args ...any)  { l.log.Info().Msgf(format, args...) }
func (l asynqLogger) Warnf(format string, args ...any)  { l.log.Warn().Msgf(format, args...) }
func (l asynqLogger) Errorf(format string, args ...any) { l.log.Error().Msgf(format, args...) }
func (l asynqLogger) Fatalf(format string, args ...any) {
	l.log.Error().Msgf("fatal: "+format, args...)
}
