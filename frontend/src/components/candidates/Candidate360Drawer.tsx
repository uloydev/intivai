import { useState, useEffect } from "react"
import { Link } from "react-router-dom"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import {
  X,
  EnvelopeSimple,
  Briefcase,
  CheckCircle,
  XCircle,
  FileText,
  Copy,
  Sparkle,
  NotePencil,
  Trophy,
} from "@phosphor-icons/react"
import { api } from "@/lib/api"
import { copyText } from "@/lib/clipboard"
import { chatInviteUrl } from "@/lib/invites"
import { stageMeta } from "@/lib/stages"
import type { Application, CandidateLifecycleStage, CreateInterviewResult, ScreeningScoreBreakdown } from "@/types/api"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Textarea } from "@/components/ui/textarea"
import { Label } from "@/components/ui/label"
import { RecommendationBadge } from "@/components/ui/RecommendationBadge"
import { cn } from "@/lib/utils"
import { toast } from "sonner"

const SCREENING_DIMENSIONS: Array<{ key: keyof ScreeningScoreBreakdown; label: string }> = [
  { key: "skills_match", label: "Skills Match" },
  { key: "experience_years", label: "Experience" },
  { key: "semantic_match", label: "Semantic Match" },
  { key: "education", label: "Education" },
  { key: "certifications", label: "Certifications" },
]

export interface Candidate360DrawerProps {
  application: Application | null
  open: boolean
  onClose: () => void
  onStageUpdate?: (appId: string, newStage: CandidateLifecycleStage, notes?: string) => void
}

export function Candidate360Drawer({
  application,
  open,
  onClose,
  onStageUpdate,
}: Candidate360DrawerProps) {
  const qc = useQueryClient()
  const [activeTab, setActiveTab] = useState<"cv" | "assessment" | "decision">("cv")
  const [currentStage, setCurrentStage] = useState<CandidateLifecycleStage | "">("")
  const [notes, setNotes] = useState("")
  const [inviteResult, setInviteResult] = useState<CreateInterviewResult | null>(null)

  useEffect(() => {
    if (application) {
      // stage is the authoritative recruiter decision (ADR-0001): null =
      // undecided. Never guess it from cv_score/interview state.
      setCurrentStage(application.stage ?? "")
      setNotes(application.recruiter_notes ?? "")
      setInviteResult(null)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [application?.id])

  const createInterview = useMutation({
    mutationFn: () => {
      if (!application) throw new Error("No candidate application selected")
      return api.post<CreateInterviewResult>("/interviews", {
        application_id: application.id,
        question_count: 3,
      })
    },
    onSuccess: (res) => {
      setInviteResult(res)
      setCurrentStage("interview_invited")
      qc.invalidateQueries({ queryKey: ["applications"] })
      qc.invalidateQueries({ queryKey: ["interviews"] })
      toast.success("Interview session created & invite link generated!")
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed to generate interview"),
  })

  const saveDecision = useMutation({
    mutationFn: () => {
      if (!application) throw new Error("No candidate application selected")
      return api.patch(`/applications/${application.id}`, {
        stage: currentStage,
        recruiter_notes: notes,
      })
    },
    onSuccess: () => {
      if (application && currentStage !== "") onStageUpdate?.(application.id, currentStage, notes)
      qc.invalidateQueries({ queryKey: ["applications"] })
      toast.success("Candidate lifecycle stage & feedback notes updated.")
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed to save decision"),
  })

  if (!open || !application) return null

  const handleSaveDecision = () => {
    saveDecision.mutate()
  }

  const inviteLink = inviteResult
    ? chatInviteUrl(inviteResult.interview_id, inviteResult.invitation_token)
    : application.interview_id
    ? chatInviteUrl(application.interview_id)
    : ""

  // Stage badge derives from the authoritative application.stage (ADR-0001),
  // not the unsaved select draft — the header must never show a lie.
  const stageBadge = stageMeta(application.stage ?? "")

  return (
    <div className="fixed inset-0 z-50 flex justify-end bg-background/80 backdrop-blur-sm animate-in fade-in duration-300">
      {/* Click outside to close backdrop */}
      <div className="flex-1 cursor-pointer" onClick={onClose} />

      {/* Slide-out Drawer Panel */}
      <div className="relative flex h-full w-full max-w-2xl flex-col border-l border-border bg-card shadow-2xl animate-in slide-in-from-right duration-300">
        {/* Drawer Header */}
        <div className="flex items-start justify-between border-b border-border p-6">
          <div className="flex items-center gap-3.5">
            <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-primary/10 text-primary font-display font-bold text-lg border border-primary/20">
              {application.candidate_name ? application.candidate_name.charAt(0).toUpperCase() : "C"}
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h2 className="font-display text-xl font-bold text-foreground">
                  {application.candidate_name || "Candidate Profile"}
                </h2>
                <Badge variant="outline" className={cn("text-xs font-semibold", stageBadge.color)}>
                  {stageBadge.label}
                </Badge>
              </div>
              <div className="flex flex-wrap items-center gap-3 text-xs text-muted-foreground mt-1">
                <span className="flex items-center gap-1">
                  <EnvelopeSimple className="h-3.5 w-3.5" />
                  {application.candidate_email || "No email provided"}
                </span>
                <span>•</span>
                <span className="flex items-center gap-1 font-medium text-foreground">
                  <Briefcase className="h-3.5 w-3.5 text-primary" />
                  {application.job_title || "General Application"}
                </span>
              </div>
            </div>
          </div>

          <Button variant="ghost" size="icon" onClick={onClose} className="rounded-full">
            <X className="h-4 w-4" />
          </Button>
        </div>

        {/* Tab Navigation */}
        <div className="flex border-b border-border/80 px-6">
          <button
            onClick={() => setActiveTab("cv")}
            className={cn(
              "flex items-center gap-2 border-b-2 py-3 px-3 text-xs font-semibold transition-colors",
              activeTab === "cv"
                ? "border-primary text-primary"
                : "border-transparent text-muted-foreground hover:text-foreground"
            )}
          >
            <FileText className="h-4 w-4" />
            <span>CV & Screening Intelligence</span>
          </button>

          <button
            onClick={() => setActiveTab("assessment")}
            className={cn(
              "flex items-center gap-2 border-b-2 py-3 px-3 text-xs font-semibold transition-colors",
              activeTab === "assessment"
                ? "border-primary text-primary"
                : "border-transparent text-muted-foreground hover:text-foreground"
            )}
          >
            <Sparkle className="h-4 w-4" />
            <span>AI Assessment & Telemetry</span>
          </button>

          <button
            onClick={() => setActiveTab("decision")}
            className={cn(
              "flex items-center gap-2 border-b-2 py-3 px-3 text-xs font-semibold transition-colors",
              activeTab === "decision"
                ? "border-primary text-primary"
                : "border-transparent text-muted-foreground hover:text-foreground"
            )}
          >
            <NotePencil className="h-4 w-4" />
            <span>Hiring Decision & Notes</span>
          </button>
        </div>

        {/* Drawer Body Scrollable Content */}
        <div className="flex-1 overflow-y-auto p-6 space-y-6">
          {/* TAB 1: CV & Screening Intelligence */}
          {activeTab === "cv" && (
            <div className="space-y-6">
              {/* Screening Score Banner */}
              <div className="flex items-center justify-between rounded-xl border border-border bg-card/60 p-4">
                <div>
                  <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    AI Resume Screening Match
                  </p>
                  <div className="flex items-center gap-2 mt-1">
                    <span className="font-display text-3xl font-bold text-foreground">
                      {application.cv_score != null ? `${application.cv_score}%` : "Pending"}
                    </span>
                    {application.passed_screening ? (
                      <Badge className="bg-emerald-500/10 text-emerald-500 border-emerald-500/30 text-xs gap-1">
                        <CheckCircle className="h-3.5 w-3.5" weight="fill" /> Passed Threshold
                      </Badge>
                    ) : (
                      <Badge variant="destructive" className="text-xs gap-1">
                        <XCircle className="h-3.5 w-3.5" weight="fill" /> Below Threshold
                      </Badge>
                    )}
                  </div>
                </div>

                <div className="text-right">
                  <span className="text-xs text-muted-foreground">Relevant Experience</span>
                  <p className="font-mono text-lg font-bold text-foreground">
                    {application.years_experience != null ? `${application.years_experience}+ Years` : "-"}
                  </p>
                </div>
              </div>

              {/* Matched Skills */}
              <div className="space-y-2.5">
                <Label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                  Verified Skills from Resume
                </Label>
                <div className="flex flex-wrap gap-1.5">
                  {(Array.isArray(application.matched_skills)
                    ? application.matched_skills
                    : []
                  ).map((skill, i) => (
                    <Badge
                      key={i}
                      variant="outline"
                      className="bg-primary/5 text-primary border-primary/20 text-xs py-1 px-2.5"
                    >
                      {skill}
                    </Badge>
                  ))}
                  {(!Array.isArray(application.matched_skills) || application.matched_skills.length === 0) && (
                    <span className="text-xs text-muted-foreground">No skills extracted</span>
                  )}
                </div>
              </div>

              {/* AI Screening Rationale — real scored dimensions, no canned copy */}
              {application.score_breakdown &&
                Object.keys(application.score_breakdown).length > 0 && (
                  <div className="space-y-3 rounded-xl border border-border/70 bg-muted/20 p-4">
                    <div className="flex items-center gap-1.5 text-xs font-semibold text-foreground">
                      <Sparkle className="h-4 w-4 text-primary" weight="fill" />
                      <span>AI Screening Recommendation</span>
                    </div>
                    <div className="grid grid-cols-2 gap-2">
                      {SCREENING_DIMENSIONS.map(({ key, label }) => {
                        const value = application.score_breakdown?.[key]
                        if (value == null) return null
                        return (
                          <div key={key} className="flex items-center justify-between rounded-lg bg-background/60 border border-border/50 px-2.5 py-1.5">
                            <span className="text-[11px] text-muted-foreground">{label}</span>
                            <span className="font-mono text-xs font-bold text-foreground">
                              {Math.round(value * 100)}%
                            </span>
                          </div>
                        )
                      })}
                    </div>
                  </div>
                )}

              {/* CV Action Link */}
              <div className="border-t border-border pt-4">
                <Button asChild variant="outline" size="sm" className="w-full gap-2 text-xs">
                  <Link to={`/cvs`}>
                    <FileText className="h-4 w-4 text-muted-foreground" />
                    <span>View CV Ingestion Records in CV Hub</span>
                  </Link>
                </Button>
              </div>
            </div>
          )}

          {/* TAB 2: AI Assessment & Telemetry */}
          {activeTab === "assessment" && (
            <div className="space-y-6">
              {/* Interview Status Card */}
              <div className="rounded-xl border border-border bg-card/60 p-4 space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    Interview Assessment State
                  </span>
                  <Badge variant="outline" className={cn("text-xs font-medium", stageBadge.color)}>
                    {stageBadge.label}
                  </Badge>
                </div>

                {application.interview_score != null ? (
                  <div className="space-y-3 pt-2">
                    <div className="flex items-center justify-between border-b border-border/50 pb-3">
                      <div>
                        <p className="text-xs text-muted-foreground">Overall Assessment Score</p>
                        <p className="font-display text-3xl font-bold text-foreground">
                          {application.interview_score} / 100
                        </p>
                      </div>
                      <div className="text-right">
                        <p className="text-xs text-muted-foreground">AI Recommendation</p>
                        <RecommendationBadge recommendation={application.recommendation} className="mt-1 capitalize" />
                      </div>
                    </div>

                    <Button asChild variant="gradient" className="w-full gap-2 text-xs font-bold">
                      <Link to={`/interviews/${application.interview_id || ""}`}>
                        <Trophy className="h-4 w-4" weight="bold" />
                        <span>Open Comprehensive Scorecard & Replay →</span>
                      </Link>
                    </Button>
                  </div>
                ) : (
                  <div className="space-y-4 pt-2">
                    <p className="text-xs text-muted-foreground leading-relaxed">
                      Dispatch an interactive AI assessment session with real-time technical probing, live coding challenge, and stage timer gates.
                    </p>

                    {inviteLink ? (
                      <div className="space-y-2">
                        <Label className="text-xs font-medium">Active Candidate Invitation Link</Label>
                        <div className="flex items-center gap-2">
                          <input
                            type="text"
                            readOnly
                            value={inviteLink}
                            className="flex-1 rounded-lg border border-border bg-muted/40 px-3 py-1.5 text-xs font-mono text-muted-foreground select-all"
                          />
                          <Button
                            size="sm"
                            variant="secondary"
                            onClick={() => copyText(inviteLink, "Candidate Invite Link")}
                            className="gap-1.5 text-xs"
                          >
                            <Copy className="h-3.5 w-3.5" /> Copy
                          </Button>
                        </div>
                      </div>
                    ) : (
                      <Button
                        onClick={() => createInterview.mutate()}
                        disabled={createInterview.isPending}
                        variant="gradient"
                        size="sm"
                        className="w-full gap-2 font-semibold"
                      >
                        <Sparkle className="h-4 w-4" weight="fill" />
                        {createInterview.isPending ? "Generating..." : "Generate AI Interview Session"}
                      </Button>
                    )}
                  </div>
                )}
              </div>
            </div>
          )}

          {/* TAB 3: Hiring Decision & Notes */}
          {activeTab === "decision" && (
            <div className="space-y-5">
              {/* Lifecycle Stage Selector */}
              <div className="space-y-2">
                <Label htmlFor="stage-select" className="text-xs font-semibold">
                  Update Candidate Stage
                </Label>
                <select
                  id="stage-select"
                  value={currentStage}
                  onChange={(e) => setCurrentStage(e.target.value as CandidateLifecycleStage)}
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-xs font-medium text-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                >
                  {currentStage === "" && (
                    <option value="" disabled>
                      — Undecided —
                    </option>
                  )}
                  <option value="applied">Applied (Inbound)</option>
                  <option value="screening_passed">Screening Passed (Qualified)</option>
                  <option value="screening_failed">Screening Failed</option>
                  <option value="interview_invited">Interview Invited / Scheduled</option>
                  <option value="interview_completed">Interview Completed (Evaluation Ready)</option>
                  <option value="offer_extended">Offer Extended</option>
                  <option value="hired">Hired 🎉</option>
                  <option value="rejected">Rejected / Archived</option>
                </select>
              </div>

              {/* Recruiter Evaluation Feedback Notes */}
              <div className="space-y-2">
                <Label htmlFor="recruiter-notes" className="text-xs font-semibold">
                  Internal Hiring Committee Notes & Feedback
                </Label>
                <Textarea
                  id="recruiter-notes"
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                  placeholder="Record hiring manager feedback, compensation targets, team fit notes..."
                  rows={6}
                  className="resize-none bg-card text-xs p-3 focus-visible:ring-primary"
                />
              </div>

              <Button
                onClick={handleSaveDecision}
                disabled={saveDecision.isPending || currentStage === ""}
                variant="gradient"
                className="w-full text-xs font-bold"
              >
                {saveDecision.isPending ? "Saving..." : "Save Candidate Decision & Notes"}
              </Button>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
