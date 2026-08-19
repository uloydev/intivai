import { useState } from "react"
import { Calculator, CheckCircle } from "@phosphor-icons/react"
import { Card } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"

const HOURS_SAVED_PER_INTERVIEW = 4.5
const ENGINEER_HOURLY_COST_USD = 85

export function RoiCalculator() {
  const [hiresPerMonth, setHiresPerMonth] = useState(15)
  const hoursSavedPerMonth = hiresPerMonth * HOURS_SAVED_PER_INTERVIEW
  const costSavingsPerMonth = hoursSavedPerMonth * ENGINEER_HOURLY_COST_USD

  return (
    <section id="calculator" className="scroll-mt-24 px-6 max-w-4xl mx-auto">
      <Card className="glass border-primary/30 p-8 md:p-10 shadow-xl shadow-primary/5 rounded-3xl space-y-6">
        <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4 border-b border-border/50 pb-6">
          <div className="space-y-1">
            <Badge variant="outline" className="text-primary border-primary/30 bg-primary/5 text-xs">
              Recruitment Efficiency Calculator
            </Badge>
            <h3 className="font-display font-bold text-2xl">Calculate Your Engineering Time Saved</h3>
          </div>
          <div className="flex items-center gap-2 text-primary">
            <Calculator className="h-8 w-8" weight="duotone" />
          </div>
        </div>

        <div className="grid gap-6 md:grid-cols-2 items-center">
          <div className="space-y-4">
            <div>
              <label className="text-xs font-semibold text-foreground uppercase tracking-wider block mb-2">
                Technical Interviews Conducted Per Month: <span className="text-primary font-bold text-base ml-1">{hiresPerMonth}</span>
              </label>
              <input
                type="range"
                min="5"
                max="100"
                step="5"
                value={hiresPerMonth}
                onChange={(e) => setHiresPerMonth(Number(e.target.value))}
                className="w-full h-2 bg-muted rounded-lg appearance-none cursor-pointer accent-primary"
              />
              <div className="flex justify-between text-[11px] text-muted-foreground mt-1">
                <span>5 interviews</span>
                <span>50 interviews</span>
                <span>100+ interviews</span>
              </div>
            </div>

            <div className="space-y-2 text-xs text-muted-foreground">
              <p className="flex items-center gap-2">
                <CheckCircle className="h-4 w-4 text-emerald-500 shrink-0" weight="fill" />
                <span>Saves {HOURS_SAVED_PER_INTERVIEW} engineering hours per candidate screened</span>
              </p>
              <p className="flex items-center gap-2">
                <CheckCircle className="h-4 w-4 text-emerald-500 shrink-0" weight="fill" />
                <span>Eliminates recruiter scheduling bottlenecks</span>
              </p>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3 p-4 rounded-2xl bg-muted/40 border border-border/50 text-center">
            <div className="p-3 bg-card rounded-xl border border-border/40">
              <span className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider block">
                Dev Hours Saved
              </span>
              <span className="font-display text-2xl sm:text-3xl font-extrabold text-primary block mt-1">
                {hoursSavedPerMonth}h
              </span>
              <span className="text-[10px] text-muted-foreground">per month</span>
            </div>
            <div className="p-3 bg-card rounded-xl border border-border/40">
              <span className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider block">
                Estimated Savings
              </span>
              <span className="font-display text-2xl sm:text-3xl font-extrabold text-emerald-500 block mt-1">
                ${costSavingsPerMonth.toLocaleString()}
              </span>
              <span className="text-[10px] text-muted-foreground">in eng bandwidth</span>
            </div>
          </div>
        </div>
      </Card>
    </section>
  )
}
