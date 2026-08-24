import { useCallback, useEffect, useMemo, useState } from "react"
import { useNavigate } from "react-router-dom"
import { MagnifyingGlass } from "@phosphor-icons/react"
import {
  Dialog,
  DialogContent,
} from "@/components/ui/dialog"
import { cn } from "@/lib/utils"

interface CommandItem {
  id: string
  label: string
  section: string
  action: () => void
  keywords?: string
}

export function CommandPalette() {
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState("")
  const [selectedIdx, setSelectedIdx] = useState(0)
  const navigate = useNavigate()

  const commands: CommandItem[] = useMemo(() => [
    { id: "dash", label: "Go to Dashboard", section: "Navigation", action: () => navigate("/dashboard"), keywords: "home main" },
    { id: "jobs", label: "Go to Jobs", section: "Navigation", action: () => navigate("/jobs"), keywords: "roles positions hiring" },
    { id: "cvs", label: "Go to CV Ingestion", section: "Navigation", action: () => navigate("/cvs"), keywords: "resume upload" },
    { id: "candidates", label: "Go to Candidates", section: "Navigation", action: () => navigate("/candidates"), keywords: "people applicants" },
    { id: "interviews", label: "Go to Interviews", section: "Navigation", action: () => navigate("/interviews"), keywords: "sessions chat" },
    { id: "rails", label: "Go to AI Rails", section: "Navigation", action: () => navigate("/company-context"), keywords: "prompt config" },
    { id: "integrations", label: "Go to Integrations", section: "Navigation", action: () => navigate("/integrations"), keywords: "webhooks settings" },
    { id: "new-job", label: "Post New Job", section: "Actions", action: () => navigate("/jobs"), keywords: "create role position" },
    { id: "upload-cv", label: "Upload Resume", section: "Actions", action: () => navigate("/cvs"), keywords: "cv resume upload" },
  ], [navigate])

  const filtered = useMemo(() => {
    if (!query) return commands
    const q = query.toLowerCase()
    return commands.filter(
      (c) => c.label.toLowerCase().includes(q) || c.keywords?.includes(q) || c.section.toLowerCase().includes(q)
    )
  }, [commands, query])

  const runCommand = useCallback((cmd: CommandItem) => {
    cmd.action()
    setOpen(false)
    setQuery("")
    setSelectedIdx(0)
  }, [])

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault()
        setOpen((prev) => !prev)
      }
    }
    window.addEventListener("keydown", handler)
    return () => window.removeEventListener("keydown", handler)
  }, [])

  useEffect(() => {
    setSelectedIdx(0)
  }, [query])

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "ArrowDown") {
      e.preventDefault()
      setSelectedIdx((i) => Math.min(i + 1, filtered.length - 1))
    } else if (e.key === "ArrowUp") {
      e.preventDefault()
      setSelectedIdx((i) => Math.max(i - 1, 0))
    } else if (e.key === "Enter" && filtered[selectedIdx]) {
      e.preventDefault()
      runCommand(filtered[selectedIdx])
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="p-0 max-w-lg gap-0 overflow-hidden" onKeyDown={handleKeyDown}>
        <div className="flex items-center border-b border-border/60 px-4">
          <MagnifyingGlass className="h-4 w-4 text-muted-foreground shrink-0" />
          <input
            autoFocus
            placeholder="Type a command..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="flex-1 bg-transparent py-3 px-3 text-sm outline-none placeholder:text-muted-foreground"
          />
          <kbd className="hidden sm:inline-flex h-5 items-center gap-1 rounded border bg-muted px-1.5 font-mono text-[10px] text-muted-foreground">
            esc
          </kbd>
        </div>
        <div className="max-h-64 overflow-y-auto p-1.5">
          {filtered.length === 0 ? (
            <p className="py-6 text-center text-sm text-muted-foreground">No results found.</p>
          ) : (
            filtered.map((cmd, i) => (
              <button
                key={cmd.id}
                type="button"
                onClick={() => runCommand(cmd)}
                className={cn(
                  "flex w-full items-center rounded-md px-3 py-2 text-sm cursor-pointer transition-colors",
                  i === selectedIdx ? "bg-primary/10 text-primary" : "text-foreground hover:bg-muted"
                )}
              >
                <span className="flex-1 text-left">{cmd.label}</span>
                <span className="text-[10px] text-muted-foreground">{cmd.section}</span>
              </button>
            ))
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
