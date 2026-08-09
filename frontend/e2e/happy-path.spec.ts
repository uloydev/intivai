import { expect, test, type Page } from "@playwright/test"

const SLUG = `e2e${Date.now().toString().slice(-8)}`
const EMAIL = `admin@${SLUG}.io`
const PASS = "secret1234"

// Progress logging: every phase prints a timestamped line to stdout so a
// long run (DeepSeek extraction, WS stream) is observable step by step.
function log(msg: string) {
  console.log(`[e2e ${new Date().toISOString().slice(11, 19)}] ${msg}`)
}

// Mirrors scripts/smoke.sh through the UI: register → job → CV upload →
// extract (real DeepSeek) → passed → interview → invite → consent → chat.
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
    log("register org " + SLUG)
    await page.goto("/register")
    await page.getByLabel("Organization name").fill("E2E Org")
    await page.getByLabel("Slug").fill(SLUG)
    await page.getByLabel("Admin email").fill(EMAIL)
    await page.getByLabel("Password").fill(PASS)
    await page.getByRole("button", { name: "Create workspace" }).click()
    await expect(page).toHaveURL(/\/jobs/)
    log("register OK → /jobs")
  })

  await test.step("create job", async () => {
    await page.getByRole("button", { name: "New job" }).click()
    await page.getByLabel("Title").fill("Go Engineer")
    await page.getByLabel("Description").fill("Go backend work")
    await page.getByLabel(/Required skills/).fill("Go, PostgreSQL")
    await page.getByRole("button", { name: "Create", exact: true }).click()
    await expect(page.getByText("Go Engineer")).toBeVisible()
    log("job created")
  })

  await test.step("upload CV", async () => {
    await page.getByRole("link", { name: "CVs" }).click()
    await page.getByLabel("Candidate name").fill("Jane E2E")
    await page.getByLabel("Email").fill("jane@e2e.io")
    await page.locator('input[type="file"]').setInputFiles("/tmp/kilo/cv.pdf")
    await page.getByRole("button", { name: "Upload" }).click()
    log("CV uploaded — waiting for parse/extract (DeepSeek)")
  })

  await test.step("wait for extraction (real DeepSeek)", async () => {
    // Poll with progress logs — DeepSeek latency varies; the badge state is
    // observable as it moves parsing → extracting → extracted.
    const deadline = Date.now() + 300_000
    let lastStatus = ""
    while (Date.now() < deadline) {
      // Name <p> → parent (name block) → parent (row); badge = first span.
      const name = page.getByText("Jane E2E", { exact: true })
      const row = name.locator("..").locator("..")
      const badge = row.locator("span").first()
      const status = (await badge.textContent().catch(() => "")) ?? ""
      if (status !== lastStatus) {
        log("CV status: " + (status || "(no row yet)"))
        lastStatus = status
      }
      if (status === "extracted") break
      await page.waitForTimeout(3_000)
    }
    await expect(page.getByText("extracted", { exact: true })).toBeVisible({ timeout: 5_000 })
    log("CV extracted")
  })

  await test.step("create interview from passed application", async () => {
    await page.getByRole("link", { name: "Interviews" }).click()
    await expect(page.getByText("Go Engineer")).toBeVisible({ timeout: 30_000 })
    log("passed application visible in interviews list")
    await page.getByRole("button", { name: "Interview", exact: true }).first().click()
    await page.getByRole("button", { name: "Create", exact: true }).click()
    await expect(page.getByLabel("Invite link")).toBeVisible({ timeout: 30_000 })
    log("interview created, invite link shown")
  })

  let candidate: Page
  await test.step("candidate opens invite + consents", async () => {
    const inviteUrl = await page.locator('input[readonly]').inputValue()
    log("invite URL: " + inviteUrl.slice(0, 60) + "…")
    candidate = await page.context().newPage()
    await candidate.goto(inviteUrl)
    await expect(candidate.getByText("Interview invitation")).toBeVisible()
    await candidate.getByLabel("I consent").check()
    await candidate.getByRole("button", { name: "Start interview" }).click()
    await expect(candidate.getByText(/Question 1 of/)).toBeVisible({ timeout: 30_000 })
    log("consent OK, WS connected, question 1 delivered")
  })

  await test.step("answer and receive streamed reply", async () => {
    await candidate.getByPlaceholder(/Type your answer/).fill(
      "I built payment services with Go, PostgreSQL and Kubernetes for five years in production.",
    )
    await candidate.getByPlaceholder(/Type your answer/).press("Enter")
    log("answer sent — waiting for next question (LLM streaming)")
    await expect(candidate.getByText(/Question 2 of/)).toBeVisible({ timeout: 120_000 })
    log("streamed reply received, question 2 delivered")
  })

  log("E2E PASSED")
})
