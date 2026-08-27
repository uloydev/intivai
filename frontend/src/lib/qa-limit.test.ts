import { describe, expect, it } from "vitest"
import { MAX_CANDIDATE_QA_LIMIT, MIN_CANDIDATE_QA_LIMIT, clampQaLimit, parseQaLimitInput } from "./qa-limit"

describe("clampQaLimit", () => {
  it("keeps values inside the 0–50 band unchanged", () => {
    expect(clampQaLimit(0)).toBe(0)
    expect(clampQaLimit(10)).toBe(10)
    expect(clampQaLimit(50)).toBe(50)
    expect(clampQaLimit(25)).toBe(25)
  })

  it("clamps negative values to the minimum", () => {
    expect(clampQaLimit(-1)).toBe(MIN_CANDIDATE_QA_LIMIT)
    expect(clampQaLimit(-100)).toBe(MIN_CANDIDATE_QA_LIMIT)
  })

  it("clamps values above 50 to the maximum", () => {
    expect(clampQaLimit(51)).toBe(MAX_CANDIDATE_QA_LIMIT)
    expect(clampQaLimit(1000)).toBe(MAX_CANDIDATE_QA_LIMIT)
  })

  it("truncates fractional input to a whole number", () => {
    expect(clampQaLimit(12.7)).toBe(12)
    expect(clampQaLimit(0.9)).toBe(0)
    expect(clampQaLimit(49.999)).toBe(49)
  })

  it("normalizes NaN to the minimum", () => {
    expect(clampQaLimit(Number.NaN)).toBe(MIN_CANDIDATE_QA_LIMIT)
  })

  it("normalizes infinities to the bounds", () => {
    expect(clampQaLimit(Infinity)).toBe(MAX_CANDIDATE_QA_LIMIT)
    expect(clampQaLimit(-Infinity)).toBe(MIN_CANDIDATE_QA_LIMIT)
  })
})

describe("parseQaLimitInput", () => {
  it("accepts a bare integer inside 0–50", () => {
    expect(parseQaLimitInput("0")).toBe(0)
    expect(parseQaLimitInput("25")).toBe(25)
    expect(parseQaLimitInput("50")).toBe(50)
  })

  it("rejects out-of-band values", () => {
    expect(parseQaLimitInput("51")).toBeNull()
    expect(parseQaLimitInput("-1")).toBeNull()
    expect(parseQaLimitInput("1000")).toBeNull()
  })

  it("rejects non-integer forms", () => {
    expect(parseQaLimitInput("12.7")).toBeNull()
    expect(parseQaLimitInput("1e2")).toBeNull()
    expect(parseQaLimitInput("+5")).toBeNull()
    expect(parseQaLimitInput("abc")).toBeNull()
    expect(parseQaLimitInput("")).toBeNull()
  })
})
