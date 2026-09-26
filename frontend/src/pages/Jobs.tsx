import { useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  Briefcase,
  Plus,
  MagnifyingGlass,
  UsersThree,
  CheckCircle,
} from "@phosphor-icons/react"
import { Link } from "react-router-dom"
import { api, ApiError } from "@/lib/api"
import {
  DEFAULT_WEIGHT_MAP,
  WEIGHT_DIMENSIONS,
  buildSaveWeights,
  isWeightMapComplete,
  isWeightSumValid,
  weightSum,
  type WeightKey,
} from "@/lib/weights"
import type { Application, Job } from "@/types/api"
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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Skeleton } from "@/components/ui/skeleton"
import { Textarea } from "@/components/ui/textarea"
import { toast } from "sonner"

const POPULAR_SKILLS = [
  "Go",
  "React",
  "TypeScript",
  "PostgreSQL",
  "Docker",
  "Kubernetes",
  "Python",
  "AWS",
  "System Design",
  "GraphQL",
  "Node.js",
  "Microservices",
]

export function JobsPage() {
  const qc = useQueryClient()
  const { data: jobs, isLoading, error } = useQuery({
    queryKey: ["jobs"],
    queryFn: () => api.get<Job[]>("/jobs"),
  })

  const { data: apps } = useQuery({
    queryKey: ["applications"],
    queryFn: () => api.get<Application[]>("/applications"),
  })

  const [open, setOpen] = useState(false)
  const [modalTab, setModalTab] = useState<"details" | "stages">("details")
  const [title, setTitle] = useState("")
  const [description, setDescription] = useState("")
  const [skills, setSkills] = useState("")
  const [minExp, setMinExp] = useState("3")
  const [search, setSearch] = useState("")
  const [filterStatus, setFilterStatus] = useState<"all" | "active" | "archived">("all")
  const [archiveTarget, setArchiveTarget] = useState<Job | null>(null)

  // Assessment Stage Pipeline state
  const [enableScreening, setEnableScreening] = useState(true)
  const [passThreshold, setPassThreshold] = useState(70)

  // Scoring weights state
  const [editingJob, setEditingJob] = useState<Job | null>(null)
  const [weights, setWeights] = useState<Record<string, number>>({ ...DEFAULT_WEIGHT_MAP })
  const [useDefaultWeights, setUseDefaultWeights] = useState(true)
  const [weightsError, setWeightsError] = useState<string | null>(null)

  const minExpNum = Number.parseInt(minExp || "0", 10)
  const minExpValid = Number.isFinite(minExpNum) && minExpNum >= 0

  const isEditing = editingJob !== null
  const weightsLocked = isEditing && editingJob.is_published === true
  const weightsControlsDisabled = weightsLocked || useDefaultWeights

  const create = useMutation({
    mutationFn: () => {
      const payload: Record<string, unknown> = {
        title,
        description,
        required_skills: skills.split(",").map((s) => s.trim()).filter(Boolean),
        min_experience: minExpNum,
        min_score_to_proceed: enableScreening ? passThreshold : 0,
      }
      // D1: never send scoring_weights when locked (published). D10: otherwise
      // always send the explicit resolution — in defaults mode that is
      // DEFAULT_WEIGHT_MAP, because PATCH nil = keep would silently leave
      // previously-saved custom weights effective while the form shows defaults.
      if (!weightsLocked) {
        payload.scoring_weights = buildSaveWeights(useDefaultWeights, weights)
      }
      if (isEditing) {
        return api.patch<Job>(`/jobs/${editingJob.id}`, payload)
      }
      return api.post<Job>("/jobs", payload)
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["jobs"] })
      closeModal()
      toast.success(isEditing ? "Job updated" : "Job role & assessment pipeline published successfully")
    },
    onError: (e) => {
      // Surface WEIGHTS_* rejections inline without closing the form (G7).
      const msg = e instanceof Error ? e.message : "Save failed"
      if (e instanceof ApiError && e.code.startsWith("WEIGHTS_")) {
        setWeightsError(msg)
      }
      toast.error(msg)
    },
  })

	const patchStatus = useMutation({
		mutationFn: ({ id, status }: { id: string; status: string }) =>
			api.patch<Job>(`/jobs/${id}`, { status }),
		onSuccess: () => {
			qc.invalidateQueries({ queryKey: ["jobs"] })
			toast.success("Job status updated")
		},
		// G7: silent failures leave recruiters believing the status changed.
		onError: (e) => toast.error(e instanceof Error ? e.message : "Status update failed"),
	})

	const patchPublished = useMutation({
		mutationFn: ({ id, is_published }: { id: string; is_published: boolean }) =>
			api.patch<Job>(`/jobs/${id}`, { is_published }),
		onSuccess: (data) => {
			qc.invalidateQueries({ queryKey: ["jobs"] })
			toast.success(data.is_published ? "Job published to careers page" : "Job removed from careers page")
		},
		// G7: silent failures leave recruiters believing the publish state changed.
		onError: (e) => toast.error(e instanceof Error ? e.message : "Publish state update failed"),
	})

  function closeModal() {
    setOpen(false)
    setEditingJob(null)
    setTitle("")
    setDescription("")
    setSkills("")
    setMinExp("3")
    setModalTab("details")
    setEnableScreening(true)
    setPassThreshold(70)
    setWeights({ ...DEFAULT_WEIGHT_MAP })
    setUseDefaultWeights(true)
    setWeightsError(null)
  }

  function openEdit(job: Job) {
    setEditingJob(job)
    setOpen(true)
    setModalTab("details")
    setTitle(job.title)
    setDescription(job.description)
    setSkills((job.required_skills ?? []).join(", "))
    setMinExp(String(job.min_experience))
    setEnableScreening((job.min_score_to_proceed ?? 0) > 0)
    setPassThreshold(job.min_score_to_proceed ?? 70)
    if (job.scoring_weights && isWeightMapComplete(job.scoring_weights)) {
      setWeights({ ...DEFAULT_WEIGHT_MAP, ...job.scoring_weights })
      setUseDefaultWeights(false)
    } else {
      setWeights({ ...DEFAULT_WEIGHT_MAP })
      setUseDefaultWeights(true)
    }
    setWeightsError(null)
  }

  function setWeight(key: WeightKey, value: number) {
    if (weightsLocked) return
    setWeights((prev) => ({ ...prev, [key]: Number.isFinite(value) ? value : 0 }))
  }

  function toggleSkill(skill: string) {
    const list = skills.split(",").map((s) => s.trim()).filter(Boolean)
    if (list.includes(skill)) {
      setSkills(list.filter((s) => s !== skill).join(", "))
    } else {
      setSkills([...list, skill].join(", "))
    }
  }

  const filteredJobs = (jobs ?? []).filter((j) => {
    const skills = j.required_skills ?? []
    const matchesSearch =
      j.title.toLowerCase().includes(search.toLowerCase()) ||
      j.description.toLowerCase().includes(search.toLowerCase()) ||
      skills.some((s) => s.toLowerCase().includes(search.toLowerCase()))
    const matchesStatus = filterStatus === "all" || j.status === filterStatus
    return matchesSearch && matchesStatus
  })

  return (
    <div className="space-y-6 animate-in fade-in duration-500">
      {/* Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="font-display text-3xl font-bold tracking-tight">Job Requisitions</h1>
          <p className="text-sm text-muted-foreground">
            Configure target competencies, experience gates, and automated CV screening rails.
          </p>
        </div>
        <Button onClick={() => setOpen(true)} className="shadow-sm">
          <Plus className="mr-1.5 h-4 w-4" weight="bold" /> Post New Job
        </Button>
      </div>

      {/* Filters & Search */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="relative w-full max-w-sm">
          <MagnifyingGlass className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search roles or skills..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9 bg-background/80"
          />
        </div>
        <div className="flex items-center gap-1.5">
          <Button
            variant={filterStatus === "all" ? "secondary" : "ghost"}
            size="sm"
            className="text-xs h-8"
            onClick={() => setFilterStatus("all")}
          >
            All Roles
          </Button>
          <Button
            variant={filterStatus === "active" ? "secondary" : "ghost"}
            size="sm"
            className="text-xs h-8 text-success"
            onClick={() => setFilterStatus("active")}
          >
            Active ({jobs?.filter((j) => j.status === "active").length ?? 0})
          </Button>
          <Button
            variant={filterStatus === "archived" ? "secondary" : "ghost"}
            size="sm"
            className="text-xs h-8 text-muted-foreground"
            onClick={() => setFilterStatus("archived")}
          >
            Archived ({jobs?.filter((j) => j.status === "archived").length ?? 0})
          </Button>
        </div>
      </div>

      {/* Job Cards */}
      {isLoading ? (
        <div className="grid gap-4 md:grid-cols-2">
          <Skeleton className="h-44 w-full rounded-xl" />
          <Skeleton className="h-44 w-full rounded-xl" />
        </div>
      ) : error ? (
        <div className="rounded-xl border border-destructive/30 bg-destructive/5 p-6 text-center text-sm text-destructive">
          {error instanceof Error ? error.message : "Failed to load jobs"}
        </div>
      ) : filteredJobs.length === 0 ? (
        <div className="rounded-2xl border border-dashed border-border/80 p-12 text-center">
          <Briefcase className="mx-auto h-12 w-12 text-muted-foreground/40 mb-3" />
          <p className="font-display font-semibold text-base">No job requisitions found</p>
          <p className="text-xs text-muted-foreground mt-1 max-w-sm mx-auto">
            Click "Post New Job" to define your first role and begin evaluating candidates.
          </p>
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {filteredJobs.map((job) => {
            const jobApps = (apps ?? []).filter((a) => a.job_id === job.id)
            const passedApps = jobApps.filter((a) => a.passed_screening)
            const applicantCount = jobApps.length
            return (
              <div
                key={job.id}
                className="rounded-xl border border-border bg-card p-5 shadow-sm transition-all hover:border-primary/40 flex flex-col justify-between"
              >
                <div>
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <Link
                        to={`/candidates?job_id=${job.id}`}
                        className="font-display text-base font-bold tracking-tight text-foreground hover:text-primary transition-colors"
                      >
                        {job.title}
                      </Link>
                      <div className="flex flex-wrap items-center gap-2 mt-1">
                        <span className="text-xs font-medium text-muted-foreground">
                          {job.min_experience}+ years experience
                        </span>
                        <span>·</span>
                        <span className="text-xs font-semibold text-primary flex items-center gap-1">
                          <UsersThree className="h-3.5 w-3.5" /> {applicantCount} applicants
                        </span>
                      </div>
                    </div>
                    <div className="flex flex-col items-end gap-1">
                      <Badge
                        className={
                          job.status === "active"
                            ? "bg-success/10 text-success border-success/20"
                            : "bg-muted text-muted-foreground"
                        }
                      >
                        {job.status}
                      </Badge>
                      {job.is_published ? (
                        <Badge variant="outline" title="Published = visible on the careers board; Active = accepting applicants" className="text-xs bg-info/10 text-info border-info/20">
                          Published
                        </Badge>
                      ) : (
                        <Badge variant="outline" title="Published = visible on the careers board; Active = accepting applicants" className="text-xs text-muted-foreground">
                          Internal
                        </Badge>
                      )}
                    </div>
                  </div>

                  {/* Relational Pipeline Badges */}
                  <div className="mt-3 flex items-center gap-2">
                    <span className="inline-flex items-center gap-1 rounded-md bg-secondary/80 px-2 py-0.5 text-[11px] font-medium text-foreground">
                      <UsersThree className="h-3 w-3 text-muted-foreground" /> {applicantCount} Applied
                    </span>
                    <span className="inline-flex items-center gap-1 rounded-md bg-success/10 border border-success/20 px-2 py-0.5 text-[11px] font-semibold text-success">
                      <CheckCircle className="h-3 w-3" weight="fill" /> {passedApps.length} Qualified
                    </span>
                  </div>

                  <p className="mt-3 line-clamp-2 text-xs leading-relaxed text-muted-foreground">{job.description}</p>

                  <div className="mt-3 flex flex-wrap items-center gap-1.5">
                    {(job.required_skills ?? []).map((s) => (
                      <span
                        key={s}
                        className="rounded-lg bg-primary/5 border border-primary/15 px-2 py-0.5 text-[11px] font-medium text-foreground"
                      >
                        {s}
                      </span>
                    ))}
                  </div>
                </div>

                <div className="mt-5 flex items-center justify-between border-t border-border/40 pt-3">
                  <Button asChild variant="ghost" size="sm" className="h-8 text-xs text-primary gap-1 font-semibold">
                    <Link to={`/candidates?job_id=${job.id}`}>
                      View Applicants ({applicantCount}) →
                    </Link>
                  </Button>
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-8 text-xs"
                      onClick={() =>
                        patchPublished.mutate({
                          id: job.id,
                          is_published: !job.is_published,
                        })
                      }
                    >
                      {job.is_published ? "Unpublish" : "Publish"}
                    </Button>
                     <Button
                      variant="outline"
                      size="sm"
                      className="h-8 text-xs"
                      onClick={() => openEdit(job)}
                    >
                      Edit
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-8 text-xs text-muted-foreground"
                      onClick={() => setArchiveTarget(job)}
                    >
                      {job.status === "active" ? "Archive" : "Activate"}
                    </Button>
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      )}

      {/* New Job Modal with Assessment Stage Selection */}
      <Dialog open={open} onOpenChange={(o) => { if (!o) closeModal() }}>
        <DialogContent className="sm:max-w-2xl lg:max-w-3xl max-h-[90vh] flex flex-col p-0 overflow-hidden">
          <div className="flex flex-col h-full max-h-[90vh]">
            <DialogHeader className="p-6 pb-3 border-b border-border bg-card shrink-0">
              <DialogTitle className="font-display text-xl font-bold flex items-center gap-2">
                <Plus className="h-5 w-5 text-primary" weight="bold" /> Create Job & Assessment Pipeline
              </DialogTitle>
              <DialogDescription className="text-xs">
                Configure role requirements, CV screening cutoff thresholds, and AI assessment stages.
              </DialogDescription>

              {/* Builder Step Tabs */}
              <div className="flex border-b border-border mt-3">
                <button
                  type="button"
                  onClick={() => setModalTab("details")}
                  className={`flex-1 border-b-2 py-2 text-xs font-semibold transition-colors text-center ${
                    modalTab === "details"
                      ? "border-primary text-primary"
                      : "border-transparent text-muted-foreground hover:text-foreground"
                  }`}
                >
                  1. Role Competencies & Details
                </button>
                <button
                  type="button"
                  onClick={() => setModalTab("stages")}
                  className={`flex-1 border-b-2 py-2 text-xs font-semibold transition-colors text-center ${
                    modalTab === "stages"
                      ? "border-primary text-primary"
                      : "border-transparent text-muted-foreground hover:text-foreground"
                  }`}
                >
                  2. Assessment Stage Pipeline
                </button>
              </div>
            </DialogHeader>

            <div className="overflow-y-auto p-6 space-y-4 flex-1">
              {modalTab === "details" ? (
                <div className="space-y-4">
                  <div className="space-y-1.5">
                    <Label htmlFor="job-title" className="text-xs font-semibold">Job Title</Label>
                    <Input
                      id="job-title"
                      value={title}
                      onChange={(e) => setTitle(e.target.value)}
                      placeholder="e.g. Senior Distributed Systems Engineer"
                      className="bg-background/80"
                    />
                  </div>

                  <div className="space-y-1.5">
                    <Label htmlFor="job-desc" className="text-xs font-semibold">Job Description & Responsibilities</Label>
                    <Textarea
                      id="job-desc"
                      value={description}
                      onChange={(e) => setDescription(e.target.value)}
                      placeholder="Key technical scope, system requirements, architecture responsibilities..."
                      rows={4}
                      className="bg-background/80"
                    />
                  </div>

                  <div className="space-y-1.5">
                    <Label htmlFor="job-skills" className="text-xs font-semibold">Required Skills (Comma separated)</Label>
                    <Input
                      id="job-skills"
                      value={skills}
                      onChange={(e) => setSkills(e.target.value)}
                      placeholder="Go, React, PostgreSQL, Docker"
                      className="bg-background/80"
                    />
                    <div className="mt-2 flex flex-wrap gap-1">
                      {POPULAR_SKILLS.map((sk) => {
                        const active = skills.split(",").map((s) => s.trim()).includes(sk)
                        return (
                          <button
                            key={sk}
                            type="button"
                            onClick={() => toggleSkill(sk)}
                            className={`text-[10px] rounded-md px-2 py-0.5 font-medium transition-colors border ${
                              active
                                ? "bg-primary text-primary-foreground border-primary"
                                : "bg-muted/70 hover:bg-muted text-muted-foreground border-border/50"
                            }`}
                          >
                            {active ? `✓ ${sk}` : `+ ${sk}`}
                          </button>
                        )
                      })}
                    </div>
                  </div>

                  <div className="space-y-1.5">
                    <Label htmlFor="job-exp" className="text-xs font-semibold">Minimum Years of Experience</Label>
                    <Input
                      id="job-exp"
                      type="number"
                      min="0"
                      max="25"
                      value={minExp}
                      onChange={(e) => setMinExp(e.target.value)}
                      className="bg-background/80"
                    />
                  </div>

                  {/* Scoring Weights — D1 lock + D4 sum rule */}
                  <div className="space-y-3 rounded-xl border border-border/80 bg-background/60 p-4">
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <Label className="text-xs font-bold">Scoring Weights</Label>
                        <p className="text-[11px] text-muted-foreground mt-0.5">
                          Tune the 5 screening dimensions. The five must sum to ~1.00 (±0.01).
                        </p>
                      </div>
                      <label className="flex shrink-0 items-center gap-1.5 text-[11px] text-muted-foreground cursor-pointer">
                        <input
                          type="checkbox"
                          checked={useDefaultWeights}
                          disabled={weightsLocked}
                          onChange={(e) => {
                            setUseDefaultWeights(e.target.checked)
                            // D10: sliders are disabled in defaults mode — show the
                            // values that will actually be saved, not stale tuning.
                            if (e.target.checked) setWeights({ ...DEFAULT_WEIGHT_MAP })
                          }}
                          className="rounded border-border text-primary h-3.5 w-3.5"
                        />
                        Use defaults
                      </label>
                    </div>

                    {weightsLocked && (
                      <p className="text-[11px] text-amber-600">
                        This job is published: weights are locked (D1) and cannot be changed.
                      </p>
                    )}

                    <div className="space-y-2.5">
                      {WEIGHT_DIMENSIONS.map((d) => (
                        <div key={d.key} className="flex items-center gap-3">
                          <span className="w-32 text-xs text-foreground">{d.label}</span>
                          <input
                            type="range"
                            min={0}
                            max={1}
                            step={0.05}
                            value={weights[d.key] ?? 0}
                            disabled={weightsControlsDisabled}
                            onChange={(e) => setWeight(d.key, Number(e.target.value))}
                            className="flex-1 h-1.5 bg-secondary rounded-lg appearance-none cursor-pointer accent-primary disabled:opacity-50"
                          />
                          <input
                            type="number"
                            min={0}
                            max={1}
                            step={0.05}
                            value={Number((weights[d.key] ?? 0).toFixed(2))}
                            disabled={weightsControlsDisabled}
                            onChange={(e) => setWeight(d.key, Number(e.target.value))}
                            className="w-16 rounded-md border border-border bg-background/80 px-2 py-1 text-xs text-right font-mono"
                          />
                        </div>
                      ))}
                    </div>

                    <div className="flex items-center justify-between border-t border-border/40 pt-2">
                      <span className="text-[11px] text-muted-foreground">Resolved sum</span>
                      <span
                        className={`font-mono text-xs font-bold ${
                          !useDefaultWeights && !isWeightSumValid(weightSum(weights))
                            ? "text-amber-600"
                            : "text-foreground"
                        }`}
                      >
                        {weightSum(weights).toFixed(2)}
                      </span>
                    </div>

                    {!useDefaultWeights && !isWeightSumValid(weightSum(weights)) && (
                      <p className="text-[11px] text-amber-600">
                        Weights must sum to ~1.00 (±0.01). The backend will reject otherwise.
                      </p>
                    )}

                    {weightsError && (
                      <p className="text-[11px] text-destructive" role="alert">
                        {weightsError}
                      </p>
                    )}
                  </div>
                </div>
              ) : (
                <div className="space-y-3.5">
                  {/* Stage 1: CV Screening */}
                  <div className="rounded-xl border border-border/80 bg-background/60 p-4 space-y-2.5">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <input
                          type="checkbox"
                          id="stage-screening"
                          checked={enableScreening}
                          onChange={(e) => setEnableScreening(e.target.checked)}
                          className="rounded border-border text-primary focus:ring-primary h-4 w-4"
                        />
                        <Label htmlFor="stage-screening" className="text-xs font-bold cursor-pointer">
                          Stage 1: Automated AI Resume Screening
                        </Label>
                      </div>
                      <Badge variant="outline" className="text-[10px]">Instant OCR + Vector Match</Badge>
                    </div>
                    {enableScreening && (
                      <div className="pl-6 pt-2 space-y-2 border-t border-border/40">
                        <div className="flex items-center justify-between text-xs">
                          <span className="text-muted-foreground">Screening Passing Threshold:</span>
                          <span className="font-mono font-bold text-primary">{passThreshold}% Match</span>
                        </div>
                        <input
                          type="range"
                          min="50"
                          max="90"
                          step="5"
                          value={passThreshold}
                          onChange={(e) => setPassThreshold(Number(e.target.value))}
                          className="w-full h-1.5 bg-secondary rounded-lg appearance-none cursor-pointer accent-primary"
                        />
                      </div>
                    )}
                  </div>

                  {/* Stage 1 info */}
                  <div className="rounded-xl border border-border/80 bg-background/60 p-4 space-y-1.5">
                    <div className="flex items-center gap-2">
                      <Badge variant="outline" className="text-[10px]">Stage 1</Badge>
                      <span className="text-xs font-bold">AI CV Screening</span>
                      <span className="ml-auto text-[11px] text-muted-foreground">Automatic, on submission</span>
                    </div>
                    <p className="text-[11px] text-muted-foreground">Semantic extraction + weighted match against the rubric.</p>
                  </div>

                  <div className="rounded-xl border border-border/80 bg-background/60 p-4 space-y-1.5">
                    <div className="flex items-center gap-2">
                      <Badge variant="outline" className="text-[10px]">Stage 2</Badge>
                      <span className="text-xs font-bold">Adaptive AI Technical & Architecture Interview</span>
                      <span className="ml-auto text-[11px] text-muted-foreground">WebSocket streaming</span>
                    </div>
                    <p className="text-[11px] text-muted-foreground">Adaptive question depth (3–8 probes) driven by answer quality.</p>
                  </div>

                  <div className="rounded-xl border border-border/80 bg-background/60 p-4 space-y-1.5">
                    <div className="flex items-center gap-2">
                      <Badge variant="outline" className="text-[10px]">Stage 3</Badge>
                      <span className="text-xs font-bold">Live Coding Sandbox Challenge</span>
                      <span className="ml-auto text-[11px] text-muted-foreground">Go / Python / TS</span>
                    </div>
                    <p className="text-[11px] text-muted-foreground">Candidate writes code and runs automated test suites in the isolated sandbox.</p>
                  </div>

                  <div className="rounded-xl border border-dashed border-border/80 bg-background/40 p-4 space-y-1.5">
                    <div className="flex items-center gap-2">
                      <Badge variant="outline" className="text-[10px]">Stage 4</Badge>
                      <span className="text-xs font-bold text-muted-foreground">AI Voice Phone Screen</span>
                      <Badge variant="secondary" className="text-[10px]">Coming soon</Badge>
                    </div>
                    <p className="text-[11px] text-muted-foreground">Full-duplex voice interviews are in pilot and not yet available on jobs.</p>
                  </div>
                </div>
              )}
            </div>

            <DialogFooter className="p-4 px-6 border-t border-border bg-card shrink-0 flex items-center justify-between sm:justify-between">
              {modalTab === "details" ? (
                <>
                  <Button variant="secondary" onClick={closeModal}>
                    Cancel
                  </Button>
                  <Button
                    variant="outline"
                    onClick={() => setModalTab("stages")}
                    disabled={!title.trim() || !minExpValid}
                    className="gap-1 text-xs font-bold"
                  >
                    Next: Configure Stages →
                  </Button>
                </>
              ) : (
                <>
                  <Button variant="secondary" onClick={() => setModalTab("details")}>
                    ← Back to Details
                  </Button>
                  <Button
                    onClick={() => create.mutate()}
                    disabled={!title.trim() || !minExpValid || create.isPending}
                    className="gap-1 text-xs font-bold shadow-sm"
                  >
                    <Plus className="h-4 w-4" weight="bold" />
                    {create.isPending
                      ? isEditing
                        ? "Saving..."
                        : "Publishing Role..."
                      : isEditing
                        ? "Save Changes"
                        : "Publish Job & Pipeline"}
                  </Button>
                </>
              )}
            </DialogFooter>
          </div>
        </DialogContent>
      </Dialog>

      {/* Archive Confirmation AlertDialog */}
      <AlertDialog open={archiveTarget !== null} onOpenChange={(o) => !o && setArchiveTarget(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{archiveTarget?.status === "active" ? "Archive" : "Activate"} Job</AlertDialogTitle>
            <AlertDialogDescription>
              {archiveTarget?.status === "active"
                ? `Archive "${archiveTarget?.title}"? It will no longer appear on the careers board.`
                : `Activate "${archiveTarget?.title}"? It will appear on the careers board again.`}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (archiveTarget) {
                  patchStatus.mutate({
                    id: archiveTarget.id,
                    status: archiveTarget.status === "active" ? "archived" : "active",
                  })
                  setArchiveTarget(null)
                }
              }}
            >
              {archiveTarget?.status === "active" ? "Archive" : "Activate"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
