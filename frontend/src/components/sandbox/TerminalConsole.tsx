import { useState } from "react"
import { Terminal, CheckCircle, XCircle, Clock, Trash2 } from "lucide-react"
import type { SandboxExecutionResult } from "@/types/api"

interface TerminalConsoleProps {
  result?: SandboxExecutionResult | null
  isRunning: boolean
  onClear: () => void
}

export function TerminalConsole({ result, isRunning, onClear }: TerminalConsoleProps) {
  const [activeTab, setActiveTab] = useState<"output" | "tests">("output")

  const totalTests = result?.test_results?.length || 0
  const passedTests = result?.test_results?.filter((t) => t.passed).length || 0

  return (
    <div className="flex flex-col h-full bg-neutral-950 border border-neutral-800 rounded-lg overflow-hidden text-xs font-mono">
      {/* Console Header */}
      <div className="flex items-center justify-between px-3 py-1.5 bg-neutral-900 border-b border-neutral-800">
        <div className="flex items-center gap-2">
          <button
            onClick={() => setActiveTab("output")}
            className={`flex items-center gap-1 px-2 py-0.5 rounded transition-colors ${
              activeTab === "output" ? "bg-neutral-800 text-neutral-100 font-semibold" : "text-neutral-400 hover:text-neutral-200"
            }`}
          >
            <Terminal className="w-3.5 h-3.5" />
            <span>Terminal Output</span>
          </button>

          {totalTests > 0 && (
            <button
              onClick={() => setActiveTab("tests")}
              className={`flex items-center gap-1.5 px-2 py-0.5 rounded transition-colors ${
                activeTab === "tests" ? "bg-neutral-800 text-neutral-100 font-semibold" : "text-neutral-400 hover:text-neutral-200"
              }`}
            >
              <span>Test Cases</span>
              <span className={`px-1.5 py-0.2 rounded-full text-[10px] font-bold ${
                result?.all_passed ? "bg-emerald-950 text-emerald-400 border border-emerald-800" : "bg-rose-950 text-rose-400 border border-rose-800"
              }`}>
                {passedTests}/{totalTests}
              </span>
            </button>
          )}
        </div>

        <div className="flex items-center gap-3">
          {result && (
            <div className="flex items-center gap-2 text-neutral-400">
              <span className="flex items-center gap-1">
                <Clock className="w-3 h-3 text-neutral-500" />
                {result.duration_ms}ms
              </span>
              {result.exit_code === 0 ? (
                <span className="text-emerald-400 flex items-center gap-1">
                  <CheckCircle className="w-3 h-3" /> Exit 0
                </span>
              ) : (
                <span className="text-rose-400 flex items-center gap-1">
                  <XCircle className="w-3 h-3" /> Exit {result.exit_code}
                </span>
              )}
            </div>
          )}

          <button
            onClick={onClear}
            className="text-neutral-500 hover:text-neutral-300 transition-colors p-1"
            title="Clear Console"
          >
            <Trash2 className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>

      {/* Console Content */}
      <div className="flex-1 p-3 overflow-y-auto font-mono text-neutral-300 space-y-2">
        {isRunning ? (
          <div className="flex items-center gap-2 text-indigo-400 animate-pulse">
            <div className="w-2 h-2 rounded-full bg-indigo-500 animate-ping" />
            <span>Compiling and executing code in sandboxed isolate...</span>
          </div>
        ) : !result ? (
          <div className="text-neutral-600 italic">
            Click "Run Code" to compile and test your solution.
          </div>
        ) : activeTab === "output" ? (
          <div>
            {result.stdout && (
              <pre className="text-neutral-100 whitespace-pre-wrap leading-relaxed">
                {result.stdout}
              </pre>
            )}
            {result.stderr && (
              <pre className="text-amber-400/90 whitespace-pre-wrap leading-relaxed">
                {result.stderr}
              </pre>
            )}
            {result.error && (
              <pre className="text-rose-400 whitespace-pre-wrap font-semibold">
                {result.error}
              </pre>
            )}
            {!result.stdout && !result.stderr && !result.error && (
              <div className="text-neutral-500 italic">Program executed with empty output.</div>
            )}
          </div>
        ) : (
          <div className="space-y-2">
            {result.test_results?.map((tr, idx) => (
              <div
                key={idx}
                className={`p-2 rounded border ${
                  tr.passed
                    ? "bg-emerald-950/20 border-emerald-800/40 text-emerald-300"
                    : "bg-rose-950/20 border-rose-800/40 text-rose-300"
                }`}
              >
                <div className="flex items-center justify-between font-bold mb-1">
                  <span>Case {idx + 1}: {tr.passed ? "Passed" : "Failed"}</span>
                  <span className="text-neutral-500 text-[11px]">{tr.duration_ms}ms</span>
                </div>
                <div className="text-[11px] text-neutral-400">
                  <div>Input: <code className="text-neutral-200">{tr.test_case.input.trim() || "(empty)"}</code></div>
                  <div>Expected: <code className="text-neutral-200">{tr.test_case.expected_output.trim()}</code></div>
                  <div>Actual: <code className="text-neutral-200">{tr.actual_output.trim() || "(empty)"}</code></div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
