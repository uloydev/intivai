import { useQuery } from "@tanstack/react-query"
import { useParams } from "react-router-dom"
import { api } from "@/lib/api"
import type { InterviewDetail } from "@/types/api"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"

export function InterviewResultPage() {
  const { id } = useParams<{ id: string }>()
  const { data: detail, isLoading, error } = useQuery({
    queryKey: ["interview", id],
    queryFn: () => api.get<InterviewDetail>(`/interviews/${id}`),
  })

  if (isLoading) {
    return (
      <div className="space-y-2">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-40 w-full" />
      </div>
    )
  }
  if (error || !detail) {
    return <p className="text-destructive">{error instanceof Error ? error.message : "Interview not found"}</p>
  }

  const evalReport = detail.evaluation

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="font-display text-2xl">
          {detail.candidate?.name ?? "Interview"} — {detail.job?.title ?? "—"}
        </h1>
        <Badge variant="secondary">{detail.status}</Badge>
      </div>

      {evalReport ? (
        <div className="rounded-md border border-border bg-card p-4 shadow-sm">
          <div className="flex flex-wrap items-center gap-4">
            <div>
              <p className="text-xs text-muted-foreground">Overall</p>
              <p className="font-display text-3xl font-semibold">{evalReport.overall_score}</p>
            </div>
            <Badge className={evalReport.recommendation === "proceed" ? "bg-accent text-accent-foreground" : "bg-amber-100 text-amber-800 dark:bg-amber-950 dark:text-amber-200"}>
              {evalReport.recommendation}
            </Badge>
          </div>
          <div className="mt-4 grid grid-cols-2 gap-3 md:grid-cols-4">
            {Object.entries(evalReport.dimensions).map(([name, dim]) => (
              <div key={name} className="rounded-md bg-muted p-3">
                <p className="text-xs capitalize text-muted-foreground">{name.replace("_", " ")}</p>
                <p className="font-display text-xl font-semibold">{dim.score}</p>
              </div>
            ))}
          </div>
          {(evalReport.strengths.length > 0 || evalReport.weaknesses.length > 0) && (
            <div className="mt-4 grid gap-3 md:grid-cols-2">
              {evalReport.strengths.length > 0 && (
                <div>
                  <p className="mb-1 text-sm font-medium text-accent">Strengths</p>
                  <ul className="list-disc space-y-0.5 pl-5 text-sm text-muted-foreground">
                    {evalReport.strengths.map((s) => (
                      <li key={s}>{s}</li>
                    ))}
                  </ul>
                </div>
              )}
              {evalReport.weaknesses.length > 0 && (
                <div>
                  <p className="mb-1 text-sm font-medium text-amber-700 dark:text-amber-300">To watch</p>
                  <ul className="list-disc space-y-0.5 pl-5 text-sm text-muted-foreground">
                    {evalReport.weaknesses.map((s) => (
                      <li key={s}>{s}</li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          )}
        </div>
      ) : (
        <p className="text-sm text-muted-foreground">Evaluation pending — it lands shortly after the interview completes.</p>
      )}

      <div className="space-y-3">
        {detail.answers.map((answer) => {
          const question = detail.questions.find((q) => q.idx === answer.idx)
          const perQ = evalReport?.per_question.find((p) => p.question_idx === answer.idx)
          return (
            <div key={answer.idx} className="rounded-md border border-border bg-card p-4 shadow-sm">
              <p className="font-medium">
                Q{answer.idx}. {question?.content ?? "—"}
                {perQ && (
                  <Badge variant="secondary" className="ml-2">
                    {perQ.score}
                  </Badge>
                )}
              </p>
              <p className="mt-1 text-sm text-muted-foreground">{answer.content}</p>
              {perQ?.rationale && <p className="mt-1 text-xs text-muted-foreground">{perQ.rationale}</p>}
            </div>
          )
        })}
      </div>
    </div>
  )
}
