// Package sidecarpb adapts the docker Runner to the gRPC SandboxService
// contract (proto/sandbox.proto).
package sidecar

import (
	"context"

	"github.com/intivai/backend/internal/sandbox/proto"
)

type GRPCServer struct {
	proto.UnimplementedSandboxServiceServer
	runner *Runner
}

func NewGRPCServer(runner *Runner) *GRPCServer {
	return &GRPCServer{runner: runner}
}

func (s *GRPCServer) Execute(ctx context.Context, req *proto.ExecutionRequest) (*proto.ExecutionResult, error) {
	if req == nil {
		req = &proto.ExecutionRequest{}
	}
	domainReq := ExecutionRequest{
		Language:   req.Language,
		Code:       req.Code,
		Stdin:      req.Stdin,
		TimeoutSec: req.TimeoutSec,
	}
	for _, tc := range req.TestCases {
		domainReq.TestCases = append(domainReq.TestCases, TestCase{
			ID:             tc.Id,
			Input:          tc.Input,
			ExpectedOutput: tc.ExpectedOutput,
			Hidden:         tc.Hidden,
		})
	}

	res, err := s.runner.Execute(ctx, domainReq)
	if err != nil {
		return nil, err
	}

	out := &proto.ExecutionResult{
		Stdout:     res.Stdout,
		Stderr:     res.Stderr,
		ExitCode:   int32(res.ExitCode),
		DurationMs: res.DurationMs,
		AllPassed:  res.AllPassed,
		Error:      res.Error,
	}
	for _, tr := range res.TestResults {
		out.TestResults = append(out.TestResults, &proto.TestCaseResult{
			TestCaseId:     tr.TestCase.ID,
			Input:          tr.TestCase.Input,
			ExpectedOutput: tr.TestCase.ExpectedOutput,
			ActualOutput:   tr.ActualOutput,
			Passed:         tr.Passed,
			DurationMs:     tr.DurationMs,
			Error:          tr.Error,
		})
	}
	return out, nil
}
