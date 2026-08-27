import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react"
import { MemoryRouter, Route, Routes } from "react-router-dom"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { StrictMode } from "react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { CandidateReviewPage } from "./CandidateReview"

vi.mock("sonner", () => ({
  toast: { error: vi.fn(), success: vi.fn() },
}))

import { toast } from "sonner"

const CV_DETAIL = {
  id: "cv-1",
  name: "Alex Dev",
  email: "alex@dev.io",
  status: "pending_review",
  cv_path: "/cvs/cv-1.pdf",
  created_at: "2026-08-27T00:00:00Z",
  cv_structured: {
    skills: ["Go", "Postgres"],
    experience_years: 5,
    education: "BS Computer Science",
    certifications: ["AWS"],
    summary: "Backend engineer with 5 years of experience.",
  },
}

function okJson(payload: unknown) {
  return new Response(JSON.stringify({ data: payload }), { status: 200 })
}

function errJson(status: number, error: string) {
  return new Response(JSON.stringify({ code: "TOKEN_INVALID", error }), { status })
}

function renderPage(url: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <StrictMode>
      <QueryClientProvider client={qc}>
        <MemoryRouter initialEntries={[url]}>
          <Routes>
            <Route path="/candidate-review/:id" element={<CandidateReviewPage />} />
            <Route path="/candidate/portal" element={<div>candidate portal</div>} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>
    </StrictMode>,
  )
}

describe("CandidateReviewPage", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })
  afterEach(() => {
    cleanup()
    vi.unstubAllGlobals()
  })

  it("renders the dynamic form prefilled from cv_structured and NO raw JSON textarea", async () => {
    const fetchMock = vi.fn().mockResolvedValue(okJson(CV_DETAIL))
    vi.stubGlobal("fetch", fetchMock)
    renderPage("/candidate-review/tok-1")

    expect(await screen.findByLabelText(/name/i)).toBeDefined()
    expect((screen.getByLabelText(/name/i) as HTMLInputElement).value).toBe("Alex Dev")
    expect((screen.getByLabelText(/email/i) as HTMLInputElement).value).toBe("alex@dev.io")
    expect((screen.getByLabelText(/experience years/i) as HTMLInputElement).value).toBe("5")
    expect((screen.getByLabelText(/education/i) as HTMLInputElement).value).toBe("BS Computer Science")
    expect((screen.getByLabelText(/professional summary/i) as HTMLTextAreaElement).value).toContain("Backend engineer")
    expect(screen.getByText("Go")).toBeDefined()
    expect(screen.getByText("AWS")).toBeDefined()
    expect(document.querySelector("textarea.font-mono")).toBeNull()
    expect(screen.queryByText(/structured data/i)).toBeNull()
  })

  it("submits the edited 7-field draft via POST confirm", async () => {
    const fetchMock = vi.fn().mockReset()
    fetchMock.mockResolvedValueOnce(okJson(CV_DETAIL))
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({ data: { status: "confirmed" } }), { status: 200 }))
    vi.stubGlobal("fetch", fetchMock)
    renderPage("/candidate-review/tok-1")

    await screen.findByLabelText(/name/i)
    const name = screen.getByLabelText(/name/i) as HTMLInputElement
    fireEvent.change(name, { target: { value: "Alex New" } })
    const skills = screen.getByLabelText(/skills/i) as HTMLInputElement
    fireEvent.change(skills, { target: { value: "Python" } })
    fireEvent.keyDown(skills, { key: "Enter" })
    fireEvent.click(screen.getByRole("button", { name: /confirm & continue/i }))

    await waitFor(() => {
      const postCall = fetchMock.mock.calls.find(([, init]) => init?.method === "POST")
      const second = postCall?.[1]
      expect(second).toBeDefined()
      expect(String(postCall?.[0])).toBe("/api/v1/public/candidate-review/tok-1/confirm")
      const body = JSON.parse(String((second as RequestInit).body))
      expect(body).toEqual({
        name: "Alex New",
        email: "alex@dev.io",
        skills: ["Go", "Postgres", "Python"],
        experience_years: 5,
        education: "BS Computer Science",
        certifications: ["AWS"],
        summary: "Backend engineer with 5 years of experience.",
      })
    })
    await waitFor(() => {
      expect(screen.getByText("candidate portal")).toBeDefined()
    })
  })

  it("shows the invalid link panel on GET 404", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(errJson(404, "review link not found")))
    renderPage("/candidate-review/tok-1")
    expect(await screen.findByText(/Review Link Invalid or Expired/i)).toBeDefined()
    expect(screen.queryByLabelText(/name/i)).toBeNull()
  })

  it("blocks submit when name is empty — no POST fired", async () => {
    const fetchMock = vi.fn().mockReset()
    fetchMock.mockResolvedValueOnce(okJson(CV_DETAIL))
    vi.stubGlobal("fetch", fetchMock)
    renderPage("/candidate-review/tok-1")

    await screen.findByLabelText(/name/i)
    fireEvent.change(screen.getByLabelText(/name/i), { target: { value: "" } })
    const button = screen.getByRole("button", { name: /confirm & continue/i }) as HTMLButtonElement
    expect(button.disabled).toBe(true)
    fireEvent.click(button)
    await new Promise((r) => setTimeout(r, 50))
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it("shows an error toast and stays on the page when POST fails", async () => {
    const fetchMock = vi.fn().mockReset()
    fetchMock.mockResolvedValueOnce(okJson(CV_DETAIL))
    fetchMock.mockResolvedValueOnce(errJson(400, "too many skills"))
    vi.stubGlobal("fetch", fetchMock)
    renderPage("/candidate-review/tok-1")

    await screen.findByLabelText(/name/i)
    fireEvent.click(screen.getByRole("button", { name: /confirm & continue/i }))

    await waitFor(() => {
      const postCall = fetchMock.mock.calls.find(([, init]) => init?.method === "POST")
      expect(postCall).toBeDefined()
    })
    expect(screen.queryByText("candidate portal")).toBeNull()
    expect(toast.error).toHaveBeenCalledWith("too many skills")
    expect(toast.success).not.toHaveBeenCalled()
  })
})
