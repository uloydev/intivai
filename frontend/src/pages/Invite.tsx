import { useState } from "react"
import { useNavigate, useParams, useSearchParams } from "react-router-dom"
import { api } from "@/lib/api"
import type { ConsentResult } from "@/types/api"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import { Label } from "@/components/ui/label"
import { toast } from "sonner"

// /invite/:interviewID?t=<invitation_token> — consent gate, then ticket
// exchange, then straight into the chat.
export function InvitePage() {
  const { id } = useParams<{ id: string }>()
  const [params] = useSearchParams()
  const token = params.get("t") ?? ""
  const navigate = useNavigate()

  const [consented, setConsented] = useState(false)
  const [busy, setBusy] = useState(false)

  async function start() {
    if (!id || !token) return
    setBusy(true)
    try {
      await api.post<ConsentResult>(`/candidate/interviews/${id}/consent`, { invitation_token: token })
      // Exchange the invitation token for a short-lived WS ticket — the chat
      // authenticates with the ticket, not the invitation token.
      const ticket = await api.post<{ ticket: string }>(`/candidate/interviews/${id}/ticket`, {
        invitation_token: token,
      })
      navigate(`/chat/${id}?t=${encodeURIComponent(ticket.ticket)}`)
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Could not start the interview")
      setBusy(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle className="font-display text-xl">Interview invitation</CardTitle>
          <CardDescription>
            Your interview is ready. It takes about 30 minutes and is conducted by an AI interviewer.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <ul className="list-disc space-y-1 pl-5 text-sm text-muted-foreground">
            <li>Answer questions one at a time — type or use the mic-free chat</li>
            <li>You can interrupt an answer and resume later from the same question</li>
            <li>Your answers are used only to evaluate your fit for this role</li>
          </ul>
          <div className="flex items-start gap-3 rounded-md border border-border p-3">
            <Checkbox
              id="consent"
              checked={consented}
              onCheckedChange={(v) => setConsented(v === true)}
              aria-label="I consent"
            />
            <Label htmlFor="consent" className="text-sm leading-relaxed">
              I consent to my answers being recorded and used for this interview's evaluation.
            </Label>
          </div>
          <p className="text-xs text-muted-foreground">
            By starting you agree to the interview privacy notice — your answers are used only for this
            role's evaluation and are accessible to you on request.
          </p>
          <Button className="w-full" onClick={start} disabled={!consented || busy || !token}>
            {busy ? "Starting…" : "Start interview"}
          </Button>
          {!token && <p className="text-sm text-destructive">This invite link is missing its token — ask the recruiter for a fresh link.</p>}
        </CardContent>
      </Card>
    </div>
  )
}
