import { test, expect } from "@playwright/test"

test.describe("Coding Sandbox & Live Pair-Programming Terminal E2E", () => {
  test("1. Candidate Chat page allows toggling Code Sandbox split view", async ({ page }) => {
    // Open candidate chat simulation with a ticket
    await page.goto("/chat/00000000-0000-0000-0000-000000000001?t=mock-ticket-token")

    // The Code Sandbox button should be visible in the header
    const sandboxToggle = page.getByRole("button", { name: /Code Sandbox/i })
    await expect(sandboxToggle).toBeVisible()

    // Toggle split view open
    await sandboxToggle.click()

    // Verify Monaco Code Editor and Terminal Console are rendered
    await expect(page.getByRole("button", { name: /Run & Test/i })).toBeVisible()
    await expect(page.getByRole("button", { name: /Terminal Console/i })).toBeVisible()
    await expect(page.getByRole("button", { name: /Test Suite/i })).toBeVisible()
    await expect(page.getByRole("combobox")).toBeVisible() // Language selector

    // Check language options
    const languageSelect = page.getByRole("combobox")
    await expect(languageSelect).toHaveValue("go")
    await languageSelect.selectOption("python")
    await expect(languageSelect).toHaveValue("python")

    // Switch to Test Suite tab
    await page.getByRole("button", { name: /Test Suite/i }).click()
    await expect(page.getByText(/Standard Input \(stdin\)/i)).toBeVisible()
    await expect(page.getByText(/Expected Output/i)).toBeVisible()
    await expect(page.getByRole("button", { name: /Add/i })).toBeVisible()

    // Toggle split view close
    const hideToggle = page.getByRole("button", { name: /Hide Code Sandbox/i })
    await expect(hideToggle).toBeVisible()
    await hideToggle.click()

    // Verify editor is hidden and full chat restored
    await expect(page.getByRole("button", { name: /Run & Test/i })).not.toBeVisible()
  })

  test("2. Voice Interview Room allows toggling Code Sandbox split view", async ({ page }) => {
    await page.goto("/voice/00000000-0000-0000-0000-000000000001")

    // Heading and Voice Evaluator card
    await expect(page.getByText(/Intivai Voice Evaluator/i)).toBeVisible()

    // Toggle Code Sandbox
    const sandboxToggle = page.getByRole("button", { name: /Code Sandbox/i })
    await expect(sandboxToggle).toBeVisible()
    await sandboxToggle.click()

    // Verify split view rendered alongside voice orb
    await expect(page.getByRole("button", { name: /Run & Test/i })).toBeVisible()
    await expect(page.getByText(/Terminal Console/i)).toBeVisible()

    // Close Sandbox
    await page.getByRole("button", { name: /Hide Code Sandbox/i }).click()
    await expect(page.getByRole("button", { name: /Run & Test/i })).not.toBeVisible()
  })
})
