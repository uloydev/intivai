import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import { CommandPalette } from "@/components/CommandPalette"
import {
  SquaresFour,
  Briefcase,
  Files,
  UsersThree,
  ChatCircleText,
  Brain,
  GearSix,
  SignOut,
  Sun,
  Moon,
  DotsThree,
  Bell,
} from "@phosphor-icons/react"
import { getSession, logout, decodePayload } from "@/lib/auth"
import { useTheme } from "@/lib/theme-context"
import { api } from "@/lib/api"
import type { RecruiterNotification } from "@/types/api"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { cn } from "@/lib/utils"
import { useState } from "react"

const nav = [
  { to: "/dashboard", label: "Dashboard", icon: SquaresFour },
  { to: "/jobs", label: "Jobs", icon: Briefcase },
  { to: "/cvs", label: "CV Ingestion", icon: Files },
  { to: "/candidates", label: "Candidates", icon: UsersThree },
  { to: "/interviews", label: "Interviews", icon: ChatCircleText },
  { to: "/company-context", label: "AI Rails", icon: Brain },
  { to: "/settings", label: "Settings", icon: GearSix },
]

const primaryNav = nav.slice(0, 4)
const overflowNav = nav.slice(4)

export function AppShell() {
  const { theme, toggle } = useTheme()
  const location = useLocation()
  const navigate = useNavigate()
  const [showMore, setShowMore] = useState(false)
  const [showNotifications, setShowNotifications] = useState(false)

  const session = getSession()
  const qc = useQueryClient()
  const { data: notifData } = useQuery({
    queryKey: ["notifications"],
    queryFn: () =>
      api.get<{ notifications: RecruiterNotification[]; unread_count: number }>("/notifications"),
    refetchInterval: 30000,
    retry: false,
    enabled: Boolean(session?.token),
  })

  const markAllRead = () => {
    api.post("/notifications/read-all").then(() => {
      qc.invalidateQueries({ queryKey: ["notifications"] })
    }).catch(() => null)
  }

  const handleNotificationClick = (n: RecruiterNotification) => {
    api.patch(`/notifications/${n.id}/read`, {}).catch(() => null)
    qc.invalidateQueries({ queryKey: ["notifications"] })
    setShowNotifications(false)
    if (n.action_url) {
      navigate(n.action_url)
    }
  }

  const orgLabel = session?.orgId ? `Workspace` : "Intivai Workspace"
  const roleLabel = (session?.role ?? "member").toUpperCase()
  const email = (() => {
    const token = session?.token
    if (!token) return "Signed out"
    const claims = decodePayload(token.split(".")[1])
    return String(claims?.email ?? claims?.sub ?? "")
  })()

  return (
    <div className="flex min-h-screen bg-background text-foreground">
      <CommandPalette />
      {/* Desktop Sidebar */}
      <aside className="hidden w-64 shrink-0 flex-col border-r border-border bg-card md:flex z-10">
        <div className="flex h-16 items-center justify-between px-6 border-b border-border/60">
          <div className="flex items-center gap-2.5">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground font-bold font-display shadow-sm">
              I
            </div>
            <span className="font-display text-lg font-bold tracking-tight text-foreground">
              Intivai
            </span>
          </div>
          <div className="flex items-center gap-1">
            <Button
              variant="ghost"
              size="icon"
              className="rounded-full h-8 w-8 text-muted-foreground hover:text-foreground relative"
              aria-label="Recruiter notifications"
              onClick={() => setShowNotifications((prev) => !prev)}
            >
              <Bell className="h-4 w-4" />
              {(notifData?.unread_count ?? 0) > 0 && (
                <span className="absolute top-1.5 right-1.5 flex h-2 w-2 rounded-full bg-primary" />
              )}
            </Button>
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
        </div>

        {/* User / Org pill — from the real session, not hardcoded */}
        <div className="mx-4 my-3 rounded-xl border border-border/60 bg-muted/40 p-3">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold truncate">{orgLabel}</span>
            <Badge variant="outline" className="text-[10px] py-0 px-1.5 border-border text-muted-foreground bg-background">
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
                    ? "bg-primary text-primary-foreground shadow-sm"
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
        <div className="p-3 border-t border-border/60">
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
        <header className="flex h-14 items-center justify-between border-b border-border bg-card px-4 md:hidden sticky top-0 z-20">
          <div className="flex items-center gap-2">
            <div className="flex h-7 w-7 items-center justify-center rounded-md bg-primary text-primary-foreground font-bold font-display text-xs">
              I
            </div>
            <span className="font-display text-base font-bold tracking-tight text-foreground">
              Intivai
            </span>
          </div>
          <div className="flex items-center gap-1">
            <Button
              variant="ghost"
              size="icon"
              className="rounded-full h-8 w-8 text-muted-foreground hover:text-foreground relative"
              aria-label="Recruiter notifications"
              onClick={() => setShowNotifications((prev) => !prev)}
            >
              <Bell className="h-4 w-4" />
              {(notifData?.unread_count ?? 0) > 0 && (
                <span className="absolute top-1.5 right-1.5 flex h-2 w-2 rounded-full bg-primary" />
              )}
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="rounded-full h-8 w-8 transition-transform duration-300"
              aria-label="Toggle dark mode"
              onClick={toggle}
            >
              <span className={cn("transition-transform duration-300 inline-block", theme === "dark" && "rotate-180")}>
                {theme === "dark" ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
              </span>
            </Button>
            <Button
              variant="ghost"
              size="icon"
              aria-label="Sign out"
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
        <main className="flex-1 p-4 sm:p-6 md:p-8 w-full max-w-7xl 2xl:max-w-[1600px] mx-auto">
          <Outlet />
        </main>

        {/* Mobile Bottom Navigation */}
        <nav className="flex border-t border-border bg-card md:hidden sticky bottom-0 z-20 pb-safe">
          {primaryNav.map((item) => {
            const active = location.pathname.startsWith(item.to)
            return (
              <NavLink
                key={item.to}
                to={item.to}
                className={cn(
                  "flex flex-1 flex-col items-center justify-center gap-1 min-h-11 py-1 text-[10px] font-medium transition-all active:scale-[0.95]",
                  active ? "text-primary font-bold" : "text-muted-foreground"
                )}
              >
                <item.icon className="h-5 w-5" weight={active ? "fill" : "bold"} />
                {item.label}
              </NavLink>
            )
          })}
          <button
            type="button"
            onClick={() => setShowMore(!showMore)}
            className={cn(
              "flex flex-1 flex-col items-center justify-center gap-1 min-h-11 py-1 text-[10px] font-medium transition-all active:scale-[0.95]",
              showMore ? "text-primary font-bold" : "text-muted-foreground"
            )}
          >
            <DotsThree className="h-5 w-5" weight="bold" />
            More
          </button>
        </nav>

        {/* Mobile More Menu */}
        {showMore && (
          <div className="md:hidden border-t border-border bg-card sticky bottom-0 z-20 p-3 space-y-1">
            {overflowNav.map((item) => {
              const active = location.pathname.startsWith(item.to)
              return (
                <NavLink
                  key={item.to}
                  to={item.to}
                  onClick={() => setShowMore(false)}
                  className={cn(
                    "flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors",
                    active ? "bg-primary/10 text-primary" : "text-muted-foreground hover:bg-muted/50"
                  )}
                >
                  <item.icon className="h-5 w-5" weight={active ? "fill" : "bold"} />
                  {item.label}
                </NavLink>
              )
            })}
          </div>
        )}

        {/* Recruiter Notifications Popover */}
        {showNotifications && (
          <div
            className="fixed inset-0 z-50 flex items-start justify-end p-4 sm:p-6"
            onClick={() => setShowNotifications(false)}
          >
            <div
              className="w-full max-w-sm rounded-xl border border-border bg-card shadow-lg p-4 space-y-3 animate-in fade-in slide-in-from-top-2 mt-12"
              onClick={(e) => e.stopPropagation()}
            >
              <div className="flex items-center justify-between border-b border-border/50 pb-2">
                <div className="flex items-center gap-2">
                  <Bell className="h-4 w-4 text-primary" weight="bold" />
                  <span className="font-display font-bold text-sm">Notifications</span>
                  {(notifData?.unread_count ?? 0) > 0 && (
                    <Badge variant="secondary" className="text-[10px] px-1.5 py-0 font-bold bg-primary/10 text-primary">
                      {notifData?.unread_count} new
                    </Badge>
                  )}
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-7 text-[11px] text-muted-foreground hover:text-foreground"
                  onClick={markAllRead}
                >
                  Mark all read
                </Button>
              </div>

              <div className="max-h-[320px] overflow-y-auto space-y-2">
                {(!notifData?.notifications || notifData.notifications.length === 0) ? (
                  <p className="text-xs text-muted-foreground py-6 text-center">
                    No notifications yet.
                  </p>
                ) : (
                  notifData.notifications.map((n) => (
                    <div
                      key={n.id}
                      onClick={() => handleNotificationClick(n)}
                      className={cn(
                        "p-2.5 rounded-lg border border-border/50 text-xs space-y-1 cursor-pointer transition-colors hover:bg-muted/60",
                        !n.read ? "bg-primary/5 border-primary/20" : "bg-card"
                      )}
                    >
                      <div className="flex items-center justify-between">
                        <span className="font-semibold text-foreground">{n.title}</span>
                        <span className="text-[10px] text-muted-foreground">
                          {new Date(n.created_at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}
                        </span>
                      </div>
                      <p className="text-muted-foreground line-clamp-2">{n.message}</p>
                    </div>
                  ))
                )}
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
