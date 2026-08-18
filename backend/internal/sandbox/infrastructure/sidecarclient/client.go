// Package sidecarclient — app-side mTLS gRPC client for the sandbox sidecar
// (ADR-0002). The app never executes code itself.
package sidecarclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/intivai/backend/internal/sandbox/domain"
	"github.com/intivai/backend/internal/sandbox/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Client struct {
	conn *grpc.ClientConn
	svc  proto.SandboxServiceClient
}

// NewClient dials the sidecar with mutual TLS (CA + client cert).
func NewClient(ctx context.Context, addr, caFile, certFile, keyFile string) (*Client, error) {
	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read sandbox CA: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("invalid sandbox CA pem")
	}
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load sandbox client cert: %w", err)
	}
	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
		MinVersion:   tls.VersionTLS13,
	})
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("dial sandbox sidecar: %w", err)
	}
	return &Client{conn: conn, svc: proto.NewSandboxServiceClient(conn)}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

// Execute implements application.CodeRunner.
func (c *Client) Execute(ctx context.Context, req domain.ExecutionRequest) (*domain.ExecutionResult, error) {
	pbReq := &proto.ExecutionRequest{
		Language:   string(req.Language),
		Code:       req.Code,
		Stdin:      req.Stdin,
		TimeoutSec: int32(req.TimeoutSec),
	}
	for _, tc := range req.TestCases {
		pbReq.TestCases = append(pbReq.TestCases, &proto.TestCase{
			Id:             tc.ID,
			Input:          tc.Input,
			ExpectedOutput: tc.ExpectedOutput,
			Hidden:         tc.Hidden,
		})
	}
	res, err := c.svc.Execute(ctx, pbReq)
	if err != nil {
		return nil, fmt.Errorf("sandbox sidecar: %w", err)
	}
	out := &domain.ExecutionResult{
		Stdout:     res.Stdout,
		Stderr:     res.Stderr,
		ExitCode:   int(res.ExitCode),
		DurationMs: res.DurationMs,
		AllPassed:  res.AllPassed,
		Error:      res.Error,
	}
	for _, tr := range res.TestResults {
		out.TestResults = append(out.TestResults, domain.TestCaseResult{
			TestCase: domain.TestCase{
				ID:             tr.TestCaseId,
				Input:          tr.Input,
				ExpectedOutput: tr.ExpectedOutput,
			},
			ActualOutput: tr.ActualOutput,
			Passed:       tr.Passed,
			DurationMs:   tr.DurationMs,
			Error:        tr.Error,
		})
	}
	return out, nil
}
