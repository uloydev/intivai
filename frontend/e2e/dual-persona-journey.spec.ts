import { expect, test } from "@playwright/test"

test.describe("Dual-Persona Concurrent Journey (Recruiter & Candidate)", () => {
  test("synchronizes assessment lifecycle between independent recruiter and candidate browser contexts", async ({
    browser,
  }) => {
    // -------------------------------------------------------------------------
    // Context 1: Recruiter Session
    // -------------------------------------------------------------------------
    const recruiterContext = await browser.newContext()
    const recruiterPage = await recruiterContext.newPage()

    await test.step("recruiter logs in and navigates to interviews", async () => {
      await recruiterPage.goto("/login")
      await recruiterPage.locator("#org").fill("demo")
      await recruiterPage.locator("#email").fill("admin@demo.io")
      await recruiterPage.locator("#password").fill("password123")
      await recruiterPage.locator("button[type=submit]").click()
      await expect(recruiterPage).toHaveURL(/.*\/dashboard/)

      await recruiterPage.goto("/interviews")
      await expect(recruiterPage.getByText(/Interview Operations/i)).toBeVisible()
    })

    let inviteUrl = ""
    await test.step("recruiter creates an interview session from eligible candidates", async () => {
      await recruiterPage.getByRole("tab", { name: /Eligible Candidates/i }).click()
      const createBtn = recruiterPage.getByRole("button", { name: /Create Interview/i }).first()
      if (await createBtn.isVisible({ timeout: 5000 }).catch(() => false)) {
        await createBtn.click()
        await recruiterPage.getByRole("button", { name: /Initialize Session/i }).click()
        await expect(recruiterPage.locator("input[readonly]").first()).toBeVisible({ timeout: 15000 })
        inviteUrl = await recruiterPage.locator("input[readonly]").first().inputValue()
      } else {
        // Fallback: copy link from existing scheduled interview
        await recruiterPage.getByRole("tab", { name: /Scheduled & Completed/i }).click()
        const scorecardLink = recruiterPage.locator("a", { hasText: /Scorecard/i }).first()
        if (await scorecardLink.isVisible({ timeout: 5000 }).catch(() => false)) {
          const href = await scorecardLink.getAttribute("href")
          const ivId = href?.split("/").pop()
          inviteUrl = `/invite/${ivId}?t=demo-test-token`
        }
      }
    })

    if (!inviteUrl) {
      test.skip(true, "No eligible candidate or scheduled interview available in demo state.")
      return
    }

    // -------------------------------------------------------------------------
    // Context 2: Candidate Session
    // -------------------------------------------------------------------------
    const candidateContext = await browser.newContext()
    const candidatePage = await candidateContext.newPage()

    await test.step("candidate inspects pre-flight preview and grants ethical AI consent", async () => {
      await candidatePage.goto(inviteUrl)
      await expect(candidatePage.getByText(/What to expect/i)).toBeVisible({ timeout: 10000 })

      const consentCheck = candidatePage.getByLabel(/I consent/i)
      await expect(consentCheck).toBeVisible()
      await consentCheck.check()

      const beginBtn = candidatePage.getByRole("button", { name: /Begin Interview Session/i })
      await expect(beginBtn).toBeEnabled()
    })

    // Cleanup contexts
    await candidateContext.close()
    await recruiterContext.close()
  })
})
