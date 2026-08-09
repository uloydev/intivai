import { describe, expect, it, vi } from "vitest"
import { ApiError, api } from "./api"

describe("api", () => {
  it("sends bearer token from storage", async () => {
    localStorage.setItem("intivai_token", "tok-123")
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ data: { ok: true } }), { status: 200 }),
    )
    vi.stubGlobal("fetch", fetchMock)
    await api.get("/jobs")
    const [, init] = fetchMock.mock.calls[0]
    expect(new Headers(init.headers).get("Authorization")).toBe("Bearer tok-123")
    vi.unstubAllGlobals()
  })

  it("normalizes error into ApiError with code", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ code: "AUTH_FAILED", error: "invalid credentials" }), { status: 401 }),
    )
    vi.stubGlobal("fetch", fetchMock)
    try {
      await api.post("/auth/login", {})
      throw new Error("should have thrown")
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError)
      const apiErr = err as ApiError
      expect(apiErr.code).toBe("AUTH_FAILED")
      expect(apiErr.message).toBe("invalid credentials")
      expect(apiErr.status).toBe(401)
    }
    vi.unstubAllGlobals()
  })

  it("wraps JSON body with content-type", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ data: null }), { status: 200 }))
    vi.stubGlobal("fetch", fetchMock)
    await api.post("/interviews", { question_count: 3 })
    const [, init] = fetchMock.mock.calls[0]
    expect(new Headers(init.headers).get("Content-Type")).toBe("application/json")
    expect(JSON.parse(init.body as string)).toEqual({ question_count: 3 })
    vi.unstubAllGlobals()
  })

  it("does not force JSON content-type for FormData", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ data: { id: "1" } }), { status: 201 }))
    vi.stubGlobal("fetch", fetchMock)
    const form = new FormData()
    form.append("file", new Blob(["pdf"], { type: "application/pdf" }), "cv.pdf")
    await api.postForm("/cvs", form)
    const [, init] = fetchMock.mock.calls[0]
    expect(new Headers(init.headers).has("Content-Type")).toBe(false)
    vi.unstubAllGlobals()
  })
})
