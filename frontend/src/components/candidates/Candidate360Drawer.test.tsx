import { render, screen, fireEvent } from "@testing-library/react"
import { MemoryRouter } from "react-router-dom"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { describe, it, expect } from "vitest"
import { Candidate360Drawer } from "./Candidate360Drawer"
import type { Application } from "@/types/api"

const mockApplication: Application = {
  id: "app-1",
  candidate_id: "cand-1",
  candidate_name: "Wahyu Miftahul",
  candidate_email: "wahyu@example.com",
  job_id: "job-1",
  job_title: "Senior Distributed Systems Engineer",
  status: "active",
  stage: "screening_passed",
  cv_score: 88,
  passed_screening: true,
  years_experience: 5,
  matched_skills: ["Go", "Distributed Systems", "Kubernetes"],
  screening_rationale: "Strong alignment with backend requirements.",
}

describe("Candidate360Drawer Component", () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })

  it("renders candidate header, screening score, and verified skills", () => {
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <Candidate360Drawer
            application={mockApplication}
            open={true}
            onClose={() => undefined}
          />
        </MemoryRouter>
      </QueryClientProvider>
    )

    expect(screen.getByText("Wahyu Miftahul")).toBeDefined()
    expect(screen.getByText("wahyu@example.com")).toBeDefined()
    expect(screen.getByText("Senior Distributed Systems Engineer")).toBeDefined()
    expect(screen.getByText("88%")).toBeDefined()
    expect(screen.getByText("Passed Threshold")).toBeDefined()
    expect(screen.getByText("5+ Years")).toBeDefined()
    expect(screen.getByText("Distributed Systems")).toBeDefined()
  })

  it("switches to Assessment tab and Decision tab seamlessly", () => {
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <Candidate360Drawer
            application={mockApplication}
            open={true}
            onClose={() => undefined}
          />
        </MemoryRouter>
      </QueryClientProvider>
    )

    // Switch to assessment tab
    fireEvent.click(screen.getByText("AI Assessment & Telemetry"))
    expect(screen.getByText("Interview Assessment State")).toBeDefined()
    expect(screen.getByText("Generate AI Interview Session")).toBeDefined()

    // Switch to decision tab
    fireEvent.click(screen.getByText("Hiring Decision & Notes"))
    expect(screen.getByText("Update Candidate Stage")).toBeDefined()
    expect(screen.getByText("Save Candidate Decision & Notes")).toBeDefined()
  })
})
