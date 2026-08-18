import { render, screen, act } from "@testing-library/react"
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { TimerGate } from "./TimerGate"

describe("TimerGate Component", () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it("renders initial stage archetype and timer badges", () => {
    render(
      <TimerGate
        sessionRemainingSec={1800}
        timeLimitSec={120}
        currentIdx={1}
        total={5}
        archetype="conversational"
        active={true}
        onExpire={() => undefined}
      />
    )

    expect(screen.getByText("Conversational Stage")).toBeDefined()
    expect(screen.getByText("Stage 1 of 5")).toBeDefined()
    expect(screen.getByText("Stage: 02:00")).toBeDefined()
    expect(screen.getByText("Session: 30:00")).toBeDefined()
  })

  it("renders coding archetype badge with correct label", () => {
    render(
      <TimerGate
        sessionRemainingSec={1500}
        timeLimitSec={600}
        currentIdx={2}
        total={5}
        archetype="coding"
        active={true}
        onExpire={() => undefined}
      />
    )

    expect(screen.getByText("Live Coding Challenge")).toBeDefined()
    expect(screen.getByText("Stage: 10:00")).toBeDefined()
  })

  it("counts down question clock and triggers auto-advance on expiry", () => {
    const handleExpire = vi.fn()
    render(
      <TimerGate
        sessionRemainingSec={1800}
        timeLimitSec={3}
        currentIdx={1}
        total={3}
        archetype="conversational"
        active={true}
        onExpire={handleExpire}
      />
    )

    // Advance 2 seconds
    act(() => {
      vi.advanceTimersByTime(2000)
    })
    expect(screen.getByText("Stage: 00:01")).toBeDefined()

    // Advance 2 more seconds (clock reaches 0, grace period starts)
    act(() => {
      vi.advanceTimersByTime(2000)
    })
    expect(screen.getByText("Stage: 00:00")).toBeDefined()
    expect(screen.getByText("Stage time expired. Auto-submitting response in:")).toBeDefined()

    // Advance past the 15-second grace period
    act(() => {
      vi.advanceTimersByTime(16000)
    })
    expect(handleExpire).toHaveBeenCalledTimes(1)
  })
})
