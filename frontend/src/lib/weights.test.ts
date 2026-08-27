import { describe, expect, it } from "vitest"
import {
  DEFAULT_WEIGHT_MAP,
  WEIGHT_DIMENSIONS,
  buildSaveWeights,
  isWeightMapComplete,
  isWeightSumValid,
  weightSum,
} from "./weights"

describe("scoring weights helper", () => {
  it("default map is complete and sums to 1.00", () => {
    expect(isWeightMapComplete(DEFAULT_WEIGHT_MAP)).toBe(true)
    expect(weightSum(DEFAULT_WEIGHT_MAP)).toBeCloseTo(1, 5)
  })

  it("sums the five dimensions", () => {
    const w = { skills_match: 0.4, experience_years: 0.2, semantic_match: 0.1, education: 0.1, certifications: 0.1 }
    expect(weightSum(w)).toBeCloseTo(0.9, 5)
  })

  it("treats missing keys as 0 and reports incomplete", () => {
    const w = { skills_match: 0.2, experience_years: 0.2, semantic_match: 0.2, education: 0.2 }
    expect(weightSum(w)).toBeCloseTo(0.8, 5)
    expect(isWeightMapComplete(w)).toBe(false)
  })

  it("accepts sum within ±0.01 tolerance", () => {
    expect(isWeightSumValid(0.99)).toBe(true)
    expect(isWeightSumValid(1.01)).toBe(true)
    expect(isWeightSumValid(1.02)).toBe(false)
    expect(isWeightSumValid(0.98)).toBe(false)
  })

  it("covers exactly the five required dimensions", () => {
    expect(WEIGHT_DIMENSIONS).toHaveLength(5)
    expect(WEIGHT_DIMENSIONS.map((d) => d.key)).toEqual([
      "skills_match",
      "experience_years",
      "semantic_match",
      "education",
      "certifications",
    ])
  })
})

describe("canonical default weights (backend parity)", () => {
  // Mirrors DefaultWeights() in backend/internal/screening/domain/scoring.go
  it("mirrors the backend canonical profile exactly", () => {
    expect(DEFAULT_WEIGHT_MAP).toEqual({
      skills_match: 0.35,
      experience_years: 0.2,
      semantic_match: 0.25,
      education: 0.1,
      certifications: 0.1,
    })
  })

  it("sums to 1 within ±0.0001", () => {
    expect(Math.abs(weightSum(DEFAULT_WEIGHT_MAP) - 1)).toBeLessThanOrEqual(0.0001)
  })
})

describe("buildSaveWeights", () => {
  const custom = {
    skills_match: 0.5,
    experience_years: 0.1,
    semantic_match: 0.3,
    education: 0.05,
    certifications: 0.05,
  }

  it("returns the explicit default map in defaults mode even when current differs", () => {
    expect(buildSaveWeights(true, custom)).toEqual(DEFAULT_WEIGHT_MAP)
  })

  it("returns current weights when tuning", () => {
    expect(buildSaveWeights(false, custom)).toEqual(custom)
  })

  it("fills missing dimensions with 0 when tuning", () => {
    expect(buildSaveWeights(false, { skills_match: 0.6 })).toEqual({
      skills_match: 0.6,
      experience_years: 0,
      semantic_match: 0,
      education: 0,
      certifications: 0,
    })
  })
})
