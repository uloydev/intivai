import { render, screen } from "@testing-library/react"
import { MemoryRouter } from "react-router-dom"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { describe, it, expect } from "vitest"
import { CompanyContextPage } from "./CompanyContext"

describe("CompanyContextPage", () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })

  it("renders page header, prompt rails tabs, and presets", () => {
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <CompanyContextPage />
        </MemoryRouter>
      </QueryClientProvider>
    )

    expect(screen.getByText("Company Intelligence & AI Interview Rails")).toBeDefined()
    expect(screen.getByText("AI Persona & Prompt Rails")).toBeDefined()
    expect(screen.getByText("Interviewer Persona Presets")).toBeDefined()
    expect(screen.getByText("Engineering Excellence")).toBeDefined()
    expect(screen.getByText("Fast-Paced Startup Architect")).toBeDefined()
  })
})
