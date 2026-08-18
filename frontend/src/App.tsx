import { BrowserRouter, Route, Routes } from "react-router-dom"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { Toaster } from "sonner"
import { AppShell } from "@/components/AppShell"
import { RequireAuth } from "@/lib/require-auth"
import { PublicLayout } from "@/components/PublicLayout"
import { LandingPage } from "@/pages/Landing"
import { CareersPage } from "@/pages/Careers"
import { CandidatePortal } from "@/pages/CandidatePortal"
import { DashboardPage } from "@/pages/Dashboard"
import { CandidatesPage } from "@/pages/Candidates"
import { ChatPage } from "@/pages/Chat"
import { InvitePage } from "@/pages/Invite"
import { CVsPage } from "@/pages/CVs"
import { InterviewResultPage } from "@/pages/InterviewResult"
import { InterviewsPage } from "@/pages/Interviews"
import { JobsPage } from "@/pages/Jobs"
import { LoginPage } from "@/pages/Login"
import { InterviewVoicePage } from "@/pages/InterviewVoice"
import { RegisterPage } from "@/pages/Register"
import { CompanyContextPage } from "@/pages/CompanyContext"

function NotFoundPage() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-2 bg-background text-center">
      <h1 className="font-display text-2xl">404</h1>
      <p className="text-sm text-muted-foreground">Page not found</p>
    </div>
  )
}

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: 1, refetchOnWindowFocus: false } },
})

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          {/* Public Landing & Careers & Candidate Portal */}
          <Route element={<PublicLayout />}>
            <Route path="/" element={<LandingPage />} />
            <Route path="/careers" element={<CareersPage />} />
            <Route path="/candidate/portal" element={<CandidatePortal />} />
          </Route>

          {/* Candidate Direct Flows */}
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
              <Route path="/interviews" element={<InterviewsPage />} />
              <Route path="/interviews/:id" element={<InterviewResultPage />} />
              <Route path="/company-context" element={<CompanyContextPage />} />
            </Route>
          </Route>

          <Route path="*" element={<NotFoundPage />} />
        </Routes>
      </BrowserRouter>
      <Toaster position="top-right" />
    </QueryClientProvider>
  )
}
