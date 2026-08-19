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

// Colors use the -600/dark:-400 light-theme pattern: -400 text is unreadable
// on a light background (~1.6:1), -600 keeps it legible in light mode.
export const STAGE_META: Record<CandidateLifecycleStage, StageMeta> = {
  hired: { label: "Hired", color: "bg-accent/10 text-accent border-accent/30" },
  offer_extended: { label: "Offer Extended", color: "bg-purple-500/10 text-purple-600 border-purple-500/30 dark:text-purple-400" },
  interview_completed: { label: "Assessment Complete", color: "bg-blue-500/10 text-blue-600 border-blue-500/30 dark:text-blue-400" },
  interview_invited: { label: "Interview Invited", color: "bg-cyan-500/10 text-cyan-700 border-cyan-500/30 dark:text-cyan-400" },
  screening_passed: { label: "Screening Passed", color: "bg-accent/10 text-accent border-accent/30" },
  screening_failed: { label: "Rejected", color: "bg-rose-500/10 text-rose-600 border-rose-500/30 dark:text-rose-400" },
  rejected: { label: "Rejected", color: "bg-rose-500/10 text-rose-600 border-rose-500/30 dark:text-rose-400" },
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
