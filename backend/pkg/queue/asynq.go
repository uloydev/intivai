package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// Task names — single source for producers + workers. Add new tasks here,
// never as bare string literals in workers.
const (
	TaskEvaluateInterview = "evaluate_interview"
)

// RedisConfig encapsulates connection parameters for Redis and Asynq.
type RedisConfig struct {
	Addr     string
	Password string
	URL      string
}

// ParseRedisConnOpt parses connection parameters into Asynq and Go-Redis options.
func ParseRedisConnOpt(cfg RedisConfig) (asynq.RedisConnOpt, *redis.Options, error) {
	if cfg.URL != "" {
		asynqOpt, err := asynq.ParseRedisURI(cfg.URL)
		if err != nil {
			return nil, nil, fmt.Errorf("parse redis uri for asynq: %w", err)
		}
		redisOpt, err := redis.ParseURL(cfg.URL)
		if err != nil {
			return nil, nil, fmt.Errorf("parse redis uri for redis: %w", err)
		}
		return asynqOpt, redisOpt, nil
	}
	addr := cfg.Addr
	if addr == "" {
		addr = "localhost:6379"
	}
	clientOpt := asynq.RedisClientOpt{
		Addr:     addr,
		Password: cfg.Password,
	}
	redisOpt := &redis.Options{
		Addr:     addr,
		Password: cfg.Password,
	}
	return clientOpt, redisOpt, nil
}

type Client struct {
	client *asynq.Client
}

func NewClient(redisAddr string) *Client {
	return NewClientWithOpt(asynq.RedisClientOpt{Addr: redisAddr})
}

func NewClientWithOpt(opt asynq.RedisConnOpt) *Client {
	return &Client{client: asynq.NewClient(opt)}
}

func (c *Client) Close() error {
	return c.client.Close()
}

// Enqueue adds a task with default retry (3) and timeout (5m). The caller's
// span context rides in the task header so worker deliveries link back to
// the producing trace (plan batch D).
func (c *Client) Enqueue(ctx context.Context, jobType string, payload any, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	task := asynq.NewTaskWithHeaders(jobType, mustMarshal(payload), map[string]string{}, opts...)
	if task == nil {
		return nil, fmt.Errorf("new task: nil task for %s", jobType)
	}
	injectTraceContext(ctx, task)
	defaults := []asynq.Option{asynq.MaxRetry(3), asynq.Timeout(5 * time.Minute)}
	return c.client.EnqueueContext(ctx, task, append(defaults, opts...)...)
}

func (c *Client) AsynqClient() *asynq.Client { return c.client }

type Server struct {
	server *asynq.Server
}

func NewServer(redisAddr string, concurrency int, log zerolog.Logger) *Server {
	return NewServerWithOpt(asynq.RedisClientOpt{Addr: redisAddr}, concurrency, log)
}

func NewServerWithOpt(opt asynq.RedisConnOpt, concurrency int, log zerolog.Logger) *Server {
	if concurrency <= 0 {
		concurrency = 10
	}
	srv := asynq.NewServer(
		opt,
		asynq.Config{
			Concurrency: concurrency,
			Logger:      asynqLogger{log: log},
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				retried, _ := asynq.GetRetryCount(ctx)
				maxRetry, _ := asynq.GetMaxRetry(ctx)
				if retried >= maxRetry {
					log.Error().Err(err).Str("type", task.Type()).Msg("dead letter: task exhausted all retries")
				} else {
					log.Warn().Err(err).Str("type", task.Type()).Int("retry", retried).Msg("task failed, will retry")
				}
			}),
		},
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

func NewRedisClient(opt *redis.Options) *redis.Client {
	return redis.NewClient(opt)
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
