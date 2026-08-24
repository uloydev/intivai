import { describe, expect, it } from "vitest"
import { REPLAY_MASKING_OPTIONS, buildSentryConfig } from "./sentry"

describe("buildSentryConfig", () => {
  it("returns null when no DSN is configured", () => {
    expect(buildSentryConfig(undefined)).toBeNull()
    expect(buildSentryConfig("")).toBeNull()
  })

  it("disables session replays entirely and keeps masked error replay (G9 decision)", () => {
    const config = buildSentryConfig("https://k@sentry.example/1")
    expect(config).not.toBeNull()
    expect(config?.replaysSessionSampleRate).toBe(0)
    expect(config?.replaysOnErrorSampleRate).toBe(1.0)
    expect(config?.tracesSampleRate).toBe(0.1)
  })

  it("attaches browser tracing + replay integrations", () => {
    const config = buildSentryConfig("https://k@sentry.example/1")
    expect(config?.integrations).toHaveLength(2)
  })

  it("forces PII masking options on the replay integration", () => {
    // The replay integration is constructed with these exact options —
    // asserted here because the integration object itself is opaque.
    expect(REPLAY_MASKING_OPTIONS.maskAllText).toBe(true)
    expect(REPLAY_MASKING_OPTIONS.maskAllInputs).toBe(true)
    expect(REPLAY_MASKING_OPTIONS.block).toContain('[aria-label="Interview transcript"]')
    for (const selector of REPLAY_MASKING_OPTIONS.block ?? []) {
      expect(selector.startsWith("[") || selector.startsWith(".")).toBe(true)
    }
  })
})
