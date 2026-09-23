import { useParams, useNavigate } from "react-router-dom"
import { useQuery, useMutation } from "@tanstack/react-query"
import { CheckCircle, Warning, UserCheck } from "@phosphor-icons/react"
import { api } from "@/lib/api"
import { Button } from "@/components/ui/button"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { toast } from "sonner"
import { Skeleton } from "@/components/ui/skeleton"
import { ResumeReviewForm } from "@/components/candidates/ResumeReviewForm"
import type { CVDetail, CVDraftProfile } from "@/types/api"

export function CandidateReviewPage() {
  const { id: token } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const { data: cv, isLoading, error } = useQuery({
    queryKey: ["candidate-review", token],
    queryFn: async () => {
      const res = await api.get<CVDetail>(`/public/candidate-review/${token}`)
      return res
    },
    retry: false,
  })

  const confirmMutation = useMutation({
    mutationFn: async (draft: CVDraftProfile) => {
      return api.post(`/public/candidate-review/${token}/confirm`, draft)
    },
    onSuccess: () => {
      toast.success("Profile confirmed and submitted for screening!")
      navigate("/candidate/portal")
    },
    onError: (e) => {
      toast.error(e instanceof Error ? e.message : "Failed to confirm profile")
    },
  })

  if (isLoading) {
    return (
      <div className="max-w-3xl mx-auto py-12 px-6 space-y-6">
        <Skeleton className="h-8 w-1/3" />
        <Skeleton className="h-[400px] w-full" />
      </div>
    )
  }

  if (error || !cv) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center gap-4 bg-background text-center px-4">
        <div className="h-16 w-16 rounded-full bg-destructive/10 flex items-center justify-center text-destructive">
          <Warning weight="fill" className="h-8 w-8" />
        </div>
        <h1 className="font-display text-2xl font-bold">Review Link Invalid or Expired</h1>
        <p className="text-sm text-muted-foreground max-w-md">
          This review link is no longer valid. The profile may have already been reviewed and processed, or the link has expired.
        </p>
        <Button onClick={() => navigate("/")} variant="outline" className="mt-4">
          Return to Homepage
        </Button>
      </div>
    )
  }

  const initial = {
    ...(cv.cv_structured ?? {
      skills: [],
      experience_years: 0,
      education: "",
      certifications: [],
      summary: "",
    }),
    name: cv.name,
    email: cv.email,
  }

  return (
    <div className="max-w-4xl mx-auto py-12 px-6 animate-in fade-in duration-500 space-y-8">
      <div className="space-y-3">
        <h1 className="font-display text-3xl font-extrabold tracking-tight">
          Review Your Extracted Profile
        </h1>
        <p className="text-muted-foreground">
          Our automated parsing pipeline has extracted the following information from your resume for <strong className="text-foreground">{cv.name}</strong>. Please review and correct any inaccuracies before we proceed with the screening.
        </p>
      </div>

      <Card className="border-border bg-card shadow-sm">
        <CardHeader className="border-b border-border/50 bg-muted/30 pb-4">
          <CardTitle className="text-lg font-display flex items-center justify-between">
            <span className="flex items-center gap-2">
              <UserCheck className="h-5 w-5 text-primary" weight="bold" />
              Your Profile Details
            </span>
          </CardTitle>
        </CardHeader>
        <CardContent className="p-6">
          <ResumeReviewForm
            initial={initial}
            onSubmit={(draft) => confirmMutation.mutate(draft)}
            disabled={confirmMutation.isPending}
          />
        </CardContent>
      </Card>

      <div className="rounded-xl border border-emerald-500/20 bg-emerald-500/5 p-4 flex gap-4">
        <CheckCircle className="h-6 w-6 text-emerald-500 shrink-0" weight="fill" />
        <div className="space-y-1">
          <h4 className="text-sm font-bold text-foreground">Ready to Submit?</h4>
          <p className="text-xs text-muted-foreground leading-relaxed">
            By confirming, this profile will be used by our scoring engine to evaluate your application against the job requirements. Ensure the details accurately reflect your experience.
          </p>
        </div>
      </div>
    </div>
  )
}
