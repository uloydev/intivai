package sidecar

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDockerRunArgsHardening(t *testing.T) {
	args := dockerRunArgs("intivai-sandbox-go", "sandbox-run-1", []string{"sh", "-c", "cat > /tmp/main.go; echo aGk= | base64 -d > /tmp/input.txt; cd /tmp && go run main.go < /tmp/input.txt"})
	joined := strings.Join(args, " ")
	for _, flag := range []string{"--network=none", "--memory=512m", "--cpus=1", "--pids-limit=128", "--read-only", "--tmpfs", "--name"} {
		require.Contains(t, joined, flag, "docker run must include %s", flag)
	}
	require.Contains(t, args, "intivai-sandbox-go")
}

func TestLanguageSpecs(t *testing.T) {
	for _, lang := range []string{"go", "python", "javascript", "typescript"} {
		spec, ok := languageSpecs[lang]
		require.True(t, ok, "%s spec missing", lang)
		require.NotEmpty(t, spec.image)
		require.NotEmpty(t, spec.filename)
		require.NotEmpty(t, spec.execCmd)
	}
	_, ok := languageSpecs["ruby"]
	require.False(t, ok)
}

func TestCompareOutputs(t *testing.T) {
	require.True(t, compareOutputs("5\n", "5"))
	require.True(t, compareOutputs("5\r\n", "5\n"))
	require.True(t, compareOutputs(" 5 \n", "5"))
	require.False(t, compareOutputs("6", "5"))
}

func TestCappedBuffer(t *testing.T) {
	var c cappedBuffer
	big := strings.Repeat("x", MaxOutputBytes+1024)
	n, _ := c.Write([]byte(big))
	require.Equal(t, len(big), n) // writer never blocks
	require.Equal(t, MaxOutputBytes, c.buf.Len())
	require.Contains(t, c.String(), "truncated")
}

// fakeDocker — records invocations; fails nothing.
type fakeDocker struct {
	stdout string
	stderr string
	err    error
	calls  [][]string
}

func (f *fakeDocker) Run(_ context.Context, args []string, stdin string, stdout, stderr *cappedBuffer) error {
	f.calls = append(f.calls, args)
	_, _ = stdout.Write([]byte(f.stdout))
	_, _ = stderr.Write([]byte(f.stderr))
	return f.err
}

func (f *fakeDocker) Kill(string) error { return nil }

func TestRunnerSingleRun(t *testing.T) {
	t.Setenv("SANDBOXD_WORKDIR", t.TempDir())
	r := &Runner{docker: &fakeDocker{stdout: "Hello, Intivai Sandbox!\n"}}
	res, err := r.Execute(context.Background(), ExecutionRequest{
		Language: "go",
		Code:     "package main\nfunc main() {}",
	})
	require.NoError(t, err)
	require.True(t, res.AllPassed)
	require.Contains(t, res.Stdout, "Hello, Intivai Sandbox!")
}

func TestRunnerTestCaseMode(t *testing.T) {
	t.Setenv("SANDBOXD_WORKDIR", t.TempDir())
	f := &fakeDocker{stdout: "5\n"}
	r := &Runner{docker: f}
	res, err := r.Execute(context.Background(), ExecutionRequest{
		Language: "python",
		Code:     "print(input())",
		TestCases: []TestCase{
			{ID: "1", Input: "2 3\n", ExpectedOutput: "5\n"},
			{ID: "2", Input: "10 20\n", ExpectedOutput: "30\n"},
		},
	})
	require.NoError(t, err)
	require.Len(t, res.TestResults, 2)
	require.True(t, res.TestResults[0].Passed)
	require.False(t, res.TestResults[1].Passed) // fake always prints 5
	require.False(t, res.AllPassed)
	require.Len(t, f.calls, 2) // one container run per test case
}

func TestRunnerUnsupportedLanguage(t *testing.T) {
	t.Setenv("SANDBOXD_WORKDIR", t.TempDir())
	r := &Runner{docker: &fakeDocker{}}
	_, err := r.Execute(context.Background(), ExecutionRequest{Language: "ruby", Code: "x"})
	require.ErrorContains(t, err, "unsupported language")
}

func TestRunnerEmptyCode(t *testing.T) {
	t.Setenv("SANDBOXD_WORKDIR", t.TempDir())
	r := &Runner{docker: &fakeDocker{}}
	_, err := r.Execute(context.Background(), ExecutionRequest{Language: "go", Code: "  "})
	require.ErrorContains(t, err, "code cannot be empty")
}

// failingDocker — kills the run on context cancellation.
type failingDocker struct {
	called bool
}

func (f *failingDocker) Run(ctx context.Context, _ []string, _ string, _, _ *cappedBuffer) error {
	f.called = true
	<-ctx.Done()
	return ctx.Err()
}

func (f *failingDocker) Kill(string) error { return nil }

func TestRunnerTimeoutKillsContainer(t *testing.T) {
	t.Setenv("SANDBOXD_WORKDIR", t.TempDir())
	f := &failingDocker{}
	r := &Runner{docker: f}
	res, err := r.Execute(context.Background(), ExecutionRequest{Language: "go", Code: "x", TimeoutSec: 1})
	require.NoError(t, err)
	require.Equal(t, -1, res.ExitCode)
	require.Contains(t, res.Error, "timed out")
}
