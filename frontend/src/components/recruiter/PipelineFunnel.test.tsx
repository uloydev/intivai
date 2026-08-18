import { render, screen } from "@testing-library/react"
import { MemoryRouter } from "react-router-dom"
import { describe, it, expect } from "vitest"
import { PipelineFunnel } from "./PipelineFunnel"

describe("PipelineFunnel Component", () => {
  it("computes conversion rates and renders stages accurately", () => {
    render(
      <MemoryRouter>
        <PipelineFunnel
          totalApplied={100}
          totalScreened={60}
          totalInterviewed={30}
          totalRecommended={15}
        />
      </MemoryRouter>
    )

    expect(screen.getByText("Recruitment Pipeline Velocity")).toBeDefined()
    expect(screen.getByText("Overall Yield: 15%")).toBeDefined()
    expect(screen.getByText("Total Sourced & Applied")).toBeDefined()
    expect(screen.getByText("100")).toBeDefined()
    expect(screen.getByText("Passed AI CV Screening")).toBeDefined()
    expect(screen.getByText("60")).toBeDefined()
    expect(screen.getByText("60% Qualification Rate")).toBeDefined()
    expect(screen.getByText("AI Assessments Completed")).toBeDefined()
    expect(screen.getByText("30")).toBeDefined()
    expect(screen.getByText("Strong Hire Recommendations")).toBeDefined()
    expect(screen.getByText("15")).toBeDefined()
  })

  it("handles zero gracefully without dividing by zero", () => {
    render(
      <MemoryRouter>
        <PipelineFunnel
          totalApplied={0}
          totalScreened={0}
          totalInterviewed={0}
          totalRecommended={0}
        />
      </MemoryRouter>
    )

    expect(screen.getByText("Overall Yield: 0%")).toBeDefined()
  })
})
