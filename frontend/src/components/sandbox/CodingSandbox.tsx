import { useState } from "react"
import { CodeEditor, STARTER_TEMPLATES } from "./CodeEditor"
import { TerminalConsole } from "./TerminalConsole"
import { TestCaseManager } from "./TestCaseManager"
import { Sparkles, Activity, Layers } from "lucide-react"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { cn } from "@/lib/utils"
import type { SandboxLanguage, SandboxTestCase, SandboxExecutionResult, AICodeReview } from "@/types/api"

interface CodingSandboxProps {
  initialLanguage?: SandboxLanguage
  initialCode?: string
  questionIdx?: number
  onCodeChange?: (language: SandboxLanguage, code: string) => void
  onExecute: (language: SandboxLanguage, code: string, testCases: SandboxTestCase[]) => Promise<SandboxExecutionResult>
  onRequestAIReview?: (language: SandboxLanguage, code: string) => Promise<AICodeReview>
  readOnly?: boolean
}

export function CodingSandbox({
  initialLanguage = "go",
  initialCode,
  onCodeChange,
  onExecute,
  onRequestAIReview,
  readOnly = false,
}: CodingSandboxProps) {
  const [language, setLanguage] = useState<SandboxLanguage>(initialLanguage)
  const [code, setCode] = useState<string>(initialCode || STARTER_TEMPLATES[initialLanguage] || "")
  const [testCases, setTestCases] = useState<SandboxTestCase[]>([
    { id: "1", input: "example input", expected_output: "example input" },
  ])
  const [result, setResult] = useState<SandboxExecutionResult | null>(null)
  const [isRunning, setIsRunning] = useState<boolean>(false)
  const [aiReview, setAiReview] = useState<AICodeReview | null>(null)
  const [isReviewing, setIsReviewing] = useState<boolean>(false)
  const [showReviewModal, setShowReviewModal] = useState<boolean>(false)
  const [bottomTab, setBottomTab] = useState<"terminal" | "tests">("terminal")

  const handleLanguageChange = (newLang: SandboxLanguage) => {
    setLanguage(newLang)
    const newCode = STARTER_TEMPLATES[newLang] || ""
    setCode(newCode)
    onCodeChange?.(newLang, newCode)
  }

  const handleCodeChange = (newCode: string) => {
    setCode(newCode)
    onCodeChange?.(language, newCode)
  }

  const handleRun = async () => {
    setIsRunning(true)
    try {
      const res = await onExecute(language, code, testCases)
      setResult(res)
      setBottomTab("terminal")
    } catch (err) {
      setResult({
        stdout: "",
        stderr: "",
        exit_code: 1,
        duration_ms: 0,
        all_passed: false,
        error: String(err),
      })
    } finally {
      setIsRunning(false)
    }
  }

  const handleAIReview = async () => {
    if (!onRequestAIReview) return
    setIsReviewing(true)
    try {
      const review = await onRequestAIReview(language, code)
      setAiReview(review)
      setShowReviewModal(true)
    } catch (err) {
      // UX stays unchanged — the AI review button just fails silently;
      // log with context so failures are visible in devtools.
      console.error("AI code review request failed", err)
    } finally {
      setIsReviewing(false)
    }
  }

  return (
    <div className="flex flex-col h-full bg-neutral-950 text-neutral-200">
      {/* Top Split Area: Monaco Code Editor */}
      <div className="h-[60%] min-h-[220px] p-2">
        <CodeEditor
          language={language}
          code={code}
          onChange={handleCodeChange}
          onLanguageChange={handleLanguageChange}
          onRun={handleRun}
          onAskAIReview={onRequestAIReview ? handleAIReview : undefined}
          isRunning={isRunning || isReviewing}
          readOnly={readOnly}
        />
      </div>

      {/* Bottom Area: Tabs for Terminal Console and Test Cases */}
      <div className="h-[40%] min-h-[160px] p-2 pt-0 flex flex-col">
        <div className="flex items-center justify-between mb-1">
          <div className="flex items-center gap-1">
            <button
              onClick={() => setBottomTab("terminal")}
              className={cn(
                "px-3 py-1 rounded text-xs font-semibold transition-colors",
                bottomTab === "terminal"
                  ? "bg-neutral-800 text-neutral-100"
                  : "text-neutral-500 hover:text-neutral-300"
              )}
            >
              Terminal Console
            </button>
            <button
              onClick={() => setBottomTab("tests")}
              className={cn(
                "px-3 py-1 rounded text-xs font-semibold transition-colors flex items-center gap-1.5",
                bottomTab === "tests"
                  ? "bg-neutral-800 text-neutral-100"
                  : "text-neutral-500 hover:text-neutral-300"
              )}
            >
              <span>Test Suite</span>
              <span className="text-[10px] px-1.5 py-0.2 rounded-full bg-neutral-900 border border-neutral-700 text-neutral-400">
                {testCases.length}
              </span>
            </button>
          </div>

          {aiReview && (
            <button
              onClick={() => setShowReviewModal(true)}
              className="flex items-center gap-1.5 text-xs text-purple-300 hover:text-purple-200 bg-purple-950/40 px-2 py-0.5 rounded border border-purple-800/40"
            >
              <Sparkles className="w-3 h-3 text-purple-400" />
              <span>Score: {aiReview.quality_score}/100</span>
            </button>
          )}
        </div>

        <div className="flex-1 min-h-0">
          {bottomTab === "terminal" ? (
            <TerminalConsole
              result={result}
              isRunning={isRunning}
              onClear={() => setResult(null)}
            />
          ) : (
            <TestCaseManager
              testCases={testCases}
              results={result?.test_results}
              onUpdateTestCases={setTestCases}
            />
          )}
        </div>
      </div>

      {/* AI Code Review Modal */}
      <Dialog open={showReviewModal && !!aiReview} onOpenChange={setShowReviewModal}>
        <DialogContent className="bg-neutral-900 border-purple-900/60 max-w-lg w-full p-5 shadow-2xl">
          {aiReview && (
            <div className="space-y-4">
              <DialogHeader className="border-b border-neutral-800 pb-3">
                <div className="flex items-center gap-2">
                  <div className="p-1.5 rounded-lg bg-purple-950 border border-purple-800 text-purple-400">
                    <Sparkles className="w-5 h-5" />
                  </div>
                  <div>
                    <DialogTitle className="font-bold text-neutral-100 text-sm">AI Algorithmic Code Review</DialogTitle>
                    <p className="text-xs text-neutral-400 mt-1">Quality & complexity inspection</p>
                  </div>
                </div>
              </DialogHeader>

              <div className="grid grid-cols-3 gap-2 text-center">
                <div className="bg-neutral-950 p-2.5 rounded-lg border border-neutral-800">
                  <div className="text-[10px] text-neutral-400 flex items-center justify-center gap-1">
                    <Activity className="w-3 h-3 text-indigo-400" />
                    Time Complexity
                  </div>
                  <div className="font-mono font-bold text-indigo-400 mt-1">{aiReview.time_complexity}</div>
                </div>
                <div className="bg-neutral-950 p-2.5 rounded-lg border border-neutral-800">
                  <div className="text-[10px] text-neutral-400 flex items-center justify-center gap-1">
                    <Layers className="w-3 h-3 text-emerald-400" />
                    Space Complexity
                  </div>
                  <div className="font-mono font-bold text-emerald-400 mt-1">{aiReview.space_complexity}</div>
                </div>
                <div className="bg-neutral-950 p-2.5 rounded-lg border border-neutral-800">
                  <div className="text-[10px] text-neutral-400">Quality Score</div>
                  <div className="font-bold text-purple-400 text-lg mt-0.5">{aiReview.quality_score}/100</div>
                </div>
              </div>

              <div>
                <div className="text-xs font-semibold text-neutral-300 mb-1">Summary</div>
                <p className="text-xs text-neutral-400 bg-neutral-950 p-2.5 rounded border border-neutral-800 leading-relaxed">
                  {aiReview.summary}
                </p>
              </div>

              {aiReview.strengths?.length > 0 && (
                <div>
                  <div className="text-xs font-semibold text-emerald-400 mb-1">Strengths</div>
                  <ul className="text-xs text-neutral-300 space-y-1 list-disc list-inside bg-emerald-950/20 p-2.5 rounded border border-emerald-900/40">
                    {aiReview.strengths.map((s, idx) => (
                      <li key={idx}>{s}</li>
                    ))}
                  </ul>
                </div>
              )}

              {aiReview.improvements?.length > 0 && (
                <div>
                  <div className="text-xs font-semibold text-amber-400 mb-1">Suggested Improvements</div>
                  <ul className="text-xs text-neutral-300 space-y-1 list-disc list-inside bg-amber-950/20 p-2.5 rounded border border-amber-900/40">
                    {aiReview.improvements.map((s, idx) => (
                      <li key={idx}>{s}</li>
                    ))}
                  </ul>
                </div>
              )}

              <button
                onClick={() => setShowReviewModal(false)}
                className="w-full py-2 bg-neutral-800 hover:bg-neutral-750 text-neutral-100 rounded-lg text-xs font-semibold transition-colors mt-2"
              >
                Close Review
              </button>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
