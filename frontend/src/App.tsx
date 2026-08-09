import { BrowserRouter, Route, Routes } from "react-router-dom"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { Toaster } from "sonner"
import { AppShell } from "@/components/AppShell"
import { RequireAuth } from "@/lib/require-auth"
import { LoginPage } from "@/pages/Login"
import { RegisterPage } from "@/pages/Register"

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: 1, refetchOnWindowFocus: false } },
})

// Placeholder pages — replaced by B2/B3 builds.
function Placeholder({ title }: { title: string }) {
  return (
    <div className="flex h-full min-h-[50vh] flex-col items-center justify-center gap-2 text-center">
      <h1 className="font-display text-xl">{title}</h1>
      <p className="text-sm text-muted-foreground">Under construction</p>
    </div>
  )
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route element={<RequireAuth />}>
            <Route element={<AppShell />}>
              <Route path="/jobs" element={<Placeholder title="Jobs" />} />
              <Route path="/cvs" element={<Placeholder title="CVs" />} />
              <Route path="/candidates" element={<Placeholder title="Candidates" />} />
              <Route path="/interviews" element={<Placeholder title="Interviews" />} />
            </Route>
          </Route>
          <Route path="*" element={<LoginPage />} />
        </Routes>
      </BrowserRouter>
      <Toaster position="top-right" />
    </QueryClientProvider>
  )
}
