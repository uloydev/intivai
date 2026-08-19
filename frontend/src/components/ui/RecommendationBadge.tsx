import { CheckCircle, WarningCircle, XCircle, Question } from "@phosphor-icons/react"
import { Badge } from "./badge"
import { cn } from "@/lib/utils"

// Shared AI recommendation pill. Colors follow the verdict: hire shades are
// emerald, reject shades are destructive, anything unresolved is amber. A
// missing/empty recommendation renders a neutral "No evaluation" pill — never
// a fabricated positive verdict.
export function RecommendationBadge({
  recommendation,
  className,
}: {
  recommendation?: string | null
  className?: string
}) {
  const normalized = (recommendation ?? "").trim().toLowerCase()

  if (!normalized) {
    return (
      <Badge variant="secondary" className={cn("bg-muted text-muted-foreground border-border text-xs font-semibold gap-1 py-1 px-2.5", className)}>
        <Question className="h-3.5 w-3.5" weight="bold" /> No evaluation
      </Badge>
    )
  }

  const label = recommendation!.toUpperCase().replace("_", " ")

  if (normalized === "proceed" || normalized === "strong_hire" || normalized === "hire") {
    return (
      <Badge className={cn("bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20 text-xs font-bold gap-1 py-1 px-2.5", className)}>
        <CheckCircle className="h-3.5 w-3.5" weight="fill" /> {label}
      </Badge>
    )
  }
  if (normalized === "reject" || normalized === "no_hire") {
    return (
      <Badge variant="destructive" className={cn("text-xs font-bold gap-1 py-1 px-2.5", className)}>
        <XCircle className="h-3.5 w-3.5" weight="fill" /> {label}
      </Badge>
    )
  }
  return (
    <Badge variant="secondary" className={cn("bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20 text-xs font-bold gap-1 py-1 px-2.5", className)}>
      <WarningCircle className="h-3.5 w-3.5" weight="fill" /> {label}
    </Badge>
  )
}
