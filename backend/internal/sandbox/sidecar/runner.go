// Package sidecar runs untrusted code in throwaway Docker containers
// (ADR-0002). The sidecar owns the Docker socket; the app reaches it over
// mTLS gRPC and never executes code itself.
package sidecar

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	MaxOutputBytes = 64 * 1024 // 64KB output cap
	// Cold `go run` compile under --cpus=1 exceeds 5s; 10s is the proto cap.
	DefaultTimeout = 10 * time.Second
	// maxPayloadBytes — code + input travel in docker stdin / argv (base64);
	// keep well under the 2MB exec ARG_MAX.
	maxPayloadBytes = 512 * 1024
)

// languageSpec — the per-language image + execution command. The code is
// piped on stdin (`cat > /tmp/<file>`), the test input is embedded base64 in
// the shell string (base64 alphabet cannot inject shell metacharacters), and
// /tmp is a writable tmpfs for caches (go build, tsx). Root fs read-only,
// network off, no bind mounts (avoids daemon-vs-container path namespaces).
type languageSpec struct {
	image    string
	filename string
	execCmd  string // run after `cat > /tmp/<file> && echo IN | base64 -d > /tmp/input.txt`
}

var languageSpecs = map[string]languageSpec{
	"go": {
		image:    "intivai-sandbox-go",
		filename: "main.go",
		execCmd:  "cd /tmp && go run main.go < /tmp/input.txt",
	},
	"python": {
		image:    "intivai-sandbox-python",
		filename: "main.py",
		execCmd:  "cd /tmp && python3 main.py < /tmp/input.txt",
	},
	"javascript": {
		image:    "intivai-sandbox-node",
		filename: "main.js",
		execCmd:  "cd /tmp && node main.js < /tmp/input.txt",
	},
	"typescript": {
		image:    "intivai-sandbox-ts",
		filename: "main.ts",
		execCmd:  "cd /tmp && tsx main.ts < /tmp/input.txt",
	},
}

func supportedLanguages() []string {
	out := make([]string, 0, len(languageSpecs))
	for lang := range languageSpecs {
		out = append(out, lang)
	}
	return out
}

// dockerRunArgs — the hardened docker run invocation. Pure so it is unit-testable.
// The container gets a deterministic name so the runner can kill it on timeout.
func dockerRunArgs(image, name string, command []string) []string {
	args := []string{
		"run", "--rm", "-i", "--name", name,
		"--network=none",
		// 512m: the go toolchain (compile+link) needs more than 256m; the
		// RUNNING program is kept under GOMEMLIMIT=256MiB.
		"--memory=512m",
		"--cpus=1",
		"--pids-limit=128",
		"--read-only",
		"--tmpfs", "/tmp:rw,size=128m,exec",
		"-e", "GOCACHE=/tmp/gocache",
		"-e", "GOPATH=/tmp/gopath",
		"-e", "GOMEMLIMIT=256MiB",
		image,
	}
	return append(args, command...)
}

// cappedBuffer keeps at most MaxOutputBytes without blocking the process.
type cappedBuffer struct {
	buf     bytes.Buffer
	dropped bool
}

func (c *cappedBuffer) Write(p []byte) (int, error) {
	remaining := MaxOutputBytes - c.buf.Len()
	if remaining > 0 {
		n := len(p)
		if n > remaining {
			n = remaining
		}
		_, _ = c.buf.Write(p[:n])
		if len(p) > n {
			c.dropped = true
		}
	} else {
		c.dropped = true
	}
	return len(p), nil
}

func (c *cappedBuffer) String() string {
	s := c.buf.String()
	if c.dropped {
		s += "\n...[output truncated due to length cap]"
	}
	return s
}

// DockerClient is the exec surface, overridable in tests.
type DockerClient interface {
	Run(ctx context.Context, args []string, stdin string, stdout, stderr *cappedBuffer) error
	Kill(containerID string) error
}

type execDocker struct{}

func (execDocker) Run(ctx context.Context, args []string, stdin string, stdout, stderr *cappedBuffer) error {
	cmd := exec.CommandContext(ctx, "docker", args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func (execDocker) Kill(containerID string) error {
	cmd := exec.Command("docker", "kill", strings.TrimSpace(containerID))
	return cmd.Run()
}

// Runner executes one ExecutionRequest by delegating to per-language
// containers. Test-case mode loops the container once per case.
type Runner struct {
	docker DockerClient
}

func NewRunner() *Runner {
	return &Runner{docker: execDocker{}}
}

type ExecutionRequest struct {
	Language   string
	Code       string
	Stdin      string
	TestCases  []TestCase
	TimeoutSec int32
}

type TestCase struct {
	ID             string
	Input          string
	ExpectedOutput string
	Hidden         bool
}

type TestCaseResult struct {
	TestCase     TestCase
	ActualOutput string
	Passed       bool
	DurationMs   int64
	Error        string
}

type ExecutionResult struct {
	Stdout      string
	Stderr      string
	ExitCode    int
	DurationMs  int64
	AllPassed   bool
	Error       string
	TestResults []TestCaseResult
}

func (r *Runner) Execute(ctx context.Context, req ExecutionRequest) (*ExecutionResult, error) {
	spec, ok := languageSpecs[req.Language]
	if !ok {
		return nil, fmt.Errorf("unsupported language %q (supported: %s)", req.Language, strings.Join(supportedLanguages(), ", "))
	}
	if strings.TrimSpace(req.Code) == "" {
		return nil, errors.New("code cannot be empty")
	}
	timeout := DefaultTimeout
	if req.TimeoutSec > 0 && req.TimeoutSec <= 10 {
		timeout = time.Duration(req.TimeoutSec) * time.Second
	}
	runOne := func(runCtx context.Context, stdin string) (*ExecutionResult, error) {
		// The test input rides base64 in argv — keep it well under ARG_MAX.
		if len(req.Code)+len(stdin) > maxPayloadBytes {
			return nil, fmt.Errorf("payload too large (max %d bytes)", maxPayloadBytes)
		}
		runName := "sandbox-" + uuid.NewString()[:12]
		input64 := base64.StdEncoding.EncodeToString([]byte(stdin))
		shell := fmt.Sprintf("cat > /tmp/%s; echo %s | base64 -d > /tmp/input.txt; %s",
			spec.filename, input64, spec.execCmd)

		var stdoutBuf, stderrBuf cappedBuffer
		runErr := r.docker.Run(runCtx, dockerRunArgs(spec.image, runName, []string{"sh", "-c", shell}), req.Code, &stdoutBuf, &stderrBuf)

		// Timed out (or parent ctx cancelled): docker run alone would leave
		// the container running — kill it by name; --rm cleans it up.
		if runCtx.Err() != nil {
			_ = r.docker.Kill(runName)
		}

		res := &ExecutionResult{
			Stdout: stdoutBuf.String(),
			Stderr: stderrBuf.String(),
		}
		if runCtx.Err() != nil {
			res.Error = fmt.Sprintf("Execution timed out (%s limit exceeded)", timeout)
			res.ExitCode = -1
			return res, nil
		}
		if runErr != nil {
			var exitErr *exec.ExitError
			if errors.As(runErr, &exitErr) {
				res.ExitCode = exitErr.ExitCode()
			} else {
				res.ExitCode = 1
			}
			if res.Stderr != "" {
				res.Error = res.Stderr
			} else {
				res.Error = runErr.Error()
			}
			res.AllPassed = false
			return res, nil
		}
		res.AllPassed = true
		return res, nil
	}

	if len(req.TestCases) > 0 {
		var results []TestCaseResult
		var total int64
		allPassed := true
		for _, tc := range req.TestCases {
			tcCtx, cancel := context.WithTimeout(ctx, timeout)
			start := time.Now()
			res, err := runOne(tcCtx, tc.Input)
			dur := time.Since(start).Milliseconds()
			cancel()
			if err != nil {
				return nil, err
			}
			total += dur
			passed := res.ExitCode == 0 && res.Error == "" && compareOutputs(res.Stdout, tc.ExpectedOutput)
			if !passed {
				allPassed = false
			}
			results = append(results, TestCaseResult{
				TestCase:     tc,
				ActualOutput: res.Stdout,
				Passed:       passed,
				DurationMs:   dur,
				Error:        res.Error,
			})
		}
		return &ExecutionResult{DurationMs: total, TestResults: results, AllPassed: allPassed}, nil
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	start := time.Now()
	res, err := runOne(execCtx, req.Stdin)
	if err != nil {
		return nil, err
	}
	res.DurationMs = time.Since(start).Milliseconds()
	return res, nil
}

// compareOutputs — trailing-whitespace-insensitive comparison (the test
// harness normalizes CRLF and trailing newlines).
func compareOutputs(actual, expected string) bool {
	norm := func(s string) string {
		s = strings.ReplaceAll(s, "\r\n", "\n")
		return strings.TrimSpace(s)
	}
	return norm(actual) == norm(expected)
}
