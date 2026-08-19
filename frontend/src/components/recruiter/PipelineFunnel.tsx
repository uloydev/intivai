import { useMemo } from "react"
import { Link } from "react-router-dom"
import {
  UsersThree,
  CheckCircle,
  ChatCircleText,
  Trophy,
  TrendUp,
} from "@phosphor-icons/react"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { cn } from "@/lib/utils"

export interface PipelineFunnelProps {
  totalApplied: number
  totalScreened: number
  totalInterviewed: number
  totalRecommended: number
}

export function PipelineFunnel({
  totalApplied,
  totalScreened,
  totalInterviewed,
  totalRecommended,
}: PipelineFunnelProps) {
  const safeApplied = Math.max(0, totalApplied)
  const safeScreened = Math.max(0, totalScreened)
  const safeInterviewed = Math.max(0, totalInterviewed)
  const safeRecommended = Math.max(0, totalRecommended)

  const screenRate = safeApplied > 0 ? Math.round((safeScreened / safeApplied) * 100) : 0
  const interviewRate = safeScreened > 0 ? Math.round((safeInterviewed / safeScreened) * 100) : 0
  const offerRate = safeInterviewed > 0 ? Math.round((safeRecommended / safeInterviewed) * 100) : 0
  const overallYield = safeApplied > 0 ? Math.round((safeRecommended / safeApplied) * 100) : 0

  const stages = useMemo(
    () => [
      {
        label: "Total Sourced & Applied",
        count: safeApplied,
        pct: 100,
        rateLabel: "Top of Funnel",
        icon: UsersThree,
        color: "from-blue-500/20 to-blue-600/30 text-blue-500 border-blue-500/30",
        barColor: "bg-blue-500",
        href: "/candidates",
      },
      {
        label: "Passed AI CV Screening",
        count: safeScreened,
        pct: safeApplied > 0 ? Math.round((safeScreened / safeApplied) * 100) : 0,
        rateLabel: `${screenRate}% Qualification Rate`,
        icon: CheckCircle,
        color: "from-cyan-500/20 to-cyan-600/30 text-cyan-500 border-cyan-500/30",
        barColor: "bg-cyan-500",
        href: "/candidates?stage=screening_passed",
      },
      {
        label: "AI Assessments Completed",
        count: safeInterviewed,
        pct: safeApplied > 0 ? Math.round((safeInterviewed / safeApplied) * 100) : 0,
        rateLabel: `${interviewRate}% Conversion from Screen`,
        icon: ChatCircleText,
        color: "from-purple-500/20 to-purple-600/30 text-purple-600 border-purple-500/30 dark:text-purple-400",
        barColor: "bg-purple-500",
        // Interview status filter — the funnel count comes from completed
        // sessions, so the drill-down must land on the same exact filter.
        href: "/interviews?status=completed",
      },
      {
        label: "Strong Hire Recommendations",
        count: safeRecommended,
        pct: safeApplied > 0 ? Math.round((safeRecommended / safeApplied) * 100) : 0,
        rateLabel: `${offerRate}% Final Candidate Yield`,
        icon: Trophy,
        color: "from-emerald-500/20 to-emerald-600/30 text-emerald-500 border-emerald-500/30",
        barColor: "bg-emerald-500",
        href: "/candidates?stage=interview_completed",
      },
    ],
    [safeApplied, safeScreened, safeInterviewed, safeRecommended, screenRate, interviewRate, offerRate]
  )

  return (
    <Card className="glass border-border/60 shadow-md">
      <CardHeader className="flex flex-row items-center justify-between pb-3">
        <div>
          <div className="flex items-center gap-2">
            <CardTitle className="font-display text-base font-bold text-foreground">
              Recruitment Pipeline Velocity
            </CardTitle>
            <Badge variant="outline" className="border-primary/30 bg-primary/5 text-primary text-xs py-0.5">
              Head of HR Overview
            </Badge>
          </div>
          <CardDescription className="text-xs text-muted-foreground mt-0.5">
            End-to-end talent conversion from inbound application to hiring recommendation.
          </CardDescription>
        </div>
        <div className="flex items-center gap-2 text-xs font-semibold text-emerald-600 dark:text-emerald-400">
          <TrendUp className="h-4 w-4" weight="bold" />
          <span>Overall Yield: {overallYield}%</span>
        </div>
      </CardHeader>

      <CardContent className="space-y-4 pt-2">
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          {stages.map((stage, idx) => {
            const Icon = stage.icon
            return (
              <Link
                key={idx}
                to={stage.href}
                className="group relative flex flex-col justify-between rounded-xl border border-border/60 bg-card/50 p-3.5 transition-all duration-300 hover:border-primary/40 hover:bg-card hover:shadow-md"
              >
                <div>
                  <div className="flex items-center justify-between">
                    <div className={cn("flex h-8 w-8 items-center justify-center rounded-lg border", stage.color)}>
                      <Icon className="h-4 w-4" weight="bold" />
                    </div>
                    <span className="font-mono text-xs font-semibold text-muted-foreground group-hover:text-primary transition-colors">
                      Step 0{idx + 1}
                    </span>
                  </div>

                  <div className="mt-3">
                    <p className="font-display text-2xl font-bold tracking-tight text-foreground">
                      {stage.count}
                    </p>
                    <p className="text-xs font-medium text-muted-foreground line-clamp-1 mt-0.5">
                      {stage.label}
                    </p>
                  </div>
                </div>

                <div className="mt-3 space-y-1.5 border-t border-border/40 pt-2.5">
                  <div className="flex items-center justify-between text-xs">
                    <span className="text-muted-foreground font-medium">{stage.rateLabel}</span>
                    <span className="font-mono font-bold text-foreground">{stage.pct}%</span>
                  </div>
                  <div className="h-1.5 w-full overflow-hidden rounded-full bg-secondary">
                    <div
                      className={cn("h-full transition-all duration-700 ease-out", stage.barColor)}
                      style={{ width: `${stage.pct}%` }}
                    />
                  </div>
                </div>
              </Link>
            )
          })}
        </div>
      </CardContent>
    </Card>
  )
}
