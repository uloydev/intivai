package session

import (
	"context"

	ivdomain "github.com/intivai/backend/internal/interview/domain"
	sbdomain "github.com/intivai/backend/internal/sandbox/domain"
)

// HandleCodeRun executes code challenges in a secure sandbox.
func (s *InterviewSession) HandleCodeRun(m ivdomain.CodeRunMessage) {
	s.historyMu.Lock()
	arch := s.archetype
	s.historyMu.Unlock()
	if arch != ivdomain.ArchetypeCoding {
		s.send(ivdomain.CodeResultMessage{
			Type:  ivdomain.MsgCodeResult,
			Error: "code execution is only allowed on coding challenges",
		})
		return
	}
	if !s.ReserveCodeRun() {
		s.send(ivdomain.CodeResultMessage{
			Type:  ivdomain.MsgCodeResult,
			Error: "a code execution is already in progress; please wait for it to finish",
		})
		return
	}
	if s.acquireCodeRun != nil && !s.acquireCodeRun() {
		s.ReleaseCodeRun()
		s.send(ivdomain.CodeResultMessage{
			Type:  ivdomain.MsgCodeResult,
			Error: "sandbox is busy; please try running again in a moment",
		})
		return
	}
	go func() {
		defer s.ReleaseCodeRun()
		if s.releaseCodeRun != nil {
			defer s.releaseCodeRun()
		}
		s.runCode(m)
	}()
}

func (s *InterviewSession) runCode(m ivdomain.CodeRunMessage) {
	if s.codeRunner == nil {
		s.send(ivdomain.CodeResultMessage{Type: ivdomain.MsgCodeResult, Error: "code execution is unavailable"})
		return
	}
	tcs := make([]sbdomain.TestCase, 0, len(m.TestCases))
	for i, tc := range m.TestCases {
		if i >= MaxSandboxTestCases {
			break
		}
		tcs = append(tcs, sbdomain.TestCase{
			ID:             tc.ID,
			Input:          tc.Input,
			ExpectedOutput: tc.ExpectedOutput,
			Hidden:         tc.Hidden,
		})
	}
	execReq := sbdomain.ExecutionRequest{
		Language:  sbdomain.Language(m.Language),
		Code:      m.Code,
		Stdin:     m.Stdin,
		TestCases: tcs,
	}
	runCtx, cancel := context.WithTimeout(s.ctx, CodeRunTimeout)
	defer cancel()

	res, err := s.codeRunner.Execute(runCtx, execReq)
	if err != nil {
		s.engine.log.Error().Err(err).Str("interview_id", s.interviewID.String()).Str("org_id", s.orgID).Msg("code runner execution failed")
		s.send(ivdomain.CodeResultMessage{
			Type:  ivdomain.MsgCodeResult,
			Error: "code execution unavailable; please retry",
		})
		return
	}
	if res == nil {
		s.engine.log.Error().Str("interview_id", s.interviewID.String()).Str("org_id", s.orgID).Msg("code runner returned nil response")
		s.send(ivdomain.CodeResultMessage{
			Type:  ivdomain.MsgCodeResult,
			Error: "code execution unavailable; please retry",
		})
		return
	}
	var rawTests []ivdomain.TestResult
	for _, tr := range res.TestResults {
		rawTests = append(rawTests, ivdomain.TestResult{
			ID:             tr.TestCase.ID,
			Passed:         tr.Passed,
			ActualOutput:   tr.ActualOutput,
			ExpectedOutput: tr.TestCase.ExpectedOutput,
			Error:          tr.Error,
		})
	}
	s.send(ivdomain.CodeResultMessage{
		Type:        ivdomain.MsgCodeResult,
		Stdout:      res.Stdout,
		Stderr:      res.Stderr,
		ExitCode:    res.ExitCode,
		DurationMs:  res.DurationMs,
		AllPassed:   res.AllPassed,
		TestResults: rawTests,
		Error:       res.Error,
	})
	if err := s.engine.svc.RecordCodingSession(s.ctx, s.orgID, s.interviewID, ivdomain.CodingSession{
		QuestionIdx: m.QuestionIdx,
		Language:    m.Language,
		Code:        m.Code,
		FinalResult: &ivdomain.ExecutionResult{
			Stdout:      res.Stdout,
			Stderr:      res.Stderr,
			ExitCode:    res.ExitCode,
			DurationMs:  res.DurationMs,
			AllPassed:   res.AllPassed,
			TestResults: rawTests,
			Error:       res.Error,
		},
	}); err != nil {
		s.engine.log.Error().Err(err).Str("interview_id", s.interviewID.String()).Str("org_id", s.orgID).Msg("record coding session failed")
	}
}
