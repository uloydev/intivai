import { NavLink, Outlet } from "react-router-dom"
import { Briefcase, Files, UsersThree, ChatCircleText, SignOut } from "@phosphor-icons/react"
import { logout } from "@/lib/auth"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

const nav = [
  { to: "/jobs", label: "Jobs", icon: Briefcase },
  { to: "/cvs", label: "CVs", icon: Files },
  { to: "/candidates", label: "Candidates", icon: UsersThree },
  { to: "/interviews", label: "Interviews", icon: ChatCircleText },
]

export function AppShell() {
  return (
    <div className="flex min-h-screen bg-background">
      <aside className="hidden w-56 shrink-0 flex-col border-r border-border bg-card md:flex">
        <div className="flex h-14 items-center px-4 font-display text-lg font-semibold text-primary">Intivai</div>
        <nav className="flex-1 space-y-1 px-2">
          {nav.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              className={({ isActive }) =>
                cn(
                  "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
                  isActive ? "bg-primary text-primary-foreground" : "text-foreground hover:bg-muted",
                )
              }
            >
              <item.icon className="h-5 w-5" />
              {item.label}
            </NavLink>
          ))}
        </nav>
        <div className="p-2">
          <Button
            variant="ghost"
            className="w-full justify-start text-muted-foreground"
            onClick={() => {
              logout()
              window.location.assign("/login")
            }}
          >
            <SignOut className="mr-2 h-4 w-4" />
            Sign out
          </Button>
        </div>
      </aside>

      {/* Mobile top bar */}
      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex h-14 items-center justify-between border-b border-border bg-card px-4 md:hidden">
          <span className="font-display text-lg font-semibold text-primary">Intivai</span>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => {
              logout()
              window.location.assign("/login")
            }}
          >
            <SignOut className="h-4 w-4" />
          </Button>
        </header>
        <main className="flex-1 p-4 md:p-6">
          <Outlet />
        </main>
        <nav className="flex border-t border-border bg-card md:hidden">
          {nav.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              className={({ isActive }) =>
                cn(
                  "flex flex-1 flex-col items-center gap-1 py-2 text-xs font-medium",
                  isActive ? "text-primary" : "text-muted-foreground",
                )
              }
            >
              <item.icon className="h-5 w-5" />
              {item.label}
            </NavLink>
          ))}
        </nav>
      </div>
    </div>
  )
}
