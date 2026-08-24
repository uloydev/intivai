import { expect, test, type Page } from "@playwright/test"

const SLUG = `e2e${Date.now().toString().slice(-8)}`
const EMAIL = `admin@${SLUG}.io`
const PASS = "secret1234"

// Progress logging: every phase prints a timestamped line to stdout so a
// long run (LLM extraction, WS stream) is observable step by step.
function log(msg: string) {
  console.log(`[e2e ${new Date().toISOString().slice(11, 19)}] ${msg}`)
}

// Mirrors scripts/smoke.sh through the UI: register → job → CV upload →
// extract (real LLM) → passed → interview → invite → consent → chat.
const pageErrors: string[] = []
test.afterEach(async () => {
  if (pageErrors.length > 0) {
    log("page console errors:\n" + pageErrors.join("\n"))
  }
})

test("full candidate journey", async ({ page }) => {
  page.on("console", (m) => {
    if (m.type() === "error") pageErrors.push(m.text())
  })
  page.on("pageerror", (e) => pageErrors.push(String(e)))

  await test.step("register org", async () => {
    await page.goto("/register")
    await page.locator("#name").fill("E2E Org")
    await page.locator("#slug").fill(SLUG)
    await page.locator("#email").fill(EMAIL)
    await page.locator("#password").fill(PASS)
    await page.locator("#confirm-password").fill(PASS)
    await page.getByRole("button", { name: /Initialize Workspace/i }).click()
    await expect(page).toHaveURL(/\/(dashboard|jobs)/)
    log("register OK → dashboard/jobs")
  })

  await test.step("create job", async () => {
    await page.goto("/jobs")
    await page.getByRole("button", { name: /Post New Job/i }).click()
    await page.locator("#job-title").fill("Go Engineer")
    await page.locator("#job-desc").fill("Go backend work")
    await page.locator("#job-skills").fill("Go, PostgreSQL")
    await page.getByRole("button", { name: /Next: Configure Stages/i }).click()
    await page.getByRole("button", { name: /Publish Job & Pipeline/i }).click({ force: true })
    await expect(page.getByText("Go Engineer")).toBeVisible()
    log("job created")
  })

  await test.step("upload CV", async () => {
    await page.goto("/cvs")
    await page.locator("#cv-name").fill("Jane E2E")
    await page.locator("#cv-email").fill("jane@e2e.io")
    await page.locator("#cv-file").setInputFiles("/tmp/kilo/cv.pdf")
    await page.getByRole("button", { name: /Ingest CV/i }).click()
    log("CV uploaded — waiting for parse/extract (LLM)")
  })

  await test.step("wait for extraction (real LLM)", async () => {
    // Poll with progress logs — LLM latency varies; the badge state is
    // observable as it moves parsing → extracting → Profile ready.
    log("waiting for extraction to complete...")
    await expect(page.getByText("Profile ready").first()).toBeVisible({ timeout: 60_000 })
    log("CV extracted")
  })

  await test.step("screen candidate for job", async () => {
    await page.getByRole("button", { name: /Screen for Role/i }).first().click()
    await page.getByRole("dialog").waitFor({ state: "visible" })
    await page.locator("#screen-job").click()
    await page.getByRole("option", { name: /Go Engineer/i }).click()
    await page.getByRole("button", { name: /Start Screening/i }).click()
    log("candidate screened for job")
  })

  await test.step("create interview from passed application", async () => {
    await page.goto("/interviews")
    await page.getByRole("tab", { name: /Eligible Candidates/i }).click()
    await expect(page.getByText("Jane E2E").first()).toBeVisible({ timeout: 30_000 })
    log("passed application visible in eligible interviews list")
    await page.getByRole("button", { name: /Create Interview/i }).first().click()
    await page.getByRole("button", { name: /Initialize Session/i }).click()
    await expect(page.locator("input[readonly]").first()).toBeVisible({ timeout: 30_000 })
    log("interview created, invite link shown")
  })

  let candidate: Page
  await test.step("candidate opens invite + consents", async () => {
    const inviteUrl = await page.locator("input[readonly]").first().inputValue()
    log("invite URL: " + inviteUrl.slice(0, 60) + "…")
    candidate = await page.context().newPage()
    await candidate.goto(inviteUrl)
    await expect(candidate.getByText(/Interview Invitation/i)).toBeVisible()
    await candidate.getByLabel(/I consent/i).check()
    await candidate.getByRole("button", { name: /Begin Interview Session/i }).click()
    await expect(candidate.getByText(/Question 1 of/i).first()).toBeVisible({ timeout: 30_000 })
    log("consent OK, WS connected, question 1 delivered")
  })

  await test.step("answer Q1, advance to Q2 (multi-turn topic dialogue)", async () => {
    await candidate.locator("#chat-input").fill(
      "I built payment services with Go, PostgreSQL and Kubernetes for five years in production.",
    )
    await candidate.locator("#chat-input").press("Enter")
    log("answer sent (reply) — waiting for LLM response")

    // Wait for the AI response (streaming finishes and response frame arrives)
    await expect(candidate.getByText(/Interrupted|Real-Time AI Session/i)).toBeVisible({ timeout: 120_000 })
    await candidate.waitForTimeout(2_000)

    // Topic is still open after reply — click the advance button to move to next question
    // The advance button is the ArrowRight icon button below the reply button.
    const advanceBtn = candidate.locator('button[aria-label="Complete topic and advance"]')
    if (await advanceBtn.isVisible().catch(() => false)) {
      // Fill a brief text so the advance button is enabled
      await candidate.locator("#chat-input").fill("That's the full context.")
      await advanceBtn.click()
      log("advance sent — waiting for Q2 (LLM streaming)")
    } else {
      // Fallback: send another answer with advance via typing and manual flow
      await candidate.locator("#chat-input").fill("Moving on to the next topic.")
      await candidate.locator("#chat-input").press("Enter")
      log("fallback: sent advance answer via Enter")
    }

    await expect(candidate.getByText(/Question 2 of/i)).toBeVisible({ timeout: 120_000 })
    log("Q2 delivered — multi-turn flow verified")
  })

  log("E2E PASSED")
})
