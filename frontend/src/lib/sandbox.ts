import { api } from "./api"
import type {
  AICodeReview,
  SandboxExecutionResult,
  SandboxLanguage,
  SandboxTestCase,
} from "@/types/api"

// Executions are CPU-bounded in the sandbox — always cap wall time at 5s.
export function runCode(
  language: SandboxLanguage,
  code: string,
  testCases: SandboxTestCase[]
): Promise<SandboxExecutionResult> {
  return api.post<SandboxExecutionResult>("/sandbox/execute", {
    language,
    code,
    test_cases: testCases,
    timeout_sec: 5,
  })
}

export function aiReview(language: SandboxLanguage, code: string, problem: string): Promise<AICodeReview> {
  return api.post<AICodeReview>("/sandbox/evaluate", {
    language,
    code,
    problem,
  })
}
