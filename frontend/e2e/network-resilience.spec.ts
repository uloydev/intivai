import { expect, test } from "@playwright/test"

test.describe("Network Disruption & Candidate Draft Preservation", () => {
  test("retains candidate draft answer and recovers gracefully during network drops", async ({
    page,
    context,
  }) => {
    // Open candidate portal / demo interview room
    await page.goto("/candidate/portal")
    await expect(page.getByText(/Candidate Portal/i)).toBeVisible()

    const emailInput = page.locator("#candidate-email")
    await expect(emailInput).toBeVisible()
    await emailInput.fill("candidate.network.test@example.com")

    // Simulate transient network disruption
    await context.setOffline(true)

    // Verify draft input remains intact despite network failure
    await expect(emailInput).toHaveValue("candidate.network.test@example.com")

    // Restore network connectivity
    await context.setOffline(false)

    // Verify application remains interactive without full reload
    await expect(emailInput).toHaveValue("candidate.network.test@example.com")
    const submitBtn = page.getByRole("button", { name: /Send Verification Code/i })
    await expect(submitBtn).toBeEnabled()
  })
})
