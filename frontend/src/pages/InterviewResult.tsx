import { useState } from "react"
import { useQuery, useQueryClient, useMutation } from "@tanstack/react-query"
import { useParams, Link } from "react-router-dom"
import {
  CheckCircle,
  XCircle,
  WarningCircle,
  Scales,
  ArrowLeft,
  DownloadSimple,
  ChatCircleText,
  ChatCircleDots,
  User,
} from "@phosphor-icons/react"
import { Code2 } from "lucide-react"
import { CodeEditor } from "@/components/sandbox/CodeEditor"
import { ProctoringCard } from "@/components/interview/ProctoringCard"
import { RecommendationBadge } from "@/components/ui/RecommendationBadge"
import { api } from "@/lib/api"
import type { InterviewDetail } from "@/types/api"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { toast } from "sonner"

export function InterviewResultPage() {
  const { id } = useParams<{ id: string }>()
  const qc = useQueryClient()
  const [exporting, setExporting] = useState(false)
  const [showRejectConfirm, setShowRejectConfirm] = useState(false)
  const [showOverrideModal, setShowOverrideModal] = useState(false)
  const [overrideRec, setOverrideRec] = useState("")
  const [overrideReason, setOverrideReason] = useState("")
  const { data: detail, isLoading, error } = useQuery({
    queryKey: ["interview", id],
    queryFn: () => api.get<InterviewDetail>(`/interviews/${id}`),
    refetchInterval: (query) =>
      query.state.data?.status !== "completed" ? 3000 : false,
  })

  // Recruiter decision actions persist the lifecycle stage on the application.
  const decision = useMutation({
    mutationFn: (stage: string) =>
      api.patch(`/applications/${detail?.application_id}`, { stage }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["applications"] })
      toast.success("Candidate status updated.")
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed to update candidate status"),
  })

  // Override AI recommendation
  const overrideDecision = useMutation({
    mutationFn: () =>
      api.put(`/interviews/${id}/decision`, { recommendation: overrideRec, reason: overrideReason }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["interview", id] })
      setShowOverrideModal(false)
      setOverrideRec("")
      setOverrideReason("")
      toast.success("Override saved")
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed to save override"),
  })

  if (isLoading) {
    return (
      <div className="space-y-4 animate-in fade-in duration-500">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-44 w-full rounded-2xl" />
        <Skeleton className="h-64 w-full rounded-2xl" />
      </div>
    )
  }

  if (error || !detail) {
    return (
      <div className="rounded-xl border border-destructive/30 bg-destructive/5 p-8 text-center space-y-3">
        <p className="text-destructive font-medium">{error instanceof Error ? error.message : "Interview not found"}</p>
        <Button asChild variant="outline" size="sm">
          <Link to="/interviews">← Back to Interviews</Link>
        </Button>
      </div>
    )
  }

  const evalReport = detail.evaluation
  const dimensionCount = evalReport ? Object.keys(evalReport.dimensions).length : 0

  return (
    <div className="space-y-6 animate-in fade-in duration-500">
      {/* Relational Breadcrumbs Navigation */}
      <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
        <Link to="/dashboard" className="hover:text-foreground transition-colors">Dashboard</Link>
        <span>/</span>
        <Link to="/jobs" className="hover:text-foreground transition-colors">Jobs</Link>
        {detail.job && (
          <>
            <span>/</span>
            <Link to={`/candidates?job_id=${detail.job.id}`} className="hover:text-foreground transition-colors">
              {detail.job.title}
            </Link>
          </>
        )}
        {detail.candidate && (
          <>
            <span>/</span>
            <Link to={`/candidates?candidate_id=${detail.candidate.id}`} className="hover:text-foreground transition-colors">
              {detail.candidate.name}
            </Link>
          </>
        )}
        <span>/</span>
        <span className="text-foreground font-medium">Evaluation Scorecard</span>
      </div>

      {/* Top Navigation Bar */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between border-b border-border/60 pb-4">
        <div className="flex items-center gap-3">
          <Button asChild variant="ghost" size="icon" className="h-9 w-9 rounded-full">
            <Link to="/interviews">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div>
            <div className="flex items-center gap-2">
              <h1 className="font-display text-2xl font-bold tracking-tight">
                {detail.candidate?.name ?? "Candidate Evaluation"}
              </h1>
              <Badge variant="secondary" className="text-xs">
                {detail.status}
              </Badge>
            </div>
            <p className="text-xs text-muted-foreground mt-0.5">
              Role: <span className="font-medium text-foreground">{detail.job?.title ?? "Position"}</span> · Candidate: <Link to={`/candidates?candidate_id=${detail.candidate?.id}`} className="text-primary hover:underline">{detail.candidate?.email}</Link>
            </p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            className="text-xs gap-1.5"
            onClick={() => {
              const url = `${window.location.origin}/interviews/${id}`
              // G11: clipboard writes reject (permissions, non-secure origin) —
              // surface it instead of an unhandled rejection.
              navigator.clipboard
                .writeText(url)
                .then(() => toast.success("Link copied to clipboard"))
                .catch(() => toast.error("Couldn't copy the link — please copy it from the address bar."))
            }}
          >
            Share
          </Button>
          <Button
            variant="outline"
            size="sm"
            className="text-xs gap-1.5"
            disabled={exporting}
            onClick={async () => {
              try {
                setExporting(true)
                toast.info("Generating Maroto evaluation PDF…")
                const blob = await api.getBlob(`/interviews/${id}/report/pdf`)
                const url = window.URL.createObjectURL(blob)
                const a = document.createElement("a")
                a.href = url
                a.download = `intivai-scorecard-${(detail.candidate?.name || "candidate").toLowerCase().replace(/\s+/g, "-")}.pdf`
                a.click()
                window.URL.revokeObjectURL(url)
                toast.success("Scorecard PDF downloaded successfully")
              } catch (err) {
                toast.error(err instanceof Error ? err.message : "Failed to download PDF report")
              } finally {
                setExporting(false)
              }
            }}
          >
            <DownloadSimple className="h-3.5 w-3.5" /> {exporting ? "Generating…" : "Export PDF"}
          </Button>
        </div>
      </div>

      {/* Executive Scorecard Header */}
      {evalReport ? (
        <Card
          className="border-border bg-card shadow-sm"
          aria-live="polite"
        >
          <CardContent className="p-6">
            <div className="flex flex-col gap-6 md:flex-row md:items-center md:justify-between border-b border-border/50 pb-6">
              <div className="flex items-center gap-5">
                <div className="flex h-20 w-20 shrink-0 flex-col items-center justify-center rounded-2xl bg-primary/10 border border-primary/20 text-primary">
                  <span className="text-xs uppercase font-bold tracking-wider">Score</span>
                  <span className="font-display text-3xl font-extrabold">{evalReport.overall_score}</span>
                  <span className="text-xs text-muted-foreground">/ 100</span>
                </div>
                <div className="space-y-1.5">
                  <div className="flex items-center gap-2">
                    <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Hiring Verdict:</span>
                    <RecommendationBadge recommendation={evalReport.recommendation} />
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-7 text-[11px] text-xs"
                      onClick={() => setShowOverrideModal(true)}
                    >
                      Override
                    </Button>
                  </div>
                  <p className="text-xs text-muted-foreground leading-relaxed max-w-xl">
                    Weighted across {dimensionCount} dimensions — automated synthesis derived from {detail.questions.length} dynamically generated competence probes, evaluating factual depth, communication clarity, and problem-solving patterns.
                  </p>
                </div>
              </div>
            </div>

            {/* Dimension Bars */}
            <div className="mt-6 grid grid-cols-2 gap-3 sm:grid-cols-4">
              {Object.entries(evalReport.dimensions).map(([name, dim]) => {
                const score = dim.score
                const weightPct = Math.round(dim.weight * 100)
                const weightedPts = Math.round(dim.score * dim.weight)
                return (
                  <div key={name} className="rounded-xl border border-border/50 bg-background/60 p-3.5 space-y-1.5">
                    <div className="flex items-center justify-between">
                      <span className="text-xs font-semibold capitalize text-muted-foreground">
                        {name.replace(/_/g, " ")}
                      </span>
                      <span className="font-display text-sm font-bold">{score}%</span>
                    </div>
                    {/* Progress Bar */}
                    <div className="h-1.5 w-full rounded-full bg-muted overflow-hidden">
                      <div
                        className="h-full rounded-full bg-primary transition-all duration-500"
                        style={{ width: `${Math.min(score, 100)}%` }}
                      />
                    </div>
                    <div className="flex items-center justify-between font-mono text-xs text-muted-foreground">
                      <span>Weight {weightPct}%</span>
                      <span title={`${score}% × ${weightPct}% weight`}>
                        {weightedPts}/{weightPct} pts
                      </span>
                    </div>
                  </div>
                )
              })}
            </div>

            {/* Strengths & Weaknesses Grid */}
            {(evalReport.strengths.length > 0 || evalReport.weaknesses.length > 0) && (
              <div className="mt-6 grid gap-4 md:grid-cols-2">
                {evalReport.strengths.length > 0 && (
                  <div className="rounded-xl border border-success/20 bg-success/5 p-4 space-y-2">
                    <p className="text-xs font-bold uppercase tracking-wider text-success flex items-center gap-1.5">
                      <CheckCircle className="h-4 w-4" weight="fill" /> Key Strengths
                    </p>
                    <ul className="space-y-1 text-xs text-foreground/90 pl-1">
                      {evalReport.strengths.map((s, idx) => (
                        <li key={idx} className="flex items-start gap-1.5">
                          <span className="text-success font-bold">•</span>
                          <span>{s}</span>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
                {evalReport.weaknesses.length > 0 && (
                  <div className="rounded-xl border border-warning/20 bg-warning/5 p-4 space-y-2">
                    <p className="text-xs font-bold uppercase tracking-wider text-warning flex items-center gap-1.5">
                      <WarningCircle className="h-4 w-4" weight="fill" /> Areas to Watch / Growth
                    </p>
                    <ul className="space-y-1 text-xs text-foreground/90 pl-1">
                      {evalReport.weaknesses.map((w, idx) => (
                        <li key={idx} className="flex items-start gap-1.5">
                          <span className="text-warning font-bold">•</span>
                          <span>{w}</span>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>
            )}

            {/* Recruiter Decision & Notes Action Section */}
            <div className="mt-6 rounded-xl border border-border/80 bg-background/50 p-4 space-y-3">
              <div className="flex items-center justify-between">
                <span className="text-xs font-bold uppercase tracking-wider text-foreground flex items-center gap-1.5">
                  <User className="h-4 w-4 text-primary" /> Recruiter Decision & Hiring Action
                </span>
                <Badge variant="outline" className="text-xs">Hiring Committee</Badge>
              </div>

              <div className="grid gap-3 sm:grid-cols-3">
                <Button
                  variant="outline"
                  size="sm"
                  className="text-xs text-success border-success/30 hover:bg-success/10 font-bold gap-1"
                  onClick={() => decision.mutate("offer_extended")}
                  disabled={decision.isPending || !detail?.application_id}
                >
                  <CheckCircle className="h-3.5 w-3.5" weight="fill" /> Extend Offer
                </Button>
                <Button
                  asChild
                  variant="outline"
                  size="sm"
                  className="text-xs text-primary border-primary/30 hover:bg-primary/10 font-bold gap-1"
                >
                  <Link to={`/candidates?candidate_id=${detail.candidate?.id}`}>
                    Candidate 360 Profile →
                  </Link>
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  className="text-xs text-destructive border-destructive/30 hover:bg-destructive/10 font-semibold gap-1"
                  onClick={() => setShowRejectConfirm(true)}
                  disabled={decision.isPending || !detail?.application_id}
                >
                  <XCircle className="h-3.5 w-3.5" weight="fill" /> Reject Candidate
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      ) : (
        <Card className="border-border p-8 text-center space-y-2 shadow-sm">
          <Scales className="mx-auto h-8 w-8 text-primary" />
          <p className="font-display font-semibold text-base">Evaluation Synthesis Pending</p>
          <p className="text-xs text-muted-foreground max-w-md mx-auto">
            The interview has concluded. The evaluation worker is currently processing transcripts against the grading rubric.
          </p>
        </Card>
      )}

      {/* AI Proctoring & Integrity Telemetry Card */}
      <ProctoringCard summary={detail.proctoring_summary} events={detail.proctoring_events} />

      {/* Coding Sessions & Sandbox Submissions */}
      {detail.coding_sessions && detail.coding_sessions.length > 0 && (
        <Card className="border-border overflow-hidden shadow-sm">
          <div className="p-6 space-y-4">
            <div className="flex items-center justify-between border-b border-border/50 pb-3">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 font-bold">
                  <Code2 className="h-5 w-5" />
                </div>
                <div>
                  <h3 className="font-display font-bold text-base tracking-tight">AI Coding & Sandbox Submissions</h3>
                  <p className="text-xs text-muted-foreground">
                    Recorded live algorithmic solutions, execution test results, and algorithmic complexity rating
                  </p>
                </div>
              </div>
              <Badge variant="outline" className="text-xs font-mono border-indigo-500/30 text-indigo-600 dark:text-indigo-400 bg-indigo-500/5">
                {detail.coding_sessions.length} Submission{detail.coding_sessions.length > 1 ? "s" : ""}
              </Badge>
            </div>

            <div className="space-y-4">
              {detail.coding_sessions.map((session, idx) => (
                <div key={idx} className="rounded-xl border border-border/60 bg-neutral-950 p-4 space-y-3">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <Badge className="bg-neutral-800 text-neutral-200 border-neutral-700 text-xs uppercase font-mono">
                        {session.language}
                      </Badge>
                      <span className="text-xs text-neutral-400">Question {session.question_idx || idx + 1}</span>
                    </div>
                    {session.submitted_at && (
                      <span className="text-xs text-neutral-500 font-mono">
                        {new Date(session.submitted_at).toLocaleTimeString()}
                      </span>
                    )}
                  </div>

                  {/* Monaco Code Display */}
                  <div className="h-44 rounded-lg overflow-hidden border border-neutral-800">
                    <CodeEditor
                      language={session.language}
                      code={session.code}
                      onChange={() => {}}
                      onLanguageChange={() => {}}
                      onRun={() => {}}
                      isRunning={false}
                      readOnly={true}
                    />
                  </div>

                  {/* AI Complexity & Quality Analysis */}
                  {session.ai_code_review && (
                    <div className="grid grid-cols-3 gap-2 bg-neutral-900/60 p-3 rounded-lg border border-neutral-800 text-xs">
                      <div className="text-center">
                        <div className="text-xs text-neutral-400">Time Complexity</div>
                        <div className="font-mono font-bold text-indigo-400 mt-0.5">
                          {session.ai_code_review.time_complexity || "—"}
                        </div>
                      </div>
                      <div className="text-center">
                        <div className="text-xs text-neutral-400">Space Complexity</div>
                        <div className="font-mono font-bold text-emerald-400 mt-0.5">
                          {session.ai_code_review.space_complexity || "—"}
                        </div>
                      </div>
                      <div className="text-center">
                        <div className="text-xs text-neutral-400">Quality Score</div>
                        <div className="font-bold text-purple-400 mt-0.5">
                          {session.ai_code_review.quality_score != null
                            ? `${session.ai_code_review.quality_score}/100`
                            : "not evaluated"}
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        </Card>
      )}

      {/* Transcript & Per-Question Analysis */}
      <div className="space-y-4">
        <h2 className="font-display text-xl font-bold tracking-tight flex items-center gap-2">
          <ChatCircleText className="h-5 w-5 text-primary" weight="bold" /> Question & Answer Transcript Analysis
        </h2>

        {detail.answers.length === 0 ? (
          <div className="rounded-xl border border-dashed border-border p-8 text-center text-xs text-muted-foreground">
            No responses recorded for this interview session.
          </div>
        ) : (
          <div className="space-y-3">
            {detail.answers.map((answer) => {
              const question = detail.questions.find((q) => q.idx === answer.idx)
              const perQ = evalReport?.per_question.find((p) => p.question_idx === answer.idx)
              return (
                <Card key={answer.idx} className="border-border overflow-hidden shadow-sm">
                  <div className="border-b border-border/40 bg-muted/30 px-4 py-3 flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <span className="flex h-6 w-6 items-center justify-center rounded-full bg-primary/10 text-primary font-bold text-xs">
                        {answer.idx}
                      </span>
                      <span className="font-display font-semibold text-xs text-foreground">
                        {question?.content ?? `Question ${answer.idx}`}
                      </span>
                    </div>
                    {perQ && (
                      <Badge className="bg-primary/10 text-primary border-primary/20 text-xs font-bold">
                        Score: {perQ.score} / 100
                      </Badge>
                    )}
                  </div>

                  <CardContent className="p-4 space-y-3">
                    {/* Candidate Answer */}
                    <div className="rounded-xl bg-background/80 border border-border/40 p-3 text-xs leading-relaxed space-y-1">
                      <div className="flex items-center gap-1.5 text-muted-foreground font-semibold text-xs">
                        <User className="h-3.5 w-3.5" /> Candidate Response:
                      </div>
                      <p className="text-foreground pl-5">{answer.content}</p>
                    </div>

                    {/* AI Evaluator Rationale */}
                    {perQ?.rationale && (
                      <div className="rounded-xl bg-muted/40 border border-border/60 p-3 text-xs space-y-1">
                        <div className="flex items-center gap-1.5 text-foreground font-semibold text-xs">
                          <Scales className="h-3.5 w-3.5 text-primary" weight="fill" /> Evaluation Rationale:
                        </div>
                        <p className="text-muted-foreground pl-5">{perQ.rationale}</p>
                      </div>
                    )}
                  </CardContent>
                </Card>
              )
            })}
          </div>
        )}
      </div>

      {/* Full Transcript */}
      {detail.questions && detail.questions.length > 0 && (
        <Card className="border-border overflow-hidden shadow-sm">
          <div className="p-6 space-y-4">
            <div className="flex items-center gap-3 border-b border-border/50 pb-3">
              <ChatCircleText className="h-5 w-5 text-primary" />
              <h3 className="font-display font-bold text-base tracking-tight">Full Transcript</h3>
            </div>
            <div className="space-y-4">
              {detail.questions.map((q, idx) => {
                const answer = detail.answers?.find((a) => a.idx === q.idx)
                return (
                  <div key={idx} className="space-y-2">
                    <div className="flex gap-3">
                      <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary text-xs font-bold">AI</div>
                      <div className="rounded-2xl rounded-tl-sm bg-primary/5 border border-primary/20 px-4 py-2.5 text-sm max-w-[85%]">
                        {q.content}
                      </div>
                    </div>
                    {answer && (
                      <div className="flex gap-3 justify-end">
                        <div className="rounded-2xl rounded-tr-sm bg-muted/40 border border-border/50 px-4 py-2.5 text-sm max-w-[85%]">
                          {answer.content}
                        </div>
                        <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-muted text-muted-foreground text-xs font-bold">C</div>
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          </div>
        </Card>
      )}

      {/* Candidate Free-Form Q&A (B4) */}
      <Card className="border-border overflow-hidden shadow-sm">
        <div className="p-6 space-y-4">
          <div className="flex items-center gap-3 border-b border-border/50 pb-3">
            <ChatCircleDots className="h-5 w-5 text-primary" weight="fill" />
            <h3 className="font-display font-bold text-base tracking-tight">Candidate Q&amp;A</h3>
            <p className="text-xs text-muted-foreground">
              Questions the candidate asked during the interview and the answers grounded in your company context.
            </p>
          </div>
          {detail.qa_pairs && detail.qa_pairs.length > 0 ? (
            <div className="space-y-3">
              {detail.qa_pairs.map((pair, idx) => (
                <div key={idx} className="rounded-xl border border-border/50 bg-background/60 p-4 space-y-2.5">
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex items-start gap-2">
                      <User className="h-3.5 w-3.5 text-primary shrink-0 mt-0.5" />
                      <p className="text-xs sm:text-sm font-medium text-foreground">{pair.question}</p>
                    </div>
                    <span className="text-[10px] font-mono text-muted-foreground shrink-0">
                      {new Date(pair.created_at).toLocaleString()}
                    </span>
                  </div>
                  <div className="flex items-start gap-2">
                    <ChatCircleDots className="h-3.5 w-3.5 text-muted-foreground shrink-0 mt-0.5" weight="fill" />
                    <p className="text-xs sm:text-sm text-muted-foreground leading-relaxed">{pair.answer}</p>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-xs text-muted-foreground rounded-xl border border-dashed border-border p-6 text-center">
              No candidate questions asked.
            </p>
          )}
        </div>
      </Card>

      {/* Reject Confirmation AlertDialog */}
      <AlertDialog open={showRejectConfirm} onOpenChange={setShowRejectConfirm}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Reject Candidate</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to reject {detail?.candidate?.name ?? "this candidate"} for {detail?.job?.title ?? "this role"}? This closes the application.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={() => decision.mutate("rejected")}
            >
              Reject
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Override AI Recommendation Modal */}
      <AlertDialog open={showOverrideModal} onOpenChange={setShowOverrideModal}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Override AI Recommendation</AlertDialogTitle>
            <AlertDialogDescription>
              The AI recommended <strong>{evalReport?.recommendation ?? "pending"}</strong>. Provide your own assessment with a reason.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="space-y-3">
            <div className="flex gap-2">
              {["strong_hire", "hire", "consider", "reject"].map((rec) => (
                <Button
                  key={rec}
                  variant={overrideRec === rec ? "default" : "outline"}
                  size="sm"
                  className="text-xs capitalize"
                  onClick={() => setOverrideRec(rec)}
                >
                  {rec.replace("_", " ")}
                </Button>
              ))}
            </div>
            <textarea
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm resize-none focus:outline-none focus:ring-2 focus:ring-ring/50"
              rows={3}
              placeholder="Reason for override (required)..."
              value={overrideReason}
              onChange={(e) => setOverrideReason(e.target.value)}
            />
          </div>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              disabled={!overrideRec || !overrideReason.trim() || overrideDecision.isPending}
              onClick={() => overrideDecision.mutate()}
            >
              {overrideDecision.isPending ? "Saving…" : "Save Override"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
