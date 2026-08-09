import { useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Link as LinkIcon } from "@phosphor-icons/react"
import { Link } from "react-router-dom"
import { api } from "@/lib/api"
import type { Application, CreateInterviewResult, InterviewListItem } from "@/types/api"
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { toast } from "sonner"

function inviteUrl(iv: CreateInterviewResult): string {
  return `${window.location.origin}/invite/${iv.interview_id}?t=${encodeURIComponent(iv.invitation_token)}`
}

export function InterviewsPage() {
  const qc = useQueryClient()
  const { data: apps, isLoading } = useQuery({
    queryKey: ["applications"],
    queryFn: () => api.get<Application[]>("/applications"),
    // Poll while scoring is still in flight — passed list depends on it.
    refetchInterval: (query) =>
      query.state.data?.some((a) => a.cv_score == null) ? 2000 : false,
  })
  const passed = (apps ?? []).filter((a) => a.passed_screening)
  const { data: createdInterviews, isLoading: loadingList } = useQuery({
    queryKey: ["interviews"],
    queryFn: () => api.get<InterviewListItem[]>("/interviews"),
  })

  const [open, setOpen] = useState(false)
  const [selectedApp, setSelectedApp] = useState<Application | null>(null)
  const [count, setCount] = useState("3")
  const [created, setCreated] = useState<CreateInterviewResult | null>(null)

  const create = useMutation({
    mutationFn: () =>
      api.post<CreateInterviewResult>("/interviews", {
        application_id: selectedApp!.id,
        question_count: parseInt(count || "3", 10),
      }),
    onSuccess: (result) => {
      setCreated(result)
      qc.invalidateQueries({ queryKey: ["applications"] })
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Create failed"),
  })

  async function copyInvite() {
    if (!created) return
    try {
      await navigator.clipboard.writeText(inviteUrl(created))
      toast.success("Invite link copied")
    } catch {
      toast.error("Copy failed — select the link manually")
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="font-display text-2xl">Interviews</h1>
        <Button onClick={() => setOpen(true)} disabled={passed.length === 0}>
          New interview
        </Button>
      </div>

      <h2 className="font-display text-lg">Created interviews</h2>
      {loadingList ? (
        <Skeleton className="h-24 w-full" />
      ) : !createdInterviews?.length ? (
        <p className="text-sm text-muted-foreground">
          None yet — create one below from a candidate who passed screening.
        </p>
      ) : (
        <div className="rounded-md border border-border bg-card shadow-sm">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Candidate</TableHead>
                <TableHead>Job</TableHead>
                <TableHead>Status</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {createdInterviews.map((iv) => (
                <TableRow key={iv.interview_id}>
                  <TableCell className="font-medium">{iv.candidate_name || "—"}</TableCell>
                  <TableCell>{iv.job_title || "—"}</TableCell>
                  <TableCell>
                    <Badge variant="secondary">{iv.status}</Badge>
                  </TableCell>
                  <TableCell className="text-right">
                    <Link to={`/interviews/${iv.interview_id}`} className="text-sm text-primary hover:underline">
                      View →
                    </Link>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      <h2 className="font-display text-lg">Start a new interview</h2>
      {passed.length === 0 && (
        <p className="text-sm text-muted-foreground">
          No passed applications yet — interviews require a candidate who passed screening.
        </p>
      )}

      {isLoading ? (
        <Skeleton className="h-40 w-full" />
      ) : (
        <div className="rounded-md border border-border bg-card shadow-sm">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Candidate</TableHead>
                <TableHead>Job</TableHead>
                <TableHead>Score</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {passed.map((app) => (
                <TableRow key={app.id}>
                  <TableCell className="font-medium">{app.candidate_name}</TableCell>
                  <TableCell>{app.job_title}</TableCell>
                  <TableCell>
                    <Badge className="bg-accent text-accent-foreground">{app.cv_score}</Badge>
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        setSelectedApp(app)
                        setCreated(null)
                        setOpen(true)
                      }}
                    >
                      Interview
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          {created ? (
            <>
              <DialogHeader>
                <DialogTitle>Interview ready</DialogTitle>
                <DialogDescription>Share the invite link — the candidate opens it, consents, and starts chatting.</DialogDescription>
              </DialogHeader>
              <div className="space-y-3">
                <Label htmlFor="invite-link">Invite link</Label>
                <div className="flex gap-2">
                  <Input id="invite-link" readOnly value={inviteUrl(created)} />
                  <Button onClick={copyInvite}>
                    <LinkIcon className="mr-2 h-4 w-4" /> Copy
                  </Button>
                </div>
                <Link to={`/interviews/${created.interview_id}`} className="text-sm text-primary hover:underline">
                  View interview →
                </Link>
              </div>
            </>
          ) : (
            <>
              <DialogHeader>
                <DialogTitle>New interview</DialogTitle>
                <DialogDescription>
                  {selectedApp?.candidate_name} — {selectedApp?.job_title} (CV score {selectedApp?.cv_score})
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-2">
                <Label htmlFor="qcount">Question count</Label>
                <Input id="qcount" type="number" min={1} max={10} value={count} onChange={(e) => setCount(e.target.value)} />
              </div>
              <DialogFooter>
                <Button variant="secondary" onClick={() => setOpen(false)}>
                  Cancel
                </Button>
                <Button onClick={() => create.mutate()} disabled={create.isPending}>
                  {create.isPending ? "Creating…" : "Create"}
                </Button>
              </DialogFooter>
            </>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
