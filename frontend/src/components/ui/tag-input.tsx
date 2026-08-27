import * as React from "react"
import { X } from "@phosphor-icons/react"

const ENTRY_MAX_LENGTH = 100
const SPLIT_RE = /[,\s]+/

interface TagInputProps {
  value: string[]
  onChange: (value: string[]) => void
  placeholder?: string
  max?: number
  label?: string
  hint?: string
  id?: string
  disabled?: boolean
}

function TagInput({
  value,
  onChange,
  placeholder,
  max,
  label,
  hint,
  id,
  disabled = false,
}: TagInputProps) {
  const fallbackId = React.useId()
  const inputId = id ?? fallbackId
  const [draft, setDraft] = React.useState("")

  const addTags = (raw: string) => {
    const tags = raw
      .split(SPLIT_RE)
      .map((t) => t.trim())
      .filter((t) => t.length > 0 && t.length <= ENTRY_MAX_LENGTH)
    if (tags.length === 0) return
    const deduped = tags.filter(
      (t) => !value.some((existing) => existing.toLowerCase() === t.toLowerCase()),
    )
    const atCap = max !== undefined && value.length >= max
    if (atCap || deduped.length === 0) return
    const room = max !== undefined ? max - value.length : deduped.length
    onChange([...value, ...deduped.slice(0, room)])
    setDraft("")
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      e.preventDefault()
      addTags(draft)
    } else if (e.key === ",") {
      e.preventDefault()
      addTags(draft)
    } else if (e.key === "Backspace" && draft.length === 0 && value.length > 0) {
      e.preventDefault()
      onChange(value.slice(0, -1))
    }
  }

  const handleRemove = (tag: string) => {
    onChange(value.filter((t) => t !== tag))
  }

  return (
    <div className="space-y-1.5">
      <div
        className="flex min-h-8 w-full flex-wrap items-center gap-1.5 rounded-lg border border-input bg-transparent px-2.5 py-1.5 transition-colors focus-within:border-ring focus-within:ring-3 focus-within:ring-ring/50 disabled:cursor-not-allowed disabled:bg-input/50 dark:bg-input/30"
      >
        {value.map((tag) => (
          <span
            key={tag}
            className="inline-flex items-center gap-1 rounded-full border border-border bg-muted px-2 py-0.5 text-xs font-medium text-foreground"
          >
            {tag}
            <button
              type="button"
              aria-label={`Remove ${tag}`}
              className="cursor-pointer text-muted-foreground hover:text-foreground"
              onClick={() => handleRemove(tag)}
              disabled={disabled}
            >
              <X className="h-3 w-3" />
            </button>
          </span>
        ))}
        <input
          id={inputId}
          aria-label={label}
          className="h-6 min-w-24 flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
          value={draft}
          placeholder={placeholder}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={handleKeyDown}
          disabled={disabled}
        />
      </div>
      {hint ? <p className="text-xs text-muted-foreground">{hint}</p> : null}
    </div>
  )
}

export { TagInput }
