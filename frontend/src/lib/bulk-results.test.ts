import { describe, expect, it } from "vitest"
import { ApiError } from "./api"
import { summarizeBulkResults } from "./bulk-results"

function fulfilled<T>(value: T): PromiseSettledResult<T> {
  return { status: "fulfilled", value }
}

function rejected(reason: unknown): PromiseSettledResult<unknown> {
  return { status: "rejected", reason }
}

describe("summarizeBulkResults", () => {
  it("reports every result as succeeded when all fulfill", () => {
    const summary = summarizeBulkResults([fulfilled(1), fulfilled(2), fulfilled(3)])
    expect(summary.succeeded).toBe(3)
    expect(summary.failed).toBe(0)
    expect(summary.reasons).toEqual([])
  })

  it("counts ApiError failures by status code", () => {
    const summary = summarizeBulkResults([
      fulfilled(1),
      rejected(new ApiError(400, "INVALID_INPUT", "bad stage")),
      rejected(new ApiError(400, "INVALID_INPUT", "bad stage again")),
      rejected(new ApiError(500, "INTERNAL", "boom")),
    ])
    expect(summary.succeeded).toBe(1)
    expect(summary.failed).toBe(3)
    expect(summary.reasons).toEqual(["2× invalid request (400)", "1× server error (500)"])
  })

  it("falls back to error class name for non-ApiError rejections", () => {
    const summary = summarizeBulkResults([
      rejected(new Error("network down")),
      rejected("string failure"),
    ])
    expect(summary.succeeded).toBe(0)
    expect(summary.failed).toBe(2)
    expect(summary.reasons).toEqual([
      "1× network down (0)",
      "1× unexpected error",
    ])
  })

  it("handles an empty result list", () => {
    const summary = summarizeBulkResults([])
    expect(summary.succeeded).toBe(0)
    expect(summary.failed).toBe(0)
    expect(summary.reasons).toEqual([])
  })
})
