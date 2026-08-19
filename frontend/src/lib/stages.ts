import type { CandidateLifecycleStage } from "@/types/api"

export interface StageMeta {
  label: string
  color: string
}

// Single source of truth for candidate lifecycle stages (ADR-0001). The stage
// is the authoritative recruiter decision; null = undecided.
export const STAGE_LADDER: CandidateLifecycleStage[] = [
  "applied",
  "screening_passed",
  "screening_failed",
  "interview_invited",
  "interview_completed",
  "offer_extended",
  "hired",
  "rejected",
]

export const STAGE_META: Record<CandidateLifecycleStage, StageMeta> = {
  hired: { label: "Hired 🎉", color: "bg-emerald-500/10 text-emerald-500 border-emerald-500/30" },
  offer_extended: { label: "Offer Extended", color: "bg-purple-500/10 text-purple-400 border-purple-500/30" },
  interview_completed: { label: "Assessment Complete", color: "bg-blue-500/10 text-blue-400 border-blue-500/30" },
  interview_invited: { label: "Interview Invited", color: "bg-cyan-500/10 text-cyan-400 border-cyan-500/30" },
  screening_passed: { label: "Screening Passed", color: "bg-emerald-500/10 text-emerald-400 border-emerald-500/30" },
  screening_failed: { label: "Rejected", color: "bg-rose-500/10 text-rose-400 border-rose-500/30" },
  rejected: { label: "Rejected", color: "bg-rose-500/10 text-rose-400 border-rose-500/30" },
  applied: { label: "Applied", color: "bg-muted text-muted-foreground border-border" },
}

export const UNDECIDED_STAGE: StageMeta = {
  label: "Undecided",
  color: "bg-muted text-muted-foreground border-border",
}

export function stageMeta(stage: CandidateLifecycleStage | "" | null | undefined): StageMeta {
  if (stage) return STAGE_META[stage] ?? UNDECIDED_STAGE
  return UNDECIDED_STAGE
}
