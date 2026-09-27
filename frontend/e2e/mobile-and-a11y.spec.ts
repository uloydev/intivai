import { expect, test } from "@playwright/test"

test.describe("Mobile Viewport & Touch Ergonomics (375px)", () => {
  test.use({ viewport: { width: 375, height: 667 } })

  test("renders responsive careers board and candidate portal with zero horizontal overflow", async ({
    page,
  }) => {
    // 1. Careers Page on Mobile Viewport
    await page.goto("/careers")
    await expect(page.getByText(/Join High-Growth Engineering Teams/i)).toBeVisible({ timeout: 10000 })

    // Check for horizontal overflow (scrollWidth should equal clientWidth)
    const hasHorizontalOverflow = await page.evaluate(() => {
      return document.documentElement.scrollWidth > document.documentElement.clientWidth
    })
    expect(hasHorizontalOverflow).toBe(false)

    // 2. Candidate Portal on Mobile Viewport
    await page.goto("/candidate/portal")
    await expect(page.getByText(/Candidate Portal/i)).toBeVisible({ timeout: 10000 })

    const portalOverflow = await page.evaluate(() => {
      return document.documentElement.scrollWidth > document.documentElement.clientWidth
    })
    expect(portalOverflow).toBe(false)

    // Verify touch action button has minimum accessible height (44px)
    const actionBtn = page.getByRole("button", { name: /Send Verification Code/i })
    await expect(actionBtn).toBeVisible()
    const box = await actionBtn.boundingBox()
    expect(box).not.toBeNull()
    if (box) {
      expect(box.height).toBeGreaterThanOrEqual(44)
    }
  })
})
