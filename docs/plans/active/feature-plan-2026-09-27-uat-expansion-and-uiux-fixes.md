# Feature Plan — Playwright UAT Expansion & UI/UX Remediation

- **Date:** 2026-09-27
- **Slug:** uat-expansion-and-uiux-fixes
- **Requirement:** Deep-dive UAT automation with Playwright covering HR epics, network resilience, proctoring telemetry, and mobile accessibility, while remediating critical UI/UX friction in Candidate Chat and Recruiter workspaces.
- **Status:** active

---

## 1. Code-Grounded Analysis & Current State Evidence

### Current State Matrix

| Area | Location | Current Behavior | Gap / Severity |
|---|---|---|---|
| **Chat Advance Trap** | `frontend/src/pages/Chat.tsx:645` & `:157` | `disabled={!input.trim() \|\| ...}` locks "Next Topic" button when input empty; `sendWithAction` exits early on empty trimmed input | **High (UX Blocker)**: Candidate on Turn 2+ cannot advance topic without typing dummy text (e.g. "Done" or "Next"). |
| **Backend Empty Answer Guard** | `backend/internal/interview/domain/interview.go:183` | `strings.TrimSpace(content) == ""` returns `ANSWER_EMPTY` error on all dialogue turns | **Medium (Domain Contradiction)**: Advance action on turn > 1 should allow concluding discussion without forcing extra text. |
| **Sandbox Mobile Responsive** | `frontend/src/pages/Chat.tsx:440-475` | Split view divides screen horizontally even on viewports `< 1024px` | **High (Mobile UX)**: Monaco editor and chat input crush into unreadable slices on mobile/tablet viewports. |
| **AI Synthesis Indicator** | `frontend/src/pages/Chat.tsx:550-600` | 2–4s pause between submit and streaming token display shows only disabled input with no feedback | **Medium (Candidate Anxiety)**: Candidate unsure if answer was delivered or if application froze. |
| **Job Rubric Sum Validation** | `frontend/src/pages/Jobs.tsx:580-600` | Sliders display warning when sum != 1.00, but publish button remains enabled | **Medium (Data Integrity)**: Recruiter can submit non-normalized weights, causing 400 rejection from backend. |
| **Residual Button Arrows (R-08)** | `frontend/src/pages/Jobs.tsx:693, 701` & `Interviews.tsx:377` | Dialog navigation and scorecard action buttons retain `→` and `←` characters | **Low (Design Standard)**: Violates Hard Gate R-08. |
| **Invite Link Dispatch** | `frontend/src/pages/Interviews.tsx:365-375` | 1-click copy with toast already functional; scorecard link button retains `→` arrow | **Closed (Verified)**: Dedicated copy button and feedback verified in code. |
| **Playwright Dual-Persona** | `frontend/e2e/happy-path.spec.ts` | Single browser context opens new page sequentially; lacks real-time multi-user synchronization | **High (UAT Gap)**: Does not verify real-time recruiter scorecard updates while candidate is actively interviewing. |
| **Playwright Resilience** | `frontend/e2e/sit-matrix.spec.ts` | Zero coverage for UAT Scenario 8 (network drop mid-interview) | **High (Beta Gate Gap)**: WS reconnect and answer draft persistence unverified. |
| **Playwright Proctoring** | `frontend/e2e/sit-matrix.spec.ts` | Zero coverage for UAT Scenario 12 (tab-switch and paste telemetry verification) | **High (Beta Gate Gap)**: Verification that proctoring alerts remain unverified and hidden from candidate unproven. |
| **Playwright Accessibility** | `frontend/e2e/*.spec.ts` | Zero automated WCAG 2.1 AA or mobile viewport apply tests | **Medium (Compliance Gap)**: Candidate mobile journey (UAT Scenario 6) unverified against 375px viewport. |

---

## 2. UI/UX Judgment & Design Evaluation

### A. Recruiter / HR Experience (Efficiency, Clarity & Governance)

1. **Pipeline Funnel (`PipelineFunnel.tsx`):**
   - *Strengths:* High executive clarity; connects high-level metrics (Applied → Screened → Interviewed → Recommended) directly to candidate filtered views via URL query params (`/candidates?stage=screening_passed`).
   - *Enhancement:* Maintain sticky funnel header on scroll in desktop view to retain pipeline context when reviewing long candidate tables.

2. **Candidate 360 Drawer (`Candidate360Drawer.tsx`):**
   - *Strengths:* Zero-context-switch inspection. Resume breakdown, skills match chips, and scoring weights remain accessible inside the table.
   - *Enhancement:* Add quick keyboard navigation (`j` / `k` or up/down arrows) to cycle candidates without closing drawer.

3. **Verbatim Quote Grounding (`InterviewResult.tsx`):**
   - *Strengths:* Eliminates AI hallucination doubt by anchoring numerical dimension scores to verbatim candidate quotes in the transcript.
   - *Enhancement:* Highlight transcript quote directly when clicking dimension card.

4. **Recruiter Governance & Override:**
   - *Strengths:* AI proposes, human decides. Modal strictly requires a justification string for audit logging before overriding verdicts.

---

### B. Candidate Experience (Friction Reduction, Trust & Anxiety)

1. **Ethical AI Consent (`Invite.tsx`):**
   - *Strengths:* Explicit consent checkbox complies with EU AI Act High-Risk AI requirements and provides transparent expectations before session begins.
   - *Fix Needed:* Add simple system audio/input diagnostic test before room entry.

2. **Passwordless Portal (`CandidatePortal.tsx`):**
   - *Strengths:* Frictionless 6-digit OTP and magic link access eliminates password management. Built-in JSON data export and account erasure complies with GDPR Articles 15 & 17.

3. **Live Chat Room (`Chat.tsx`):**
   - *Critical UX Flaw:* The "Next Topic" advance trap. A candidate who provides a comprehensive initial answer cannot advance when the AI asks a minor follow-up unless they type placeholder text.
   - *Mobile Viewport Defect:* Screen crushing when opening the code sandbox on mobile screens. A segmented tab switch (`[ Chat ] | [ Code Sandbox ]`) is mandatory below 1024px.
   - *Feedback Gap:* The silent gap between candidate answer submission and LLM token delivery causes candidate panic. A pulsing synthesis state ("AI Evaluator formulating question...") must render instantly.

---

## 3. Architectural & Decision Register

| Decision | Context | Choice | Rationale |
|---|---|---|---|
| **D1: Empty Input Advance** | Candidate finishes topic on Turn 2+ without additional comments. | Allow "Next Topic" click with empty input if `topicTurn >= 1`. If empty, send default marker text `"[Topic concluded]"` or update domain to accept empty on advance. | Forcing dummy input confuses candidates and pollutes transcripts with filler words. |
| **D2: Sandbox Mobile Layout** | Code sandbox opened on small viewport (<1024px). | Render segmented tab switcher (`[ Chat ]` / `[ Code Editor ]`) on `< 1024px`; keep split-pane on `≥ 1024px`. | Split panes on mobile screens render both editor and chat unreadable. Segmented tabs preserve full touch area. |
| **D3: Playwright Multi-Context** | Verifying recruiter + candidate live coordination. | Use `browser.newContext()` to create independent, concurrent browser instances. | Accurately tests cross-user WebSocket events and tenant boundary isolation without cookie collisions. |
| **D4: Network Drop Simulation** | Fulfilling UAT Scenario 8 (offline reconnect). | Utilize Playwright `context.setOffline(true/false)` while candidate has draft text in textarea. | Validates WebSocket auto-reconnect, ticket token re-use, and client draft persistence under true network disruption. |
| **D5: Telemetry Testing** | Fulfilling UAT Scenario 12 (proctoring events). | Dispatch synthetic `visibilitychange` and clipboard paste events via Playwright. | Proves that proctoring alerts appear in recruiter scorecard with "Unverified" label while remaining invisible to candidate. |
| **D6: Automated Accessibility** | Ensuring candidate portal and chat comply with WCAG 2.1 AA. | Integrate axe-core/playwright and assert zero critical/serious violations on key flows. | Candidate-facing portals must be legally accessible across screen readers and keyboard navigation. |

---

## 4. Implementation Phases

### Phase 1: Candidate Chat Room UI/UX Remediation (TDD)
- **Task 1.1:** Fix "Next Topic" Advance Logic (`frontend/src/pages/Chat.tsx`).
  - Allow `advanceTopic` button to be active when `input.trim() === ""` provided `topicTurn >= 1`.
  - When input is empty, submit `"[Topic concluded by candidate]"` to satisfy backend non-empty answer constraint without failing validation.
  - Dynamically update button label: `"Submit & Advance"` when input has text; `"Conclude Topic"` when input is empty.
  - Unit test in `frontend/src/pages/Chat.test.tsx`.
- **Task 1.2:** Add AI Synthesis Indicator (`frontend/src/pages/Chat.tsx`).
  - Render an active pulsing message bubble when `pendingAnswer` is true or during turn transitions.
  - Disappear instantly upon arrival of first streaming token.
- **Task 1.3:** Responsive Code Sandbox Tab Switch (`frontend/src/pages/Chat.tsx`).
  - Below `lg` (1024px): replace horizontal split with full-width segmented tab control.
  - Show unread indicator dot on Chat tab when AI streams response while user is on Sandbox tab.

---

### Phase 2: Recruiter Workspace UI/UX Hardening
- **Task 2.1:** Job Rubric 100% Sum Enforcement (`frontend/src/pages/Jobs.tsx`).
  - Compute `canPublish` guard requiring `(useDefaultWeights || isWeightSumValid(weightSum(weights)))`.
  - Disable "Publish Job & Pipeline" button when custom weights are selected and sum != 1.00.
  - Unit test in frontend/src/pages/Jobs.tsx.
- **Task 2.2:** Eliminate Residual Button Arrows per R-08 (`frontend/src/pages/Jobs.tsx` & `frontend/src/pages/Interviews.tsx`).
  - Replace `"Next: Configure Stages →"` with `"Next: Configure Stages"`.
  - Replace `"← Back to Details"` with `"Back to Details"`.
  - Replace `"View Scorecard →"` with `"View Scorecard"`.

---

### Phase 3: Playwright UAT Suite Expansion
- **Task 3.1:** Multi-Persona Dual-Context Journey (planned frontend/e2e/dual-persona-journey.spec.ts).
  - Context 1 (Recruiter): Creates job with 100% rubric, screens candidate, generates invite link.
  - Context 2 (Candidate): Opens invite link, completes consent, answers Question 1, advances topic with empty input, runs Go sandbox code.
  - Context 1 (Recruiter): Observes live completion, inspects scorecard quotes, overrides AI verdict with audit reason.
- **Task 3.2:** Network Drop & Draft Preservation Spec (planned frontend/e2e/network-resilience.spec.ts).
  - Connects to chat session.
  - Enters draft text.
  - `setOffline(true)`: asserts input locks, amber "Reconnecting..." badge appears, draft text remains.
  - `setOffline(false)`: asserts socket reconnects, draft text intact, sends answer successfully.
- **Task 3.3:** Proctoring Telemetry Spec (planned frontend/e2e/proctoring-telemetry.spec.ts).
  - Triggers tab switch (`visibilitychange`) and clipboard paste.
  - Asserts recruiter scorecard displays "Unverified" events; asserts candidate chat interface never mentions proctoring.
- **Task 3.4:** Mobile Viewport & Accessibility Spec (planned frontend/e2e/mobile-and-a11y.spec.ts).
  - Configures 375x667 viewport (iPhone SE).
  - Tests public apply modal on `/careers` and OTP login on `/candidate/portal`.
  - Runs axe-core/playwright accessibility audit asserting 0 violations.

---

## 5. File Ownership & Staging Plan

```
# Phase 1: Chat UI/UX Fixes
frontend/src/pages/Chat.tsx
frontend/src/pages/Chat.test.tsx

# Phase 2: Recruiter UI/UX Fixes
frontend/src/pages/Jobs.tsx
frontend/src/pages/Jobs.test.tsx
frontend/src/pages/Interviews.tsx

# Phase 3: Playwright UAT Specs & Deps
frontend/package.json
frontend/e2e/dual-persona-journey.spec.ts
frontend/e2e/network-resilience.spec.ts
frontend/e2e/proctoring-telemetry.spec.ts
frontend/e2e/mobile-and-a11y.spec.ts
docs/engineering/test-matrices/uat-matrix.md
```

---

## 6. Definition of Done & Acceptance Gates

- [ ] `Chat.tsx` permits empty-input topic advancement when `topicTurn >= 1`.
- [ ] Responsive segmented tab switch functions on screen widths `< 1024px`.
- [ ] Real-time synthesis indicator displays during AI question generation latency.
- [ ] `Jobs.tsx` validates rubric weights to exactly 100% before allowing publish.
- [ ] Residual button arrows eliminated from `Jobs.tsx` and `Interviews.tsx` per R-08.
- [ ] Dual-persona Playwright spec passes against live local stack.
- [ ] Network resilience spec verifies WS reconnect and draft answer retention.
- [ ] Proctoring spec verifies unverified telemetry visibility on recruiter scorecard.
- [ ] Mobile 375px viewport and axe-core accessibility audits pass clean.
- [ ] `make check && make coverage && npm run test` passes 100% green.
