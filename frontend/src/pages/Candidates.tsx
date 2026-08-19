import { useState, useEffect } from "react"
import { useQuery } from "@tanstack/react-query"
import { useParams, useSearchParams } from "react-router-dom"
import {
  UsersThree,
  MagnifyingGlass,
  CheckCircle,
  XCircle,
  Briefcase,
  Eye,
} from "@phosphor-icons/react"
import { api } from "@/lib/api"
import { stageMeta } from "@/lib/stages"
import type { Application, CandidateLifecycleStage, Job } from "@/types/api"
import { Candidate360Drawer } from "@/components/candidates/Candidate360Drawer"
import { toast } from "sonner"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { cn } from "@/lib/utils"

function scorePill(app: Application) {
  if (app.cv_score == null) {
    // cv_score is null until the pipeline produces one — the pill must say
    // WHY, not guess "scoring" (pending_review blocks scoring entirely).
    switch (app.cv_status) {
      case "pending_review":
        return (
          <Badge variant="secondary" className="bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20 text-xs">
            Pending review
          </Badge>
        )
      case "failed_ocr":
      case "failed_extract":
        return (
          <Badge variant="destructive" className="font-semibold text-xs gap-1">
            <XCircle className="h-3 w-3" weight="fill" /> Extraction failed
          </Badge>
        )
      case "new":
      case "parsing":
      case "extracting":
        return (
          <Badge variant="secondary" className="bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20 animate-pulse text-xs">
            Scoring…
          </Badge>
        )
      default:
        return (
          <Badge variant="secondary" className="text-xs">
            Not scored
          </Badge>
        )
    }
  }
  if (app.passed_screening) {
    return (
      <Badge className="bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20 font-bold text-xs gap-1">
        <CheckCircle className="h-3 w-3" weight="fill" /> {app.cv_score}% Match
      </Badge>
    )
  }
  return (
    <Badge variant="destructive" className="font-semibold text-xs gap-1">
      <XCircle className="h-3 w-3" weight="fill" /> {app.cv_score}% Match
    </Badge>
  )
}

function stagePill(app: Application) {
  // stage is the authoritative recruiter decision (ADR-0001); null = undecided
  const meta = stageMeta(app.stage ?? "")
  const isCompleted = app.stage === "interview_completed"
  return (
    <Badge className={cn("text-[11px]", meta.color)}>
      {meta.label}
      {isCompleted ? ` (${app.interview_score ?? "-"}/100)` : ""}
    </Badge>
  )
}

export function CandidatesPage() {
  const { id: routeId } = useParams<{ id: string }>()
  const [searchParams, setSearchParams] = useSearchParams()
  // Filter state IS the URL — derive directly, no sync effects.
  const selectedJob = searchParams.get("job_id") ?? "all"
  const statusFilter = searchParams.get("stage") ?? "all"
  const candidateParam = searchParams.get("candidate_id") || searchParams.get("application_id") || routeId

  const { data: apps, isLoading, error } = useQuery({
    queryKey: ["applications"],
    queryFn: () => api.get<Application[]>("/applications"),
    refetchInterval: (query) =>
      query.state.data?.some((a) => a.cv_score == null) ? 2000 : false,
  })

  const { data: jobs } = useQuery({
    queryKey: ["jobs"],
    queryFn: () => api.get<Job[]>("/jobs"),
  })

  const [search, setSearch] = useState("")
  const [selectedApp, setSelectedApp] = useState<Application | null>(null)
  const [drawerOpen, setDrawerOpen] = useState(false)

  // Open drawer if candidate_id or routeId is provided
  useEffect(() => {
    if (candidateParam && apps) {
      const match = apps.find(
        (a) =>
          a.candidate_id === candidateParam ||
          a.id === candidateParam ||
          a.candidate_email === candidateParam
      )
      if (match) {
        setSelectedApp(match)
        setDrawerOpen(true)
      } else {
        toast.info("Candidate application profile not found or pending screening.")
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [candidateParam, apps])

  const handleJobChange = (jobId: string) => {
    const next = new URLSearchParams(searchParams)
    if (jobId === "all") next.delete("job_id")
    else next.set("job_id", jobId)
    setSearchParams(next)
  }

  const handleStageChange = (stage: string) => {
    const next = new URLSearchParams(searchParams)
    if (stage === "all") next.delete("stage")
    else next.set("stage", stage)
    setSearchParams(next)
  }

  const handleStageUpdate = (appId: string, newStage: CandidateLifecycleStage, notes?: string) => {
    if (selectedApp && selectedApp.id === appId) {
      setSelectedApp({
        ...selectedApp,
        stage: newStage,
        recruiter_notes: notes,
      })
    }
  }

  const filteredApps = (apps ?? []).filter((app) => {
    const name = (app.candidate_name || "").toLowerCase()
    const email = (app.candidate_email || "").toLowerCase()
    const title = (app.job_title || "").toLowerCase()
    const q = (search || "").toLowerCase()
    const matchesSearch = name.includes(q) || email.includes(q) || title.includes(q)
    const matchesJob = selectedJob === "all" || app.job_id === selectedJob
    
    let matchesStatus = true
    if (statusFilter === "passed" || statusFilter === "screening_passed") {
      matchesStatus = Boolean(app.passed_screening)
    } else if (statusFilter === "rejected" || statusFilter === "screening_failed") {
      matchesStatus = Boolean(app.cv_score != null && !app.passed_screening)
    } else if (statusFilter === "interview_completed") {
      matchesStatus = app.interview_score != null || app.status === "completed"
    } else if (statusFilter !== "all") {
      matchesStatus = app.stage === statusFilter
    }

    return matchesSearch && matchesJob && matchesStatus
  })

  return (
    <div className="space-y-6 animate-in fade-in duration-500">
      {/* Header */}
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="font-display text-3xl font-bold tracking-tight">Candidate Screening Pool</h1>
          <p className="text-sm text-muted-foreground">
            Semantic CV match rankings, qualification scoring, and interview progression.
          </p>
        </div>
      </div>

      {/* Filter Bar */}
      <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div className="flex flex-1 flex-col gap-3 sm:flex-row sm:items-center max-w-xl">
          <div className="relative w-full">
            <MagnifyingGlass className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="Search candidate name or email..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-9 bg-background/80"
            />
          </div>
          <select
            className="rounded-md border border-input bg-background px-3 py-2 text-xs shadow-sm focus:outline-none focus:ring-1 focus:ring-primary shrink-0"
            value={selectedJob}
            onChange={(e) => handleJobChange(e.target.value)}
          >
            <option value="all">All Roles</option>
            {jobs?.map((j) => (
              <option key={j.id} value={j.id}>
                {j.title}
              </option>
            ))}
          </select>
        </div>

        <div className="flex flex-wrap items-center gap-1.5 shrink-0">
          <Button
            variant={statusFilter === "all" ? "secondary" : "ghost"}
            size="sm"
            className="text-xs h-8"
            onClick={() => handleStageChange("all")}
          >
            All ({apps?.length ?? 0})
          </Button>
          <Button
            variant={statusFilter === "screening_passed" || statusFilter === "passed" ? "secondary" : "ghost"}
            size="sm"
            className="text-xs h-8 text-emerald-600 dark:text-emerald-400"
            onClick={() => handleStageChange("screening_passed")}
          >
            Passed ({apps?.filter((a) => a.passed_screening).length ?? 0})
          </Button>
          <Button
            variant={statusFilter === "interview_completed" ? "secondary" : "ghost"}
            size="sm"
            className="text-xs h-8 text-blue-500"
            onClick={() => handleStageChange("interview_completed")}
          >
            Evaluated ({apps?.filter((a) => a.interview_score != null).length ?? 0})
          </Button>
          <Button
            variant={statusFilter === "rejected" ? "secondary" : "ghost"}
            size="sm"
            className="text-xs h-8 text-destructive"
            onClick={() => handleStageChange("rejected")}
          >
            Below Gate ({apps?.filter((a) => a.cv_score != null && !a.passed_screening).length ?? 0})
          </Button>
        </div>
      </div>

      {/* Candidates Table */}
      {isLoading ? (
        <Skeleton className="h-64 w-full rounded-xl" />
      ) : error ? (
        <div className="rounded-xl border border-destructive/30 bg-destructive/5 p-6 text-center text-sm text-destructive">
          {error instanceof Error ? error.message : "Failed to load candidates"}
        </div>
      ) : filteredApps.length === 0 ? (
        <div className="rounded-2xl border border-dashed border-border/80 p-12 text-center">
          <UsersThree className="mx-auto h-12 w-12 text-muted-foreground/40 mb-3" />
          <p className="font-display font-semibold text-base">No candidates matched</p>
          <p className="text-xs text-muted-foreground mt-1 max-w-sm mx-auto">
            Upload candidate resumes in the CVs tab and screen them against target roles to see them here.
          </p>
        </div>
      ) : (
        <div className="rounded-xl border border-border/60 bg-card shadow-sm overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow className="bg-muted/40">
                <TableHead>Candidate Profile</TableHead>
                <TableHead>Target Role</TableHead>
                <TableHead>CV Match</TableHead>
                <TableHead>Talent Lifecycle Stage</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredApps.map((app) => (
                <TableRow
                  key={app.id}
                  tabIndex={0}
                  role="button"
                  className="cursor-pointer transition-colors hover:bg-muted/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60"
                  onClick={() => {
                    setSelectedApp(app)
                    setDrawerOpen(true)
                  }}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" || e.key === " ") {
                      e.preventDefault()
                      setSelectedApp(app)
                      setDrawerOpen(true)
                    }
                  }}
                >
                  <TableCell>
                    <div className="flex items-center gap-3">
                      <div className="flex h-9 w-9 items-center justify-center rounded-full bg-primary/10 text-primary font-bold text-xs">
                        {app.candidate_name ? app.candidate_name.charAt(0).toUpperCase() : "C"}
                      </div>
                      <div>
                        <p className="font-display font-semibold text-sm">{app.candidate_name || "Candidate"}</p>
                        <p className="text-xs text-muted-foreground">{app.candidate_email || "No email"}</p>
                      </div>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1.5 text-xs font-medium">
                      <Briefcase className="h-3.5 w-3.5 text-muted-foreground" />
                      <span>{app.job_title || "General Application"}</span>
                    </div>
                  </TableCell>
                  <TableCell>{scorePill(app)}</TableCell>
                  <TableCell>{stagePill(app)}</TableCell>
                  <TableCell className="text-right" onClick={(e) => e.stopPropagation()}>
                    <Button
                      size="sm"
                      variant="outline"
                      className="h-8 text-xs text-primary border-primary/30 hover:bg-primary/10 gap-1.5"
                      onClick={() => {
                        setSelectedApp(app)
                        setDrawerOpen(true)
                      }}
                    >
                      <Eye className="h-3.5 w-3.5" /> Candidate 360 →
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {/* Candidate 360 Slide-Out Drawer */}
      <Candidate360Drawer
        application={selectedApp}
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        onStageUpdate={handleStageUpdate}
      />
    </div>
  )
}
