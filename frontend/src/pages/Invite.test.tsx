import { act, fireEvent, render, screen, waitFor } from "@testing-library/react"
import { MemoryRouter, Route, Routes } from "react-router-dom"
import { StrictMode } from "react"
import { describe, expect, it, vi, afterEach } from "vitest"
import { InvitePage } from "./Invite"

function renderInvite(url: string) {
  return render(
    <StrictMode>
      <MemoryRouter initialEntries={[url]}>
        <Routes>
          <Route path="/invite/:id" element={<InvitePage />} />
          <Route path="/chat/:id" element={<div>chat page</div>} />
        </Routes>
      </MemoryRouter>
    </StrictMode>,
  )
}

// A Response body can be consumed once — always hand out a fresh one.
const okJson = () => new Response(JSON.stringify({ data: {} }), { status: 200 })

describe("InvitePage", () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    sessionStorage.clear()
  })

  it("DD: magic-link autostart does NOT tick consent nor start on its own", () => {
    const fetchMock = vi.fn().mockImplementation(okJson)
    vi.stubGlobal("fetch", fetchMock)
    renderInvite("/invite/iv-1?t=inv-tok&auto=1")

    const checkbox = screen.getByRole("checkbox", { name: "I consent" })
    // Radix checkbox: state lives on aria-checked, not an input's checked.
    expect(checkbox.getAttribute("aria-checked")).toBe("false")
    // No consent/ticket POST until the candidate explicitly consents.
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it("autostarts exactly once (StrictMode-safe) after explicit consent", async () => {
    const fetchMock = vi.fn().mockImplementation(okJson)
    vi.stubGlobal("fetch", fetchMock)
    renderInvite("/invite/iv-1?t=inv-tok&auto=1")

    await act(async () => {
      fireEvent.click(screen.getByRole("checkbox", { name: "I consent" }))
    })

    await waitFor(() => {
      // consent + ticket mint = exactly two POSTs, even under StrictMode
      expect(fetchMock).toHaveBeenCalledTimes(2)
    })
    await waitFor(() => {
      expect(screen.getByText("chat page")).toBeDefined()
    })
  })

  it("stores the invitation token for mid-session ws_ticket re-mint (G2)", async () => {
    const fetchMock = vi.fn().mockImplementation((path: unknown, init?: RequestInit) => {
      void init
      if (String(path).endsWith("/ticket")) {
        return Promise.resolve(new Response(JSON.stringify({ data: { ticket: "ws-tkt" } }), { status: 200 }))
      }
      return Promise.resolve(okJson())
    })
    vi.stubGlobal("fetch", fetchMock)
    renderInvite("/invite/iv-1?t=inv-tok-42&auto=1")

    await act(async () => {
      fireEvent.click(screen.getByRole("checkbox", { name: "I consent" }))
    })

    await waitFor(() => {
      expect(screen.getByText("chat page")).toBeDefined()
    })
    expect(sessionStorage.getItem("intivai_invitation_iv-1")).toContain("inv-tok-42")
  })
})
