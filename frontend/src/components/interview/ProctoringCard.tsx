import { CheckCircle, Info, ShieldCheck, WarningCircle } from "@phosphor-icons/react"
import { Badge } from "@/components/ui/badge"
import { Card } from "@/components/ui/card"
import { cn } from "@/lib/utils"
import type { ProctoringEvent, ProctoringSummary } from "@/types/api"

interface ProctoringCardProps {
  summary?: ProctoringSummary
  events?: ProctoringEvent[]
}

// Client-side integrity telemetry: the summary is assembled from browser
// events reported over the WS/REST pipe, so the score is always
// client-reported and can never be treated as verified.
export function ProctoringCard({ summary, events }: ProctoringCardProps) {
  const resolved = summary ?? {
    integrity_score: 0,
    risk_level: "low" as const,
    tab_switch_count: 0,
    total_away_duration_sec: 0,
    paste_event_count: 0,
    suspicious_paste_count: 0,
    audio_anomaly_count: 0,
    flags: [] as string[],
  }

  const isClean = resolved.integrity_score >= 85
  const isMed = resolved.integrity_score >= 60 && resolved.integrity_score < 85

  return (
    <Card className="glass border-border/60 overflow-hidden shadow-sm">
      <div className="p-6 space-y-5">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 border-b border-border/50 pb-4">
          <div className="flex items-center gap-3">
            <div
              className={cn(
                "flex h-10 w-10 items-center justify-center rounded-xl font-bold",
                isClean
                  ? "bg-emerald-500/10 text-emerald-500"
                  : isMed
                  ? "bg-amber-500/10 text-amber-500"
                  : "bg-destructive/10 text-destructive"
              )}
            >
              <ShieldCheck className="h-5 w-5" weight="bold" />
            </div>
            <div>
              <h3 className="font-display font-bold text-base tracking-tight">AI Proctoring & Integrity Audit</h3>
              <p className="text-xs text-muted-foreground">
                Real-time telemetry tracking window focus, clipboard behavior, and audio integrity
              </p>
            </div>
          </div>

          <div className="flex flex-col items-end gap-1">
            <div className="flex items-center gap-2.5">
              <Badge
                variant="outline"
                className={cn(
                  "font-mono text-xs px-2.5 py-1 font-bold",
                  isClean
                    ? "border-emerald-500/30 text-emerald-600 dark:text-emerald-400 bg-emerald-500/5"
                    : isMed
                    ? "border-amber-500/30 text-amber-600 dark:text-amber-400 bg-amber-500/5"
                    : "border-destructive/30 text-destructive bg-destructive/5"
                )}
              >
                Integrity: {resolved.integrity_score}/100
              </Badge>
              <Badge
                className={cn(
                  "text-[10px] font-bold uppercase tracking-wider",
                  isClean
                    ? "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400"
                    : isMed
                    ? "bg-amber-500/15 text-amber-600 dark:text-amber-400"
                    : "bg-destructive/15 text-destructive"
                )}
              >
                {resolved.risk_level} Risk
              </Badge>
            </div>
            {summary && (
              <span className="text-[10px] font-medium text-muted-foreground">
                Client-reported · unverified
              </span>
            )}
          </div>
        </div>

        {/* Quick Metrics Grid */}
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <div className="rounded-xl border border-border/40 bg-muted/20 p-3">
            <span className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider block">
              Tab Switches
            </span>
            <span className="font-display text-lg font-bold text-foreground mt-0.5 block">
              {resolved.tab_switch_count}
            </span>
          </div>
          <div className="rounded-xl border border-border/40 bg-muted/20 p-3">
            <span className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider block">
              Time Out of Focus
            </span>
            <span className="font-display text-lg font-bold text-foreground mt-0.5 block">
              {resolved.total_away_duration_sec}s
            </span>
          </div>
          <div className="rounded-xl border border-border/40 bg-muted/20 p-3">
            <span className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider block">
              Clipboard Pastes
            </span>
            <span className="font-display text-lg font-bold text-foreground mt-0.5 block">
              {resolved.paste_event_count}{" "}
              <span className="text-xs font-normal text-muted-foreground">
                ({resolved.suspicious_paste_count} large)
              </span>
            </span>
          </div>
          <div className="rounded-xl border border-border/40 bg-muted/20 p-3">
            <span className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider block">
              Audio Anomalies
            </span>
            <span className="font-display text-lg font-bold text-foreground mt-0.5 block">
              {resolved.audio_anomaly_count}
            </span>
          </div>
        </div>

        {/* Flags / Audit Details */}
        {resolved.flags && resolved.flags.length > 0 ? (
          <div className="rounded-xl border border-amber-500/20 bg-amber-500/5 p-4 space-y-2">
            <p className="text-xs font-bold text-amber-600 dark:text-amber-400 flex items-center gap-1.5 uppercase tracking-wider">
              <WarningCircle className="h-4 w-4" weight="fill" /> Flagged Integrity Telemetry
            </p>
            <ul className="space-y-1 text-xs text-foreground/90 pl-1">
              {resolved.flags.map((flag, idx) => (
                <li key={idx} className="flex items-start gap-1.5">
                  <span className="text-amber-500 font-bold">•</span>
                  <span>{flag}</span>
                </li>
              ))}
            </ul>
          </div>
        ) : resolved.integrity_score === 0 && (!events || events.length === 0) ? (
          <div className="rounded-xl border border-muted-foreground/20 bg-muted/50 p-3.5 flex items-center gap-2 text-xs text-muted-foreground">
            <Info className="h-4 w-4 shrink-0" weight="fill" />
            <span>No telemetry data available for this session.</span>
          </div>
        ) : (
          <div className="rounded-xl border border-emerald-500/20 bg-emerald-500/5 p-3.5 flex items-center gap-2 text-xs text-emerald-600 dark:text-emerald-400">
            <CheckCircle className="h-4 w-4 shrink-0" weight="fill" />
            <span>No integrity anomalies detected.</span>
          </div>
        )}

        {/* Telemetry Event Stream */}
        {events && events.length > 0 && (
          <div className="pt-2 space-y-2">
            <p className="text-xs font-bold text-muted-foreground uppercase tracking-wider">
              Chronological Telemetry Stream ({events.length} events)
            </p>
            <div className="max-h-40 overflow-y-auto rounded-xl border border-border/40 bg-background/50 divide-y divide-border/30 text-xs">
              {events.map((ev, idx) => (
                <div key={idx} className="p-2.5 flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="font-mono text-[10px] text-muted-foreground">
                      {new Date(ev.timestamp).toLocaleTimeString()}
                    </span>
                    <Badge variant="outline" className="text-[10px] py-0 px-1.5 capitalize">
                      {ev.type.replace("_", " ")}
                    </Badge>
                    {ev.question_idx && (
                      <span className="text-muted-foreground text-[11px]">Q{ev.question_idx}</span>
                    )}
                  </div>
                  {ev.details && Object.keys(ev.details).length > 0 && (
                    <span className="text-[11px] text-muted-foreground truncate max-w-xs">
                      {JSON.stringify(ev.details)}
                    </span>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </Card>
  )
}
