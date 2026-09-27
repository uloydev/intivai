import { useState, useEffect, useMemo, useRef, useDeferredValue } from "react"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import { useParams, useSearchParams } from "react-router-dom"
import {
  UsersThree,
  MagnifyingGlass,
  CheckCircle,
  XCircle,
  Briefcase,
  Eye,
  CaretUp,
  CaretDown,
  CaretUpDown,
  PaperPlaneTilt,
} from "@phosphor-icons/react"
import { api } from "@/lib/api"
import { summarizeBulkResults } from "@/lib/bulk-results"
import { stageMeta } from "@/lib/stages"
import type { Application, CandidateLifecycleStage, Job } from "@/types/api"
import { Candidate360Drawer } from "@/components/candidates/Candidate360Drawer"
import { toast } from "sonner"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
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

const PAGE_SIZE = 50

type SortKey = "score" | "date"
type SortDir = "asc" | "desc"

const BULK_STAGE_OPTIONS: Array<{ stage: CandidateLifecycleStage; label: string }> = [
  { stage: "screening_passed", label: "Mark as Screening Passed" },
  { stage: "interview_invited", label: "Move to Interview Invited" },
  { stage: "offer_extended", label: "Extend Offer" },
  { stage: "hired", label: "Mark as Hired" },
  { stage: "rejected", label: "Reject Candidates" },
]

function scorePill(app: Application) {
  if (app.cv_score == null) {
    // cv_score is null until the pipeline produces one — the pill must say
    // WHY, not guess "scoring" (pending_review blocks scoring entirely).
    switch (app.cv_status) {
      case "pending_review":
        return (
          <Badge variant="secondary" className="bg-warning/10 text-warning border-warning/20 text-xs">
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
          <Badge variant="secondary" className="bg-warning/10 text-warning border-warning/20 text-xs">
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
  const bd = app.score_breakdown
  return (
    <div className="space-y-1">
      {app.passed_screening ? (
        <Badge className="bg-success/10 text-success border-success/20 font-bold text-xs gap-1">
          <CheckCircle className="h-3 w-3" weight="fill" /> <span className="font-mono tabular-nums">{app.cv_score}%</span> Match
        </Badge>
      ) : (
        <Badge variant="destructive" className="font-semibold text-xs gap-1">
          <XCircle className="h-3 w-3" weight="fill" /> <span className="font-mono tabular-nums">{app.cv_score}%</span> Match
        </Badge>
      )}
      {bd && (bd.skills_match != null || bd.experience_years != null || bd.semantic_match != null) && (
        <div className="flex items-center gap-1 text-[10px] font-mono text-muted-foreground" title="Skills · Experience · Semantic breakdown">
          {bd.skills_match != null && <span>S:{(bd.skills_match * 100).toFixed(0)}%</span>}
          {bd.experience_years != null && <span>· E:{(bd.experience_years * 100).toFixed(0)}%</span>}
          {bd.semantic_match != null && <span>· V:{(bd.semantic_match * 100).toFixed(0)}%</span>}
        </div>
      )}
    </div>
  )
}

function stagePill(app: Application) {
  // stage is the authoritative recruiter decision (ADR-0001); null = undecided
  const meta = stageMeta(app.stage ?? "")
  const isCompleted = app.stage === "interview_completed"
  return (
    <Badge className={cn("text-xs font-medium", meta.color)}>
      {meta.label}
      {isCompleted ? <span className="ml-1 font-mono text-[11px] tabular-nums">({app.interview_score ?? "-"}/100)</span> : ""}
    </Badge>
  )
}

// Module scope (G11): a component definition inside a render body remounts
// on every render and recreates closures for no benefit.
function SortIcon({ active, dir }: { active: boolean; dir: SortDir }) {
  if (!active) return <CaretUpDown className="h-3.5 w-3.5 opacity-50" />
  return dir === "asc" ? <CaretUp className="h-3.5 w-3.5" /> : <CaretDown className="h-3.5 w-3.5" />
}

export function CandidatesPage() {
  const { id: routeId } = useParams<{ id: string }>()
  const [searchParams, setSearchParams] = useSearchParams()
  const qc = useQueryClient()
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
  const deferredSearch = useDeferredValue(search)
  const [selectedApp, setSelectedApp] = useState<Application | null>(null)
  const [selectedAppIds, setSelectedAppIds] = useState<Set<string>>(new Set())
  const [bulkProcessing, setBulkProcessing] = useState(false)
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [page, setPage] = useState(1)
  const [sortKey, setSortKey] = useState<SortKey>("score")
  const [sortDir, setSortDir] = useState<SortDir>("desc")

  // G11: deep-linked missing candidate — toast once per param, not on every
  // background poll tick.
  const missingCandidateToastedRef = useRef<string | null>(null)

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
        missingCandidateToastedRef.current = null
        setSelectedApp(match)
        setDrawerOpen(true)
      } else if (missingCandidateToastedRef.current !== candidateParam) {
        missingCandidateToastedRef.current = candidateParam
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

  // G11: structural param type — both MouseEvent and ChangeEvent satisfy it,
  // so no `as unknown as` casts are needed at call sites.
  const toggleSelect = (id: string, e?: { stopPropagation(): void }) => {
    e?.stopPropagation()
    setSelectedAppIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const toggleSelectAll = () => {
    if (pageApps.every((a) => selectedAppIds.has(a.id))) {
      setSelectedAppIds((prev) => {
        const next = new Set(prev)
        pageApps.forEach((a) => next.delete(a.id))
        return next
      })
    } else {
      setSelectedAppIds((prev) => {
        const next = new Set(prev)
        pageApps.forEach((a) => next.add(a.id))
        return next
      })
    }
  }

  // G8: Promise.allSettled masks per-item failures — report succeeded/failed
  // counts and grouped reasons honestly instead of a blanket success toast.
  const handleBulkStageChange = async (targetStage: CandidateLifecycleStage) => {
    if (selectedAppIds.size === 0) return
    setBulkProcessing(true)
    try {
      const ids = Array.from(selectedAppIds)
      const results = await Promise.allSettled(
        ids.map((id) => api.patch(`/applications/${id}`, { stage: targetStage }))
      )
      const summary = summarizeBulkResults(results)
      qc.invalidateQueries({ queryKey: ["applications"] })
      const stageLabel = stageMeta(targetStage).label
      if (summary.succeeded === 0) {
        toast.error(`No candidates were moved to ${stageLabel}. Failures: ${summary.reasons.join("; ")}.`)
      } else if (summary.failed > 0) {
        toast.warning(
          `Updated ${summary.succeeded} candidate${summary.succeeded > 1 ? "s" : ""} to ${stageLabel}; ${summary.failed} failed (${summary.reasons.join("; ")}).`
        )
        setSelectedAppIds(new Set())
      } else {
        toast.success(`Updated ${summary.succeeded} candidate${summary.succeeded > 1 ? "s" : ""} to ${stageLabel}.`)
        setSelectedAppIds(new Set())
      }
    } finally {
      setBulkProcessing(false)
    }
  }

  const handleBulkInvite = async () => {
    if (selectedAppIds.size === 0) return
    setBulkProcessing(true)
    try {
      const eligibleApps = (apps ?? []).filter(
        (a) => selectedAppIds.has(a.id) && !a.interview_id
      )
      if (eligibleApps.length === 0) {
        toast.info("All selected candidates already have interview invitations.")
        setBulkProcessing(false)
        return
      }
      const results = await Promise.allSettled(
        eligibleApps.map((a) =>
          api.post("/interviews", {
            application_id: a.id,
            question_count: 3,
          })
        )
      )
      const summary = summarizeBulkResults(results)
      qc.invalidateQueries({ queryKey: ["applications"] })
      qc.invalidateQueries({ queryKey: ["interviews"] })
      if (summary.succeeded === 0) {
        toast.error(`No interview invitations were generated. Failures: ${summary.reasons.join("; ")}.`)
      } else if (summary.failed > 0) {
        toast.warning(
          `Generated interview sessions for ${summary.succeeded} candidate${summary.succeeded > 1 ? "s" : ""}; ${summary.failed} failed (${summary.reasons.join("; ")}).`
        )
        setSelectedAppIds(new Set())
      } else {
        toast.success(`Generated interview sessions for ${summary.succeeded} candidate${summary.succeeded > 1 ? "s" : ""}.`)
        setSelectedAppIds(new Set())
      }
    } finally {
      setBulkProcessing(false)
    }
  }

  const filteredApps = (apps ?? []).filter((app) => {
    const name = (app.candidate_name || "").toLowerCase()
    const email = (app.candidate_email || "").toLowerCase()
    const title = (app.job_title || "").toLowerCase()
    const q = (deferredSearch || "").toLowerCase()
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

  const sortedApps = useMemo(() => {
    const list = [...filteredApps]
    list.sort((a, b) => {
      let cmp = 0
      if (sortKey === "score") {
        const av = a.cv_score ?? -1
        const bv = b.cv_score ?? -1
        cmp = av === bv ? 0 : av < bv ? -1 : 1
      } else {
        const at = a.applied_at ? new Date(a.applied_at).getTime() : -1
        const bt = b.applied_at ? new Date(b.applied_at).getTime() : -1
        cmp = at === bt ? 0 : at < bt ? -1 : 1
      }
      return sortDir === "desc" ? -cmp : cmp
    })
    return list
  }, [filteredApps, sortKey, sortDir])

  // Reset pagination whenever the filter/sort window changes.
  useEffect(() => {
    setPage(1)
  }, [deferredSearch, selectedJob, statusFilter])

  const pageCount = Math.max(1, Math.ceil(sortedApps.length / PAGE_SIZE))
  const safePage = Math.min(page, pageCount)
  const pageApps = sortedApps.slice((safePage - 1) * PAGE_SIZE, safePage * PAGE_SIZE)

  const toggleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"))
    } else {
      setSortKey(key)
      setSortDir(key === "score" ? "desc" : "desc")
    }
  }

  const sortableHead = (label: string, key: SortKey, className?: string) => (
    <TableHead
      className={className}
      aria-sort={
        sortKey === key ? (sortDir === "asc" ? "ascending" : "descending") : "none"
      }
    >
      <button
        type="button"
        onClick={() => toggleSort(key)}
        className={cn(
          "inline-flex items-center gap-1 rounded font-semibold uppercase tracking-wider transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/60",
          sortKey === key ? "text-foreground" : "text-muted-foreground hover:text-foreground"
        )}
      >
        {label}
        <SortIcon active={sortKey === key} dir={sortDir} />
      </button>
    </TableHead>
  )

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
          <Select value={selectedJob} onValueChange={handleJobChange}>
            <SelectTrigger className="w-full sm:w-56 h-8 text-xs bg-background/80 shrink-0">
              <SelectValue placeholder="All Roles" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Roles</SelectItem>
              {jobs?.map((j) => (
                <SelectItem key={j.id} value={j.id}>
                  {j.title}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
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
            className="text-xs h-8 text-success"
            onClick={() => handleStageChange("screening_passed")}
          >
            Passed ({apps?.filter((a) => a.passed_screening).length ?? 0})
          </Button>
          <Button
            variant={statusFilter === "interview_completed" ? "secondary" : "ghost"}
            size="sm"
            className="text-xs h-8 text-info"
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
        <>
          {/* Desktop Table */}
          <div className="hidden md:block rounded-xl border border-border/60 bg-card shadow-sm overflow-hidden">
            <Table>
              <TableHeader>
                <TableRow className="bg-muted/40">
                  <TableHead className="w-12 text-center">
                    <input
                      type="checkbox"
                      aria-label="Select all candidates"
                      checked={pageApps.length > 0 && pageApps.every((a) => selectedAppIds.has(a.id))}
                      onChange={toggleSelectAll}
                      className="h-4 w-4 rounded border-border accent-primary cursor-pointer align-middle"
                    />
                  </TableHead>
                  <TableHead>Candidate Profile</TableHead>
                  <TableHead>Target Role</TableHead>
                  {sortableHead("CV Match", "score")}
                  {sortableHead("Applied", "date")}
                  <TableHead>Talent Lifecycle Stage</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
            <TableBody>
              {pageApps.map((app) => (
                <TableRow
                  key={app.id}
                  tabIndex={0}
                  role="button"
                  className={cn(
                    "cursor-pointer transition-colors hover:bg-muted/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60",
                    selectedAppIds.has(app.id) && "bg-primary/5 hover:bg-primary/10"
                  )}
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
                  <TableCell className="w-12 text-center" onClick={(e) => e.stopPropagation()}>
                    <input
                      type="checkbox"
                      aria-label={`Select ${app.candidate_name || "candidate"}`}
                      checked={selectedAppIds.has(app.id)}
                      onChange={(e) => toggleSelect(app.id, e)}
                      className="h-4 w-4 rounded border-border accent-primary cursor-pointer align-middle"
                    />
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-3">
                      <div className="flex h-9 w-9 items-center justify-center rounded-full bg-primary/10 text-primary font-bold text-xs">
                        {app.candidate_name ? app.candidate_name.charAt(0).toUpperCase() : "C"}
                      </div>
                      <div>
                        <p className="font-display font-semibold text-sm">{app.candidate_name || "Candidate"}</p>
                        <p className="text-xs text-muted-foreground">{app.candidate_email || "-"}</p>
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
                  <TableCell>
                    <span className="text-xs font-mono text-muted-foreground tabular-nums">
                      {app.applied_at ? new Date(app.applied_at).toLocaleDateString() : "-"}
                    </span>
                  </TableCell>
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
                      <Eye className="h-3.5 w-3.5" /> Candidate 360
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>

          {/* Pagination footer */}
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between border-t border-border/60 px-4 py-3">
            <p className="text-xs text-muted-foreground">
              {sortedApps.length === 0
                ? "0 candidates"
                : `Showing ${(safePage - 1) * PAGE_SIZE + 1}–${Math.min(safePage * PAGE_SIZE, sortedApps.length)} of ${sortedApps.length} candidates`}
            </p>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                className="h-8 text-xs"
                disabled={safePage <= 1}
                onClick={() => setPage(safePage - 1)}
              >
                Previous
              </Button>
              <span className="text-xs text-muted-foreground">
                Page {safePage} of {pageCount}
              </span>
              <Button
                variant="outline"
                size="sm"
                className="h-8 text-xs"
                disabled={safePage >= pageCount}
                onClick={() => setPage(safePage + 1)}
              >
                Next
              </Button>
            </div>
          </div>
          </div>

          {/* Mobile Card Layout */}
          <div className="md:hidden space-y-3">
            {pageApps.map((app) => (
              <div
                key={app.id}
                className="p-4 bg-card border border-border/60 rounded-xl shadow-sm cursor-pointer hover:border-primary/40 transition-colors"
                onClick={() => { setSelectedApp(app); setDrawerOpen(true) }}
              >
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-3">
                    <div className="flex h-9 w-9 items-center justify-center rounded-full bg-primary/10 text-primary font-bold text-xs">
                      {app.candidate_name ? app.candidate_name.charAt(0).toUpperCase() : "C"}
                    </div>
                    <div>
                      <p className="font-display font-semibold text-sm">{app.candidate_name || "Candidate"}</p>
                      <p className="text-xs text-muted-foreground">{app.candidate_email || "-"}</p>
                    </div>
                  </div>
                  <input
                    type="checkbox"
                    aria-label={`Select ${app.candidate_name || "candidate"}`}
                    checked={selectedAppIds.has(app.id)}
                    onChange={(e) => toggleSelect(app.id, e)}
                    className="h-4 w-4 rounded border-border accent-primary cursor-pointer"
                  />
                </div>
                <div className="flex items-center justify-between text-xs">
                  <span className="text-muted-foreground">{app.job_title || "General Application"}</span>
                  {app.cv_score !== null && app.cv_score !== undefined && (
                    <Badge variant={app.cv_score >= 70 ? "success" : "secondary"} size="sm">{Math.round(app.cv_score)}%</Badge>
                  )}
                </div>
              </div>
            ))}
          </div>
        </>
      )}

      {/* Floating Bulk Actions Bar */}
      {selectedAppIds.size > 0 && (
        <div className="fixed bottom-6 left-1/2 -translate-x-1/2 z-40 bg-card border border-border rounded-xl shadow-lg p-3 sm:px-6 flex flex-wrap items-center gap-3 sm:gap-4 animate-in slide-in-from-bottom-5">
          <div className="flex items-center gap-2">
            <span className="flex h-6 w-6 items-center justify-center rounded-full bg-primary text-primary-foreground font-bold text-xs">
              {selectedAppIds.size}
            </span>
            <span className="text-xs font-semibold text-foreground">
              Candidate{selectedAppIds.size > 1 ? "s" : ""} Selected
            </span>
          </div>

          <div className="h-4 w-px bg-border/80 hidden sm:block" />

          {/* Bulk Stage Selector */}
          <Select onValueChange={(val) => handleBulkStageChange(val as CandidateLifecycleStage)} disabled={bulkProcessing}>
            <SelectTrigger className="h-8 text-xs bg-background/80 w-44">
              <SelectValue placeholder="Advance Stage…" />
            </SelectTrigger>
            <SelectContent>
              {BULK_STAGE_OPTIONS.map((opt) => (
                <SelectItem key={opt.stage} value={opt.stage}>
                  {opt.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          {/* Bulk Invite Button */}
          <Button
            size="sm"
            variant="default"
            disabled={bulkProcessing}
            onClick={handleBulkInvite}
            className="h-8 text-xs gap-1.5 shadow-sm"
          >
            <PaperPlaneTilt className="h-3.5 w-3.5" />
            <span>Generate Invites</span>
          </Button>

          {/* Clear Selection */}
          <Button
            size="sm"
            variant="ghost"
            disabled={bulkProcessing}
            onClick={() => setSelectedAppIds(new Set())}
            className="h-8 text-xs text-muted-foreground hover:text-foreground"
          >
            Clear
          </Button>
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
