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

  it("resolves undefined on 204 no content instead of throwing", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }))
    vi.stubGlobal("fetch", fetchMock)
    await expect(api.delete("/cvs/cv-1")).resolves.toBeUndefined()
    vi.unstubAllGlobals()
  })

  it("still unwraps the data envelope on 200", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ data: { id: "cv-1" } }), { status: 200 }),
    )
    vi.stubGlobal("fetch", fetchMock)
    await expect(api.get<{ id: string }>("/cvs/cv-1")).resolves.toEqual({ id: "cv-1" })
    vi.unstubAllGlobals()
  })

  it("getBlob sends recruiter bearer token and a timeout signal", async () => {
    localStorage.setItem("intivai_token", "tok-blob")
    const fetchMock = vi.fn().mockResolvedValue(new Response("pdf-bytes", { status: 200 }))
    vi.stubGlobal("fetch", fetchMock)
    await api.getBlob("/interviews/iv-1/report/pdf")
    const [path, init] = fetchMock.mock.calls[0]
    expect(path).toContain("/interviews/iv-1/report/pdf")
    expect(new Headers(init.headers).get("Authorization")).toBe("Bearer tok-blob")
    expect(init.signal).toBeInstanceOf(AbortSignal)
    vi.unstubAllGlobals()
  })

  it("getBlob throws ApiError with backend code on failure", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ code: "NOT_FOUND", error: "no report" }), { status: 404 }),
    )
    vi.stubGlobal("fetch", fetchMock)
    try {
      await api.getBlob("/interviews/iv-1/report/pdf")
      throw new Error("should have thrown")
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError)
      const apiErr = err as ApiError
      expect(apiErr.status).toBe(404)
      expect(apiErr.code).toBe("NOT_FOUND")
      expect(apiErr.message).toBe("no report")
    }
    vi.unstubAllGlobals()
  })

  it("getBlob uses the candidate token partition for candidate paths and clears it on 401", async () => {
    localStorage.setItem("intivai_token", "recruiter-tok")
    localStorage.setItem("intivai_candidate_token", "candidate-tok")
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ code: "UNAUTHORIZED", error: "expired" }), { status: 401 }),
    )
    vi.stubGlobal("fetch", fetchMock)
    try {
      await api.getBlob("/candidate/portal/export/file")
      throw new Error("should have thrown")
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError)
      const [, init] = fetchMock.mock.calls[0]
      expect(new Headers(init.headers).get("Authorization")).toBe("Bearer candidate-tok")
      expect(localStorage.getItem("intivai_candidate_token")).toBeNull()
      expect(localStorage.getItem("intivai_token")).toBe("recruiter-tok")
    }
    vi.unstubAllGlobals()
  })
})
