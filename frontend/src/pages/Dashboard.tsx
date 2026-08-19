import { useQuery } from "@tanstack/react-query"
import { Link } from "react-router-dom"
import {
  Briefcase,
  Files,
  UsersThree,
  ChatCircleText,
  TrendUp,
  Sparkle,
  Plus,
  ArrowRight,
  MicrophoneStage,
  Lightning,
} from "@phosphor-icons/react"
import { PipelineFunnel } from "@/components/recruiter/PipelineFunnel"
import { api } from "@/lib/api"
import type { Application, CVListItem, InterviewListItem, Job } from "@/types/api"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { cn } from "@/lib/utils"

export function DashboardPage() {
  const { data: jobs, isLoading: loadingJobs } = useQuery({
    queryKey: ["jobs"],
    queryFn: () => api.get<Job[]>("/jobs"),
  })

  const { data: cvs, isLoading: loadingCVs } = useQuery({
    queryKey: ["cvs"],
    queryFn: () => api.get<CVListItem[]>("/cvs"),
  })

  const { data: apps, isLoading: loadingApps } = useQuery({
    queryKey: ["applications"],
    queryFn: () => api.get<Application[]>("/applications"),
  })

  const { data: interviews, isLoading: loadingInterviews } = useQuery({
    queryKey: ["interviews"],
    queryFn: () => api.get<InterviewListItem[]>("/interviews"),
  })

  const { data: health } = useQuery({
    queryKey: ["health"],
    queryFn: async () => {
      try {
        const res = await fetch("/ready")
        return res.ok ? "healthy" : "degraded"
      } catch {
        return "offline"
      }
    },
    refetchInterval: 10000,
  })

  const activeJobs = jobs?.filter((j) => j.status === "active") ?? []
  const totalCVs = cvs?.length ?? 0
  const totalApps = apps?.length ?? 0
  const passedApps = apps?.filter((a) => a.passed_screening) ?? []
  const passRate = totalApps > 0 ? Math.round((passedApps.length / totalApps) * 100) : null
  const completedInterviews = interviews?.filter((i) => i.status === "completed") ?? []
  // Only the real AI verdict counts — never infer "strong hire" from a score
  // heuristic that the product has not defined.
  const strongHires = completedInterviews.filter((i) => i.evaluation?.recommendation === "proceed")

  return (
    <div className="space-y-8 animate-in fade-in duration-500">
      {/* Header with Quick Actions */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <div className="flex items-center gap-2">
            <h1 className="font-display text-3xl font-bold tracking-tight text-foreground">Recruitment Command Center</h1>
            <Badge variant="outline" className="gap-1 border-primary/30 bg-primary/5 text-primary text-xs py-0.5">
              <Sparkle className="h-3 w-3" weight="fill" /> AI Powered
            </Badge>
          </div>
          <p className="text-sm text-muted-foreground mt-1">
            Real-time screening intelligence, automated candidate evaluations, and voice interview telemetry.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2.5">
          <Button asChild variant="outline" size="sm" className="rounded-lg shadow-sm">
            <Link to="/cvs">
              <Plus className="mr-1.5 h-4 w-4" /> Upload CV
            </Link>
          </Button>
          <Button asChild variant="gradient" size="sm" className="rounded-lg shadow-md shadow-primary/20">
            <Link to="/jobs">
              <Briefcase className="mr-1.5 h-4 w-4" /> Post New Job
            </Link>
          </Button>
        </div>
      </div>

      {/* Recruitment Pipeline Velocity Funnel (Head of HR Overview) */}
      <PipelineFunnel
        totalApplied={totalApps}
        totalScreened={passedApps.length}
        totalInterviewed={completedInterviews.length}
        totalRecommended={strongHires.length}
      />

      {/* KPI Metrics Grid */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {/* Metric 1: Active Jobs */}
        <Card className="glass relative overflow-hidden border-border/60 transition-all hover:border-primary/40 hover:shadow-lg hover:shadow-primary/5">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Active Roles</CardTitle>
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-primary">
              <Briefcase className="h-4 w-4" weight="bold" />
            </div>
          </CardHeader>
          <CardContent>
            {loadingJobs ? (
              <Skeleton className="h-8 w-16" />
            ) : (
              <div className="flex items-baseline justify-between">
                <span className="font-display text-3xl font-bold">{activeJobs.length}</span>
                <span className="text-xs font-medium text-muted-foreground">{jobs?.length ?? 0} total</span>
              </div>
            )}
            <p className="mt-1 text-xs text-muted-foreground">Positions actively accepting applicants</p>
          </CardContent>
        </Card>

        {/* Metric 2: CVs Ingested */}
        <Card className="glass relative overflow-hidden border-border/60 transition-all hover:border-primary/40 hover:shadow-lg hover:shadow-primary/5">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">CVs Ingested</CardTitle>
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-blue-500/10 text-blue-500">
              <Files className="h-4 w-4" weight="bold" />
            </div>
          </CardHeader>
          <CardContent>
            {loadingCVs ? (
              <Skeleton className="h-8 w-16" />
            ) : (
              <div className="flex items-baseline justify-between">
                <span className="font-display text-3xl font-bold">{totalCVs}</span>
                <Badge variant="secondary" className="bg-blue-500/10 text-blue-600 dark:text-blue-400 text-[10px]">
                  OCR + Parsed
                </Badge>
              </div>
            )}
            <p className="mt-1 text-xs text-muted-foreground">Auto-extracted candidate profiles</p>
          </CardContent>
        </Card>

        {/* Metric 3: Screening Pass Rate */}
        <Card className="glass relative overflow-hidden border-border/60 transition-all hover:border-primary/40 hover:shadow-lg hover:shadow-primary/5">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Screening Pass Rate (all roles)</CardTitle>
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-emerald-500/10 text-emerald-500">
              <TrendUp className="h-4 w-4" weight="bold" />
            </div>
          </CardHeader>
          <CardContent>
            {loadingApps ? (
              <Skeleton className="h-8 w-16" />
            ) : (
              <div className="flex items-baseline justify-between">
                <span className="font-display text-3xl font-bold text-emerald-600 dark:text-emerald-400">
                  {passRate !== null ? `${passRate}%` : "—"}
                </span>
                {passRate !== null && (
                  <span className="text-xs font-medium text-muted-foreground">{passedApps.length}/{totalApps} passed</span>
                )}
              </div>
            )}
            <p className="mt-1 text-xs text-muted-foreground">Match threshold benchmark</p>
          </CardContent>
        </Card>

        {/* Metric 4: AI Interviews */}
        <Card className="glass relative overflow-hidden border-border/60 transition-all hover:border-primary/40 hover:shadow-lg hover:shadow-primary/5">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Interviews Run</CardTitle>
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-500/10 text-indigo-500">
              <ChatCircleText className="h-4 w-4" weight="bold" />
            </div>
          </CardHeader>
          <CardContent>
            {loadingInterviews ? (
              <Skeleton className="h-8 w-16" />
            ) : (
              <div className="flex items-baseline justify-between">
                <span className="font-display text-3xl font-bold">{interviews?.length ?? 0}</span>
                <Badge variant="secondary" className="bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 text-[10px]">
                  {completedInterviews.length} Evaluated
                </Badge>
              </div>
            )}
            <p className="mt-1 text-xs text-muted-foreground">Real-time chat & voice sessions</p>
          </CardContent>
        </Card>
      </div>

      {/* Main Grid: Pipeline Activity + System Health */}
      <div className="grid gap-6 lg:grid-cols-3">
        {/* Left 2 Cols: Active Job Openings & Screened Candidates */}
        <div className="space-y-6 lg:col-span-2">
          {/* Active Job Openings */}
          <Card className="glass border-border/60">
            <CardHeader className="flex flex-row items-center justify-between">
              <div>
                <CardTitle className="font-display text-lg">Active Job Roles</CardTitle>
                <CardDescription>Open positions with AI skill rails & screening weights</CardDescription>
              </div>
              <Button asChild variant="ghost" size="sm" className="gap-1 text-primary">
                <Link to="/jobs">
                  View all <ArrowRight className="h-3.5 w-3.5" />
                </Link>
              </Button>
            </CardHeader>
            <CardContent>
              {loadingJobs ? (
                <div className="space-y-2">
                  <Skeleton className="h-14 w-full" />
                  <Skeleton className="h-14 w-full" />
                </div>
              ) : activeJobs.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-8 text-center">
                  <Briefcase className="h-10 w-10 text-muted-foreground/40 mb-2" />
                  <p className="font-medium text-sm">No active roles</p>
                  <p className="text-xs text-muted-foreground mt-0.5">Post a job to begin matching candidates</p>
                </div>
              ) : (
                <div className="space-y-2.5">
                  {activeJobs.slice(0, 3).map((job) => {
                    const jobApps = (apps ?? []).filter((a) => a.job_id === job.id)
                    return (
                      <div
                        key={job.id}
                        className="flex items-center justify-between rounded-xl border border-border/50 bg-background/50 p-3.5 transition-all hover:bg-muted/40 hover:border-primary/30"
                      >
                        <div className="min-w-0 pr-4">
                          <Link
                            to={`/candidates?job_id=${job.id}`}
                            className="font-display font-semibold text-sm truncate text-foreground hover:text-primary transition-colors inline-block"
                          >
                            {job.title}
                          </Link>
                          <div className="mt-1 flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground">
                            {(job.required_skills ?? []).slice(0, 3).map((skill) => (
                              <span key={skill} className="rounded-md bg-muted px-1.5 py-0.5 text-[10px] font-medium">
                                {skill}
                              </span>
                            ))}
                            {(job.required_skills ?? []).length > 3 && (
                              <span className="text-[10px] text-muted-foreground">+{(job.required_skills ?? []).length - 3} more</span>
                            )}
                            <span>· {job.min_experience}+ yrs exp</span>
                          </div>
                        </div>
                        <Button asChild variant="outline" size="sm" className="shrink-0 h-8 text-xs gap-1">
                          <Link to={`/candidates?job_id=${job.id}`}>
                            Applicants ({jobApps.length}) →
                          </Link>
                        </Button>
                      </div>
                    )
                  })}
                </div>
              )}
            </CardContent>
          </Card>

          {/* Recent Candidates & Screening */}
          <Card className="glass border-border/60">
            <CardHeader className="flex flex-row items-center justify-between">
              <div>
                <CardTitle className="font-display text-lg">Candidate Screening Feed</CardTitle>
                <CardDescription>Latest scored applications and status</CardDescription>
              </div>
              <Button asChild variant="ghost" size="sm" className="gap-1 text-primary">
                <Link to="/candidates">
                  View all <ArrowRight className="h-3.5 w-3.5" />
                </Link>
              </Button>
            </CardHeader>
            <CardContent>
              {loadingApps ? (
                <div className="space-y-2">
                  <Skeleton className="h-12 w-full" />
                  <Skeleton className="h-12 w-full" />
                </div>
              ) : !apps?.length ? (
                <div className="flex flex-col items-center justify-center py-8 text-center">
                  <UsersThree className="h-10 w-10 text-muted-foreground/40 mb-2" />
                  <p className="font-medium text-sm">No applications scored yet</p>
                  <p className="text-xs text-muted-foreground mt-0.5">Upload a CV in the CVs tab to screen candidates</p>
                </div>
              ) : (
                <div className="space-y-2.5">
                  {apps.slice(0, 4).map((app) => (
                    <div
                      key={app.id}
                      className="flex items-center justify-between rounded-xl border border-border/50 bg-background/50 p-3 transition-all hover:bg-muted/40"
                    >
                      <div className="min-w-0 pr-2">
                        <p className="font-medium text-sm truncate">{app.candidate_name}</p>
                        <p className="text-xs text-muted-foreground truncate">{app.job_title}</p>
                      </div>
                      <div className="flex items-center gap-2">
                        {app.cv_score != null ? (
                          <Badge
                            className={
                              app.passed_screening
                                ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20"
                                : "bg-destructive/10 text-destructive border-destructive/20"
                            }
                          >
                            {app.cv_score}% Match
                          </Badge>
                        ) : (
                          <Badge variant="secondary" className="text-xs">
                            Scoring…
                          </Badge>
                        )}
                        {app.passed_screening && (
                          <Button asChild size="sm" variant="ghost" className="h-7 text-xs text-primary">
                            <Link to={`/interviews?invite=${encodeURIComponent(app.candidate_id)}`}>Invite →</Link>
                          </Button>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Right 1 Col: Quick Mode Launchers & Telemetry */}
        <div className="space-y-6">
          {/* Quick Launch Interview Modes */}
          <Card className="glass border-primary/20 bg-gradient-to-br from-primary/5 via-card to-card">
            <CardHeader>
              <CardTitle className="font-display text-lg flex items-center gap-2">
                <Lightning className="h-5 w-5 text-primary" weight="fill" /> Quick Interview Modes
              </CardTitle>
              <CardDescription>Candidate test portals & live simulators</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="rounded-xl border border-primary/20 bg-primary/5 p-3.5 space-y-2">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <ChatCircleText className="h-4 w-4 text-primary" weight="bold" />
                    <span className="text-sm font-semibold">Chat Interview Mode</span>
                  </div>
                  <Badge variant="secondary" className="text-[10px]">WebSocket</Badge>
                </div>
                <p className="text-xs text-muted-foreground">
                  AI interviewer generates adaptive questions and streams token-by-token evaluation.
                </p>
              </div>

              <div className="rounded-xl border border-blue-500/20 bg-blue-500/5 p-3.5 space-y-2">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <MicrophoneStage className="h-4 w-4 text-blue-500" weight="bold" />
                    <span className="text-sm font-semibold">Voice Interview Mode</span>
                  </div>
                  <Badge variant="secondary" className="text-[10px] bg-blue-500/10 text-blue-600 dark:text-blue-400">WebRTC + Whisper</Badge>
                </div>
                <p className="text-xs text-muted-foreground">
                  Full duplex audio stream with Speech-to-Text and Edge synthesized voice output.
                </p>
              </div>

              <Button asChild variant="gradient" size="sm" className="w-full text-xs mt-1 shadow-md shadow-primary/20">
                <Link to="/interviews">Manage Interview Sessions →</Link>
              </Button>
            </CardContent>
          </Card>

          {/* System Telemetry & Health */}
          <Card className="glass border-border/60">
            <CardHeader className="pb-3">
              <CardTitle className="font-display text-sm flex items-center justify-between">
                <span>Infrastructure Telemetry</span>
                <span className="flex h-2 w-2 relative">
                  <span
                    className={cn(
                      "animate-ping absolute inline-flex h-full w-full rounded-full opacity-75",
                      health === "healthy" ? "bg-emerald-400" : "bg-amber-400"
                    )}
                  />
                  <span
                    className={cn(
                      "relative inline-flex rounded-full h-2 w-2",
                      health === "healthy" ? "bg-emerald-500" : "bg-amber-500"
                    )}
                  />
                </span>
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-2.5 text-xs">
              <div className="flex items-center justify-between py-1">
                <span className="text-muted-foreground">Application, Postgres, Redis & MinIO</span>
                {health === "offline" ? (
                  <Badge variant="outline" className="text-[10px] text-destructive border-destructive/30">
                    Offline
                  </Badge>
                ) : health === "degraded" ? (
                  <Badge variant="outline" className="text-[10px] text-amber-500 border-amber-500/30">
                    Degraded
                  </Badge>
                ) : (
                  <Badge variant="outline" className="text-[10px] text-emerald-500 border-emerald-500/30">
                    Online
                  </Badge>
                )}
              </div>
              <p className="text-[11px] text-muted-foreground leading-relaxed">
                Live readiness probe — Postgres (RLS), Redis, and MinIO are pinged every 10 seconds via /ready.
              </p>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}
