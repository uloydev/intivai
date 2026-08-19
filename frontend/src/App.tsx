import { BrowserRouter, Route, Routes } from "react-router-dom"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { Toaster } from "sonner"
import React, { Suspense } from "react"
import { AppShell } from "@/components/AppShell"
import { ErrorBoundary } from "@/components/ErrorBoundary"
import { RequireAuth } from "@/lib/require-auth"
import { PublicLayout } from "@/components/PublicLayout"

const LandingPage = React.lazy(() => import("@/pages/Landing").then(m => ({ default: m.LandingPage })))
const CareersPage = React.lazy(() => import("@/pages/Careers").then(m => ({ default: m.CareersPage })))
const CandidatePortal = React.lazy(() => import("@/pages/CandidatePortal").then(m => ({ default: m.CandidatePortal })))
const DashboardPage = React.lazy(() => import("@/pages/Dashboard").then(m => ({ default: m.DashboardPage })))
const CandidatesPage = React.lazy(() => import("@/pages/Candidates").then(m => ({ default: m.CandidatesPage })))
const CandidateReviewPage = React.lazy(() => import("@/pages/CandidateReview").then(m => ({ default: m.CandidateReviewPage })))
const ChatPage = React.lazy(() => import("@/pages/Chat").then(m => ({ default: m.ChatPage })))
const InvitePage = React.lazy(() => import("@/pages/Invite").then(m => ({ default: m.InvitePage })))
const CVsPage = React.lazy(() => import("@/pages/CVs").then(m => ({ default: m.CVsPage })))
const InterviewResultPage = React.lazy(() => import("@/pages/InterviewResult").then(m => ({ default: m.InterviewResultPage })))
const InterviewsPage = React.lazy(() => import("@/pages/Interviews").then(m => ({ default: m.InterviewsPage })))
const JobsPage = React.lazy(() => import("@/pages/Jobs").then(m => ({ default: m.JobsPage })))
const LoginPage = React.lazy(() => import("@/pages/Login").then(m => ({ default: m.LoginPage })))
const InterviewVoicePage = React.lazy(() => import("@/pages/InterviewVoice").then(m => ({ default: m.InterviewVoicePage })))
const RegisterPage = React.lazy(() => import("@/pages/Register").then(m => ({ default: m.RegisterPage })))
const CompanyContextPage = React.lazy(() => import("@/pages/CompanyContext").then(m => ({ default: m.CompanyContextPage })))

function NotFoundPage() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-2 bg-background text-center">
      <h1 className="font-display text-2xl">404</h1>
      <p className="text-sm text-muted-foreground">Page not found</p>
    </div>
  )
}

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: 1, refetchOnWindowFocus: false, staleTime: 30_000 } },
})

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ErrorBoundary>
        <BrowserRouter>
          <Suspense fallback={<div className="flex h-screen w-full items-center justify-center text-sm text-muted-foreground">Loading...</div>}>
            <Routes>
              {/* Public Landing & Careers & Candidate Portal */}
              <Route element={<PublicLayout />}>
                <Route path="/" element={<LandingPage />} />
                <Route path="/careers" element={<CareersPage />} />
                <Route path="/candidate/portal" element={<CandidatePortal />} />
              </Route>

              {/* Candidate Direct Flows */}
              <Route path="/candidate-review/:id" element={<CandidateReviewPage />} />
              <Route path="/invite/:id" element={<InvitePage />} />
              <Route path="/chat/:id" element={<ChatPage />} />
              <Route path="/voice/:id" element={<InterviewVoicePage />} />

              {/* Auth */}
              <Route path="/login" element={<LoginPage />} />
              <Route path="/register" element={<RegisterPage />} />

              {/* Recruiter Authed Workspace */}
              <Route element={<RequireAuth />}>
                <Route element={<AppShell />}>
                  <Route path="/dashboard" element={<DashboardPage />} />
                  <Route path="/jobs" element={<JobsPage />} />
                  <Route path="/cvs" element={<CVsPage />} />
                  <Route path="/candidates" element={<CandidatesPage />} />
                  <Route path="/candidates/:id" element={<CandidatesPage />} />
                  <Route path="/interviews" element={<InterviewsPage />} />
                  <Route path="/interviews/:id" element={<InterviewResultPage />} />
                  <Route path="/interviews/:id/result" element={<InterviewResultPage />} />
                  <Route path="/company-context" element={<CompanyContextPage />} />
                </Route>
              </Route>

              <Route path="*" element={<NotFoundPage />} />
            </Routes>
          </Suspense>
        </BrowserRouter>
        <Toaster position="top-right" />
      </ErrorBoundary>
    </QueryClientProvider>
  )
}
