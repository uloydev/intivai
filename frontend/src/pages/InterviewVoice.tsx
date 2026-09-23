import { useParams, Link } from "react-router-dom"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { MicrophoneSlash, ArrowLeft } from "@phosphor-icons/react"

// G4 decision: this page previously sent the recruiter localStorage JWT as
// ?ticket= on the voice WebSocket — the wrong credential type in the URL.
// The only browser-safe voice credential is a candidate ws_ticket minted via
// POST /candidate/interviews/:id/ticket, which requires an invitation_token
// this public route does not have (and no recruiter mint-by-interview-id
// endpoint exists). Until a proper credential path exists, the page renders
// a disabled state with guidance instead of attempting an unauthorized
// connection. NEVER put the auth JWT into the URL.
export function InterviewVoicePage() {
  const { id } = useParams<{ id: string }>()

  return (
    <div className="flex flex-col h-screen bg-background text-foreground overflow-hidden">
      <header className="flex h-14 items-center justify-between border-b border-border bg-card px-5 shrink-0">
        <div className="flex items-center gap-3">
          <Link
            to="/interviews"
            className="flex h-8 w-8 items-center justify-center rounded-lg border border-border/70 text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
          >
            <ArrowLeft className="h-4 w-4" />
          </Link>
          <span className="font-display font-bold text-sm tracking-tight text-foreground">
            Intivai Voice &amp; Coding Room
          </span>
        </div>
      </header>

      <div className="flex-1 flex items-center justify-center p-6 overflow-y-auto">
        <Card className="w-full max-w-md border-border shadow-sm">
          <CardHeader className="text-center pb-2">
            <div className="mx-auto mb-2 flex h-12 w-12 items-center justify-center rounded-xl bg-muted text-muted-foreground border border-border">
              <MicrophoneSlash size={24} />
            </div>
            <Badge variant="outline" className="mx-auto gap-1 border-warning/30 bg-warning/10 text-warning text-xs py-0.5">
              Voice Session Unavailable
            </Badge>
            <CardTitle className="font-display text-xl font-bold tracking-tight">
              Recruiter access required
            </CardTitle>
            <CardDescription className="text-xs">
              This voice session cannot be started from here.
            </CardDescription>
          </CardHeader>

          <CardContent className="space-y-3 p-5 text-xs text-muted-foreground leading-relaxed">
            <p>
              Live voice interviews are joined through the candidate&apos;s personal invite link,
              which carries the secure session credentials required to open the audio channel.
            </p>
            {id && (
              <p className="font-mono text-[10px] text-muted-foreground/80 break-all">
                Session: {id}
              </p>
            )}
            <ul className="list-disc pl-4 space-y-1">
              <li>
                Candidates: reopen the invite link you received and choose the voice session there.
              </li>
              <li>
                Recruiters: share the interview invitation from the Interviews workspace — the
                candidate launches the voice room from their own link.
              </li>
            </ul>
            <Link
              to="/interviews"
              className="inline-block pt-1 text-primary hover:underline font-medium"
            >
              ← Back to Interviews
            </Link>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
