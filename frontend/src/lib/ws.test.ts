import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { ChatClient, type ChatFrame } from "./ws"

class FakeWebSocket {
  static OPEN = 1
  readyState = 0
  sent: string[] = []
  onmessage: ((ev: { data: string }) => void) | null = null
  onclose: ((ev: { wasClean: boolean }) => void) | null = null
  onerror: (() => void) | null = null
  static instances: FakeWebSocket[] = []
  url: string

  constructor(url: string) {
    this.url = url
    FakeWebSocket.instances.push(this)
    this.readyState = 1
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
}

describe("ChatClient", () => {
  beforeEach(() => {
    FakeWebSocket.instances = []
    vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket)
  })
  afterEach(() => vi.unstubAllGlobals())

  it("connects with the ticket as a query param", () => {
    const frames: ChatFrame[] = []
    new ChatClient({ ticket: "tkt", onFrame: (f) => frames.push(f), onClose: () => undefined }).connect("iv-1")
    expect(FakeWebSocket.instances[0].url).toContain("/candidate/interviews/iv-1/chat?ticket=tkt")
  })

  it("parses frames in order", () => {
    const frames: ChatFrame[] = []
    const client = new ChatClient({ ticket: "t", onFrame: (f) => frames.push(f), onClose: () => undefined })
    client.connect("iv-1")
    const ws = FakeWebSocket.instances[0]
    ws.emit({ type: "interview.start", session_id: "s1", total_questions: 3 })
    ws.emit({ type: "question", content: "Q1", idx: 1 })
    ws.emit({ type: "token", content: "Hel" })
    ws.emit({ type: "token", content: "lo" })
    ws.emit({ type: "response", content: "Hello" })
    expect(frames.map((f) => f.type)).toEqual(["interview.start", "question", "token", "token", "response"])
    expect((frames[0] as { total_questions: number }).total_questions).toBe(3)
  })

  it("sends answer/interrupt/resume frames", () => {
    const client = new ChatClient({ ticket: "t", onFrame: () => undefined, onClose: () => undefined })
    client.connect("iv-1")
    const ws = FakeWebSocket.instances[0]
    client.answer("my answer")
    client.answer("advance answer", "advance")
    client.interrupt()
    client.resume("s1")
    expect(ws.sent).toEqual([
      JSON.stringify({ type: "answer", content: "my answer", action: "reply" }),
      JSON.stringify({ type: "answer", content: "advance answer", action: "advance" }),
      JSON.stringify({ type: "interrupt" }),
      JSON.stringify({ type: "resume", session_id: "s1" }),
    ])
  })

  it("J10: sends candidate_question frames via sendCandidateQuestion", () => {
    const client = new ChatClient({ ticket: "t", onFrame: () => undefined, onClose: () => undefined })
    client.connect("iv-1")
    const ws = FakeWebSocket.instances[0]
    client.sendCandidateQuestion("What does the team use for CI?")
    client.sendCandidateQuestion("How many engineers are in the org?")
    expect(ws.sent).toEqual([
      JSON.stringify({ type: "candidate_question", content: "What does the team use for CI?" }),
      JSON.stringify({ type: "candidate_question", content: "How many engineers are in the org?" }),
    ])
  })

  it("J10: returns false when the socket is not open", () => {
    const client = new ChatClient({ ticket: "t", onFrame: () => undefined, onClose: () => undefined })
    client.connect("iv-1")
    const ws = FakeWebSocket.instances[0]
    ws.readyState = 3
    expect(client.sendCandidateQuestion("Q?")).toBe(false)
  })

  it("J10: parses qa_answer frames (answered and refused)", () => {
    const frames: ChatFrame[] = []
    const client = new ChatClient({ ticket: "t", onFrame: (f) => frames.push(f), onClose: () => undefined })
    client.connect("iv-1")
    const ws = FakeWebSocket.instances[0]
    ws.emit({ type: "qa_answer", question: "What is the salary range?", answer: "The range is in the posting.", refused: false })
    ws.emit({ type: "qa_answer", question: "How many people?", answer: "No data about that.", refused: true })
    expect(frames).toEqual([
      { type: "qa_answer", question: "What is the salary range?", answer: "The range is in the posting.", refused: false },
      { type: "qa_answer", question: "How many people?", answer: "No data about that.", refused: true },
    ])
  })

  it("sends candidate_question only as typed content", () => {
    const client = new ChatClient({ ticket: "t", onFrame: () => undefined, onClose: () => undefined })
    client.connect("iv-1")
    const ws = FakeWebSocket.instances[0]
    expect(client.sendCandidateQuestion("")).toBe(true)
    expect(JSON.parse(ws.sent[0])).toEqual({ type: "candidate_question", content: "" })
  })

  it("reports close reason", () => {
    const reasons: string[] = []
    const client = new ChatClient({ ticket: "t", onFrame: () => undefined, onClose: (r) => reasons.push(r) })
    client.connect("iv-1")
    const ws = FakeWebSocket.instances[0]
    ws.onclose?.({ wasClean: false })
    expect(reasons).toEqual(["error"])
  })
it("notifies close exactly once when error+close both fire", () => {
  const reasons: string[] = []
  const client = new ChatClient({ ticket: "t", onFrame: () => undefined, onClose: (r) => reasons.push(r) })
  client.connect("iv-1")
  const ws = FakeWebSocket.instances[0]
  ws.onerror?.()
  ws.onclose?.({ wasClean: false })
  expect(reasons).toEqual(["error"])
})

it("isOpen reflects socket state and answer reports delivery", () => {
  const client = new ChatClient({ ticket: "t", onFrame: () => undefined, onClose: () => undefined })
  client.connect("iv-1")
  const ws = FakeWebSocket.instances[0]
  expect(client.isOpen()).toBe(true)
  expect(client.answer("x")).toBe(true)
  ws.readyState = 3
  expect(client.isOpen()).toBe(false)
  expect(client.answer("y")).toBe(false)
})

})
