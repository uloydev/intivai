export const WEIGHT_DIMENSIONS = [
  { key: "skills_match", label: "Skills Match" },
  { key: "experience_years", label: "Experience (Years)" },
  { key: "semantic_match", label: "Semantic Match" },
  { key: "education", label: "Education" },
  { key: "certifications", label: "Certifications" },
] as const

export type WeightKey = (typeof WEIGHT_DIMENSIONS)[number]["key"]

// D10: mirrors DefaultWeights() in backend/internal/screening/domain/scoring.go
// (skills 0.35 / experience 0.20 / semantic 0.25 / education 0.10 / certifications 0.10,
// summing to 1.00). Key names are the backend weight-map contract validated in
// backend/internal/job/domain/job.go (SetScoringWeights). Keep both sides in sync.
export const DEFAULT_WEIGHT_MAP: Record<WeightKey, number> = {
  skills_match: 0.35,
  experience_years: 0.2,
  semantic_match: 0.25,
  education: 0.1,
  certifications: 0.1,
}

export function buildSaveWeights(
  useDefaults: boolean,
  current: Record<string, number>,
): Record<WeightKey, number> {
  const out = {} as Record<WeightKey, number>
  for (const d of WEIGHT_DIMENSIONS) {
    out[d.key] = useDefaults ? DEFAULT_WEIGHT_MAP[d.key] : Number(current[d.key]) || 0
  }
  return out
}

export const WEIGHT_SUM_TOLERANCE = 0.01

export function weightSum(weights: Record<string, number>): number {
  return WEIGHT_DIMENSIONS.reduce((acc, d) => acc + (Number(weights[d.key]) || 0), 0)
}

export function isWeightMapComplete(weights: Record<string, number>): boolean {
  return WEIGHT_DIMENSIONS.every(
    (d) => typeof weights[d.key] === "number" && Number.isFinite(weights[d.key]),
  )
}

export function isWeightSumValid(sum: number): boolean {
  // Small epsilon absorbs binary float error at the inclusive ±0.01 boundary
  // (e.g. 0.99 arrives as 0.9899999…1, a hair over the limit).
  return Math.abs(sum - 1) <= WEIGHT_SUM_TOLERANCE + 1e-9
}
