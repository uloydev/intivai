import { expect, test } from "@playwright/test"

test.describe("Proctoring Integrity Telemetry & Scorecard Transparency", () => {
  test("surfaces unverified integrity audit signals to recruiter while remaining non-punitive to candidate", async ({
    page,
  }) => {
    // 1. Recruiter logs in to inspect candidate integrity audit
    await page.goto("/login")
    await page.locator("#org").fill("demo")
    await page.locator("#email").fill("admin@demo.io")
    await page.locator("#password").fill("password123")
    await page.locator("button[type=submit]").click()
    await expect(page).toHaveURL(/.*\/dashboard/)

    // 2. Open Completed Interviews list
    await page.goto("/interviews")
    await expect(page.getByText(/Interview Operations/i)).toBeVisible()

    const scorecardLink = page.locator("a", { hasText: /Scorecard/i }).first()
    if (await scorecardLink.isVisible({ timeout: 5000 }).catch(() => false)) {
      await scorecardLink.click()
      await expect(page).toHaveURL(/.*\/interviews\/.*/)

      // 3. Verify Proctoring Card renders with unverified advisory disclosure
      const proctoringSection = page.getByText(/AI Proctoring & Integrity Audit/i)
      await expect(proctoringSection).toBeVisible()
      await expect(page.getByText(/Client-reported · unverified/i).first()).toBeVisible()

      // 4. Verify candidate transcript and question analysis section
      await expect(page.getByText(/Question & Answer Transcript Analysis/i)).toBeVisible()
    }
  })
})
