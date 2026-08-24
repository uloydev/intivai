import { ApiError } from "./api"

export interface BulkResultSummary {
  succeeded: number
  failed: number
  /** Human-readable failure reasons grouped by kind, e.g. "2× invalid request (400)". */
  reasons: string[]
}

function reasonLabel(reason: unknown): string {
  if (reason instanceof ApiError) {
    if (reason.status >= 400 && reason.status < 500) {
      return `invalid request (${reason.status})`
    }
    return `server error (${reason.status})`
  }
  if (reason instanceof Error) return `${reason.message} (0)`
  return "unexpected error"
}

// Promise.allSettled masks per-item failures — this restores honest reporting:
// recruiters must see how many operations failed and why (G8).
export function summarizeBulkResults(results: readonly PromiseSettledResult<unknown>[]): BulkResultSummary {
  let succeeded = 0
  const failures: string[] = []
  for (const r of results) {
    if (r.status === "fulfilled") {
      succeeded += 1
    } else {
      failures.push(reasonLabel(r.reason))
    }
  }

  const counts = new Map<string, number>()
  for (const label of failures) {
    counts.set(label, (counts.get(label) ?? 0) + 1)
  }
  const reasons = Array.from(counts.entries()).map(([label, n]) => `${n}× ${label}`)

  return { succeeded, failed: failures.length, reasons }
}
