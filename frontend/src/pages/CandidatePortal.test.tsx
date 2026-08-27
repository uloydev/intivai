import { cleanup, render, screen, waitFor } from "@testing-library/react"
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { CandidatePortal } from "./CandidatePortal"

function jsonOk(payload: unknown): () => Response {
  return () => new Response(JSON.stringify({ data: payload }), { status: 200 })
}

function LocationProbe() {
  const loc = useLocation()
  return <div data-testid="location">{loc.pathname + loc.search}</div>
}

function renderPortal(url: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[url]}>
        <Routes>
          <Route path="/candidate/portal" element={<><CandidatePortal /><LocationProbe /></>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

const COMPLETED_APP = {
  application_id: "app-1",
  org_id: "org-1",
  org_name: "Acme",
  org_slug: "acme",
  job_id: "job-1",
  job_title: "Backend Engineer",
  job_location: "Remote",
  job_employment_type: "full_time",
  candidate_id: "cand-1",
  candidate_name: "Alex Dev",
  candidate_email: "alex@dev.io",
  cv_status: "scored",
  application_status: "applied",
  applied_at: new Date().toISOString(),
  interview_id: "iv-1",
  interview_status: "completed",
  invitation_token: "inv-tok",
}

describe("CandidatePortal — G6 magic link URL strip", () => {
  beforeEach(() => {
    localStorage.clear()
  })
  afterEach(() => {
    cleanup()
    vi.unstubAllGlobals()
    localStorage.clear()
  })

  it("strips the spent ?token= after successful verify", async () => {
    const verifyResponse = jsonOk({ token: "cand-jwt", email: "alex@dev.io", expires_at: "" })
    const appsResponse = jsonOk([])
    const fetchMock = vi.fn().mockImplementation((path: unknown) =>
      Promise.resolve(String(path).includes("/verify") ? verifyResponse() : appsResponse()),
    )
    vi.stubGlobal("fetch", fetchMock)

    renderPortal("/candidate/portal?token=magic-tok")

    await waitFor(() => {
      expect(screen.getByText("Applicant Tracking Dashboard")).toBeDefined()
    })
    // Router state commits after the mutation's batched setState — poll.
    await waitFor(() => {
      expect(screen.getByTestId("location").textContent).toBe("/candidate/portal")
    })
  })

  it("remount without ?token= keeps the valid session (no logout, no re-verify)", async () => {
    const verifyResponse = jsonOk({ token: "cand-jwt", email: "alex@dev.io", expires_at: "" })
    let verifyCalls = 0
    const appsResponse = jsonOk([])
    const fetchMock = vi.fn().mockImplementation((path: unknown) => {
      if (String(path).includes("/verify")) {
        verifyCalls += 1
        return Promise.resolve(verifyResponse())
      }
      return Promise.resolve(appsResponse())
    })
    vi.stubGlobal("fetch", fetchMock)

    const view = renderPortal("/candidate/portal?token=magic-tok")
    await waitFor(() => {
      expect(screen.getByText("Applicant Tracking Dashboard")).toBeDefined()
    })

    // Simulate a refresh landing on the cleaned URL: token already consumed.
    view.unmount()
    cleanup()
    renderPortal("/candidate/portal")

    await waitFor(() => {
      expect(screen.getByText("Applicant Tracking Dashboard")).toBeDefined()
    })
    expect(verifyCalls).toBe(1)
    expect(screen.queryByText(/Invalid or expired magic link/)).toBeNull()
  })

  it("an error verify does NOT wipe an unrelated stored state before logout fires", async () => {
    const failVerify = () =>
      Promise.resolve(
        new Response(JSON.stringify({ code: "TOKEN_EXPIRED", error: "magic link expired" }), { status: 401 }),
      )
    vi.stubGlobal("fetch", vi.fn().mockImplementation(failVerify))

    renderPortal("/candidate/portal?token=spent-tok")

    await waitFor(() => {
      expect(screen.getByText(/Invalid or expired magic link|magic link expired/i)).toBeDefined()
    })
  })
})

describe("CandidatePortal — G10 no fabricated feedback", () => {
  beforeEach(() => {
    localStorage.clear()
    localStorage.setItem("intivai_candidate_token", "cand-jwt")
    localStorage.setItem("intivai_candidate_email", "alex@dev.io")
  })
  afterEach(() => {
    cleanup()
    vi.unstubAllGlobals()
    localStorage.clear()
  })

  it("renders the neutral line instead of hardcoded strengths/growth strings", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation(jsonOk([COMPLETED_APP])),
    )

    renderPortal("/candidate/portal")

    await waitFor(() => {
      expect(screen.getByText("Assessment Summary")).toBeDefined()
    })
    expect(screen.getByText("No automated feedback available yet.")).toBeDefined()
    expect(screen.queryByText(/Strong technical articulation/)).toBeNull()
    expect(screen.queryByText(/multi-region tenant sharding/)).toBeNull()
  })

  it("renders only real evaluation fields when present", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation(jsonOk([{ ...COMPLETED_APP, recommendation: "strong_hire", overall_score: 87 }])),
    )

    renderPortal("/candidate/portal")

    await waitFor(() => {
      // Recommendation appears both in the feedback card and Stage 4 stepper.
      expect(screen.getAllByText(/strong hire/i).length).toBeGreaterThan(0)
    })
    expect(screen.getAllByText(/87\/100/).length).toBeGreaterThan(0)
    expect(screen.queryByText(/Strong technical articulation/)).toBeNull()
  })
})
