export const MIN_CANDIDATE_QA_LIMIT = 0
export const MAX_CANDIDATE_QA_LIMIT = 50

// G11/J13: the org-level candidate Q&A cap is an integer in [0, 50]. Clamp
// non-integer input to a whole number; NaN means "no meaningful value" — fall
// back to the minimum so an empty/garbage field never saves a runaway cap.
export function clampQaLimit(value: number): number {
  if (Number.isNaN(value)) return MIN_CANDIDATE_QA_LIMIT
  if (value === Infinity) return MAX_CANDIDATE_QA_LIMIT
  if (value === -Infinity) return MIN_CANDIDATE_QA_LIMIT
  return Math.min(
    MAX_CANDIDATE_QA_LIMIT,
    Math.max(MIN_CANDIDATE_QA_LIMIT, Math.trunc(value)),
  )
}

// Validation for user-typed input (J13): only a bare integer inside [0, 50]
// is accepted — decimals, negatives, signs, and overflow are all rejected so
// the save button never sends a value the backend would refuse.
export function parseQaLimitInput(raw: string): number | null {
  const trimmed = raw.trim()
  if (!/^\d+$/.test(trimmed)) return null
  const n = Number(trimmed)
  if (n < MIN_CANDIDATE_QA_LIMIT || n > MAX_CANDIDATE_QA_LIMIT) return null
  return n
}
