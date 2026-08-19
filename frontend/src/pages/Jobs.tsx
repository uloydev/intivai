import { useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  Briefcase,
  Plus,
  MagnifyingGlass,
  Sparkle,
  UsersThree,
  CheckCircle,
} from "@phosphor-icons/react"
import { Link } from "react-router-dom"
import { api } from "@/lib/api"
import type { Application, Job } from "@/types/api"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
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

  // Assessment Stage Pipeline state
  const [enableScreening, setEnableScreening] = useState(true)
  const [passThreshold, setPassThreshold] = useState(70)

  const minExpNum = Number.parseInt(minExp || "0", 10)
  const minExpValid = Number.isFinite(minExpNum) && minExpNum >= 0

  const create = useMutation({
    mutationFn: () =>
      api.post<Job>("/jobs", {
        title,
        description,
        required_skills: skills.split(",").map((s) => s.trim()).filter(Boolean),
        min_experience: minExpNum,
        min_score_to_proceed: enableScreening ? passThreshold : 0,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["jobs"] })
      setOpen(false)
      setTitle("")
      setDescription("")
      setSkills("")
      setMinExp("3")
      setModalTab("details")
      toast.success("Job role & assessment pipeline published successfully")
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Create failed"),
  })

	const patchStatus = useMutation({
		mutationFn: ({ id, status }: { id: string; status: string }) =>
			api.patch<Job>(`/jobs/${id}`, { status }),
		onSuccess: () => {
			qc.invalidateQueries({ queryKey: ["jobs"] })
			toast.success("Job status updated")
		},
	})

	const patchPublished = useMutation({
		mutationFn: ({ id, is_published }: { id: string; is_published: boolean }) =>
			api.patch<Job>(`/jobs/${id}`, { is_published }),
		onSuccess: (data) => {
			qc.invalidateQueries({ queryKey: ["jobs"] })
			toast.success(data.is_published ? "Job published to careers page" : "Job removed from careers page")
		},
	})

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
        <Button onClick={() => setOpen(true)} variant="gradient" className="shadow-md shadow-primary/20">
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
            className="text-xs h-8 text-emerald-600 dark:text-emerald-400"
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
                className="glass rounded-xl border border-border/60 p-5 shadow-sm transition-all hover:border-primary/40 hover:shadow-md flex flex-col justify-between"
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
                            ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20"
                            : "bg-muted text-muted-foreground"
                        }
                      >
                        {job.status}
                      </Badge>
                      {job.is_published ? (
                        <Badge variant="outline" title="Published = visible on the careers board; Active = accepting applicants" className="text-xs bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-500/20">
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
                    <span className="inline-flex items-center gap-1 rounded-md bg-emerald-500/10 border border-emerald-500/20 px-2 py-0.5 text-[11px] font-semibold text-emerald-500">
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
                      className="h-8 text-xs text-muted-foreground"
                      onClick={() =>
                        patchStatus.mutate({
                          id: job.id,
                          status: job.status === "active" ? "archived" : "active",
                        })
                      }
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
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-w-xl">
          <DialogHeader>
            <DialogTitle className="font-display text-xl flex items-center gap-2">
              <Sparkle className="h-5 w-5 text-primary" weight="fill" /> Create Job & Assessment Pipeline
            </DialogTitle>
            <DialogDescription>
              Configure role requirements, CV screening cutoff thresholds, and AI assessment stages.
            </DialogDescription>
          </DialogHeader>

          {/* Builder Step Tabs */}
          <div className="flex border-b border-border my-2">
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

          {modalTab === "details" ? (
            <div className="space-y-4 py-1">
              <div className="space-y-1.5">
                <Label htmlFor="job-title" className="text-xs font-semibold">Job Title</Label>
                <Input
                  id="job-title"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  placeholder="e.g. Senior Distributed Systems Engineer"
                />
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="job-desc" className="text-xs font-semibold">Job Description & Responsibilities</Label>
                <Textarea
                  id="job-desc"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Key technical scope, system requirements, architecture responsibilities..."
                  rows={3}
                />
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="job-skills" className="text-xs font-semibold">Required Skills (Comma separated)</Label>
                <Input
                  id="job-skills"
                  value={skills}
                  onChange={(e) => setSkills(e.target.value)}
                  placeholder="Go, React, PostgreSQL, Docker"
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
                />
              </div>
            </div>
          ) : (
            <div className="space-y-3.5 py-1">
              {/* Stage 1: CV Screening */}
              <div className="rounded-xl border border-border/80 bg-background/60 p-3.5 space-y-2">
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
                  <div className="pl-6 pt-1 space-y-1.5 border-t border-border/40">
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

              {/* The assessment pipeline is fixed for every interview — these
                  descriptions replace the old configurable toggles, which were
                  never persisted to the backend. */}
              <div className="rounded-xl border border-border/80 bg-background/60 p-3.5 space-y-2">
                <div className="flex items-center gap-2">
                  <Badge variant="outline" className="text-[10px]">Stage 1</Badge>
                  <span className="text-xs font-bold">AI CV Screening</span>
                  <span className="ml-auto text-[11px] text-muted-foreground">Automatic, on submission</span>
                </div>
                <p className="text-[11px] text-muted-foreground">Semantic extraction + weighted match against the rubric below.</p>
              </div>

              <div className="rounded-xl border border-border/80 bg-background/60 p-3.5 space-y-2">
                <div className="flex items-center gap-2">
                  <Badge variant="outline" className="text-[10px]">Stage 2</Badge>
                  <span className="text-xs font-bold">Adaptive AI Technical & Architecture Interview</span>
                  <span className="ml-auto text-[11px] text-muted-foreground">WebSocket streaming</span>
                </div>
                <p className="text-[11px] text-muted-foreground">Adaptive question depth (3–8 probes) driven by answer quality.</p>
              </div>

              <div className="rounded-xl border border-border/80 bg-background/60 p-3.5 space-y-2">
                <div className="flex items-center gap-2">
                  <Badge variant="outline" className="text-[10px]">Stage 3</Badge>
                  <span className="text-xs font-bold">Live Coding Sandbox Challenge</span>
                  <span className="ml-auto text-[11px] text-muted-foreground">Go / Python / TS</span>
                </div>
                <p className="text-[11px] text-muted-foreground">Candidate writes code and runs automated test suites in the isolated sandbox.</p>
              </div>

              <div className="rounded-xl border border-dashed border-border/80 bg-background/40 p-3.5 space-y-2">
                <div className="flex items-center gap-2">
                  <Badge variant="outline" className="text-[10px]">Stage 4</Badge>
                  <span className="text-xs font-bold text-muted-foreground">AI Voice Phone Screen</span>
                  <Badge variant="secondary" className="text-[10px]">Coming soon</Badge>
                </div>
                <p className="text-[11px] text-muted-foreground">Full-duplex voice interviews are in pilot and not yet available on jobs.</p>
              </div>
            </div>
          )}

          <DialogFooter className="flex items-center justify-between sm:justify-between pt-2">
            {modalTab === "details" ? (
              <>
                <Button variant="secondary" onClick={() => setOpen(false)}>
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
                  variant="gradient"
                  onClick={() => create.mutate()}
                  disabled={!title.trim() || !minExpValid || create.isPending}
                  className="gap-1 text-xs font-bold shadow-md shadow-primary/20"
                >
                  <Sparkle className="h-4 w-4" weight="fill" />
                  {create.isPending ? "Publishing Role..." : "Publish Job & Pipeline"}
                </Button>
              </>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
