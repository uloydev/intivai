import { useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  ChatCircleText,
  Copy,
  Plus,
  ArrowSquareOut,
  Sparkle,
  CheckCircle,
  Play,
  UsersThree,
} from "@phosphor-icons/react"
import { Link } from "react-router-dom"
import { api } from "@/lib/api"
import { copyText } from "@/lib/clipboard"
import { chatInviteUrl } from "@/lib/invites"
import type { Application, CreateInterviewResult, InterviewListItem } from "@/types/api"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { RecommendationBadge } from "@/components/ui/RecommendationBadge"
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

export function InterviewsPage() {
  const qc = useQueryClient()
  const [tab, setTab] = useState<"interviews" | "eligible">("interviews")

  const { data: apps, isLoading: loadingApps } = useQuery({
    queryKey: ["applications"],
    queryFn: () => api.get<Application[]>("/applications"),
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
      qc.invalidateQueries({ queryKey: ["interviews"] })
      toast.success("Interview session initialized")
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Create failed"),
  })

  return (
    <div className="space-y-6 animate-in fade-in duration-500">
      {/* Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="font-display text-3xl font-bold tracking-tight">AI Interview Operations</h1>
          <p className="text-sm text-muted-foreground">
            Configure dynamic questioning rails, dispatch invites, and monitor real-time AI sessions.
          </p>
        </div>
        <Button
          onClick={() => {
            if (passed.length > 0) {
              setSelectedApp(passed[0])
              setCreated(null)
              setOpen(true)
            }
          }}
          disabled={passed.length === 0}
          variant="gradient"
          className="shadow-md shadow-primary/20"
        >
          <Plus className="mr-1.5 h-4 w-4" weight="bold" /> New Interview Session
        </Button>
      </div>

      {/* Navigation Tabs */}
      <div className="flex items-center gap-2 border-b border-border/60 pb-3">
        <Button
          variant={tab === "interviews" ? "secondary" : "ghost"}
          size="sm"
          className="text-xs gap-1.5"
          onClick={() => setTab("interviews")}
        >
          <ChatCircleText className="h-4 w-4" weight="bold" />
          Active & Completed Sessions ({createdInterviews?.length ?? 0})
        </Button>
        <Button
          variant={tab === "eligible" ? "secondary" : "ghost"}
          size="sm"
          className="text-xs gap-1.5"
          onClick={() => setTab("eligible")}
        >
          <UsersThree className="h-4 w-4" weight="bold" />
          Screened & Eligible Candidates ({passed.length})
        </Button>
      </div>

      {tab === "interviews" && (
        <div className="space-y-4">
          {loadingList ? (
            <Skeleton className="h-48 w-full rounded-xl" />
          ) : !createdInterviews?.length ? (
            <div className="rounded-2xl border border-dashed border-border/80 p-12 text-center">
              <ChatCircleText className="mx-auto h-12 w-12 text-muted-foreground/40 mb-3" />
              <p className="font-display font-semibold text-base">No interviews generated yet</p>
              <p className="text-xs text-muted-foreground mt-1 max-w-sm mx-auto">
                {passed.length > 0
                  ? "You have candidates who passed screening ready for interviews! Switch to 'Eligible Candidates' tab to invite them."
                  : "Upload CVs and screen candidates first. Once they pass screening, you can schedule interviews."}
              </p>
            </div>
          ) : (
            <div className="rounded-xl border border-border/60 bg-card shadow-sm overflow-hidden">
              <Table>
                <TableHeader>
                  <TableRow className="bg-muted/40">
                    <TableHead>Candidate</TableHead>
                    <TableHead>Role</TableHead>
                    <TableHead>Assessment State</TableHead>
                    <TableHead>Assessment Score</TableHead>
                    <TableHead>AI Recommendation</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {createdInterviews.map((iv) => (
                    <TableRow key={iv.interview_id} className="transition-colors hover:bg-muted/40">
                      <TableCell className="font-medium">
                        <div className="flex items-center gap-2.5">
                          <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/10 text-primary font-bold text-xs">
                            {(iv.candidate_name || "C").charAt(0).toUpperCase()}
                          </div>
                          <div>
                            <Link
                              to={`/candidates?candidate_id=${iv.candidate_id}`}
                              className="text-sm font-semibold text-foreground hover:text-primary transition-colors block"
                            >
                              {iv.candidate_name || "Candidate"}
                            </Link>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">{iv.job_title || "—"}</TableCell>
                      <TableCell>
                        <Badge
                          variant={iv.status === "completed" ? "default" : "secondary"}
                          className={
                            iv.status === "completed"
                              ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20 text-xs"
                              : "text-xs"
                          }
                        >
                          {iv.status === "completed" ? "Completed" : iv.status}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        {iv.evaluation ? (
                          <span className="font-display font-bold text-sm text-foreground">
                            {iv.evaluation.overall_score} / 100
                          </span>
                        ) : (
                          <span className="text-xs text-muted-foreground">—</span>
                        )}
                      </TableCell>
                      <TableCell>
                        {iv.evaluation ? (
                          <RecommendationBadge recommendation={iv.evaluation.recommendation} className="capitalize" />
                        ) : (
                          <span className="text-xs text-muted-foreground">Pending</span>
                        )}
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex items-center justify-end gap-2">
                          <Button asChild size="sm" variant="ghost" className="h-8 text-xs text-primary gap-1">
                            <Link to={`/interviews/${iv.interview_id}`}>
                              Scorecard <ArrowSquareOut className="h-3.5 w-3.5" />
                            </Link>
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </div>
      )}

      {tab === "eligible" && (
        <div className="space-y-4">
          {loadingApps ? (
            <Skeleton className="h-48 w-full rounded-xl" />
          ) : passed.length === 0 ? (
            <div className="rounded-2xl border border-dashed border-border/80 p-12 text-center">
              <UsersThree className="mx-auto h-12 w-12 text-muted-foreground/40 mb-3" />
              <p className="font-display font-semibold text-base">No candidates currently passed screening</p>
              <p className="text-xs text-muted-foreground mt-1 max-w-sm mx-auto">
                Candidates must achieve the minimum matching score during CV screening to be eligible for interviews.
              </p>
            </div>
          ) : (
            <div className="rounded-xl border border-border/60 bg-card shadow-sm overflow-hidden">
              <Table>
                <TableHeader>
                  <TableRow className="bg-muted/40">
                    <TableHead>Candidate</TableHead>
                    <TableHead>Target Role</TableHead>
                    <TableHead>CV Score</TableHead>
                    <TableHead className="text-right">Action</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {passed.map((app) => (
                    <TableRow key={app.id} className="transition-colors hover:bg-muted/40">
                      <TableCell className="font-medium">{app.candidate_name}</TableCell>
                      <TableCell className="text-xs text-muted-foreground">{app.job_title}</TableCell>
                      <TableCell>
                        <Badge className="bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20 font-bold">
                          {app.cv_score} / 100
                        </Badge>
                      </TableCell>
                      <TableCell className="text-right">
                        <Button
                          variant="gradient"
                          size="sm"
                          className="h-8 text-xs shadow-sm"
                          onClick={() => {
                            setSelectedApp(app)
                            setCreated(null)
                            setOpen(true)
                          }}
                        >
                          <Play className="mr-1.5 h-3.5 w-3.5" weight="fill" /> Create Interview
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </div>
      )}

      {/* Create & Invite Modal */}
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-w-md">
          {created ? (
            <>
              <DialogHeader>
                <DialogTitle className="font-display text-lg flex items-center gap-2 text-emerald-600 dark:text-emerald-400">
                  <CheckCircle className="h-6 w-6" weight="fill" /> Interview Session Active!
                </DialogTitle>
                <DialogDescription>
                  Share the secure candidate invite link below to initiate the evaluation.
                </DialogDescription>
              </DialogHeader>

              <div className="space-y-4 py-2">
                <div className="rounded-xl border border-primary/20 bg-primary/5 p-3.5 space-y-2">
                  <div className="flex items-center justify-between">
                    <Label className="text-xs font-semibold flex items-center gap-1.5">
                      <ChatCircleText className="h-4 w-4 text-primary" weight="bold" /> Chat Interview Link
                    </Label>
                    <Badge variant="secondary" className="text-[10px]">Candidate Portal</Badge>
                  </div>
                  <div className="flex gap-1.5">
                    <Input readOnly value={chatInviteUrl(created.interview_id, created.invitation_token)} className="text-xs bg-background" />
                    <Button size="sm" variant="outline" onClick={() => copyText(chatInviteUrl(created.interview_id, created.invitation_token), "Chat link")}>
                      <Copy className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                </div>

                <div className="flex justify-between items-center pt-2">
                  <Button asChild variant="ghost" size="sm" className="text-xs text-primary gap-1">
                    <Link to={`/interviews/${created.interview_id}`}>
                      View Scorecard →
                    </Link>
                  </Button>
                  <Button asChild variant="default" size="sm" className="text-xs">
                    <Link to={`/invite/${created.interview_id}?t=${encodeURIComponent(created.invitation_token)}`} target="_blank">
                      Test As Candidate <ArrowSquareOut className="ml-1 h-3.5 w-3.5" />
                    </Link>
                  </Button>
                </div>
              </div>
            </>
          ) : (
            <>
              <DialogHeader>
                <DialogTitle className="font-display text-lg flex items-center gap-2">
                  <Sparkle className="h-5 w-5 text-primary" weight="fill" /> Setup Candidate Interview
                </DialogTitle>
                <DialogDescription>
                  Configure the scope for <span className="font-semibold text-foreground">{selectedApp?.candidate_name}</span> applying for <span className="font-semibold text-foreground">{selectedApp?.job_title}</span>.
                </DialogDescription>
              </DialogHeader>

              <div className="space-y-4 py-3">
                <div className="space-y-1.5">
                  <Label htmlFor="qcount" className="text-xs font-semibold">Number of Dynamic AI Questions</Label>
                  <Input
                    id="qcount"
                    type="number"
                    min={1}
                    max={10}
                    value={count}
                    onChange={(e) => setCount(e.target.value)}
                    className="bg-background/80"
                  />
                  <p className="text-[11px] text-muted-foreground">
                    DeepSeek will synthesize CV-gap questions tailored to the candidate's missing competencies.
                  </p>
                </div>
              </div>

              <DialogFooter>
                <Button variant="secondary" onClick={() => setOpen(false)}>
                  Cancel
                </Button>
                <Button variant="gradient" onClick={() => create.mutate()} disabled={create.isPending}>
                  {create.isPending ? "Generating Rails…" : "Initialize Session"}
                </Button>
              </DialogFooter>
            </>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
