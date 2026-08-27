import { act, fireEvent, render, screen, waitFor } from "@testing-library/react"
import { MemoryRouter, Route, Routes } from "react-router-dom"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { StrictMode } from "react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { ChatPage } from "./Chat"

vi.mock("sonner", () => ({
  toast: { error: vi.fn(), info: vi.fn(), success: vi.fn(), warning: vi.fn() },
}))

// ChatPage pulls useChatSession which pings sonner on missing creds etc. We
// provide real creds so the useEffect chain boots the ChatClient; the socket
// is stubbed below.
class FakeWebSocket {
  static OPEN = 1
  static instances: FakeWebSocket[] = []
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
    void Promise.resolve().then(() => {
      this.readyState = 1
      this.onopen?.()
    })
  }

  send(data: string) {
    this.sent.push(data)
  }

  close() {
    this.readyState = 3
  }

  emit(frame: unknown) {
    this.onmessage?.({ data: JSON.stringify(frame) })
  }

  start(sessionId = "s-1") {
    this.emit({ type: "interview.start", session_id: sessionId, total_questions: 3 })
  }
}

function lastInstance(): FakeWebSocket {
  return FakeWebSocket.instances[FakeWebSocket.instances.length - 1]
}

// ChatPage uses window.matchMedia for reduced motion — jsdom has no impl.
const matchMediaStub = () => ({
  matches: false,
  addEventListener: () => undefined,
  removeEventListener: () => undefined,
  addListener: () => undefined,
  removeListener: () => undefined,
  dispatchEvent: () => false,
})

function ChatRoute() {
  return <ChatPage />
}

function renderChat(url = "/chat/iv-1?t=ws-tkt") {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[url]}>
        <Routes>
          <Route path="/chat/:id" element={<ChatRoute />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

function renderChatStrict(url = "/chat/iv-1?t=ws-tkt") {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <StrictMode>
      <QueryClientProvider client={qc}>
        <MemoryRouter initialEntries={[url]}>
          <Routes>
            <Route path="/chat/:id" element={<ChatRoute />} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>
    </StrictMode>,
  )
}

// ChatPage needs sessionStorage/invite token absent → the WS mock still works
// (no re-mint path).
describe("ChatPage — G11 input clamps", () => {
  beforeEach(() => {
    FakeWebSocket.instances = []
    sessionStorage.clear()
    localStorage.clear()
    vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket)
    vi.stubGlobal("matchMedia", matchMediaStub)
    // scrollTo/scrollIntoView not implemented in jsdom.
    Element.prototype.scrollTo = () => undefined
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.clearAllMocks()
  })

  it("caps the answer textarea at 4000 characters", async () => {
    renderChat()
    await waitFor(() => {
      expect(screen.getByLabelText("Your answer")).toBeDefined()
    })
    const answer = screen.getByLabelText("Your answer") as HTMLTextAreaElement
    expect(answer.getAttribute("maxlength")).toBe("4000")
    // Typing beyond the cap is truncated by the browser attribute.
    fireEvent.change(answer, { target: { value: "x".repeat(4200) } })
    expect(answer.value.length).toBeLessThanOrEqual(4200)
  })

  it("caps the candidate question input at 1000 characters", async () => {
    renderChat()
    await waitFor(() => {
      expect(screen.getByText("Ask a question about the role")).toBeDefined()
    })
    fireEvent.click(screen.getByText("Ask a question about the role"))
    const qa = screen.getByLabelText("Send question") // exists after open
    expect(qa).toBeDefined()
    const input = screen.getByPlaceholderText(/Ask about the role/) as HTMLTextAreaElement
    expect(input.getAttribute("maxlength")).toBe("1000")
  })

  it("J10: sends the candidate question and renders the grounded answer in the transcript", async () => {
    renderChat()
    await waitFor(() => {
      expect(screen.getByText("Ask a question about the role")).toBeDefined()
    })
    fireEvent.click(screen.getByText("Ask a question about the role"))
    const input = screen.getByPlaceholderText(/Ask about the role/) as HTMLTextAreaElement
    fireEvent.change(input, { target: { value: "What tech stack?" } })
    fireEvent.click(screen.getByLabelText("Send question"))

    // The ws frame went out with the typed question.
    await waitFor(() => {
      const frames = lastInstance().sent.map((s) => JSON.parse(s))
      expect(frames.some((f) => f.type === "candidate_question" && f.content === "What tech stack?")).toBe(true)
    })
    // Question bubble appears immediately (the qa input collapses).
    expect(screen.getByText("What tech stack?")).toBeDefined()

    // Server answers with a qa_answer frame → the transcript shows the answer.
    await act(async () => {
      lastInstance().emit({
        type: "qa_answer",
        question: "What tech stack?",
        answer: "Go and TypeScript.",
        refused: false,
      })
    })
    expect(screen.getByText("Grounded Answer")).toBeDefined()
    expect(screen.getByText("Go and TypeScript.")).toBeDefined()
  })

  it("J10: renders the refusal copy when qa_answer arrives refused", async () => {
    renderChat()
    await waitFor(() => {
      expect(screen.getByText("Ask a question about the role")).toBeDefined()
    })
    fireEvent.click(screen.getByText("Ask a question about the role"))
    const input = screen.getByPlaceholderText(/Ask about the role/) as HTMLTextAreaElement
    fireEvent.change(input, { target: { value: "Another one?" } })
    fireEvent.click(screen.getByLabelText("Send question"))

    await act(async () => {
      lastInstance().emit({
        type: "qa_answer",
        question: "Another one?",
        answer: "You've reached the limit of questions.",
        refused: true,
      })
    })
    expect(screen.getByText("Question limit reached")).toBeDefined()
  })

  it("G11 StrictMode: opens exactly ONE websocket connection, no duplicate resume", async () => {
    renderChatStrict()
    // Effects run twice in StrictMode — the cleanup must close the first
    // client, and the second effect must reuse the SAME ChatClient (ref), so
    // exactly one WebSocket instance ever exists.
    await waitFor(() => {
      expect(FakeWebSocket.instances.length).toBe(1)
    })
    expect(FakeWebSocket.instances.length).toBe(1)
    // After the handshake, no second socket was created and the resume replay
    // frame was sent once (session was never started here, so no resume).
    expect(lastInstance().sent.filter((s) => JSON.parse(s).type === "resume")).toHaveLength(0)
    // Single connection → single open → no reconnect loop.
    await act(async () => {
      lastInstance().start()
    })
    expect(FakeWebSocket.instances.length).toBe(1)
  })
})
