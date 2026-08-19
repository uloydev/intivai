import { useState } from "react"
import { useNavigate, useParams, useSearchParams } from "react-router-dom"
import {
  Sparkle,
  ArrowRight,
  SpinnerGap,
  Clock,
  CheckCircle,
} from "@phosphor-icons/react"
import { api } from "@/lib/api"
import type { ConsentResult } from "@/types/api"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import { Label } from "@/components/ui/label"
import { Badge } from "@/components/ui/badge"
import { toast } from "sonner"

export function InvitePage() {
  const { id } = useParams<{ id: string }>()
  const [params] = useSearchParams()
  const token = params.get("t") ?? ""
  const navigate = useNavigate()

  const [consented, setConsented] = useState(false)
  const [busy, setBusy] = useState(false)
  // Persistent failure state — an expired or already-consumed invitation is
  // not a transient toast: the page must tell the candidate what happened and
  // how to get a fresh link, and must not re-enable the button.
  const [failed, setFailed] = useState(false)

  async function start() {
    if (!id || !token) return
    setBusy(true)
    setFailed(false)
    try {
      await api.post<ConsentResult>(`/candidate/interviews/${id}/consent`, { invitation_token: token })
      const ticket = await api.post<{ ticket: string }>(`/candidate/interviews/${id}/ticket`, {
        invitation_token: token,
      })
      navigate(`/chat/${id}?t=${encodeURIComponent(ticket.ticket)}`)
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Could not start the interview")
      setFailed(true)
      setBusy(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-[radial-gradient(ellipse_at_top,_var(--tw-gradient-stops))] from-primary/10 via-background to-background p-4 animate-in fade-in duration-500">
      <div className="w-full max-w-lg space-y-4">
        <Card className="glass border-primary/20 shadow-2xl shadow-primary/10 overflow-hidden relative">
          <CardHeader className="text-center pb-2">
            <div className="mx-auto mb-2 flex h-12 w-12 items-center justify-center rounded-2xl bg-primary text-primary-foreground font-bold font-display text-xl shadow-lg shadow-primary/25">
              I
            </div>
            <div className="flex items-center justify-center gap-1.5 mb-1">
              <Badge variant="outline" className="gap-1 border-primary/30 bg-primary/5 text-primary text-xs py-0.5">
                <Sparkle className="h-3 w-3" weight="fill" /> Candidate Portal
              </Badge>
            </div>
            <CardTitle className="font-display text-2xl font-bold tracking-tight">Interview Invitation</CardTitle>
            <CardDescription className="text-xs">
              Welcome! You've been invited to complete an interactive AI screening interview.
            </CardDescription>
          </CardHeader>

          <CardContent className="space-y-5 pt-2">
            {/* Guide box */}
            <div className="rounded-xl border border-border/60 bg-muted/30 p-4 space-y-2.5 text-xs text-muted-foreground">
              <p className="font-semibold text-foreground flex items-center gap-1.5 text-xs">
                <Clock className="h-4 w-4 text-primary" /> What to expect:
              </p>
              <ul className="space-y-1.5 pl-1">
                <li className="flex items-start gap-2">
                  <CheckCircle className="h-4 w-4 text-primary shrink-0 mt-0.5" weight="fill" />
                  <span>Answer dynamic technical questions one at a time via keyboard.</span>
                </li>
                <li className="flex items-start gap-2">
                  <CheckCircle className="h-4 w-4 text-primary shrink-0 mt-0.5" weight="fill" />
                  <span>Session takes approximately 15-20 minutes with automatic save & resume support.</span>
                </li>
                <li className="flex items-start gap-2">
                  <CheckCircle className="h-4 w-4 text-primary shrink-0 mt-0.5" weight="fill" />
                  <span>Your responses are evaluated against core competencies and delivered directly to the hiring team.</span>
                </li>
              </ul>
            </div>

            {/* GDPR Consent Box */}
            <div className="flex items-start gap-3 rounded-xl border border-primary/20 bg-primary/5 p-3.5">
              <Checkbox
                id="consent"
                checked={consented}
                onCheckedChange={(v) => setConsented(v === true)}
                className="mt-0.5"
                aria-label="I consent"
              />
              <div className="space-y-1">
                <Label htmlFor="consent" className="text-xs leading-relaxed text-foreground cursor-pointer font-medium">
                  I consent to my answers being evaluated by AI, and I acknowledge that the following telemetry is
                  collected during the session: tab switching, paste events, and window focus loss.
                </Label>
                <p className="text-[10px] text-muted-foreground leading-relaxed">
                  Flagged events are reviewed by a human recruiter before any hiring decision is made.
                </p>
              </div>
            </div>

            <Button
              className="w-full font-semibold shadow-md shadow-primary/20"
              variant="gradient"
              size="lg"
              onClick={start}
              disabled={!consented || busy || !token || failed}
            >
              {busy ? (
                <>
                  <SpinnerGap className="mr-2 h-4 w-4 animate-spin" /> Preparing AI Rails…
                </>
              ) : (
                <>
                  Begin Interview Session <ArrowRight className="ml-2 h-4 w-4" weight="bold" />
                </>
              )}
            </Button>

            {failed && (
              <div
                role="alert"
                className="rounded-xl border border-destructive/30 bg-destructive/5 p-4 text-center text-xs text-destructive space-y-1"
              >
                <p className="font-semibold">This invitation has expired or already been used.</p>
                <p className="text-destructive/80">Contact the hiring team to request a new one.</p>
              </div>
            )}

            {!token && (
              <div className="rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-center text-xs text-destructive">
                This invitation link is missing its authorization token. Please request a fresh invite from the recruiter.
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
