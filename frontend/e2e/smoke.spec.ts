import { expect, test } from "@playwright/test"

test("vite serves the app", async ({ page }) => {
  await page.goto("/login")
  await expect(page.getByText("Intivai")).toBeVisible()
})
