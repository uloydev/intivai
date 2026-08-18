import { useState } from "react"
import { Link } from "react-router-dom"
import { ArrowRight, SpinnerGap } from "@phosphor-icons/react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { api, ApiError } from "@/lib/api"
import { login } from "@/lib/auth"
import type { RegisterResult } from "@/types/api"

export function RegisterPage() {
  const [name, setName] = useState("")
  const [slug, setSlug] = useState("")
  const [slugTouched, setSlugTouched] = useState(false)
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      await api.post<RegisterResult>("/auth/register", {
        name: name.trim(),
        slug: slug.trim(),
        admin_email: email.trim(),
        admin_password: password,
      })
      await login(slug.trim(), email.trim(), password)
      window.location.assign("/dashboard")
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Registration failed")
    } finally {
      setLoading(false)
    }
  }

  function handleNameChange(val: string) {
    setName(val)
    // Auto-derive the slug from the CURRENT name until the user edits the
    // slug field manually (the old code compared against the previous name,
    // so auto-derivation silently stopped after the first keystroke).
    if (!slugTouched) {
      setSlug(val.toLowerCase().replace(/[^a-z0-9]/g, "-"))
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-[radial-gradient(ellipse_at_top,_var(--tw-gradient-stops))] from-primary/10 via-background to-background p-4 animate-in fade-in duration-500">
      <div className="w-full max-w-md space-y-4">
        <Card className="glass border-primary/20 shadow-2xl shadow-primary/10 overflow-hidden relative">
          <CardHeader className="text-center pb-2">
            <div className="mx-auto mb-2 flex h-12 w-12 items-center justify-center rounded-2xl bg-primary text-primary-foreground font-bold font-display text-xl shadow-lg shadow-primary/25">
              I
            </div>
            <CardTitle className="font-display text-2xl font-bold tracking-tight">Create Workspace</CardTitle>
            <CardDescription className="text-xs">
              Deploy your AI recruitment pipeline and candidate evaluation engine
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4 pt-2">
            <form onSubmit={onSubmit} className="space-y-3.5">
              <div className="space-y-1.5">
                <Label htmlFor="name" className="text-xs font-semibold">Company / Organization Name</Label>
                <Input
                  id="name"
                  value={name}
                  onChange={(e) => handleNameChange(e.target.value)}
                  placeholder="e.g. Acme Technologies"
                  className="bg-background/80"
                  required
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="slug" className="text-xs font-semibold">Workspace Slug</Label>
                <Input
                  id="slug"
                  value={slug}
                  onChange={(e) => {
                    setSlugTouched(true)
                    setSlug(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ""))
                  }}
                  placeholder="acme"
                  className="bg-background/80 font-mono text-xs"
                  required
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="email" className="text-xs font-semibold">Admin Email</Label>
                <Input
                  id="email"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="admin@acme.com"
                  className="bg-background/80"
                  required
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="password" className="text-xs font-semibold">Master Password</Label>
                <Input
                  id="password"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="At least 8 characters"
                  className="bg-background/80"
                  minLength={8}
                  required
                />
              </div>

              {error && (
                <div className="rounded-lg border border-destructive/30 bg-destructive/5 p-2.5 text-xs text-destructive">
                  {error}
                </div>
              )}

              <Button type="submit" variant="gradient" className="w-full font-semibold shadow-md shadow-primary/20 mt-2" disabled={loading}>
                {loading ? <SpinnerGap className="mr-2 h-4 w-4 animate-spin" /> : <ArrowRight className="mr-2 h-4 w-4" />}
                Initialize Workspace
              </Button>
            </form>

            <div className="border-t border-border/40 pt-3 text-center text-xs text-muted-foreground">
              Already have a workspace?{" "}
              <Link to="/login" className="text-primary font-semibold hover:underline">
                Sign in
              </Link>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
