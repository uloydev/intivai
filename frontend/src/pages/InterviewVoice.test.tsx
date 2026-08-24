import { render, screen } from "@testing-library/react"
import { MemoryRouter } from "react-router-dom"
import { afterEach, describe, expect, it, vi } from "vitest"
import { InterviewVoicePage } from "./InterviewVoice"

describe("InterviewVoicePage — G4 credential gating", () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    localStorage.clear()
  })

  it("renders a disabled state with guidance instead of connecting", () => {
    render(
      <MemoryRouter initialEntries={["/voice/iv-1"]}>
        <InterviewVoicePage />
      </MemoryRouter>,
    )

    expect(screen.getByText("Voice Session Unavailable")).toBeDefined()
    expect(screen.getByText("Recruiter access required")).toBeDefined()
    expect(screen.queryByText("Start Voice Interview")).toBeNull()
  })

  it("never places the recruiter auth JWT into any URL or connection attempt", () => {
    localStorage.setItem("intivai_token", "recruiter-jwt-secret")
    const WSProbe = vi.fn()
    vi.stubGlobal("WebSocket", WSProbe)

    render(
      <MemoryRouter initialEntries={["/voice/iv-1"]}>
        <InterviewVoicePage />
      </MemoryRouter>,
    )

    // No socket is ever constructed and the JWT never appears in the DOM.
    expect(WSProbe).not.toHaveBeenCalled()
    expect(document.body.innerHTML).not.toContain("recruiter-jwt-secret")
  })
})
