import React, { useState } from "react"
import { Plus, Trash2, CheckCircle2, XCircle, Clock } from "lucide-react"
import type { SandboxTestCase, SandboxTestCaseResult } from "@/types/api"

interface TestCaseManagerProps {
  testCases: SandboxTestCase[]
  results?: SandboxTestCaseResult[]
  onUpdateTestCases: (testCases: SandboxTestCase[]) => void
}

export function TestCaseManager({ testCases, results, onUpdateTestCases }: TestCaseManagerProps) {
  const [activeTab, setActiveTab] = useState<number>(0)

  const addTestCase = () => {
    const newCase: SandboxTestCase = {
      id: String(testCases.length + 1),
      input: "",
      expected_output: "",
    }
    const updated = [...testCases, newCase]
    onUpdateTestCases(updated)
    setActiveTab(updated.length - 1)
  }

  const removeTestCase = (idx: number, e: React.MouseEvent) => {
    e.stopPropagation()
    if (testCases.length <= 1) return
    const updated = testCases.filter((_, i) => i !== idx)
    onUpdateTestCases(updated)
    if (activeTab >= updated.length) {
      setActiveTab(Math.max(0, updated.length - 1))
    }
  }

  const updateCurrent = (field: "input" | "expected_output", value: string) => {
    const updated = [...testCases]
    if (updated[activeTab]) {
      updated[activeTab] = { ...updated[activeTab], [field]: value }
      onUpdateTestCases(updated)
    }
  }

  const currentCase = testCases[activeTab]
  const currentResult = results?.find((r) => r.test_case.id === currentCase?.id)

  return (
    <div className="flex flex-col h-full bg-neutral-900 border border-neutral-800 rounded-lg overflow-hidden text-xs">
      {/* Test Cases Tab Bar */}
      <div className="flex items-center gap-1 p-1.5 bg-neutral-950 border-b border-neutral-800 overflow-x-auto">
        {testCases.map((tc, idx) => {
          const res = results?.find((r) => r.test_case.id === tc.id)
          return (
            <button
              key={tc.id || idx}
              onClick={() => setActiveTab(idx)}
              className={`flex items-center gap-1.5 px-2.5 py-1 rounded transition-colors ${
                activeTab === idx
                  ? "bg-neutral-800 text-white font-medium shadow-sm"
                  : "text-neutral-400 hover:text-neutral-200 hover:bg-neutral-900"
              }`}
            >
              {res ? (
                res.passed ? (
                  <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400 shrink-0" />
                ) : (
                  <XCircle className="w-3.5 h-3.5 text-rose-400 shrink-0" />
                )
              ) : (
                <span className="w-2 h-2 rounded-full bg-neutral-600 shrink-0" />
              )}
              <span>Case {idx + 1}</span>
              {testCases.length > 1 && (
                <span
                  onClick={(e) => removeTestCase(idx, e)}
                  className="hover:text-rose-400 p-0.5 rounded ml-0.5 text-neutral-500"
                >
                  <Trash2 className="w-3 h-3" />
                </span>
              )}
            </button>
          )
        })}
        <button
          onClick={addTestCase}
          className="flex items-center gap-1 px-2 py-1 text-neutral-400 hover:text-emerald-400 hover:bg-neutral-900 rounded transition-colors ml-auto"
          title="Add Test Case"
        >
          <Plus className="w-3.5 h-3.5" />
          <span>Add</span>
        </button>
      </div>

      {/* Test Case Editor Panel */}
      {currentCase && (
        <div className="p-3 flex-1 overflow-y-auto space-y-3">
          <div>
            <div className="text-neutral-400 font-medium mb-1">Standard Input (stdin):</div>
            <textarea
              value={currentCase.input}
              onChange={(e) => updateCurrent("input", e.target.value)}
              placeholder="e.g. 2 3"
              rows={2}
              className="w-full font-mono bg-neutral-950 border border-neutral-800 rounded p-2 text-neutral-200 focus:outline-none focus:border-indigo-500 transition-colors"
            />
          </div>

          <div>
            <div className="text-neutral-400 font-medium mb-1">Expected Output:</div>
            <textarea
              value={currentCase.expected_output}
              onChange={(e) => updateCurrent("expected_output", e.target.value)}
              placeholder="e.g. 5"
              rows={2}
              className="w-full font-mono bg-neutral-950 border border-neutral-800 rounded p-2 text-neutral-200 focus:outline-none focus:border-indigo-500 transition-colors"
            />
          </div>

          {currentResult && (
            <div className={`p-2.5 rounded border ${
              currentResult.passed ? "bg-emerald-950/30 border-emerald-800/60" : "bg-rose-950/30 border-rose-800/60"
            }`}>
              <div className="flex items-center justify-between font-semibold mb-1">
                <span className={currentResult.passed ? "text-emerald-400" : "text-rose-400"}>
                  {currentResult.passed ? "✓ Test Passed" : "✗ Test Failed"}
                </span>
                <span className="flex items-center gap-1 text-neutral-400">
                  <Clock className="w-3 h-3" />
                  {currentResult.duration_ms}ms
                </span>
              </div>
              <div className="text-neutral-400">Actual Output:</div>
              <pre className="font-mono text-neutral-200 bg-neutral-950/80 p-1.5 rounded mt-0.5 whitespace-pre-wrap">
                {currentResult.actual_output || "(empty)"}
              </pre>
              {currentResult.error && (
                <div className="text-rose-400 mt-1 font-mono text-[11px]">
                  {currentResult.error}
                </div>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
