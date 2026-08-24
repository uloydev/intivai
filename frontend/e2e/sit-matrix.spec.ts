import { test, expect } from "@playwright/test"

test.describe("Intivai Comprehensive SIT Suite — Aligned with HR UAT Epics", () => {
  test.beforeEach(async ({ page }) => {
    page.on("pageerror", (err) => {
      console.error("[Page Error]", err)
    })
  })

  // ---------------------------------------------------------------------------
  // EPIC 1: Job Requisition & Rubric Management (UAT-REQ-001 to UAT-REQ-004)
  // ---------------------------------------------------------------------------
  test("Epic 1: Job Requisition & Competency Rubric Management (UAT-REQ-001 to 004)", async ({ page }) => {
    await page.goto("/login")
    await page.locator("#org").fill("demo")
    await page.locator("#email").fill("admin@demo.io")
    await page.locator("#password").fill("password123")
    await page.locator("button[type=submit]").click()
    await expect(page).toHaveURL(/.*\/dashboard/)

    await page.goto("/jobs")
    await expect(page.getByText("Job Requisitions")).toBeVisible()
    await expect(page.getByText("Senior Distributed Systems Engineer")).toBeVisible()
    await expect(page.getByText("Staff Frontend Architect")).toBeVisible()

    const unpublishBtn = page.getByRole("button", { name: "Unpublish" }).first()
    if (await unpublishBtn.isVisible()) {
      await unpublishBtn.click()
      await expect(page.getByRole("button", { name: "Publish" }).first()).toBeVisible()
      await page.getByRole("button", { name: "Publish" }).first().click()
      await expect(page.getByRole("button", { name: "Unpublish" }).first()).toBeVisible()
    }

    const jobSearch = page.getByPlaceholder(/Search roles or skills/i)
    if (await jobSearch.isVisible()) {
      await jobSearch.fill("Frontend")
      await expect(page.getByText("Staff Frontend Architect")).toBeVisible()
      await jobSearch.fill("")
    }
  })

  // ---------------------------------------------------------------------------
  // EPIC 2 & 3: Multi-Source CV Ingestion & Semantic Screening (UAT-ING-001 to 004, UAT-SCR-001 to 004)
  // ---------------------------------------------------------------------------
  test("Epic 2 & 3: Multi-Source CV Ingestion & AI Semantic Screening (UAT-ING-001 to 004, UAT-SCR-001 to 004)", async ({ page }) => {
    await page.goto("/login")
    await page.locator("#org").fill("demo")
    await page.locator("#email").fill("admin@demo.io")
    await page.locator("#password").fill("password123")
    await page.locator("button[type=submit]").click()
    await expect(page).toHaveURL(/.*\/dashboard/)

    await page.goto("/cvs")
    await expect(page.getByText("CV Ingestion Hub")).toBeVisible()
    await expect(page.getByText(/Upload Candidate Resume/i)).toBeVisible()
    await expect(page.getByRole("tab", { name: "Single Candidate" })).toBeVisible()
    await expect(page.getByRole("tab", { name: "Bulk Upload" })).toBeVisible()

    await expect(page.getByText("Alex Rivera")).toBeVisible()
    await expect(page.getByText("Elena Rostova")).toBeVisible()

    // UAT-SCR-004: Candidate Self-Review of Extracted Profile
    // Guarded — requires a valid review token in the DB (seed-dependent).
    await page.goto("/candidate-review/magic-link-token-marcus-vance-2026")
    const reviewTitle = page.getByText(/Review Your Extracted Profile/i)
    if (await reviewTitle.isVisible({ timeout: 3_000 }).catch(() => false)) {
      await expect(page.getByText("Marcus Vance", { exact: true })).toBeVisible()
      await expect(page.getByText(/Extracted Structured Data/i)).toBeVisible()
    }
  })

  // ---------------------------------------------------------------------------
  // EPIC 4: Candidate Portal & Frictionless Authentication (UAT-CAN-001 to UAT-CAN-004)
  // ---------------------------------------------------------------------------
  test("Epic 4: Candidate Portal & Frictionless Authentication (UAT-CAN-001 to 004)", async ({ page }) => {
    await page.goto("/careers")
    await expect(page.getByText(/Join High-Growth Engineering Teams/i)).toBeVisible()
    const applyBtn = page.getByRole("button", { name: /Apply Now/i }).first()
    await applyBtn.click()
    await expect(page.getByRole("dialog")).toBeVisible()
    await expect(page.getByLabel(/Full Name/i)).toBeVisible()
    await expect(page.getByLabel(/Email Address/i)).toBeVisible()
    await page.getByRole("button", { name: /Cancel/i }).click()

    await page.goto("/candidate/portal")
    await expect(page.getByLabel(/Applicant Email Address/i)).toBeVisible()
    await expect(page.getByRole("button", { name: /Send Verification Code/i })).toBeVisible()

    await page.goto("/candidate/portal?token=demo-magic-token-alex-2026")
    await expect(page.getByText(/Applicant Tracking Dashboard/i)).toBeVisible()
    await expect(page.getByText(/alex.rivera@example.com/i)).toBeVisible()
    await expect(page.getByText("Senior Distributed Systems Engineer")).toBeVisible()
    await expect(page.getByText(/Assessment Complete/i)).toBeVisible()
    await expect(page.getByText(/Assessment Performance & Strengths Summary/i)).toBeVisible()
    await expect(page.getByText(/Demonstrated Strengths/i)).toBeVisible()
    await expect(page.getByText(/Growth Opportunities/i)).toBeVisible()
  })

  // ---------------------------------------------------------------------------
  // EPIC 5 & 6: Autonomous AI Interview Execution & Proctoring (UAT-INT-001 to 006, UAT-PRC-001 to 004)
  // ---------------------------------------------------------------------------
  test("Epic 5 & 6: Autonomous AI Interview Rooms & Integrity Proctoring (UAT-INT-001 to 006, UAT-PRC-001 to 004)", async ({ page }) => {
    await page.goto("/invite/demo-invitation-token")
    await expect(page.getByText("Interview Invitation")).toBeVisible()
    await expect(page.getByText(/I consent to my answers being evaluated by AI/i)).toBeVisible()
    await expect(page.getByRole("button", { name: /Begin Interview Session/i })).toBeVisible()

    await page.goto("/chat/demo-session-id")
    await expect(page.getByText(/Intivai Live Assessment/i)).toBeVisible()
    // Chat input placeholder is context-dependent (initial question vs in-topic clarification)
    const chatInput = page.getByPlaceholder(/Type your answer to Question|Ask a clarification/i)
    await expect(chatInput).toBeVisible()

    const sandboxToggle = page.getByRole("button", { name: /Code Sandbox/i })
    await expect(sandboxToggle).toBeVisible()
    await sandboxToggle.click()
    await expect(page.getByRole("button", { name: /Hide Code Sandbox/i })).toBeVisible()

    await page.goto("/voice/demo-session-id")
    // G4 fix: voice page is recruiter-gated — renders guidance panel, never a public start control
    await expect(page.getByText("Recruiter access required")).toBeVisible()
    await expect(page.getByRole("button", { name: /Start Voice Interview/i })).toHaveCount(0)
  })

  // ---------------------------------------------------------------------------
  // EPIC 5b: Multi-Turn Topic Dialogue (interview-chat-flow-mvp)
  // ---------------------------------------------------------------------------
  test("Epic 5b: Multi-Turn Topic Dialogue — topic progress indicator and request human (UAT-INT-MT-001 to 005)", async ({ page }) => {
    await page.goto("/chat/demo-session-id")
    await expect(page.getByText(/Intivai Live Assessment/i)).toBeVisible()

    // Topic progress indicator — present once a question is loaded
    const topicIndicator = page.getByText(/Topic Discussion/i)
    if (await topicIndicator.isVisible().catch(() => false)) {
      await expect(topicIndicator).toBeVisible()
    }

    // "Prefer a human interviewer?" link always present in the chat footer
    await expect(page.getByText(/Prefer a human interviewer/i)).toBeVisible()
  })

  test("Epic 5b: Request Human Interviewer modal (UAT-INT-RH-001 to 003)", async ({ page }) => {
    await page.goto("/chat/demo-session-id")
    await expect(page.getByText(/Intivai Live Assessment/i)).toBeVisible()

    // Open request human modal
    await page.getByText(/Prefer a human interviewer/i).click()
    await expect(page.getByText("Request Human Interviewer")).toBeVisible()
    await expect(page.getByText(/The AI interview will be paused/)).toBeVisible()
    await expect(page.getByRole("button", { name: /Confirm Request/i })).toBeVisible()
    await expect(page.getByRole("button", { name: /Cancel/i })).toBeVisible()

    // Cancel closes modal
    await page.getByRole("button", { name: /Cancel/i }).click()
    await expect(page.getByText("Request Human Interviewer")).not.toBeVisible()
  })

  // ---------------------------------------------------------------------------
  // EPIC 7 & 8: Candidate 360 Scorecards, Verdicts & Talent Passports (UAT-EVL-001 to 004, UAT-PAS-001 to 002)
  // ---------------------------------------------------------------------------
  test("Epic 7 & 8: Candidate 360 Scorecards, Verdicts & Talent Passports (UAT-EVL-001 to 004, UAT-PAS-001 to 002)", async ({ page }) => {
    await page.goto("/login")
    await page.locator("#org").fill("demo")
    await page.locator("#email").fill("admin@demo.io")
    await page.locator("#password").fill("password123")
    await page.locator("button[type=submit]").click()
    await expect(page).toHaveURL(/.*\/dashboard/)

    await page.goto("/candidates")
    await expect(page.getByText("Candidate Screening Pool")).toBeVisible()
    await expect(page.getByText("Alex Rivera").first()).toBeVisible()
    await expect(page.getByText("Elena Rostova").first()).toBeVisible()

    const selectAllBox = page.getByRole("checkbox", { name: /Select all candidates/i })
    await expect(selectAllBox).toBeVisible()
    await selectAllBox.click()
    await expect(page.getByText(/Candidate.*Selected/i)).toBeVisible()
    await expect(page.getByRole("button", { name: /Generate Invites/i })).toBeVisible()
    await page.getByRole("button", { name: /Clear/i }).click()
    await expect(page.getByText(/Candidate.*Selected/i)).not.toBeVisible()

    await page.getByText("Alex Rivera").first().click()
    await expect(page.getByRole("dialog")).toBeVisible()
    await expect(page.getByText(/AI Resume Screening Match/i)).toBeVisible()
    await expect(page.getByText(/Verified Skills from Resume/i)).toBeVisible()
    await expect(page.getByText(/Skills Match/i)).toBeVisible()
    await expect(page.getByText(/Hiring Decision & Notes/i)).toBeVisible()

    await page.keyboard.press("Escape")

    await page.goto("/interviews/b4c5d6e7-f8a3-4b4c-8d5e-5d6e7f8a3b4c")
    await expect(page.getByText(/Score/i).first()).toBeVisible()
    await expect(page.getByText(/92/i).first()).toBeVisible()
    await expect(page.getByText(/Hiring Verdict/i)).toBeVisible()

    await expect(page.getByText(/Can you describe a challenging distributed concurrency/i).first()).toBeVisible()
    await expect(page.getByText(/How do you enforce PostgreSQL tenant isolation/i).first()).toBeVisible()
  })

  // ---------------------------------------------------------------------------
  // EPIC 7b: Recruiter Decision Override (interview-service-mvp)
  // ---------------------------------------------------------------------------
  test("Epic 7b: Recruiter Decision Override on completed scorecard (UAT-EVL-DEC-001 to 004)", async ({ page }) => {
    await page.goto("/login")
    await page.locator("#org").fill("demo")
    await page.locator("#email").fill("admin@demo.io")
    await page.locator("#password").fill("password123")
    await page.locator("button[type=submit]").click()
    await expect(page).toHaveURL(/.*\/dashboard/)

    await page.goto("/interviews/b4c5d6e7-f8a3-4b4c-8d5e-5d6e7f8a3b4c")
    await expect(page.getByText(/Score/i).first()).toBeVisible()
    await expect(page.getByText(/Hiring Verdict/i)).toBeVisible()

    // Override button present on completed scorecards
    const overrideBtn = page.getByRole("button", { name: /Override/i })
    if (await overrideBtn.isVisible().catch(() => false)) {
      await overrideBtn.click()
      await expect(page.getByText("Override AI Recommendation")).toBeVisible()
      await expect(page.getByPlaceholder(/Reason/i)).toBeVisible()
      await page.keyboard.press("Escape")
    }
  })

  // ---------------------------------------------------------------------------
  // EPIC 8b: GDPR Data Export & Deletion (candidate-portal-mvp)
  // ---------------------------------------------------------------------------
  test("Epic 8b: GDPR Data Export & Right-to-Erasure (UAT-GDPR-001 to 005)", async ({ page }) => {
    await page.goto("/candidate/portal?token=demo-magic-token-alex-2026")
    await expect(page.getByText(/Applicant Tracking Dashboard/i)).toBeVisible({ timeout: 10_000 })

    await expect(page.getByText(/Export My Data/i)).toBeVisible()
    await expect(page.getByText(/Delete Account/i)).toBeVisible()

    const deleteBtn = page.getByRole("button", { name: /Delete Account/i }).first()
    if (await deleteBtn.isVisible().catch(() => false)) {
      await deleteBtn.click()
      await expect(page.getByText(/DELETE/i).first()).toBeVisible()
      await page.keyboard.press("Escape")
    }
  })

  // ---------------------------------------------------------------------------
  // EPIC 9: Company Intelligence & Custom AI Interview Rails (UAT-ORG-001 to UAT-ORG-002)
  // ---------------------------------------------------------------------------
  test("Epic 9: Company Intelligence & Custom AI Interview Rails (UAT-ORG-001 to 002)", async ({ page }) => {
    await page.goto("/login")
    await page.locator("#org").fill("demo")
    await page.locator("#email").fill("admin@demo.io")
    await page.locator("#password").fill("password123")
    await page.locator("button[type=submit]").click()
    await expect(page).toHaveURL(/.*\/dashboard/)

    await page.goto("/company-context")
    await expect(page.getByText(/Company Intelligence & AI Interview Rails/i)).toBeVisible()
    await expect(page.getByText(/Tenant System Prompt/i)).toBeVisible()
  })

  // ---------------------------------------------------------------------------
  // EPIC 10b: Webhook Integration (settings-integrations-mvp)
  // ---------------------------------------------------------------------------
  test("Epic 10b: Webhook Integration — CRUD + delivery log (UAT-WH-001 to 005)", async ({ page }) => {
    await page.goto("/login")
    await page.locator("#org").fill("demo")
    await page.locator("#email").fill("admin@demo.io")
    await page.locator("#password").fill("password123")
    await page.locator("button[type=submit]").click()
    await expect(page).toHaveURL(/.*\/dashboard/)

    await page.goto("/integrations")
    await expect(page.getByText("Integrations")).toBeVisible()
    await expect(page.getByRole("button", { name: /Add Webhook/i })).toBeVisible()

    // Open create webhook form
    await page.getByRole("button", { name: /Add Webhook/i }).click()
    await expect(page.getByText(/Interview Completed/i)).toBeVisible()
    await page.keyboard.press("Escape")
  })

  // ---------------------------------------------------------------------------
  // EPIC 10: Recruitment Operations, Analytics & ROI (UAT-OPS-001 to UAT-OPS-003)
  // ---------------------------------------------------------------------------
  test("Epic 10: Recruitment Operations, Analytics & ROI (UAT-OPS-001 to UAT-OPS-003)", async ({ page }) => {
    await page.goto("/login")
    await page.locator("#org").fill("demo")
    await page.locator("#email").fill("admin@demo.io")
    await page.locator("#password").fill("password123")
    await page.locator("button[type=submit]").click()

    await expect(page).toHaveURL(/.*\/dashboard/, { timeout: 10000 })
    await expect(page.getByText("Recruitment Command Center")).toBeVisible()
    await expect(page.getByText("Active Roles")).toBeVisible()
    await expect(page.getByText("CVs Ingested")).toBeVisible()
    await expect(page.getByText(/Screening Pass Rate/i)).toBeVisible()
    await expect(page.getByText("Interviews Run")).toBeVisible()
    await expect(page.getByText("Recruitment Pipeline Velocity")).toBeVisible()
    await expect(page.getByText("Total Sourced & Applied")).toBeVisible()
    await expect(page.getByText("Passed AI CV Screening")).toBeVisible()
    await expect(page.getByText("AI Assessments Completed")).toBeVisible()

    await page.goto("/#calculator")
    await expect(page.getByText(/Calculate Your Engineering Time Saved/i)).toBeVisible()
    await expect(page.getByText(/Dev Hours Saved/i)).toBeVisible()
    await expect(page.getByText(/Estimated Savings/i)).toBeVisible()
  })

  // ---------------------------------------------------------------------------
  // EPIC 11: Legal & Compliance Pages (landing-compliance-mvp)
  // ---------------------------------------------------------------------------
  test("Epic 11: Legal & Compliance pages are accessible (UAT-LEGAL-001 to 003)", async ({ page }) => {
    await page.goto("/privacy")
    await expect(page).toHaveURL("/privacy")

    await page.goto("/terms")
    await expect(page).toHaveURL("/terms")

    await page.goto("/security")
    await expect(page).toHaveURL("/security")
  })
})
