import { useState } from "react"
import { Link, useLocation } from "react-router-dom"
import { ArrowRight, SpinnerGap, Key } from "@phosphor-icons/react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { ApiError } from "@/lib/api"
import { login } from "@/lib/auth"

export function LoginPage() {
  const location = useLocation()
  const from = (location.state as { from?: string } | null)?.from ?? "/dashboard"
  const [orgSlug, setOrgSlug] = useState("")
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      await login(orgSlug.trim(), email.trim(), password)
      window.location.assign(from)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Login failed")
    } finally {
      setLoading(false)
    }
  }

  function fillDemo() {
    setOrgSlug("demo")
    setEmail("admin@demo.io")
    setPassword("password123")
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-[radial-gradient(ellipse_at_top,_var(--tw-gradient-stops))] from-primary/10 via-background to-background p-4 animate-in fade-in duration-500">
      <div className="w-full max-w-md space-y-4">
        <Card className="glass border-primary/20 shadow-2xl shadow-primary/10 overflow-hidden relative">
          <CardHeader className="text-center pb-2">
            <div className="mx-auto mb-2 flex h-12 w-12 items-center justify-center rounded-2xl bg-primary text-primary-foreground font-bold font-display text-xl shadow-lg shadow-primary/25">
              I
            </div>
            <CardTitle className="font-display text-2xl font-bold tracking-tight">Intivai Workspace</CardTitle>
            <CardDescription className="text-xs">
              AI recruitment screening & intelligent interview platform
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4 pt-2">
            {/* 1-click Demo Fill */}
            <div className="rounded-xl border border-primary/20 bg-primary/5 p-3 flex items-center justify-between">
              <div>
                <p className="text-xs font-semibold flex items-center gap-1 text-primary">
                  <Key className="h-3.5 w-3.5" /> Testing Demo Workspace?
                </p>
                <p className="text-[11px] text-muted-foreground">demo / admin@demo.io</p>
              </div>
              <Button size="sm" variant="outline" className="h-7 text-xs border-primary/30 text-primary hover:bg-primary/10" onClick={fillDemo}>
                Auto-fill
              </Button>
            </div>

            <form onSubmit={onSubmit} className="space-y-3.5">
              <div className="space-y-1.5">
                <Label htmlFor="org" className="text-xs font-semibold">Organization Slug</Label>
                <Input
                  id="org"
                  value={orgSlug}
                  onChange={(e) => setOrgSlug(e.target.value)}
                  placeholder="e.g. demo or acme"
                  autoComplete="username"
                  className="bg-background/80"
                  required
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="email" className="text-xs font-semibold">Admin / Recruiter Email</Label>
                <Input
                  id="email"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="admin@demo.io"
                  autoComplete="email"
                  className="bg-background/80"
                  required
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="password" className="text-xs font-semibold">Password</Label>
                <Input
                  id="password"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  autoComplete="current-password"
                  className="bg-background/80"
                  required
                />
              </div>

              {error && (
                <div className="rounded-lg border border-destructive/30 bg-destructive/5 p-2.5 text-xs text-destructive">
                  {error}
                </div>
              )}

              <Button type="submit" variant="gradient" className="w-full font-semibold shadow-md shadow-primary/20 mt-1" disabled={loading}>
                {loading ? <SpinnerGap className="mr-2 h-4 w-4 animate-spin" /> : <ArrowRight className="mr-2 h-4 w-4" />}
                Sign In to Console
              </Button>
            </form>

            <div className="border-t border-border/40 pt-3 text-center text-xs text-muted-foreground">
              New organization?{" "}
              <Link to="/register" className="text-primary font-semibold hover:underline">
                Create new workspace
              </Link>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
