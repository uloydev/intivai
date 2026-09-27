import { useEffect, useRef, useState } from "react"
import Editor from "@monaco-editor/react"
import { Play, RotateCcw, Code2, Check, ChevronDown } from "lucide-react"
import { cn } from "@/lib/utils"
import type { SandboxLanguage } from "@/types/api"
import { toast } from "sonner"
import { STARTER_TEMPLATES } from "./starter-templates"

interface CodeEditorProps {
  language: SandboxLanguage
  code: string
  onChange: (value: string) => void
  onLanguageChange: (lang: SandboxLanguage) => void
  onRun: () => void
  onAskAIReview?: () => void
  isRunning: boolean
  readOnly?: boolean
}

export function CodeEditor({
  language,
  code,
  onChange,
  onLanguageChange,
  onRun,
  onAskAIReview,
  isRunning,
  readOnly = false,
}: CodeEditorProps) {
  const [copied, setCopied] = useState(false)
  const [confirmReset, setConfirmReset] = useState(false)
  const resetTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const copiedTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    return () => {
      if (resetTimerRef.current) clearTimeout(resetTimerRef.current)
      if (copiedTimerRef.current) clearTimeout(copiedTimerRef.current)
    }
  }, [])

  const handleReset = () => {
    if (confirmReset) {
      onChange(STARTER_TEMPLATES[language] || "")
      setConfirmReset(false)
    } else {
      setConfirmReset(true)
      if (resetTimerRef.current) clearTimeout(resetTimerRef.current)
      resetTimerRef.current = setTimeout(() => {
        setConfirmReset(false)
        resetTimerRef.current = null
      }, 3000)
    }
  }

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(code)
      setCopied(true)
      if (copiedTimerRef.current) clearTimeout(copiedTimerRef.current)
      copiedTimerRef.current = setTimeout(() => {
        setCopied(false)
        copiedTimerRef.current = null
      }, 2000)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to copy code")
    }
  }

  return (
    <div className="flex flex-col h-full bg-neutral-950 border border-neutral-800 rounded-lg overflow-hidden shadow-sm">
      {/* Editor Top Toolbar */}
      <div className="flex items-center justify-between px-3 py-2 bg-neutral-900 border-b border-neutral-800">
        <div className="flex items-center gap-2">
          {/* Language Selector */}
          <div className="relative inline-block">
            <select
              value={language}
              disabled={readOnly}
              onChange={(e) => onLanguageChange(e.target.value as SandboxLanguage)}
              className="appearance-none bg-neutral-800 hover:bg-neutral-700 text-neutral-100 text-xs font-semibold px-3 py-1.5 pr-7 rounded border border-neutral-700 focus:outline-none focus:border-indigo-500 cursor-pointer"
            >
              <option value="go">Go 1.26</option>
              <option value="python">Python 3.12</option>
              <option value="typescript">TypeScript</option>
              <option value="javascript">JavaScript / Node</option>
            </select>
            <ChevronDown className="w-3.5 h-3.5 text-neutral-400 absolute right-2 top-2 pointer-events-none" />
          </div>

          {!readOnly && (
            <button
              onClick={handleReset}
              className="flex items-center gap-1 text-xs text-neutral-400 hover:text-neutral-200 px-2 py-1 rounded hover:bg-neutral-800 transition-colors"
              title="Reset code template"
            >
              <RotateCcw className="w-3 h-3" />
              <span>{confirmReset ? "Click again to confirm" : "Reset"}</span>
            </button>
          )}
        </div>

        <div className="flex items-center gap-2">
          {onAskAIReview && !readOnly && (
            <button
              onClick={onAskAIReview}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium text-foreground bg-muted/60 hover:bg-muted border border-border/80 transition-all shadow-sm"
              title="Request Algorithmic & Complexity Feedback"
            >
              <Code2 className="w-3.5 h-3.5 text-primary" />
              <span>Code Review</span>
            </button>
          )}

          {!readOnly && (
            <button
              onClick={onRun}
              disabled={isRunning}
              className={cn(
                "flex items-center gap-1.5 px-4 py-1.5 rounded text-xs font-bold text-white transition-all shadow-sm",
                isRunning
                  ? "bg-emerald-800 opacity-60 cursor-not-allowed"
                  : "bg-emerald-600 hover:bg-emerald-500 active:scale-95"
              )}
            >
              <Play className={cn("w-3.5 h-3.5 fill-current", isRunning && "animate-spin motion-reduce:animate-none")} />
              <span>{isRunning ? "Running..." : "Run & Test"}</span>
            </button>
          )}

          {readOnly && (
            <button
              onClick={handleCopy}
              className="flex items-center gap-1 text-xs text-neutral-400 hover:text-neutral-200 px-2.5 py-1 rounded hover:bg-neutral-800 transition-colors"
            >
              {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : null}
              <span>{copied ? "Copied!" : "Copy Code"}</span>
            </button>
          )}
        </div>
      </div>

      {/* Monaco Editor Core */}
      <div className="flex-1 min-h-0 bg-[#1e1e1e]">
        <Editor
          height="100%"
          language={language}
          value={code}
          theme="vs-dark"
          options={{
            readOnly,
            fontSize: 13,
            fontFamily: "'Fira Code', 'Geist Mono', Consolas, Monaco, monospace",
            minimap: { enabled: false },
            scrollBeyondLastLine: false,
            automaticLayout: true,
            tabSize: language === "python" ? 4 : 2,
            wordWrap: "on",
            lineNumbers: "on",
            cursorBlinking: "smooth",
            smoothScrolling: true,
          }}
          onChange={(val) => onChange(val || "")}
        />
      </div>
    </div>
  )
}
