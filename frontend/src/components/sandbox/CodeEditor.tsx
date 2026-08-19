import { useState } from "react"
import Editor from "@monaco-editor/react"
import { Play, RotateCcw, Sparkles, Check, ChevronDown } from "lucide-react"
import type { SandboxLanguage } from "@/types/api"

export const STARTER_TEMPLATES: Record<SandboxLanguage, string> = {
  go: `package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Solve implements your algorithmic solution
func Solve(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	// TODO: implement logic here
	return strings.Join(lines, " ")
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	fmt.Println(Solve(lines))
}
`,
  python: `import sys

def solve():
    """Implement your algorithmic solution here"""
    lines = sys.stdin.read().strip().split()
    if not lines:
        return
    # TODO: implement logic here
    print(" ".join(lines))

if __name__ == "__main__":
    solve()
`,
  typescript: `function solve(input: string): string {
    // TODO: implement your solution
    return input.trim();
}

const readline = require("readline");
const rl = readline.createInterface({ input: process.stdin });
let lines: string[] = [];

rl.on("line", (line: string) => lines.push(line));
rl.on("close", () => {
    console.log(solve(lines.join("\\n")));
});
`,
  javascript: `function solve(input) {
    // TODO: implement your solution
    return input.trim();
}

const readline = require("readline");
const rl = readline.createInterface({ input: process.stdin });
let lines = [];

rl.on("line", (line) => lines.push(line));
rl.on("close", () => {
    console.log(solve(lines.join("\\n")));
});
`,
}

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

  const handleReset = () => {
    if (window.confirm("Reset code to starter template? Your current edits will be overwritten.")) {
      onChange(STARTER_TEMPLATES[language] || "")
    }
  }

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(code)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch (e) {
      alert("Failed to copy to clipboard")
    }
  }

  return (
    <div className="flex flex-col h-full bg-neutral-950 border border-neutral-800 rounded-lg overflow-hidden shadow-2xl">
      {/* Editor Top Toolbar */}
      <div className="flex items-center justify-between px-3 py-2 bg-neutral-900/90 backdrop-blur border-b border-neutral-800">
        <div className="flex items-center gap-2">
          {/* Language Selector */}
          <div className="relative inline-block">
            <select
              value={language}
              disabled={readOnly}
              onChange={(e) => onLanguageChange(e.target.value as SandboxLanguage)}
              className="appearance-none bg-neutral-800 hover:bg-neutral-750 text-neutral-100 text-xs font-semibold px-3 py-1.5 pr-7 rounded border border-neutral-700 focus:outline-none focus:border-indigo-500 cursor-pointer"
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
              <span>Reset</span>
            </button>
          )}
        </div>

        <div className="flex items-center gap-2">
          {onAskAIReview && !readOnly && (
            <button
              onClick={onAskAIReview}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium text-purple-300 bg-purple-950/50 hover:bg-purple-900/60 border border-purple-800/60 transition-all shadow-sm"
              title="Request AI Algorithmic & Complexity Feedback"
            >
              <Sparkles className="w-3.5 h-3.5 text-purple-400" />
              <span>AI Code Review</span>
            </button>
          )}

          {!readOnly && (
            <button
              onClick={onRun}
              disabled={isRunning}
              className={`flex items-center gap-1.5 px-4 py-1.5 rounded text-xs font-bold text-white shadow-lg transition-all ${
                isRunning
                  ? "bg-emerald-800 opacity-60 cursor-not-allowed"
                  : "bg-emerald-600 hover:bg-emerald-500 active:scale-95 shadow-emerald-950/50"
              }`}
            >
              <Play className={`w-3.5 h-3.5 fill-current ${isRunning ? "animate-spin" : ""}`} />
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
          language={language === "typescript" ? "typescript" : language === "javascript" ? "javascript" : language === "python" ? "python" : "go"}
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
