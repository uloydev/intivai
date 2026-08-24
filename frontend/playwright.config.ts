import { defineConfig } from "@playwright/test"

export default defineConfig({
  testDir: "./e2e",
  // LLM turns can take 60s+ with reasoning models (thinking before streaming);
  // the happy-path journey chains multiple LLM round-trips in one test.
  timeout: 420_000,
  expect: { timeout: 120_000 },
  workers: 1,
  use: {
    baseURL: "http://localhost:5173",
  },
  projects: [{ name: "chromium", use: { browserName: "chromium" } }],
})
