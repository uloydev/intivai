import React, { useState } from "react"
import { Link, Outlet, useLocation, useNavigate } from "react-router-dom"
import {
  Briefcase,
  ArrowRight,
  Sun,
  Moon,
  ShieldCheck,
  Sparkle,
  Cpu,
  List,
  X,
  Calculator,
  Question,
} from "@phosphor-icons/react"
import type { Icon } from "@phosphor-icons/react"
import { useTheme } from "@/lib/theme"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import { getToken } from "@/lib/auth"

interface PublicNavItem {
  to: string
  label: string
  section?: string
  icon?: Icon
  mobileClass?: string
  mobileIconClass?: string
  desktopIconClass?: string
}

// Single source of truth for both header lists — desktop links and the mobile
// drawer render from the same array.
const NAV: PublicNavItem[] = [
  { to: "/", label: "Platform" },
  { to: "/careers", label: "Careers & Jobs", icon: Briefcase, mobileClass: "text-foreground", mobileIconClass: "text-primary" },
  {
    to: "/candidate/portal",
    label: "Track Applications",
    icon: ShieldCheck,
    mobileClass: "text-cyan-400 hover:text-cyan-300",
    mobileIconClass: "text-cyan-400",
    desktopIconClass: "text-cyan-500",
  },
  { to: "/#demo", label: "AI Evaluator", section: "demo", icon: Sparkle, mobileIconClass: "text-primary" },
  { to: "/#how-it-works", label: "How it Works", section: "how-it-works" },
  { to: "/#features", label: "Intelligence", section: "features", icon: Cpu },
  { to: "/#calculator", label: "ROI Calculator", section: "calculator", icon: Calculator },
  { to: "/#faq", label: "FAQ", section: "faq", icon: Question },
]

export function PublicLayout() {
  const { theme, toggle } = useTheme()
  const location = useLocation()
  const navigate = useNavigate()
  const authenticated = !!getToken()
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)

  const handleSectionClick = (sectionId: string, e: React.MouseEvent) => {
    setMobileMenuOpen(false)
    if (location.pathname === "/") {
      e.preventDefault()
      window.history.pushState(null, "", `/#${sectionId}`)
      const el = document.getElementById(sectionId)
      if (el) {
        el.scrollIntoView({ behavior: "smooth" })
      }
    } else {
      navigate(`/#${sectionId}`)
    }
  }

  const renderNavLink = (item: PublicNavItem, viewport: "desktop" | "mobile") => {
    const Icon = item.icon
    const isMobile = viewport === "mobile"
    const active =
      !item.section &&
      (item.to === "/"
        ? location.pathname === "/" && !location.hash
        : location.pathname.startsWith(item.to))
    const className = isMobile
      ? cn(
          "py-1.5 hover:text-primary transition-colors",
          item.mobileClass ?? (item.section ? "text-muted-foreground" : "text-foreground"),
          Icon && "flex items-center gap-2"
        )
      : cn(
          "transition-colors hover:text-primary",
          Icon && "flex items-center gap-1.5",
          active ? "text-primary font-bold" : "text-muted-foreground"
        )
    return (
      <Link
        key={item.to}
        to={item.to}
        onClick={(e) => {
          if (isMobile) setMobileMenuOpen(false)
          if (item.section) {
            handleSectionClick(item.section, e)
          } else if (item.to === "/" && location.pathname === "/") {
            e.preventDefault()
            window.scrollTo({ top: 0, behavior: "smooth" })
          }
        }}
        className={className}
      >
        {Icon && (
          <Icon
            className={isMobile ? cn("h-4 w-4", item.mobileIconClass) : cn("h-3.5 w-3.5", item.desktopIconClass)}
          />
        )}
        {item.label}
      </Link>
    )
  }

  return (
    <div className="flex min-h-screen flex-col bg-background text-foreground selection:bg-primary/20 selection:text-primary">
      {/* Public Header */}
      <header className="sticky top-0 z-50 flex h-16 items-center justify-between border-b border-border/50 bg-background/80 px-6 backdrop-blur-xl md:px-12">
        <Link
          to="/"
          onClick={(e) => {
            if (location.pathname === "/") {
              e.preventDefault()
              window.scrollTo({ top: 0, behavior: "smooth" })
            }
          }}
          className="flex items-center gap-2.5"
        >
          <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-primary text-primary-foreground font-bold font-display text-lg shadow-lg shadow-primary/25">
            I
          </div>
          <span className="font-display text-xl font-bold tracking-tight bg-gradient-to-r from-primary via-blue-500 to-indigo-500 bg-clip-text text-transparent">
            Intivai
          </span>
        </Link>

        {/* Desktop Navigation Links */}
        <nav className="hidden items-center gap-7 text-xs font-semibold lg:flex">
          {NAV.map((item) => renderNavLink(item, "desktop"))}
        </nav>

        {/* Right Action Icons & Auth / Workspace Buttons */}
        <div className="flex items-center gap-3">
          <Button
            variant="ghost"
            size="icon"
            className="rounded-full h-8 w-8 text-muted-foreground hover:text-foreground"
            aria-label="Toggle dark mode"
            onClick={toggle}
          >
            {theme === "dark" ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
          </Button>

          {authenticated ? (
            <Button asChild variant="gradient" size="sm" className="shadow-md shadow-primary/20 text-xs hidden sm:inline-flex">
              <Link to="/dashboard">
                Workspace <ArrowRight className="ml-1.5 h-3.5 w-3.5" />
              </Link>
            </Button>
          ) : (
            <div className="hidden sm:flex items-center gap-2">
              <Button asChild variant="ghost" size="sm" className="text-xs font-medium">
                <Link to="/login">Sign In</Link>
              </Button>
              <Button asChild variant="gradient" size="sm" className="text-xs shadow-md shadow-primary/20">
                <Link to="/register">Get Started Free</Link>
              </Button>
            </div>
          )}

          {/* Mobile Menu Toggle Button */}
          <Button
            variant="ghost"
            size="icon"
            className="lg:hidden rounded-lg h-9 w-9 text-muted-foreground"
            onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
            aria-label="Toggle Navigation Menu"
          >
            {mobileMenuOpen ? <X className="h-5 w-5" /> : <List className="h-5 w-5" />}
          </Button>
        </div>
      </header>

      {/* Mobile Navigation Drawer */}
      {mobileMenuOpen && (
        <div className="lg:hidden border-b border-border/60 bg-background/95 backdrop-blur-xl px-6 py-5 space-y-4 animate-in slide-in-from-top-2 duration-200 z-40 sticky top-16">
          <nav className="flex flex-col space-y-3 text-sm font-semibold">
            {NAV.map((item) => renderNavLink(item, "mobile"))}
          </nav>

          <div className="pt-3 border-t border-border/50 flex flex-col gap-2">
            {authenticated ? (
              <Button asChild variant="gradient" className="w-full justify-center">
                <Link to="/dashboard" onClick={() => setMobileMenuOpen(false)}>
                  Go to Workspace <ArrowRight className="ml-2 h-4 w-4" />
                </Link>
              </Button>
            ) : (
              <div className="grid grid-cols-2 gap-2">
                <Button asChild variant="outline" className="w-full">
                  <Link to="/login" onClick={() => setMobileMenuOpen(false)}>
                    Sign In
                  </Link>
                </Button>
                <Button asChild variant="gradient" className="w-full shadow-md shadow-primary/20">
                  <Link to="/register" onClick={() => setMobileMenuOpen(false)}>
                    Get Started
                  </Link>
                </Button>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Page Body */}
      <main className="flex-1">
        <Outlet />
      </main>

      {/* Public Footer */}
      <footer className="border-t border-border/60 bg-muted/20 px-6 py-12 md:px-12">
        <div className="mx-auto max-w-6xl">
          <div className="grid gap-8 md:grid-cols-4">
            <div className="space-y-3 md:col-span-2">
              <div className="flex items-center gap-2">
                <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-primary text-primary-foreground font-bold font-display text-xs">
                  I
                </div>
                <span className="font-display text-lg font-bold">Intivai</span>
              </div>
              <p className="text-xs text-muted-foreground max-w-sm leading-relaxed">
                Next-generation autonomous AI recruitment platform. Delivering real-time semantic CV matching, token-streamed technical interviews, and bias-free candidate evaluations.
              </p>
              <div className="flex items-center gap-2 text-xs text-muted-foreground pt-1">
                <ShieldCheck className="h-4 w-4 text-emerald-500" />
                <span>SOC 2 Type II & GDPR Compliant</span>
              </div>
            </div>

            <div className="space-y-2.5">
              <p className="font-display text-xs font-bold uppercase tracking-wider text-foreground">Platform</p>
              <ul className="space-y-2 text-xs text-muted-foreground">
                <li>
                  <Link to="/careers" className="hover:text-primary transition-colors">
                    Public Job Board
                  </Link>
                </li>
                <li>
                  <Link to="/login" className="hover:text-primary transition-colors">
                    Recruiter Console
                  </Link>
                </li>
                <li>
                  <Link
                    to="/#features"
                    onClick={(e) => handleSectionClick("features", e)}
                    className="hover:text-primary transition-colors"
                  >
                    WebRTC Voice & Chat Interviews
                  </Link>
                </li>
                <li>
                  <Link
                    to="/#features"
                    onClick={(e) => handleSectionClick("features", e)}
                    className="hover:text-primary transition-colors"
                  >
                    Semantic CV Screening
                  </Link>
                </li>
                <li>
                  <Link
                    to="/#calculator"
                    onClick={(e) => handleSectionClick("calculator", e)}
                    className="hover:text-primary transition-colors"
                  >
                    ROI & Dev Hours Calculator
                  </Link>
                </li>
              </ul>
            </div>

            <div className="space-y-2.5">
              <p className="font-display text-xs font-bold uppercase tracking-wider text-foreground">Candidates</p>
              <ul className="space-y-2 text-xs text-muted-foreground">
                <li>
                  <Link to="/careers" className="hover:text-primary transition-colors">
                    Explore Open Roles
                  </Link>
                </li>
                <li>
                  <Link to="/candidate/portal" className="hover:text-primary transition-colors text-cyan-400">
                    Track Application & Status
                  </Link>
                </li>
                <li>
                  <Link
                    to="/#demo"
                    onClick={(e) => handleSectionClick("demo", e)}
                    className="hover:text-primary transition-colors"
                  >
                    Interactive AI Evaluator
                  </Link>
                </li>
                <li>
                  <Link
                    to="/#how-it-works"
                    onClick={(e) => handleSectionClick("how-it-works", e)}
                    className="hover:text-primary transition-colors"
                  >
                    Interview Preparation
                  </Link>
                </li>
                <li>
                  <Link
                    to="/#security"
                    onClick={(e) => handleSectionClick("security", e)}
                    className="hover:text-primary transition-colors"
                  >
                    Privacy & Data Consent
                  </Link>
                </li>
                <li>
                  <Link
                    to="/#faq"
                    onClick={(e) => handleSectionClick("faq", e)}
                    className="hover:text-primary transition-colors"
                  >
                    Candidate FAQs
                  </Link>
                </li>
              </ul>
            </div>
          </div>

          <div className="mt-12 flex flex-col items-center justify-between gap-4 border-t border-border/40 pt-6 sm:flex-row text-xs text-muted-foreground">
            <p>© {new Date().getFullYear()} Intivai Inc. All rights reserved.</p>
            <div className="flex items-center gap-6">
              <Link to="/#security" onClick={(e) => handleSectionClick("security", e)} className="hover:text-primary">
                Privacy Policy
              </Link>
              <Link to="/#security" onClick={(e) => handleSectionClick("security", e)} className="hover:text-primary">
                Terms of Service
              </Link>
              <Link to="/#security" onClick={(e) => handleSectionClick("security", e)} className="hover:text-primary">
                Security
              </Link>
            </div>
          </div>
        </div>
      </footer>
    </div>
  )
}
