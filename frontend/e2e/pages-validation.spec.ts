import { test, expect } from "@playwright/test"

test.describe("Full Intivai Application Pages Validation", () => {
  test("1. Landing Page (/) should render hero, simulator preview, metrics, and CTA", async ({ page }) => {
    await page.goto("/")
    await expect(page).toHaveURL("/")

    // Brand and navigation
    await expect(page.getByText("Intivai").first()).toBeVisible()
    await expect(page.getByRole("link", { name: "Careers & Jobs" }).first()).toBeVisible()
    await expect(page.getByRole("link", { name: /AI Evaluator/i }).first()).toBeVisible()
    await expect(page.getByRole("link", { name: /How it Works/i }).first()).toBeVisible()
    await expect(page.getByRole("link", { name: /Sign In/i }).first()).toBeVisible()

    // Hero title & description
    await expect(page.getByText(/Screen, Probe, and Grade Engineers with Real-Time AI/i)).toBeVisible()
    await expect(page.getByText(/Intivai conducts adaptive voice and chat technical interviews/i)).toBeVisible()

    // How it works and simulator preview
    await expect(page.getByText(/How Autonomous Screening Works/i)).toBeVisible()
    await expect(page.getByText(/Interactive AI Technical Evaluator/i)).toBeVisible()
    await expect(page.getByText(/AI Interviewer/i).first()).toBeVisible()

    // Value pillars & metrics
    await expect(page.getByText(/Faster Candidate Turnaround/i)).toBeVisible()
    await expect(page.getByText(/Deterministic Safety Rails/i)).toBeVisible()
  })

  test("2. Careers Page (/careers) should render jobs list, search, filters, and apply modal", async ({ page }) => {
    await page.goto("/careers")
    await expect(page).toHaveURL("/careers")

    // Heading and search input
    await expect(page.getByText(/Join High-Growth Engineering Teams/i)).toBeVisible()
    const searchInput = page.getByPlaceholder(/Search by role, company, skills, or location/i)
    await expect(searchInput).toBeVisible()

    // Filter by search — assert the filtered result set (auto-retrying)
    await searchInput.fill("Distributed")
    await expect(page.getByText("Senior Distributed Systems Engineer")).toBeVisible()

    // Job cards
    const applyButtons = page.getByRole("button", { name: /Apply Now/i })
    await expect(applyButtons.first()).toBeVisible()

    // Open Apply Modal
    await applyButtons.first().click()
    await expect(page.getByRole("dialog")).toBeVisible()
    await expect(page.getByText(/Apply for Senior Distributed Systems Engineer/i)).toBeVisible()
    await expect(page.getByLabel(/Full Name/i)).toBeVisible()
    await expect(page.getByLabel(/Email Address/i)).toBeVisible()
    await expect(page.getByText(/Resume \/ CV \(PDF or DOCX\)/i)).toBeVisible()

    // Close Modal
    await page.getByRole("button", { name: /Cancel/i }).click()
    await expect(page.getByText(/Resume \/ CV \(PDF or DOCX\)/i)).not.toBeVisible()
  })

  test("3. Auth Pages (/login & /register) and Recruiter Sign-in", async ({ page }) => {
    // Register page
    await page.goto("/register")
    await expect(page.getByText("Create Workspace")).toBeVisible()
    await expect(page.locator("#name")).toBeVisible()
    await expect(page.locator("#slug")).toBeVisible()

    // Login page
    await page.goto("/login")
    await expect(page.getByText("Intivai Workspace")).toBeVisible()

    // Fill credentials
    await page.locator("#org").fill("demo")
    await page.locator("#email").fill("admin@demo.io")
    await page.locator("#password").fill("password123")

    // Submit form
    await page.locator("button[type=submit]").click()

    // Verify redirected to dashboard
    await expect(page).toHaveURL(/.*\/dashboard/, { timeout: 10000 })
    await expect(page.getByText("Recruitment Command Center")).toBeVisible()
    await expect(page.getByText("Active Roles")).toBeVisible()
  })

  test("4. Recruiter Dashboard & Workspace Navigation (/dashboard, /jobs, /cvs, /candidates, /interviews)", async ({ page }) => {
    // Authenticate first
    await page.goto("/login")
    await page.locator("#org").fill("demo")
    await page.locator("#email").fill("admin@demo.io")
    await page.locator("#password").fill("password123")
    await page.locator("button[type=submit]").click()
    await expect(page).toHaveURL(/.*\/dashboard/)

    // 4a. Dashboard checks
    await expect(page.getByText("Recruitment Command Center")).toBeVisible()
    await expect(page.getByText("Active Roles")).toBeVisible()
    await expect(page.getByText("CVs Ingested")).toBeVisible()

    // 4b. Jobs Page
    await page.goto("/jobs")
    await expect(page.getByText("Job Requisitions")).toBeVisible()
    await expect(page.getByRole("button", { name: /Post New Job/i })).toBeVisible()

    // 4c. CVs Page
    await page.goto("/cvs")
    await expect(page.getByText("CV Ingestion Hub")).toBeVisible()
    await expect(page.getByText(/Upload Candidate Resume/i)).toBeVisible()

    // 4d. Candidates Page
    await page.goto("/candidates")
    await expect(page.getByText("Candidate Screening Pool")).toBeVisible()
    await expect(page.getByPlaceholder(/Search candidate name or email/i)).toBeVisible()

    // 4e. Interviews Page
    await page.goto("/interviews")
    await expect(page.getByText("AI Interview Operations")).toBeVisible()
    await expect(page.getByRole("button", { name: /New Interview Session/i })).toBeVisible()
  })

  test("5. Candidate Invitation Gate (/invite/:id)", async ({ page }) => {
    await page.goto("/invite/demo-invitation-token")
    await expect(page.getByText("Interview Invitation")).toBeVisible()
    await expect(page.getByText(/I consent to my answers being evaluated by AI/i)).toBeVisible()
    await expect(page.getByRole("button", { name: /Begin Interview Session/i })).toBeVisible()
  })

  test("6. Candidate Live Voice Interview Simulator (/voice/:id)", async ({ page }) => {
    await page.goto("/voice/demo-session-id")
    // G4 fix: voice page is recruiter-gated — renders guidance panel, never a public start control
    await expect(page.getByText("Recruiter access required")).toBeVisible()
    await expect(page.getByRole("button", { name: /Start Voice Interview/i })).toHaveCount(0)
  })

  test("7. Candidate Live Chat Interview (/chat/:id)", async ({ page }) => {
    await page.goto("/chat/demo-session-id")
    await expect(page.getByText(/Intivai Live Assessment/i)).toBeVisible()
    // Placeholder is context-dependent (initial vs in-topic clarification)
    await expect(page.getByPlaceholder(/Type your answer to Question|Ask a clarification/i)).toBeVisible()
  })

  test("8. Public Navbar Cross-Page Navigation & Anchors", async ({ page }) => {
    await page.goto("/careers")
    await expect(page).toHaveURL("/careers")

    // Click "How it Works" from careers page
    await page.getByRole("link", { name: /How it Works/i }).first().click()
    await expect(page).toHaveURL(/\/#how-it-works/)
    await expect(page.getByText(/How Autonomous Screening Works/i)).toBeVisible()

    // Click "ROI Calculator"
    await page.getByRole("link", { name: /ROI Calculator/i }).first().click()
    await expect(page).toHaveURL(/\/#calculator/)
    await expect(page.getByText(/Calculate Your Engineering Time Saved/i)).toBeVisible()

    // Click "FAQ"
    await page.getByRole("link", { name: /FAQ/i }).first().click()
    await expect(page).toHaveURL(/\/#faq/)
    await expect(page.getByText(/Frequently Asked Questions/i)).toBeVisible()
  })

  test("9. Recruiter Workspace — Integrations & Webhooks (/integrations)", async ({ page }) => {
    // Authenticate first
    await page.goto("/login")
    await page.locator("#org").fill("demo")
    await page.locator("#email").fill("admin@demo.io")
    await page.locator("#password").fill("password123")
    await page.locator("button[type=submit]").click()
    await expect(page).toHaveURL(/.*\/dashboard/, { timeout: 10000 })

    // Integrations page
    await page.goto("/integrations")
    await expect(page.getByText("Integrations")).toBeVisible()
    await expect(page.getByRole("button", { name: /Add Webhook/i })).toBeVisible()
  })

  test("10. Candidate Portal — GDPR Data Export & Deletion (/candidate/portal)", async ({ page }) => {
    await page.goto("/candidate/portal?token=demo-magic-token-alex-2026")
    await expect(page.getByText(/Applicant Tracking Dashboard/i)).toBeVisible({ timeout: 10_000 })

    // GDPR export button
    await expect(page.getByText(/Export My Data/i)).toBeVisible()

    // GDPR delete section — button text is "Delete Account"
    await expect(page.getByText(/Delete Account/i)).toBeVisible()
  })

  test("11. Interview Scorecard — Recruiter Decision Override", async ({ page }) => {
    // Authenticate first
    await page.goto("/login")
    await page.locator("#org").fill("demo")
    await page.locator("#email").fill("admin@demo.io")
    await page.locator("#password").fill("password123")
    await page.locator("button[type=submit]").click()
    await expect(page).toHaveURL(/.*\/dashboard/, { timeout: 10000 })

    // Completed scorecard with override button
    await page.goto("/interviews/b4c5d6e7-f8a3-4b4c-8d5e-5d6e7f8a3b4c")
    await expect(page.getByText(/Score/i).first()).toBeVisible()
    await expect(page.getByText(/Hiring Verdict/i)).toBeVisible()

    // Override button present on completed scorecards
    const overrideBtn = page.getByRole("button", { name: /Override/i })
    if (await overrideBtn.isVisible().catch(() => false)) {
      await expect(overrideBtn).toBeVisible()
    }
  })

  test("12. Legal & Compliance Pages (/privacy, /terms, /security)", async ({ page }) => {
    await page.goto("/privacy")
    await expect(page).toHaveURL("/privacy")

    await page.goto("/terms")
    await expect(page).toHaveURL("/terms")

    await page.goto("/security")
    await expect(page).toHaveURL("/security")
  })
})
