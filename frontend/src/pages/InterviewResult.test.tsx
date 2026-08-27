import { cleanup, render, screen, waitFor } from "@testing-library/react"
import { MemoryRouter, Route, Routes } from "react-router-dom"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { afterEach, describe, expect, it, vi } from "vitest"
import { InterviewResultPage } from "./InterviewResult"
import type { InterviewDetail } from "@/types/api"

function jsonOk(payload: unknown): () => Response {
  return () => new Response(JSON.stringify({ data: payload }), { status: 200 })
}

function makeDetail(overrides: Partial<InterviewDetail> = {}): InterviewDetail {
  return {
    interview_id: "iv-1",
    application_id: "app-1",
    status: "completed",
    context_version: 1,
    total_questions: 1,
    questions: [{ idx: 1, content: "Tell me about Go", category: "technical" }],
    answers: [{ idx: 1, content: "Five years of Go.", answered_at: "2026-08-01T10:00:00Z" }],
    evaluation: null,
    created_at: "2026-08-01T09:00:00Z",
    ...overrides,
  }
}

function renderResult(detail: InterviewDetail) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  vi.stubGlobal(
    "fetch",
    vi.fn().mockImplementation(jsonOk(detail)),
  )
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={["/interviews/iv-1"]}>
        <Routes>
          <Route path="/interviews/:id" element={<InterviewResultPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe("InterviewResultPage — candidate Q&A section", () => {
  afterEach(() => {
    cleanup()
    vi.unstubAllGlobals()
  })

  it("renders the candidate Q&A pairs with question, answer and timestamp", async () => {
    renderResult(
      makeDetail({
        qa_pairs: [
          {
            question: "Is relocation required?",
            answer: "This role is fully remote.",
            created_at: "2026-08-01T09:30:00Z",
          },
          {
            question: "What is the stack?",
            answer: "Go and Postgres.",
            created_at: "2026-08-01T09:31:00Z",
          },
        ],
      }),
    )

    await waitFor(() => {
      expect(screen.getByText("Candidate Q&A")).toBeDefined()
    })
    expect(screen.getByText("Is relocation required?")).toBeDefined()
    expect(screen.getByText("This role is fully remote.")).toBeDefined()
    expect(screen.getByText("What is the stack?")).toBeDefined()
    expect(screen.getByText("Go and Postgres.")).toBeDefined()
  })

  it("renders an honest empty state when no questions were asked", async () => {
    renderResult(makeDetail({ qa_pairs: [] }))

    await waitFor(() => {
      expect(screen.getByText("Candidate Q&A")).toBeDefined()
    })
    expect(screen.getByText("No candidate questions asked.")).toBeDefined()
  })
})
