import { BrowserRouter, Route, Routes } from "react-router-dom"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { Toaster } from "sonner"
import { AppShell } from "@/components/AppShell"
import { RequireAuth } from "@/lib/require-auth"
import { CandidatesPage } from "@/pages/Candidates"
import { ChatPage } from "@/pages/Chat"
import { InvitePage } from "@/pages/Invite"
import { CVsPage } from "@/pages/CVs"
import { InterviewResultPage } from "@/pages/InterviewResult"
import { InterviewsPage } from "@/pages/Interviews"
import { JobsPage } from "@/pages/Jobs"
import { LoginPage } from "@/pages/Login"
import { RegisterPage } from "@/pages/Register"

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: 1, refetchOnWindowFocus: false } },
})

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/invite/:id" element={<InvitePage />} />
          <Route path="/chat/:id" element={<ChatPage />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route element={<RequireAuth />}>
            <Route element={<AppShell />}>
              <Route path="/jobs" element={<JobsPage />} />
              <Route path="/cvs" element={<CVsPage />} />
              <Route path="/candidates" element={<CandidatesPage />} />
              <Route path="/interviews" element={<InterviewsPage />} />
              <Route path="/interviews/:id" element={<InterviewResultPage />} />
            </Route>
          </Route>
          <Route path="*" element={<LoginPage />} />
        </Routes>
      </BrowserRouter>
      <Toaster position="top-right" />
    </QueryClientProvider>
  )
}
