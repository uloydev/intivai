import { useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { api } from "@/lib/api"
import type { Application, CandidateReport } from "@/types/api"
import { Badge } from "@/components/ui/badge"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"

function scorePill(app: Application) {
  if (app.cv_score == null) {
    return <Badge variant="secondary" className="bg-amber-100 text-amber-800 dark:bg-amber-950 dark:text-amber-200">scoring…</Badge>
  }
  if (app.passed_screening) return <Badge className="bg-accent text-accent-foreground">{app.cv_score} — passed</Badge>
  return <Badge variant="destructive">{app.cv_score} — rejected</Badge>
}

export function CandidatesPage() {
  const { data: apps, isLoading, error } = useQuery({
    queryKey: ["applications"],
    queryFn: () => api.get<Application[]>("/applications"),
    refetchInterval: (query) =>
      query.state.data?.some((a) => a.cv_score == null) ? 2000 : false,
  })

  const [selectedId, setSelectedId] = useState<string | null>(null)
  const report = useQuery({
    queryKey: ["candidate-report", selectedId],
    queryFn: () => api.get<CandidateReport>(`/candidates/${selectedId}/report`),
    enabled: selectedId !== null,
  })
  const selected = report.data ?? null

  return (
    <div className="space-y-4">
      <h1 className="font-display text-2xl">Candidates</h1>

      {isLoading ? (
        <Skeleton className="h-64 w-full" />
      ) : error ? (
        <p className="text-destructive">{error instanceof Error ? error.message : "Failed to load candidates"}</p>
      ) : !apps?.length ? (
        <p className="py-16 text-center text-sm text-muted-foreground">No candidates yet — upload a CV first</p>
      ) : (
        <div className="rounded-md border border-border bg-card shadow-sm">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Candidate</TableHead>
                <TableHead>Job</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Score</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {apps.map((app) => (
                <TableRow key={app.id} className="cursor-pointer" onClick={() => setSelectedId(app.candidate_id)}>
                  <TableCell>
                    <p className="font-medium">{app.candidate_name}</p>
                    <p className="text-xs text-muted-foreground">{app.candidate_email}</p>
                  </TableCell>
                  <TableCell>{app.job_title}</TableCell>
                  <TableCell>
                    <Badge variant="secondary">{app.status}</Badge>
                  </TableCell>
                  <TableCell className="text-right">{scorePill(app)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      <Dialog open={selectedId !== null} onOpenChange={(open) => !open && setSelectedId(null)}>
        <DialogContent className="max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle className="font-display">{selected?.candidate.name}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            {selected?.interviews.length === 0 && (
              <p className="text-sm text-muted-foreground">No interviews yet.</p>
            )}
            {selected?.interviews.map((iv) => (
              <div key={iv.interview_id} className="rounded-md border border-border p-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">Interview {iv.status}</span>
                  <Badge variant="secondary">{iv.status}</Badge>
                </div>
                {iv.evaluation ? (
                  <div className="mt-2 space-y-1 text-sm">
                    <p>
                      Overall: <span className="font-display font-semibold">{iv.evaluation.overall_score}</span> —{" "}
                      <span className="text-muted-foreground">{iv.evaluation.recommendation}</span>
                    </p>
                    <p className="text-xs text-muted-foreground">
                      technical {iv.evaluation.dimensions.technical?.score ?? "–"} · communication{" "}
                      {iv.evaluation.dimensions.communication?.score ?? "–"} · problem solving{" "}
                      {iv.evaluation.dimensions.problem_solving?.score ?? "–"} · culture{" "}
                      {iv.evaluation.dimensions.culture_fit?.score ?? "–"}
                    </p>
                  </div>
                ) : (
                  <p className="mt-2 text-xs text-muted-foreground">Evaluation pending</p>
                )}
              </div>
            ))}
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
