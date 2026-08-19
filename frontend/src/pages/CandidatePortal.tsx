import { useCallback, useEffect, useRef, useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useSearchParams, Link } from "react-router-dom"
import { api, ApiError } from "@/lib/api"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import type {
  CandidateApplicationItem,
  CandidateOTPResponse,
  CandidateVerifyResponse,
} from "@/types/api"

const TOKEN_KEY = "intivai_candidate_token"
const EMAIL_KEY = "intivai_candidate_email"
const OTP_DEFAULT_TTL_SEC = 600

function applicationStatusLabel(status: string | null | undefined): string {
  switch (status) {
    case "applied":
      return "Application received"
    case "screening":
    case "under_review":
      return "Under review"
    case "passed":
      return "Screening passed"
    case "rejected":
      return "Application not proceeding"
    default:
      if (!status) return "Application received"
      return status.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase())
  }
}

function SubmitButton({
  loading,
  loadingLabel,
  label,
  disabled,
}: {
  loading: boolean
  loadingLabel: string
  label: string
  disabled?: boolean
}) {
  return (
    <Button
      type="submit"
      variant="gradient"
      className="w-full h-12 rounded-xl font-semibold shadow-lg shadow-primary/25 disabled:opacity-50 flex items-center justify-center gap-2"
      disabled={loading || disabled}
    >
      {loading ? (
        <>
          <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
          <span>{loadingLabel}</span>
        </>
      ) : (
        <span>{label}</span>
      )}
    </Button>
  )
}

export function CandidatePortal() {
  const [searchParams] = useSearchParams()
  const qc = useQueryClient()
  const [email, setEmail] = useState(localStorage.getItem(EMAIL_KEY) || "")
  const [otpCode, setOtpCode] = useState("")
  const [step, setStep] = useState<"email" | "otp" | "dashboard">(
    localStorage.getItem(TOKEN_KEY) ? "dashboard" : "email"
  )
  const [error, setError] = useState<string | null>(null)
  const [infoMsg, setInfoMsg] = useState<string | null>(null)
  const [otpExpiresAt, setOtpExpiresAt] = useState<number | null>(null)
  const [otpRemainingSec, setOtpRemainingSec] = useState(0)
  const emptyAutoRefetchRef = useRef(false)

  const handleLogout = useCallback(() => {
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(EMAIL_KEY)
    setStep("email")
    setOtpCode("")
    setOtpExpiresAt(null)
    qc.removeQueries({ queryKey: ["candidate-applications"] })
  }, [qc])

  // Magic Link exchange: the raw ?token= is NOT a JWT — swap it for a real
  // candidate token first, then the applications query (enabled on dashboard)
  // takes over.
  const magicVerify = useMutation({
    mutationFn: (token: string) =>
      api.post<CandidateVerifyResponse>("/public/candidate/auth/verify", { token }),
    onSuccess: (res) => {
      localStorage.setItem(TOKEN_KEY, res.token)
      localStorage.setItem(EMAIL_KEY, res.email)
      setEmail(res.email)
      setStep("dashboard")
      qc.invalidateQueries({ queryKey: ["candidate-applications", res.email] })
    },
    onError: (err) => {
      setError(err instanceof ApiError ? err.message : "Invalid or expired magic link.")
      handleLogout()
    },
  })

  useEffect(() => {
    const magicToken = searchParams.get("token")
    if (magicToken) {
      magicVerify.mutate(magicToken)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams])

  const sendOtp = useMutation({
    mutationFn: (emailValue: string) =>
      api.post<CandidateOTPResponse>("/public/candidate/auth/otp", { email: emailValue }),
    onSuccess: (data, emailValue) => {
      setStep("otp")
      setOtpCode("")
      setError(null)
      setOtpExpiresAt(Date.now() + (data.expires_in || OTP_DEFAULT_TTL_SEC) * 1000)
      setInfoMsg(`A 6-digit verification code has been dispatched to ${emailValue}.`)
    },
    onError: (err) => {
      setError(err instanceof ApiError ? err.message : "Failed to dispatch verification code. Please try again.")
    },
  })

  const verify = useMutation({
    mutationFn: ({ emailValue, code }: { emailValue: string; code: string }) =>
      api.post<CandidateVerifyResponse>("/public/candidate/auth/verify", {
        email: emailValue,
        code,
      }),
    onSuccess: (res) => {
      localStorage.setItem(TOKEN_KEY, res.token)
      localStorage.setItem(EMAIL_KEY, res.email)
      setEmail(res.email)
      setOtpCode("")
      setOtpExpiresAt(null)
      setStep("dashboard")
      qc.invalidateQueries({ queryKey: ["candidate-applications", res.email] })
    },
    onError: (err) => {
      setError(err instanceof ApiError ? err.message : "Invalid or expired verification code.")
    },
  })

  const appsQuery = useQuery({
    queryKey: ["candidate-applications", email],
    queryFn: () => api.get<CandidateApplicationItem[]>("/candidate/portal/applications"),
    enabled: step === "dashboard",
  })

  const applications = appsQuery.data ?? []

  // Live OTP expiry countdown under the code input.
  useEffect(() => {
    if (!otpExpiresAt) {
      setOtpRemainingSec(0)
      return
    }
    const tick = () => setOtpRemainingSec(Math.max(0, Math.ceil((otpExpiresAt - Date.now()) / 1000)))
    tick()
    const t = setInterval(tick, 1000)
    return () => clearInterval(t)
  }, [otpExpiresAt])

  const otpExpired = otpExpiresAt !== null && otpRemainingSec <= 0

  // Expired/revoked candidate token: the api layer already dropped the stored
  // token on 401 — bounce back to the email step.
  useEffect(() => {
    if (appsQuery.error instanceof ApiError && appsQuery.error.status === 401) {
      handleLogout()
    }
  }, [appsQuery.error, handleLogout])

  // A just-completed apply may not be visible for a few seconds (async
  // screening pipeline). Refetch once after 10s while the list is empty so we
  // never imply the email has no applications in that window.
  useEffect(() => {
    if (step !== "dashboard" || applications.length > 0 || emptyAutoRefetchRef.current) return
    emptyAutoRefetchRef.current = true
    const t = setTimeout(() => {
      appsQuery.refetch()
    }, 10_000)
    return () => clearTimeout(t)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [step, applications.length, appsQuery])

  function handleSendOTP(e?: React.FormEvent) {
    e?.preventDefault()
    const normalized = email.trim().toLowerCase()
    if (!normalized || !normalized.includes("@")) {
      setError("Please enter a valid email address.")
      return
    }
    setError(null)
    setInfoMsg(null)
    sendOtp.mutate(normalized)
  }

  function handleVerifyOTP(e: React.FormEvent) {
    e.preventDefault()
    if (otpExpired) {
      setError("Code expired — request a new one.")
      return
    }
    if (!otpCode.trim() || otpCode.length !== 6) {
      setError("Please enter the complete 6-digit verification code.")
      return
    }
    setError(null)
    verify.mutate({ emailValue: email.trim().toLowerCase(), code: otpCode.trim() })
  }

  const authBusy = sendOtp.isPending || verify.isPending || magicVerify.isPending
  const formatCountdown = (secs: number) => {
    const mins = Math.floor(secs / 60)
    const rem = secs % 60
    return `${mins.toString().padStart(2, "0")}:${rem.toString().padStart(2, "0")}`
  }

  return (
    <div className="min-h-[calc(100vh-4rem)] bg-background text-foreground py-12 px-4 sm:px-6 lg:px-8 relative overflow-hidden">
      {/* Background glow effects */}
      <div className="absolute top-1/4 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[350px] bg-indigo-600/10 rounded-full blur-3xl pointer-events-none" />
      <div className="absolute bottom-10 right-10 w-[400px] h-[250px] bg-cyan-600/10 rounded-full blur-3xl pointer-events-none" />

      <div className="max-w-4xl mx-auto relative z-10">
        {step !== "dashboard" ? (
          /* Authentication Screen */
          <div className="max-w-md mx-auto">
            <div className="text-center mb-8">
              <div className="inline-flex items-center justify-center w-12 h-12 rounded-xl bg-gradient-to-tr from-primary to-blue-500 text-primary-foreground font-bold text-xl shadow-lg shadow-primary/25 mb-4">
                ✦
              </div>
              <h1 className="text-3xl font-extrabold tracking-tight text-foreground sm:text-4xl">
                Candidate Portal
              </h1>
              <p className="mt-2 text-sm text-muted-foreground">
                Track your job applications, screening scores, and launch your AI assessment sessions.
              </p>
            </div>

            <div className="bg-card/80 border border-border rounded-2xl p-8 backdrop-blur-xl shadow-2xl">
              {error && (
                <div className="mb-6 p-4 rounded-xl bg-destructive/10 border border-destructive/30 text-destructive text-sm flex items-start gap-3">
                  <span className="text-destructive font-bold">✕</span>
                  <span>{error}</span>
                </div>
              )}

              {infoMsg && (
                <div className="mb-6 p-4 rounded-xl bg-primary/10 border border-primary/20 text-primary text-sm flex items-start gap-3">
                  <span className="text-primary font-bold">ℹ</span>
                  <span>{infoMsg}</span>
                </div>
              )}

              {step === "email" ? (
                <form onSubmit={handleSendOTP} className="space-y-5">
                  <div>
                    <label htmlFor="candidate-email" className="block text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2">
                      Applicant Email Address
                    </label>
                    <Input
                      id="candidate-email"
                      type="email"
                      required
                      value={email}
                      onChange={(e) => setEmail(e.target.value)}
                      placeholder="e.g. alex.dev@gmail.com"
                      className="w-full h-12 rounded-xl bg-background/60"
                    />
                    <p className="mt-2 text-xs text-muted-foreground">
                      We'll send you a passwordless 6-digit verification code & magic login link.
                    </p>
                  </div>

                  <SubmitButton
                    loading={authBusy}
                    loadingLabel="Sending Security Code..."
                    label="Send Verification Code →"
                  />

                  <div className="pt-4 border-t border-border/80 flex items-center justify-between text-xs text-muted-foreground">
                    <Link to="/careers" className="hover:text-primary transition-colors">
                      ← Browse Open Positions
                    </Link>
                  </div>
                </form>
              ) : (
                <form onSubmit={handleVerifyOTP} className="space-y-5">
                  <div>
                    <div className="flex items-center justify-between mb-2">
                      <label htmlFor="otp-code" className="block text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                        6-Digit Security Code
                      </label>
                      <button
                        type="button"
                        onClick={() => { setStep("email"); setOtpExpiresAt(null); setError(null); }}
                        className="text-xs text-primary hover:underline"
                      >
                        Change Email
                      </button>
                    </div>
                    <Input
                      id="otp-code"
                      type="text"
                      maxLength={6}
                      required
                      autoFocus
                      value={otpCode}
                      onChange={(e) => setOtpCode(e.target.value.replace(/\D/g, ""))}
                      placeholder="• • • • • •"
                      className="w-full h-14 rounded-xl text-center text-2xl tracking-[0.5em] font-mono bg-background/60"
                    />
                    {otpExpiresAt !== null ? (
                      otpExpired ? (
                        <p className="mt-2 text-xs text-destructive text-center">
                          Code expired — request a new one.
                        </p>
                      ) : (
                        <p className="mt-2 text-xs text-muted-foreground text-center">
                          Code expires in <strong className="text-foreground font-mono">{formatCountdown(otpRemainingSec)}</strong>. Check your email.
                        </p>
                      )
                    ) : (
                      <p className="mt-2 text-xs text-muted-foreground text-center">
                        Check your email for the verification code.
                      </p>
                    )}
                  </div>

                  <SubmitButton
                    loading={authBusy}
                    loadingLabel="Verifying..."
                    label="Access Candidate Portal →"
                    disabled={otpCode.length !== 6 || otpExpired}
                  />

                  <div className="text-center">
                    <button
                      type="button"
                      onClick={handleSendOTP}
                      className="text-xs text-muted-foreground hover:text-primary transition-colors"
                    >
                      Didn't receive the code? Resend
                    </button>
                  </div>
                </form>
              )}
            </div>
          </div>
        ) : (
          /* Authenticated Candidate Dashboard */
          <div className="space-y-8">
            {/* Header bar */}
            <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 p-6 bg-card/80 border border-border rounded-2xl backdrop-blur-xl">
              <div>
                <div className="flex items-center gap-2 mb-1">
                  <span className="inline-block w-2.5 h-2.5 rounded-full bg-emerald-500 animate-pulse" />
                  <h2 className="text-xl font-bold text-foreground">Applicant Tracking Dashboard</h2>
                </div>
                <p className="text-sm text-muted-foreground">
                  Signed in as <span className="text-primary font-medium">{email}</span>
                </p>
              </div>

              <div className="flex items-center gap-3">
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  onClick={() => appsQuery.refetch()}
                  disabled={appsQuery.isFetching}
                  className="text-xs flex items-center gap-1.5"
                >
                  <span className={appsQuery.isFetching ? "animate-spin" : ""}>↻</span> Refresh
                </Button>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={handleLogout}
                  className="text-xs text-destructive hover:text-destructive hover:bg-destructive/10"
                >
                  Sign Out
                </Button>
              </div>
            </div>

            {/* Applications List */}
            {appsQuery.isLoading && !appsQuery.data ? (
              <div className="p-12 text-center bg-card/60 border border-border rounded-2xl">
                <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin mx-auto mb-4" />
                <p className="text-muted-foreground text-sm">Retrieving your application history...</p>
              </div>
            ) : appsQuery.error ? (
              <div className="p-12 text-center bg-card/60 border border-destructive/30 rounded-2xl space-y-3">
                <div className="w-16 h-16 rounded-full bg-destructive/10 text-destructive flex items-center justify-center text-2xl mx-auto">
                  ⚠
                </div>
                <p className="text-sm text-destructive">
                  Unable to load applications at this time. Please refresh or try again later.
                </p>
              </div>
            ) : applications.length === 0 ? (
              <div className="p-12 text-center bg-card/60 border border-border rounded-2xl space-y-4">
                <div className="w-16 h-16 rounded-full bg-muted text-muted-foreground flex items-center justify-center text-2xl mx-auto">
                  📋
                </div>
                <div>
                  <h3 className="text-lg font-semibold text-foreground">No Applications Found</h3>
                  <p className="text-sm text-muted-foreground mt-1">
                    Applications may take a few minutes to appear while your profile is processed.
                  </p>
                  <p className="text-sm text-muted-foreground mt-1">
                    You haven't submitted any job applications under this email address yet.
                  </p>
                </div>
                <Button asChild variant="gradient" size="sm" className="shadow-md shadow-primary/20">
                  <Link to="/careers">
                    Explore Open Careers →
                  </Link>
                </Button>
              </div>
            ) : (
              <div className="space-y-6">
                {appsQuery.isFetching && (
                  <p className="text-xs text-muted-foreground text-center animate-pulse">
                    Refreshing…
                  </p>
                )}
                {applications.map((app) => {
                  const isCompleted = app.interview_status === "completed"
                  const hasInterviewTicket = Boolean(app.invitation_token)
                  const isInterviewReady = hasInterviewTicket && app.interview_status !== "completed"

                  return (
                    <div
                      key={app.application_id}
                      className="p-6 bg-card border border-border rounded-2xl shadow-xl transition-all hover:border-primary/40"
                    >
                      {/* Top Job Info Header */}
                      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4 pb-6 border-b border-border/80">
                        <div>
                          <div className="flex items-center gap-2 mb-1.5 flex-wrap">
                            <span className="px-2.5 py-0.5 rounded-full text-xs font-semibold bg-primary/10 text-primary border border-primary/20">
                              {app.org_name}
                            </span>
                            <span className="px-2.5 py-0.5 rounded-full text-xs font-medium bg-muted text-muted-foreground border border-border">
                              {app.job_employment_type}
                            </span>
                            <span className="px-2.5 py-0.5 rounded-full text-xs font-medium bg-muted text-muted-foreground border border-border">
                              {app.job_location}
                            </span>
                          </div>
                          <h3 className="text-xl font-bold text-foreground">{app.job_title}</h3>
                          <p className="text-xs text-muted-foreground mt-1">
                            Applied on {new Date(app.applied_at).toLocaleDateString(undefined, { dateStyle: "long" })}
                          </p>
                        </div>

                        {/* Action launch buttons */}
                        <div>
                          {isInterviewReady ? (
                            <Link
                              to={`/invite/${app.interview_id}?t=${encodeURIComponent(app.invitation_token ?? "")}`}
                              className="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl text-sm font-semibold text-white bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 active:scale-[0.98] transition-all shadow-lg shadow-emerald-600/25 animate-pulse"
                            >
                              <span>Launch AI Interview</span> →
                            </Link>
                          ) : isCompleted ? (
                            <div className="text-right">
                              <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-semibold bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/30">
                                ✓ Assessment Complete
                              </span>
                              {app.overall_score !== null && app.overall_score !== undefined && (
                                <p className="text-xs text-muted-foreground mt-1">
                                  Score: <strong className="text-foreground">{Math.round(app.overall_score)}/100</strong>
                                </p>
                              )}
                            </div>
                          ) : (
                            <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-medium bg-muted text-muted-foreground border border-border">
                              Status: {applicationStatusLabel(app.application_status)}
                            </span>
                          )}
                        </div>
                      </div>

                      {/* 4-Stage Pipeline Stepper */}
                      <div className="pt-6">
                        <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-4">
                          Application Progress
                        </h4>

                        <div className="grid grid-cols-1 sm:grid-cols-4 gap-4">
                          {/* Stage 1: Submitted */}
                          <div className="p-3.5 rounded-xl bg-muted/40 border border-emerald-500/30">
                            <div className="flex items-center gap-2 text-xs font-semibold text-emerald-600 dark:text-emerald-400 mb-1">
                              <span>✓</span> Stage 1: Submitted
                            </div>
                            <p className="text-xs text-muted-foreground">Application & CV received</p>
                          </div>

                          {/* Stage 2: AI CV Screening */}
                          <div
                            className={cn(
                              "p-3.5 rounded-xl bg-muted/40 border",
                              app.passed_screening
                                ? "border-emerald-500/30"
                                : app.cv_score !== null && app.cv_score !== undefined
                                ? "border-amber-500/30"
                                : "border-border"
                            )}
                          >
                            <div className="flex items-center justify-between text-xs font-semibold mb-1">
                              <span className={app.passed_screening ? "text-emerald-600 dark:text-emerald-400" : "text-muted-foreground"}>
                                {app.passed_screening ? "✓" : "•"} Stage 2: CV Screen
                              </span>
                              {app.cv_score !== null && app.cv_score !== undefined && (
                                <span className="text-xs font-mono font-bold text-primary">
                                  {app.cv_score.toFixed(0)}%
                                </span>
                              )}
                            </div>
                            <p className="text-xs text-muted-foreground">
                              {app.passed_screening
                                ? "Screening benchmark met"
                                : app.cv_score !== null && app.cv_score !== undefined
                                ? "Screening benchmark not met"
                                : "Profile matching in progress"}
                            </p>
                          </div>

                          {/* Stage 3: AI Interview */}
                          <div
                            className={cn(
                              "p-3.5 rounded-xl bg-muted/40 border",
                              isCompleted
                                ? "border-emerald-500/30"
                                : isInterviewReady
                                ? "border-primary/60 ring-1 ring-primary/30"
                                : "border-border"
                            )}
                          >
                            <div className="flex items-center gap-2 text-xs font-semibold mb-1">
                              <span
                                className={
                                  isCompleted
                                    ? "text-emerald-600 dark:text-emerald-400"
                                    : isInterviewReady
                                    ? "text-primary"
                                    : "text-muted-foreground"
                                }
                              >
                                {isCompleted ? "✓" : isInterviewReady ? "⚡" : "•"} Stage 3: AI Interview
                              </span>
                            </div>
                            <p className="text-xs text-muted-foreground">
                              {isCompleted
                                ? "Session finished"
                                : isInterviewReady
                                ? "Ready to begin assessment"
                                : "Pending invitation"}
                            </p>
                          </div>

                          {/* Stage 4: Decision */}
                          <div
                            className={cn(
                              "p-3.5 rounded-xl bg-muted/40 border",
                              isCompleted && app.recommendation
                                ? "border-emerald-500/30"
                                : "border-border"
                            )}
                          >
                            <div className="flex items-center gap-2 text-xs font-semibold mb-1">
                              <span className={isCompleted ? "text-emerald-600 dark:text-emerald-400" : "text-muted-foreground"}>
                                {isCompleted ? "✓" : "•"} Stage 4: Decision
                              </span>
                            </div>
                            <p className="text-xs text-muted-foreground">
                              {isCompleted && app.recommendation
                                ? `Evaluation: ${app.recommendation.replace("_", " ")}`
                                : "Awaiting interview review"}
                            </p>
                          </div>
                        </div>
                      </div>
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
