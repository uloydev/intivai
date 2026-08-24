import { act, renderHook } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import type { ChatFrame } from "./ws"
import { storeInvitationToken } from "./interview-ticket"
import { useChatSession } from "./useChatSession"

vi.mock("sonner", () => ({
  toast: {
    error: vi.fn(),
    info: vi.fn(),
    success: vi.fn(),
    warning: vi.fn(),
  },
}))

import { toast } from "sonner"

type Behavior = "open" | "reject"

class FakeWebSocket {
  static instances: FakeWebSocket[] = []
  static behaviors: Behavior[] = []
  readyState = 0
  sent: string[] = []
  url: string
  onopen: (() => void) | null = null
  onmessage: ((ev: { data: string }) => void) | null = null
  onerror: (() => void) | null = null
  onclose: ((ev: { wasClean: boolean }) => void) | null = null

  constructor(url: string) {
    this.url = url
    FakeWebSocket.instances.push(this)
    const behavior = FakeWebSocket.behaviors[FakeWebSocket.instances.length - 1] ?? "open"
    // Lifecycle events fire ASYNC (promise-job level, immune to fake timers)
    // exactly like a real socket: ChatClient assigns handlers after the
    // constructor returns, so synchronous delivery would be lost.
    void Promise.resolve().then(() => {
      if (behavior === "open") {
        this.readyState = 1
        this.onopen?.()
      } else {
        this.onerror?.()
        this.onclose?.({ wasClean: false })
      }
    })
  }

  send(data: string) {
    this.sent.push(data)
  }

  close() {
    this.readyState = 3
  }

  emit(frame: ChatFrame) {
    this.onmessage?.({ data: JSON.stringify(frame) })
  }

  start(sessionId = "s-1") {
    this.emit({ type: "interview.start", session_id: sessionId, total_questions: 3 })
  }

  drop() {
    this.onclose?.({ wasClean: false })
  }
}

/** Drains nested promise jobs (mint .then chains, deferred socket events). */
async function flushAsync(): Promise<void> {
  for (let i = 0; i < 8; i += 1) {
    await Promise.resolve()
  }
}

const TICKET_MINT_PATH = "/candidate/interviews/iv-1/ticket"

function mintResponse(ticket: string): Response {
  return new Response(JSON.stringify({ data: { ticket } }), { status: 200 })
}

function lastInstance(): FakeWebSocket {
  return FakeWebSocket.instances[FakeWebSocket.instances.length - 1]
}

describe("useChatSession", () => {
  beforeEach(() => {
    FakeWebSocket.instances = []
    FakeWebSocket.behaviors = []
    sessionStorage.clear()
    localStorage.clear()
    vi.useFakeTimers()
    vi.stubGlobal(
      "WebSocket",
      FakeWebSocket as unknown as typeof WebSocket,
    )
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
    vi.clearAllMocks()
  })

  describe("G2 — single re-mint on expired-ticket handshake rejection", () => {
    it("mints a fresh ticket once and retries with it after an unopened connection", async () => {
      storeInvitationToken("iv-1", "inv-tok")
      FakeWebSocket.behaviors = ["reject", "open"]
      const fetchMock = vi.fn().mockResolvedValue(mintResponse("fresh-ticket"))
      vi.stubGlobal("fetch", fetchMock)

      const { result } = renderHook(() =>
        useChatSession({ id: "iv-1", ticket: "expired-ticket" }),
      )

      expect(FakeWebSocket.instances).toHaveLength(1)

      await act(async () => {
        vi.advanceTimersByTime(1100)
        await flushAsync()
      })

      expect(fetchMock).toHaveBeenCalledTimes(1)
      const [path, init] = fetchMock.mock.calls[0]
      expect(String(path)).toContain(TICKET_MINT_PATH)
      expect(JSON.parse(init.body as string)).toEqual({ invitation_token: "inv-tok" })
      // Retry used the freshly minted credential, never the original one.
      expect(FakeWebSocket.instances).toHaveLength(2)
      expect(lastInstance().url).toContain("ticket=fresh-ticket")
      expect(result.current.reconnecting).toBe(false)
    })

    it("never mints twice — further failures ride the reconnect budget", async () => {
      storeInvitationToken("iv-1", "inv-tok")
      FakeWebSocket.behaviors = ["reject", "reject", "reject", "reject"]
      const fetchMock = vi.fn().mockResolvedValue(mintResponse("fresh-ticket"))
      vi.stubGlobal("fetch", fetchMock)

      renderHook(() => useChatSession({ id: "iv-1", ticket: "expired-ticket" }))

      // Re-mint retry + two more budget reconnects.
      await act(async () => {
        vi.advanceTimersByTime(1100)
        await flushAsync()
        vi.advanceTimersByTime(2100)
        await flushAsync()
        vi.advanceTimersByTime(4100)
        await flushAsync()
      })

      expect(fetchMock).toHaveBeenCalledTimes(1)
      expect(FakeWebSocket.instances.length).toBeGreaterThanOrEqual(3)
    })

    it("does not re-mint when no stored invitation token exists", async () => {
      FakeWebSocket.behaviors = ["reject", "reject"]
      const fetchMock = vi.fn()
      vi.stubGlobal("fetch", fetchMock)

      renderHook(() => useChatSession({ id: "iv-1", ticket: "expired-ticket" }))

      await act(async () => {
        await flushAsync()
        vi.advanceTimersByTime(1100)
        await flushAsync()
      })

      expect(fetchMock).not.toHaveBeenCalled()
      expect(FakeWebSocket.instances).toHaveLength(2)
    })

    it("re-arms the re-mint allowance after a successful connection", async () => {
      storeInvitationToken("iv-1", "inv-tok")
      // reject (re-mint #1) → open → clean drop → reject (re-mint #2) → open
      FakeWebSocket.behaviors = ["reject", "open", "reject", "open"]
      let mintCount = 0
      const fetchMock = vi.fn().mockImplementation(() => {
        mintCount += 1
        return Promise.resolve(mintResponse(`fresh-ticket-${mintCount}`))
      })
      vi.stubGlobal("fetch", fetchMock)

      renderHook(() => useChatSession({ id: "iv-1", ticket: "expired-ticket" }))

      // Episode 1: original ticket rejected pre-upgrade → re-mint → retry opens.
      await act(async () => {
        vi.advanceTimersByTime(1100)
        await flushAsync()
      })
      expect(fetchMock).toHaveBeenCalledTimes(1)
      expect(lastInstance().url).toContain("ticket=fresh-ticket-1")

      // Healthy session traffic re-arms the allowance…
      act(() => {
        lastInstance().start()
      })

      // …but an opened-then-dropped socket rides the normal budget (no mint).
      await act(async () => {
        lastInstance().drop()
        await flushAsync()
      })
      expect(fetchMock).toHaveBeenCalledTimes(1)

      // Budget reconnect lands; THAT attempt is rejected pre-upgrade → new
      // episode allows the single re-mint again.
      await act(async () => {
        vi.advanceTimersByTime(2100)
        await flushAsync()
      })
      expect(fetchMock).toHaveBeenCalledTimes(2)
      expect(lastInstance().url).toContain("ticket=fresh-ticket-2")
    })
  })

  describe("G3 — reconnect budget resets on successful start/resume", () => {
    it("survives more than MAX_RECONNECTS drop→connect-success cycles", async () => {
      FakeWebSocket.behaviors = ["open"]
      const { result } = renderHook(() =>
        useChatSession({ id: "iv-1", ticket: "tkt" }),
      )

      const cycles = 8 // > MAX_RECONNECTS (5)
      for (let i = 0; i < cycles; i += 1) {
        await act(async () => {
          lastInstance().drop()
          vi.advanceTimersByTime(10_500)
          await flushAsync()
        })
        // Every fresh connection completes the handshake.
        act(() => {
          lastInstance().start()
        })
        expect(result.current.disconnected).toBe(false)
      }
      expect(FakeWebSocket.instances.length).toBe(cycles + 1)
    })
  })

  describe("D19 — turn_in_progress error frame surfaces gentle notice", () => {
    it("shows retryable info instead of a generic error toast", async () => {
      FakeWebSocket.behaviors = ["open"]
      const { result } = renderHook(() =>
        useChatSession({ id: "iv-1", ticket: "tkt" }),
      )
      await act(async () => {
        lastInstance().start()
      })

      await act(async () => {
        lastInstance().emit({
          type: "error",
          code: "turn_in_progress",
          message: "a turn is already in progress",
        })
      })

      expect(toast.info).toHaveBeenCalledWith("Interviewer is still responding — hold on a moment")
      expect(toast.error).not.toHaveBeenCalledWith("a turn is already in progress")
      // Input must recover so the candidate can retry after the turn ends.
      expect(result.current.pendingAnswer).toBe(false)
    })

    it("keeps the generic error path for other codes", async () => {
      FakeWebSocket.behaviors = ["open"]
      renderHook(() => useChatSession({ id: "iv-1", ticket: "tkt" }))
      await act(async () => {
        lastInstance().start()
      })

      await act(async () => {
        lastInstance().emit({
          type: "error",
          code: "CONSENT_REQUIRED",
          message: "consent missing",
        })
      })

      expect(toast.error).toHaveBeenCalledWith("consent missing")
      expect(toast.info).not.toHaveBeenCalled()
    })
  })
})
