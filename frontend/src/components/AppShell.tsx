import { NavLink, Outlet, useLocation } from "react-router-dom"
import {
  SquaresFour,
  Briefcase,
  Files,
  UsersThree,
  ChatCircleText,
  Brain,
  SignOut,
  Sun,
  Moon,
} from "@phosphor-icons/react"
import { getSession, logout } from "@/lib/auth"
import { useTheme } from "@/lib/theme"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { cn } from "@/lib/utils"
import { useMemo } from "react"

const nav = [
  { to: "/dashboard", label: "Dashboard", icon: SquaresFour },
  { to: "/jobs", label: "Jobs", icon: Briefcase },
  { to: "/cvs", label: "CV Ingestion", icon: Files },
  { to: "/candidates", label: "Candidates", icon: UsersThree },
  { to: "/interviews", label: "Interviews", icon: ChatCircleText },
  { to: "/company-context", label: "AI Rails", icon: Brain },
]

export function AppShell() {
  const { theme, toggle } = useTheme()
  const location = useLocation()

  const session = useMemo(() => getSession(), [location.pathname])
  const orgLabel = session?.orgId ? `Workspace ${session.orgId.slice(0, 8)}` : "Intivai Workspace"
  const roleLabel = (session?.role ?? "member").toUpperCase()
  const email = (() => {
    const token = session?.token
    if (!token) return "Signed out"
    try {
      const claims = JSON.parse(atob(token.split(".")[1]))
      return String(claims.email ?? claims.sub ?? "")
    } catch {
      return ""
    }
  })()

  return (
    <div className="flex min-h-screen bg-background bg-[radial-gradient(ellipse_at_top,_var(--tw-gradient-stops))] from-primary/5 via-background to-background text-foreground">
      {/* Desktop Sidebar */}
      <aside className="hidden w-64 shrink-0 flex-col border-r border-border/50 bg-background/70 backdrop-blur-xl md:flex z-10">
        <div className="flex h-16 items-center justify-between px-6 border-b border-border/40">
          <div className="flex items-center gap-2.5">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground font-bold font-display shadow-md shadow-primary/20">
              I
            </div>
            <span className="font-display text-lg font-bold tracking-tight bg-gradient-to-r from-primary via-blue-500 to-indigo-500 bg-clip-text text-transparent">
              Intivai
            </span>
          </div>
          <Button
            variant="ghost"
            size="icon"
            className="rounded-full h-8 w-8 text-muted-foreground hover:text-foreground"
            aria-label="Toggle dark mode"
            onClick={toggle}
          >
            {theme === "dark" ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
          </Button>
        </div>

        {/* User / Org pill — from the real session, not hardcoded */}
        <div className="mx-4 my-3 rounded-xl border border-border/50 bg-muted/30 p-3">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold truncate">{orgLabel}</span>
            <Badge variant="outline" className="text-[10px] py-0 px-1.5 border-primary/30 text-primary bg-primary/5">
              {roleLabel}
            </Badge>
          </div>
          <p className="text-[11px] text-muted-foreground truncate mt-0.5">{email}</p>
        </div>

        {/* Navigation Items */}
        <nav className="flex-1 space-y-1 px-3 pt-1">
          {nav.map((item) => {
            const active = location.pathname.startsWith(item.to)
            return (
              <NavLink
                key={item.to}
                to={item.to}
                className={cn(
                  "flex items-center gap-3 rounded-xl px-3 py-2.5 text-xs font-semibold transition-all active:scale-[0.98]",
                  active
                    ? "bg-primary text-primary-foreground shadow-md shadow-primary/20"
                    : "text-muted-foreground hover:bg-muted/60 hover:text-foreground"
                )}
              >
                <item.icon className="h-4 w-4" weight={active ? "fill" : "bold"} />
                {item.label}
              </NavLink>
            )
          })}
        </nav>

        {/* Sign Out */}
        <div className="p-3 border-t border-border/40">
          <Button
            variant="ghost"
            size="sm"
            className="w-full justify-start text-xs text-muted-foreground hover:bg-muted/60 hover:text-destructive rounded-lg transition-all active:scale-[0.98]"
            onClick={() => {
              logout()
              window.location.assign("/login")
            }}
          >
            <SignOut className="mr-2.5 h-4 w-4" />
            Sign out
          </Button>
        </div>
      </aside>

      {/* Main Content Area */}
      <div className="flex min-w-0 flex-1 flex-col relative">
        {/* Mobile Top Bar */}
        <header className="flex h-14 items-center justify-between border-b border-border/50 bg-background/80 backdrop-blur-xl px-4 md:hidden sticky top-0 z-20">
          <div className="flex items-center gap-2">
            <div className="flex h-7 w-7 items-center justify-center rounded-md bg-primary text-primary-foreground font-bold font-display text-xs">
              I
            </div>
            <span className="font-display text-base font-bold tracking-tight bg-gradient-to-r from-primary to-blue-500 bg-clip-text text-transparent">
              Intivai
            </span>
          </div>
          <div className="flex items-center gap-1.5">
            <Button
              variant="ghost"
              size="icon"
              className="rounded-full h-8 w-8"
              aria-label="Toggle dark mode"
              onClick={toggle}
            >
              {theme === "dark" ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="rounded-full h-8 w-8 text-muted-foreground hover:text-destructive"
              onClick={() => {
                logout()
                window.location.assign("/login")
              }}
            >
              <SignOut className="h-4 w-4" />
            </Button>
          </div>
        </header>

        {/* Page Container */}
        <main className="flex-1 p-4 md:p-8 w-full max-w-6xl mx-auto">
          <Outlet />
        </main>

        {/* Mobile Bottom Navigation */}
        <nav className="flex border-t border-border/50 bg-background/80 backdrop-blur-xl md:hidden sticky bottom-0 z-20 pb-safe">
          {nav.map((item) => {
            const active = location.pathname.startsWith(item.to)
            return (
              <NavLink
                key={item.to}
                to={item.to}
                className={cn(
                  "flex flex-1 flex-col items-center gap-1 py-2.5 text-[10px] font-medium transition-all active:scale-[0.95]",
                  active ? "text-primary font-bold" : "text-muted-foreground"
                )}
              >
                <item.icon className="h-5 w-5" weight={active ? "fill" : "bold"} />
                {item.label}
              </NavLink>
            )
          })}
        </nav>
      </div>
    </div>
  )
}
