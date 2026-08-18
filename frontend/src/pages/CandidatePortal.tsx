import { useState, useEffect } from "react"
import { useSearchParams, Link } from "react-router-dom"
import { api, ApiError } from "../lib/api"
import type { CandidateApplicationItem, CandidateVerifyResponse } from "../types/api"

export function CandidatePortal() {
  const [searchParams] = useSearchParams()
  const [email, setEmail] = useState(localStorage.getItem("intivai_candidate_email") || "")
  const [otpCode, setOtpCode] = useState("")
  const [step, setStep] = useState<"email" | "otp" | "dashboard">(
    localStorage.getItem("intivai_candidate_token") ? "dashboard" : "email"
  )
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [infoMsg, setInfoMsg] = useState<string | null>(null)
  const [applications, setApplications] = useState<CandidateApplicationItem[]>([])
  const [fetchingApps, setFetchingApps] = useState(false)

  // Handle direct Magic Link URL token (?token=...): the raw magic token is
  // NOT a JWT — exchange it for a real candidate token first.
  useEffect(() => {
    const magicToken = searchParams.get("token")
    if (magicToken) {
      ;(async () => {
        setLoading(true)
        setError(null)
        try {
          const res = await api.post<CandidateVerifyResponse>("/public/candidate/auth/verify", {
            token: magicToken,
          })
          localStorage.setItem("intivai_candidate_token", res.token)
          localStorage.setItem("intivai_candidate_email", res.email)
          setEmail(res.email)
          setStep("dashboard")
          loadApplications()
        } catch (err) {
          setError(err instanceof ApiError ? err.message : "Invalid or expired magic link.")
          handleLogout()
        } finally {
          setLoading(false)
        }
      })()
    } else if (localStorage.getItem("intivai_candidate_token")) {
      loadApplications()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams])

  async function handleSendOTP(e: React.FormEvent) {
    e.preventDefault()
    if (!email.trim() || !email.includes("@")) {
      setError("Please enter a valid email address.")
      return
    }
    setLoading(true)
    setError(null)
    setInfoMsg(null)
    try {
      await api.post("/public/candidate/auth/otp", { email: email.trim().toLowerCase() })
      setStep("otp")
      setInfoMsg(`A 6-digit verification code has been dispatched to ${email}.`)
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError("Failed to dispatch verification code. Please try again.")
      }
    } finally {
      setLoading(false)
    }
  }

  async function handleVerifyOTP(e: React.FormEvent) {
    e.preventDefault()
    if (!otpCode.trim() || otpCode.length !== 6) {
      setError("Please enter the complete 6-digit verification code.")
      return
    }
    setLoading(true)
    setError(null)
    try {
      const res = await api.post<CandidateVerifyResponse>("/public/candidate/auth/verify", {
        email: email.trim().toLowerCase(),
        code: otpCode.trim(),
      })
      localStorage.setItem("intivai_candidate_token", res.token)
      localStorage.setItem("intivai_candidate_email", res.email)
      setStep("dashboard")
      loadApplications()
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError("Invalid or expired verification code.")
      }
    } finally {
      setLoading(false)
    }
  }

  async function loadApplications() {
    setFetchingApps(true)
    setError(null)
    try {
      const data = await api.get<CandidateApplicationItem[]>("/candidate/portal/applications")
      setApplications(data || [])
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        handleLogout()
      } else {
        setError("Unable to load applications at this time.")
      }
    } finally {
      setFetchingApps(false)
    }
  }

  function handleLogout() {
    localStorage.removeItem("intivai_candidate_token")
    localStorage.removeItem("intivai_candidate_email")
    setStep("email")
    setOtpCode("")
    setApplications([])
  }

  return (
    <div className="min-h-[calc(100vh-4rem)] bg-slate-950 text-slate-100 py-12 px-4 sm:px-6 lg:px-8 relative overflow-hidden">
      {/* Background glow effects */}
      <div className="absolute top-1/4 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[350px] bg-indigo-600/10 rounded-full blur-3xl pointer-events-none" />
      <div className="absolute bottom-10 right-10 w-[400px] h-[250px] bg-cyan-600/10 rounded-full blur-3xl pointer-events-none" />

      <div className="max-w-4xl mx-auto relative z-10">
        {step !== "dashboard" ? (
          /* Authentication Screen */
          <div className="max-w-md mx-auto">
            <div className="text-center mb-8">
              <div className="inline-flex items-center justify-center w-12 h-12 rounded-xl bg-gradient-to-tr from-indigo-600 to-cyan-500 text-white font-bold text-xl shadow-lg shadow-indigo-500/20 mb-4">
                ✦
              </div>
              <h1 className="text-3xl font-extrabold tracking-tight text-white sm:text-4xl">
                Candidate Portal
              </h1>
              <p className="mt-2 text-sm text-slate-400">
                Track your job applications, screening scores, and launch your AI assessment sessions.
              </p>
            </div>

            <div className="bg-slate-900/80 border border-slate-800 rounded-2xl p-8 backdrop-blur-xl shadow-2xl">
              {error && (
                <div className="mb-6 p-4 rounded-xl bg-rose-950/50 border border-rose-800/80 text-rose-300 text-sm flex items-start gap-3">
                  <span className="text-rose-400 font-bold">✕</span>
                  <span>{error}</span>
                </div>
              )}

              {infoMsg && (
                <div className="mb-6 p-4 rounded-xl bg-indigo-950/50 border border-indigo-800/80 text-indigo-300 text-sm flex items-start gap-3">
                  <span className="text-indigo-400 font-bold">ℹ</span>
                  <span>{infoMsg}</span>
                </div>
              )}

              {step === "email" ? (
                <form onSubmit={handleSendOTP} className="space-y-5">
                  <div>
                    <label htmlFor="candidate-email" className="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-2">
                      Applicant Email Address
                    </label>
                    <input
                      id="candidate-email"
                      type="email"
                      required
                      value={email}
                      onChange={(e) => setEmail(e.target.value)}
                      placeholder="e.g. alex.dev@gmail.com"
                      className="w-full px-4 py-3 bg-slate-950/60 border border-slate-800 rounded-xl text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent transition-all"
                    />
                    <p className="mt-2 text-xs text-slate-500">
                      We'll send you a passwordless 6-digit verification code & magic login link.
                    </p>
                  </div>

                  <button
                    type="submit"
                    disabled={loading}
                    className="w-full py-3.5 px-4 rounded-xl font-semibold text-white bg-gradient-to-r from-indigo-600 to-indigo-700 hover:from-indigo-500 hover:to-indigo-600 active:scale-[0.99] transition-all shadow-lg shadow-indigo-600/25 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
                  >
                    {loading ? (
                      <>
                        <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                        <span>Sending Security Code...</span>
                      </>
                    ) : (
                      <span>Send Verification Code →</span>
                    )}
                  </button>

                  <div className="pt-4 border-t border-slate-800/80 flex items-center justify-between text-xs text-slate-400">
                    <Link to="/careers" className="hover:text-indigo-400 transition-colors">
                      ← Browse Open Positions
                    </Link>
                  </div>
                </form>
              ) : (
                <form onSubmit={handleVerifyOTP} className="space-y-5">
                  <div>
                    <div className="flex items-center justify-between mb-2">
                      <label htmlFor="otp-code" className="block text-xs font-semibold text-slate-300 uppercase tracking-wider">
                        6-Digit Security Code
                      </label>
                      <button
                        type="button"
                        onClick={() => { setStep("email"); setError(null); }}
                        className="text-xs text-indigo-400 hover:text-indigo-300"
                      >
                        Change Email
                      </button>
                    </div>
                    <input
                      id="otp-code"
                      type="text"
                      maxLength={6}
                      required
                      autoFocus
                      value={otpCode}
                      onChange={(e) => setOtpCode(e.target.value.replace(/\D/g, ""))}
                      placeholder="• • • • • •"
                      className="w-full px-4 py-3 bg-slate-950/60 border border-slate-800 rounded-xl text-center text-2xl tracking-[0.5em] font-mono text-white placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent transition-all"
                    />
                    <p className="mt-2 text-xs text-slate-500 text-center">
                      Code expires in 10 minutes. Check your email.
                    </p>
                  </div>

                  <button
                    type="submit"
                    disabled={loading || otpCode.length !== 6}
                    className="w-full py-3.5 px-4 rounded-xl font-semibold text-white bg-gradient-to-r from-indigo-600 to-indigo-700 hover:from-indigo-500 hover:to-indigo-600 active:scale-[0.99] transition-all shadow-lg shadow-indigo-600/25 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
                  >
                    {loading ? (
                      <>
                        <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                        <span>Verifying...</span>
                      </>
                    ) : (
                      <span>Access Candidate Portal →</span>
                    )}
                  </button>

                  <div className="text-center">
                    <button
                      type="button"
                      onClick={handleSendOTP}
                      className="text-xs text-slate-400 hover:text-indigo-400 transition-colors"
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
            <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 p-6 bg-slate-900/80 border border-slate-800 rounded-2xl backdrop-blur-xl">
              <div>
                <div className="flex items-center gap-2 mb-1">
                  <span className="inline-block w-2.5 h-2.5 rounded-full bg-emerald-500 animate-pulse" />
                  <h2 className="text-xl font-bold text-white">Applicant Tracking Dashboard</h2>
                </div>
                <p className="text-sm text-slate-400">
                  Signed in as <span className="text-indigo-300 font-medium">{email}</span>
                </p>
              </div>

              <div className="flex items-center gap-3">
                <button
                  type="button"
                  onClick={loadApplications}
                  disabled={fetchingApps}
                  className="px-4 py-2 text-xs font-semibold text-slate-300 bg-slate-800/80 hover:bg-slate-700 rounded-xl transition-colors flex items-center gap-1.5"
                >
                  <span className={fetchingApps ? "animate-spin" : ""}>↻</span> Refresh
                </button>
                <button
                  type="button"
                  onClick={handleLogout}
                  className="px-4 py-2 text-xs font-semibold text-rose-400 hover:text-rose-300 bg-rose-950/30 hover:bg-rose-950/60 border border-rose-900/40 rounded-xl transition-colors"
                >
                  Sign Out
                </button>
              </div>
            </div>

            {/* Applications List */}
            {fetchingApps ? (
              <div className="p-12 text-center bg-slate-900/40 border border-slate-800/60 rounded-2xl">
                <div className="w-8 h-8 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin mx-auto mb-4" />
                <p className="text-slate-400 text-sm">Retrieving your application history...</p>
              </div>
            ) : applications.length === 0 ? (
              <div className="p-12 text-center bg-slate-900/40 border border-slate-800/60 rounded-2xl space-y-4">
                <div className="w-16 h-16 rounded-full bg-slate-800/80 text-slate-400 flex items-center justify-center text-2xl mx-auto">
                  📋
                </div>
                <div>
                  <h3 className="text-lg font-semibold text-white">No Applications Found</h3>
                  <p className="text-sm text-slate-400 mt-1">
                    You haven't submitted any job applications under this email address yet.
                  </p>
                </div>
                <Link
                  to="/careers"
                  className="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl text-sm font-semibold text-white bg-indigo-600 hover:bg-indigo-500 transition-colors shadow-lg shadow-indigo-600/20"
                >
                  Explore Open Careers →
                </Link>
              </div>
            ) : (
              <div className="space-y-6">
                {applications.map((app) => {
                  const isCompleted = app.interview_status === "completed"
                  const hasInterviewTicket = Boolean(app.invitation_token)
                  const isInterviewReady = hasInterviewTicket && app.interview_status !== "completed"

                  return (
                    <div
                      key={app.application_id}
                      className="p-6 bg-slate-900/90 border border-slate-800 rounded-2xl backdrop-blur-xl shadow-xl transition-all hover:border-slate-700"
                    >
                      {/* Top Job Info Header */}
                      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4 pb-6 border-b border-slate-800/80">
                        <div>
                          <div className="flex items-center gap-2 mb-1.5 flex-wrap">
                            <span className="px-2.5 py-0.5 rounded-full text-xs font-semibold bg-indigo-950 text-indigo-300 border border-indigo-800/60">
                              {app.org_name}
                            </span>
                            <span className="px-2.5 py-0.5 rounded-full text-xs font-medium bg-slate-800 text-slate-300">
                              {app.job_employment_type}
                            </span>
                            <span className="px-2.5 py-0.5 rounded-full text-xs font-medium bg-slate-800 text-slate-300">
                              {app.job_location}
                            </span>
                          </div>
                          <h3 className="text-xl font-bold text-white">{app.job_title}</h3>
                          <p className="text-xs text-slate-400 mt-1">
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
                              <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-semibold bg-emerald-950/80 text-emerald-300 border border-emerald-800/60">
                                ✓ Assessment Complete
                              </span>
                              {app.overall_score !== null && app.overall_score !== undefined && (
                                <p className="text-xs text-slate-400 mt-1">
                                  Score: <strong className="text-white">{app.overall_score.toFixed(1)}/100</strong>
                                </p>
                              )}
                            </div>
                          ) : (
                            <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-medium bg-slate-800 text-slate-300">
                              Status: {app.application_status}
                            </span>
                          )}
                        </div>
                      </div>

                      {/* 4-Stage Pipeline Stepper */}
                      <div className="pt-6">
                        <h4 className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-4">
                          Application Progress
                        </h4>

                        <div className="grid grid-cols-1 sm:grid-cols-4 gap-4">
                          {/* Stage 1: Submitted */}
                          <div className="p-3.5 rounded-xl bg-slate-950/60 border border-emerald-800/40">
                            <div className="flex items-center gap-2 text-xs font-semibold text-emerald-400 mb-1">
                              <span>✓</span> Stage 1: Submitted
                            </div>
                            <p className="text-xs text-slate-400">Application & CV received</p>
                          </div>

                          {/* Stage 2: AI CV Screening */}
                          <div
                            className={`p-3.5 rounded-xl bg-slate-950/60 border ${
                              app.passed_screening
                                ? "border-emerald-800/40"
                                : app.cv_score !== null && app.cv_score !== undefined
                                ? "border-amber-800/40"
                                : "border-slate-800"
                            }`}
                          >
                            <div className="flex items-center justify-between text-xs font-semibold mb-1">
                              <span className={app.passed_screening ? "text-emerald-400" : "text-slate-300"}>
                                {app.passed_screening ? "✓" : "•"} Stage 2: CV Screen
                              </span>
                              {app.cv_score !== null && app.cv_score !== undefined && (
                                <span className="text-xs font-mono font-bold text-indigo-300">
                                  {app.cv_score.toFixed(0)}%
                                </span>
                              )}
                            </div>
                            <p className="text-xs text-slate-400">
                              {app.passed_screening ? "Screening benchmark met" : "Profile matching in progress"}
                            </p>
                          </div>

                          {/* Stage 3: AI Interview */}
                          <div
                            className={`p-3.5 rounded-xl bg-slate-950/60 border ${
                              isCompleted
                                ? "border-emerald-800/40"
                                : isInterviewReady
                                ? "border-indigo-600/60 ring-1 ring-indigo-500/30"
                                : "border-slate-800"
                            }`}
                          >
                            <div className="flex items-center gap-2 text-xs font-semibold mb-1">
                              <span
                                className={
                                  isCompleted
                                    ? "text-emerald-400"
                                    : isInterviewReady
                                    ? "text-indigo-400"
                                    : "text-slate-500"
                                }
                              >
                                {isCompleted ? "✓" : isInterviewReady ? "⚡" : "•"} Stage 3: AI Interview
                              </span>
                            </div>
                            <p className="text-xs text-slate-400">
                              {isCompleted
                                ? "Session finished"
                                : isInterviewReady
                                ? "Ready to begin assessment"
                                : "Pending invitation"}
                            </p>
                          </div>

                          {/* Stage 4: Decision */}
                          <div
                            className={`p-3.5 rounded-xl bg-slate-950/60 border ${
                              isCompleted && app.recommendation
                                ? "border-emerald-800/40"
                                : "border-slate-800"
                            }`}
                          >
                            <div className="flex items-center gap-2 text-xs font-semibold mb-1">
                              <span className={isCompleted ? "text-emerald-400" : "text-slate-500"}>
                                {isCompleted ? "✓" : "•"} Stage 4: Decision
                              </span>
                            </div>
                            <p className="text-xs text-slate-400">
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
