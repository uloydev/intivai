import { cleanup, render, screen, waitFor, fireEvent } from "@testing-library/react"
import { MemoryRouter } from "react-router-dom"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { SettingsPage } from "./Settings"

function renderSettings() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <SettingsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

const ADMIN_JWT = `x.${btoa(JSON.stringify({ sub: "u1", org_id: "org-1", role: "admin", exp: 9999999999 }))}.x`
const RECRUITER_JWT = `x.${btoa(JSON.stringify({ sub: "u1", org_id: "org-1", role: "recruiter", exp: 9999999999 }))}.x`

describe("SettingsPage — candidate Q&A limit (J13)", () => {
  beforeEach(() => {
    localStorage.clear()
  })
  afterEach(() => {
    cleanup()
    vi.unstubAllGlobals()
    localStorage.clear()
  })

  it("admin sees the input with no known limit and an honest placeholder/error notice", async () => {
    localStorage.setItem("intivai_token", ADMIN_JWT)
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("no GET endpoint")))
    renderSettings()

    // The GET has no backend route — the query settles into an honest error
    // notice instead of a fabricated default.
    await waitFor(() => {
      expect(screen.getByText(/current limit is not available/i)).toBeDefined()
    })
    expect(screen.getAllByText("Candidate Q&A limit").length).toBeGreaterThan(0)
    expect(screen.getByRole("button", { name: /save limit/i })).toBeDefined()
    expect((screen.getByLabelText("Candidate Q&A limit") as HTMLInputElement).placeholder).toBe("10")
  })

  it("saves a valid limit via PUT /orgs/:orgId/settings/candidate-qa-limit", async () => {
    localStorage.setItem("intivai_token", ADMIN_JWT)
    const fetchMock = vi.fn().mockImplementation((_: unknown, init?: RequestInit) =>
      Promise.resolve(
        init?.method === "PUT"
          ? new Response(JSON.stringify({ data: { candidate_qa_limit: 25 } }), { status: 200 })
          : Promise.reject(new Error("no GET")),
      ),
    )
    vi.stubGlobal("fetch", fetchMock)
    renderSettings()

    const input = await screen.findByLabelText("Candidate Q&A limit")
    fireEvent.change(input, { target: { value: "25" } })
    fireEvent.click(screen.getByRole("button", { name: /save limit/i }))

    await waitFor(() => {
      expect(
        fetchMock.mock.calls.some(([path, init]) =>
          init?.method === "PUT" && String(path).includes("/orgs/org-1/settings/candidate-qa-limit"),
        ),
      ).toBe(true)
    })
    const putCall = fetchMock.mock.calls.find(([, init]) => init?.method === "PUT")
    expect(String(putCall?.[0])).toBe("/api/v1/orgs/org-1/settings/candidate-qa-limit")
    const putInit = putCall?.[1] as RequestInit | undefined
    expect(JSON.parse(String(putInit?.body))).toEqual({ candidate_qa_limit: 25 })
  })

  it("blocks out-of-band values locally and refuses to call the API", async () => {
    localStorage.setItem("intivai_token", ADMIN_JWT)
    const fetchMock = vi.fn().mockImplementation((_: unknown, init?: RequestInit) =>
      Promise.resolve(
        init?.method === "PUT"
          ? new Response(JSON.stringify({ data: { candidate_qa_limit: 5 } }), { status: 200 })
          : Promise.reject(new Error("no GET")),
      ),
    )
    vi.stubGlobal("fetch", fetchMock)
    renderSettings()

    const input = await screen.findByLabelText("Candidate Q&A limit")
    fireEvent.change(input, { target: { value: "51" } })
    fireEvent.click(screen.getByRole("button", { name: /save limit/i }))

    await waitFor(() => {
      expect(screen.getByText("Enter a whole number from 0–50.")).toBeDefined()
    })
    expect(fetchMock.mock.calls.some(([, init]) => init?.method === "PUT")).toBe(false)
  })

  it("hides the control from non-admin roles", async () => {
    localStorage.setItem("intivai_token", RECRUITER_JWT)
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("no GET endpoint")))
    renderSettings()

    expect(screen.queryByText("Candidate Q&A limit")).toBeNull()
    expect(screen.getByText(/organization admins/i)).toBeDefined()
  })
})
