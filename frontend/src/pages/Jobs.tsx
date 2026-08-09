import { useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Plus } from "@phosphor-icons/react"
import { api } from "@/lib/api"
import type { Job } from "@/types/api"
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

export function JobsPage() {
  const qc = useQueryClient()
  const { data: jobs, isLoading, error } = useQuery({
    queryKey: ["jobs"],
    queryFn: () => api.get<Job[]>("/jobs"),
  })
  const [open, setOpen] = useState(false)
  const [title, setTitle] = useState("")
  const [description, setDescription] = useState("")
  const [skills, setSkills] = useState("")
  const [minExp, setMinExp] = useState("0")

  const minExpNum = Number.parseInt(minExp || "0", 10)
  const minExpValid = Number.isFinite(minExpNum) && minExpNum >= 0

  const create = useMutation({
    mutationFn: () =>
      api.post<Job>("/jobs", {
        title,
        description,
        required_skills: skills.split(",").map((s) => s.trim()).filter(Boolean),
        min_experience: minExpNum,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["jobs"] })
      setOpen(false)
      setTitle("")
      setDescription("")
      setSkills("")
      setMinExp("0")
      toast.success("Job created")
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Create failed"),
  })

  const patchStatus = useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) =>
      api.patch<Job>(`/jobs/${id}`, { status }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["jobs"] }),
  })

  if (error) {
    return (
      <div className="flex flex-col items-center gap-2 py-16 text-center">
        <p className="text-destructive">{error instanceof Error ? error.message : "Failed to load jobs"}</p>
        <Button variant="secondary" onClick={() => qc.invalidateQueries({ queryKey: ["jobs"] })}>
          Retry
        </Button>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="font-display text-2xl">Jobs</h1>
        <Button onClick={() => setOpen(true)}>
          <Plus className="mr-2 h-4 w-4" /> New job
        </Button>
      </div>

      {isLoading ? (
        <div className="space-y-2">
          <Skeleton className="h-20 w-full" />
          <Skeleton className="h-20 w-full" />
        </div>
      ) : !jobs?.length ? (
        <div className="flex flex-col items-center gap-2 py-16 text-center">
          <p className="font-display text-lg">No jobs yet</p>
          <p className="text-sm text-muted-foreground">Create a job to start screening candidates</p>
        </div>
      ) : (
        <div className="grid gap-3 md:grid-cols-2">
          {jobs.map((job) => (
            <div key={job.id} className="rounded-md border border-border bg-card p-4 shadow-sm">
              <div className="flex items-start justify-between gap-2">
                <div>
                  <h2 className="font-display text-base font-semibold">{job.title}</h2>
                  <p className="mt-1 line-clamp-2 text-sm text-muted-foreground">{job.description}</p>
                </div>
                <Badge
                  variant={job.status === "active" ? "default" : "secondary"}
                  className={job.status === "active" ? "bg-accent text-accent-foreground" : undefined}
                >
                  {job.status}
                </Badge>
              </div>
              <div className="mt-3 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                {job.required_skills.map((s) => (
                  <span key={s} className="rounded-full bg-muted px-2 py-0.5">
                    {s}
                  </span>
                ))}
                <span>{job.min_experience}+ yrs</span>
              </div>
              <div className="mt-3 flex gap-2">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => patchStatus.mutate({ id: job.id, status: job.status === "active" ? "archived" : "active" })}
                >
                  {job.status === "active" ? "Archive" : "Activate"}
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>New job</DialogTitle>
            <DialogDescription>Skills drive CV-gap questions and screening weights.</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="job-title">Title</Label>
              <Input id="job-title" value={title} onChange={(e) => setTitle(e.target.value)} placeholder="Senior Go Engineer" />
            </div>
            <div className="space-y-2">
              <Label htmlFor="job-desc">Description</Label>
              <Textarea id="job-desc" value={description} onChange={(e) => setDescription(e.target.value)} rows={4} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="job-skills">Required skills (comma separated)</Label>
              <Input id="job-skills" value={skills} onChange={(e) => setSkills(e.target.value)} placeholder="Go, PostgreSQL, Kubernetes" />
            </div>
            <div className="space-y-2">
              <Label htmlFor="job-exp">Min experience (years)</Label>
              <Input id="job-exp" type="number" value={minExp} onChange={(e) => setMinExp(e.target.value)} />
            </div>
          </div>
          <DialogFooter>
            <Button variant="secondary" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button onClick={() => create.mutate()} disabled={!title.trim() || !minExpValid || create.isPending}>
              {create.isPending ? "Creating…" : "Create"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
