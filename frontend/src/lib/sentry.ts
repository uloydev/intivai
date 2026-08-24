import * as Sentry from "@sentry/react"

// G9 PII decision (config-level, simplest correct): candidate interview
// transcripts and portal data must never leave the browser unmasked.
//  - Session replays are DISABLED entirely (rate 0) — continuous capture of
//    chat/portal screens is not worth the residual risk.
//  - Error-triggered replays stay ON (1.0) for debugging, but with
//    maskAllText + maskAllInputs and sensitive containers blocked, so replay
//    frames carry layout, never words or keystrokes.
export const REPLAY_MASKING_OPTIONS: {
  maskAllText: boolean
  maskAllInputs: boolean
  block: string[]
} = {
  maskAllText: true,
  maskAllInputs: true,
  block: ['[aria-label="Interview transcript"]', "[data-sentry-block]"],
}

export function buildSentryConfig(dsn: string | undefined) {
  if (!dsn) return null
  return {
    dsn,
    integrations: [
      Sentry.browserTracingIntegration(),
      Sentry.replayIntegration(REPLAY_MASKING_OPTIONS),
    ],
    tracesSampleRate: 0.1,
    replaysSessionSampleRate: 0,
    replaysOnErrorSampleRate: 1.0,
  }
}
