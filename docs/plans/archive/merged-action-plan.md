# Merged Action Plan — Competitive Gaps + UI/UX Debt

> Status: archived 2026-08-24. Superseded by `docs/FINDINGS.md` — all task
> statuses were migrated there (section F, IDs F1–F17). This file is kept as
> a frozen reference for the detailed per-task acceptance criteria and the
> Penpot/Magic-UI workflow recipes. Do not edit; do not track new status here.

**Sources:** `COMPETITIVE_REVIEW_ACTION_PLAN.md` + `UI_UX_REVAMP_PLAN.md` (both deleted 2026-08-24 — fully absorbed by this plan)
**Date:** 2026-08-22
**Goal:** Resolve competitive gaps AND UI/UX debt in single coordinated execution
# Intivai — Merged Action Plan

**Sources:** `COMPETITIVE_REVIEW_ACTION_PLAN.md` + `UI_UX_REVAMP_PLAN.md`
**Date:** 2026-08-22
**Goal:** Resolve competitive gaps AND UI/UX debt in single coordinated execution
**Format:** Agent-friendly — each task has exact file paths, acceptance criteria, and testing commands

---

## Execution Rules

1. **TDD mandatory** — write failing test first, then implement, then verify
2. **Run `make check` after every task** — must be green before moving on
3. **Run `make test-integration-dev` after schema changes** — must be green
4. **One concern per commit** — never bundle unrelated changes
5. **Dependencies are explicit** — do NOT start a task before its dependencies are marked complete
6. **Status tracking** — update the checkbox `[x]` when task is fully complete (code + tests + `make check` green)
7. **Merged tasks** — when a competitive task merges into a UI/UX phase, both acceptance criteria must be met

---

## Merge Strategy

### Overlap Map

| Competitive Task | UI/UX Phase | Resolution |
|---|---|---|
| P0-2 Mobile responsive | Phase 3.3 Chat, 3.5 Candidates, 3.8 AppShell | **Absorb** into UI/UX phases |
| P0-3 Hide proctoring | Phase 3.3 Chat, 3.7 InterviewResult | **Absorb** into UI/UX phases |
| P0-7 Full transcript | Phase 3.7 InterviewResult | **Absorb** into UI/UX phases |
| P0-8 Connection indicator | Phase 3.3 Chat | **Absorb** into UI/UX phases |
| P0-6 Override AI | Phase 3.7 InterviewResult | **Absorb** into UI/UX phases |
| P0-10 PRD update | None | **Independent** — run anytime |

### Conflict-Free Tasks (Run Parallel)

| Plan | Task | No Overlap With |
|---|---|---|
| Competitive | P0-1 Magic link | Any UI/UX |
| Competitive | P0-4 Webhooks | Any UI/UX |
| Competitive | P0-5 Data portal | Any UI/UX |
| Competitive | P0-9 Request human | Any UI/UX |
| UI/UX | Phase 0 Design tokens | Any competitive |
| UI/UX | Phase 1 Penpot designs | Any competitive |
| UI/UX | Phase 2 Component upgrades | Any competitive |
| UI/UX | Phase 3.1 Landing | Any competitive |
| UI/UX | Phase 3.2 Dashboard | Any competitive |
| UI/UX | Phase 3.4 Jobs | Any competitive |
| UI/UX | Phase 3.6 CVs | Any competitive |
| UI/UX | Phase 3.9 PublicLayout | Any competitive |
| UI/UX | Phase 4 Cross-cutting | Any competitive |

---

## Week 1: Foundation + Independent Backend

**Goal:** Design tokens + competitive tasks with zero UI overlap

### Week 1A: UI/UX Phase 0 — Design Token System

**Tool:** ui-ux-pro-max

| Task | Description | Deliverable | Effort |
|------|-------------|-------------|--------|
| 0.1 | Generate design system via CLI | `python3 scripts/search.py "hr recruitment saas dashboard" --design-system -p "Intivai"` | 30min |
| 0.2 | Persist as master source of truth | `design-system/MASTER.md` | 30min |
| 0.3 | Create page-specific overrides | `design-system/pages/landing.md`, `dashboard.md`, `chat.md` | 1hr |
| 0.4 | Migrate `index.css` tokens | Add semantic tokens: `--color-success`, `--color-warning`, `--color-info`, `--shadow-card`, `--shadow-elevated`, `--duration-fast/normal/slow`, `--easing-standard` | 1hr |
| 0.5 | Typography audit | Decide: real display typeface (DM Sans, Sora) or remove `font-display` alias | 30min |
| 0.6 | Button size scale | Proper `sm`/`md`/`lg`/`xl` with meaningful height differences | 30min |
| 0.7 | Badge size variants | `sm`/`md`/`lg` with appropriate padding | 15min |
| 0.8 | Card hover states | Default hover effect + glass variant contrast fix for light mode | 15min |

**Token Migration Map:**

```css
/* BEFORE (scattered in 50+ places) */
bg-emerald-500/10 text-emerald-600 dark:text-emerald-400
bg-amber-500/10 text-amber-600 dark:text-amber-400
bg-red-500/10 text-red-600 dark:text-red-400
bg-blue-500/10 text-blue-600 dark:text-blue-400

/* AFTER (single source of truth) */
bg-success/10 text-success
bg-warning/10 text-warning
bg-destructive/10 text-destructive
bg-info/10 text-info
```

```css
/* NEW CSS VARIABLES */
:root {
  --color-success: #059669;
  --color-success-foreground: #065f46;
  --color-warning: #d97706;
  --color-warning-foreground: #92400e;
  --color-info: #2563eb;
  --color-info-foreground: #1e40af;

  --shadow-card: 0 1px 3px rgba(0,0,0,0.1), 0 1px 2px rgba(0,0,0,0.06);
  --shadow-elevated: 0 10px 15px -3px rgba(0,0,0,0.1), 0 4px 6px -4px rgba(0,0,0,0.1);
  --shadow-glow-primary: 0 0 20px rgba(30,58,95,0.15);

  --duration-fast: 150ms;
  --duration-normal: 300ms;
  --duration-slow: 500ms;
  --easing-standard: cubic-bezier(0.4, 0, 0.2, 1);
  --easing-spring: cubic-bezier(0.34, 1.56, 0.64, 1);
}

.dark {
  --color-success: #10b981;
  --color-success-foreground: #d1fae5;
  --color-warning: #f59e0b;
  --color-warning-foreground: #fef3c7;
  --color-info: #3b82f6;
  --color-info-foreground: #dbeafe;
}
```

**Acceptance criteria:**
- [ ] `design-system/MASTER.md` exists with full token spec
- [ ] `index.css` has all semantic color tokens
- [ ] `index.css` has shadow tokens
- [ ] `index.css` has typography scale tokens
- [ ] `index.css` has animation tokens
- [ ] Button component has proper size scale
- [ ] Badge component has size variants
- [ ] Card glass variant has adequate light mode contrast
- [ ] `make check` green

**Testing:**
```bash
make check
# Visual: verify all pages render with new tokens
# Dark mode: verify contrast parity
```

**Dependencies:** None

---

### Week 1B: Competitive P0-10 — PRD Title Update

**Why:** "Autonomous AI Recruitment Platform" misrepresents the product.

**Files to modify:**
- `PRD_SUMMARY.md` — line 4: change title
- `AI_Interviewer_Phases.md` — update phase descriptions
- `AI_Interviewer_Research.md` — section 6: update positioning
- `README.md` — update project description

**Implementation:**
1. Replace all instances of "Autonomous AI Recruitment Platform" with "AI Technical Interview Platform"
2. Update executive summary: lead with "technical interview" not "recruitment"
3. Add positioning statement: "Intivai is an AI-powered technical interview platform that conducts live conversational interviews with coding assessments — replacing manual screening calls and take-home coding tests."
4. Update competitive positioning to emphasize: "Not an ATS. An AI interview platform that integrates with your existing ATS."

**Acceptance criteria:**
- [ ] PRD title updated to "AI Technical Interview Platform"
- [ ] Executive summary reflects technical interview positioning
- [ ] Competitive section acknowledges "not an ATS"
- [ ] `make check` green

**Testing:**
```bash
grep -r "Recruitment Platform" PRD_SUMMARY.md AI_Interviewer_Phases.md README.md
# Should return 0 matches (all replaced)
```

**Dependencies:** None

**Effort:** 1 hour

---

### Week 1C: Competitive P0-1 — Magic Link Authentication

**Why:** Candidates abandon when forced to enter 6-digit codes. Hireflix uses zero-friction magic links.

**Files to modify:**
- `internal/iam/domain/otp.go` — keep OTP code but make magic link the primary flow
- `internal/iam/api/candidate_auth_handler.go` — add magic link endpoint
- `internal/iam/application/magic_link_worker.go` — new: email magic link on interview invite
- `frontend/src/pages/candidate/Portal.tsx` — remove OTP input UI, redirect to magic link flow
- `frontend/src/pages/candidate/Interview.tsx` — auto-auth from magic link token in URL

**Implementation:**
1. Interview invitation already sends email → add magic link token to email body
2. Magic link = `https://app.intivai.com/candidate/interview/{id}?token={magic_token}`
3. Candidate clicks link → backend validates token → auto-authenticates → starts interview
4. Token: `crypto/rand` 32-byte, URL-safe, single-use, 24h expiry
5. Keep OTP as fallback (for candidates who don't receive email), but UI should default to magic link

**Acceptance criteria:**
- [ ] Candidate receives interview email with clickable magic link
- [ ] Clicking link opens interview page with zero additional auth steps
- [ ] Magic link is single-use (second click shows "already started" message)
- [ ] Magic link expires after 24 hours
- [ ] OTP flow still works as fallback (manual code entry option visible)
- [ ] `make check` green
- [ ] Playwright test: email → click link → interview starts (no manual auth)

**Testing:**
```bash
make check
# Manual: send interview invite, click magic link in email, verify interview loads
```

**Dependencies:** None

**Effort:** 2-3 hours

---

### Week 1D: Competitive P0-4 — ATS Integration Webhooks

**Why:** HR teams won't switch ATS just for interviews. Intivai must push data to existing ATS.

**Files to create/modify:**
- `internal/integration/domain/webhook.go` — new: webhook entity + event types
- `internal/integration/application/webhook_worker.go` — new: async webhook delivery
- `internal/integration/api/webhook_handler.go` — new: CRUD for webhook configs
- `internal/integration/infrastructure/delivery/http_webhook.go` — new: HTTP POST delivery
- `pkg/db/migrations/0XX_webhooks.up.sql` — new: webhook_configs table
- `frontend/src/pages/settings/Integrations.tsx` — new: webhook configuration UI

**Implementation:**
1. Webhook events:
   - `interview.completed` — payload: `{interview_id, candidate_id, score, recommendation, transcript_url}`
   - `candidate.screened` — payload: `{candidate_id, score, passed, breakdown}`
   - `candidate.advanced` — payload: `{candidate_id, stage, timestamp}`
2. Webhook config per org:
   - `url` (POST endpoint)
   - `events[]` (which events to send)
   - `secret` (HMAC-SHA256 signature for verification)
   - `active` (enable/disable)
3. Delivery: async via Asynq worker, 3 retries with exponential backoff
4. Signature: `X-Intivai-Signature: sha256={hmac}` header
5. Generic webhook (not ATS-specific) — works with Greenhouse, Lever, Workable, Zapier, custom

**Acceptance criteria:**
- [ ] Webhook config CRUD API: `POST/GET/PUT/DELETE /api/v1/webhooks`
- [ ] Webhook delivery on `interview.completed` event
- [ ] HMAC-SHA256 signature in `X-Intivai-Signature` header
- [ ] Retry logic: 3 attempts, exponential backoff
- [ ] Webhook logs visible in settings page (last 10 deliveries, status, timestamp)
- [ ] `make check` green
- [ ] Integration test: mock webhook endpoint, verify delivery + signature

**Testing:**
```bash
make check
make test-integration-dev
# Manual: configure webhook → run interview → verify POST received at mock endpoint
```

**Dependencies:** None

**Effort:** 6-8 hours

---

### Week 1E: Competitive P0-5 — Candidate Data Portal

**Why:** GDPR compliance + candidate trust. Candidates want to see what data Intivai holds and delete it.

**Files to create/modify:**
- `internal/candidate/api/data_portal_handler.go` — new: candidate data export + delete endpoints
- `internal/candidate/application/export_candidate_data.go` — new: JSON export of all candidate data
- `internal/candidate/application/delete_candidate_data.go` — new: cascade delete all candidate data
- `pkg/db/migrations/0XX_data_requests.up.sql` — new: data_requests table (audit trail)
- `frontend/src/pages/candidate/DataPortal.tsx` — new: candidate self-service page

**Implementation:**
1. Candidate access: magic link from interview invitation email → `/candidate/data/{token}`
2. Two actions:
   - **Export:** Download JSON with: CV text, extracted profile, interview transcript, scores, consent record
   - **Delete:** Permanently delete all candidate data (CV, transcript, scores, recordings)
3. Delete requires confirmation: "Type DELETE to confirm"
4. Data request logged to `data_requests` table (timestamp, action, candidate_id, completed_at)
5. Cascade delete: candidates → applications → interviews → questions → scores → recordings

**Acceptance criteria:**
- [ ] Candidate can access data portal via magic link
- [ ] Export generates JSON with all candidate data
- [ ] Delete removes all candidate data (cascade)
- [ ] Delete requires confirmation text input
- [ ] Data request logged in `data_requests` table
- [ ] `make check` green

**Testing:**
```bash
make check
# Manual: request export → verify JSON contains all data
# Manual: request delete → verify data removed from DB
```

**Dependencies:** None

**Effort:** 4-6 hours

---

## Week 2: Design + Components + Remaining Backend

**Goal:** Penpot designs + shadcn/Magic UI components + competitive P0-9

### Week 2A: UI/UX Phase 1 — Visual Design (Penpot)

**Tool:** Penpot MCP

**Screen Priority Matrix:**

| Priority | Screen | Reason | Complexity |
|----------|--------|--------|------------|
| P0 | Landing page | First impression, conversion | High |
| P0 | Dashboard | Daily recruiter use | Medium |
| P0 | Chat interview | Candidate-facing, brand | High |
| P1 | Public layout (header/footer) | Brand consistency | Low |
| P1 | Jobs list | Core workflow | Medium |
| P1 | Candidates list | Core workflow | Medium |
| P1 | Interview result | Evaluation sharing | Medium |
| P2 | AppShell (sidebar) | Navigation refinement | Low |
| P2 | CVs page | Utility page | Low |
| P2 | Login/Register | Auth flow | Low |

**Per-Screen Design Deliverables:**

Each screen gets:
1. **Desktop layout** — exact dimensions, spacing grid, component placement
2. **Mobile layout** — breakpoint behavior, stacked vs side-by-side
3. **Dark mode variant** — token-mapped colors, shadow adjustments
4. **State variants** — hover, focus, active, disabled, loading, empty, error
5. **Micro-interactions** — button presses, card hovers, page transitions
6. **Typography spec** — heading levels, body text, labels, captions

**Penpot MCP Workflow:**

```
1. Agent reads current screen code via Penpot MCP
2. Agent creates/modifies design elements in Penpot
3. Agent reports exact specs back (dimensions, colors, spacing)
4. Code is updated to match Penpot design precisely
5. Agent validates code matches design via MCP read-back
```

**Acceptance criteria:**
- [ ] All P0 screens have Penpot designs (Landing, Dashboard, Chat)
- [ ] All P1 screens have Penpot designs (PublicLayout, Jobs, Candidates, InterviewResult)
- [ ] All P2 screens have Penpot designs (AppShell, CVs, Login/Register)
- [ ] Each design includes desktop + mobile + dark mode variants
- [ ] Each design includes state variants (hover, focus, active, disabled, loading, empty, error)
- [ ] Design specs exported as CSS tokens for implementation

**Dependencies:** Phase 0 (tokens) complete

**Effort:** 4-6 hours (design time)

---

### Week 2B: UI/UX Phase 2 — Component Upgrades

**Tool:** Magic UI MCP + shadcn CLI

**Core Component Upgrades:**

| Component | Current State | Upgrade | Source |
|-----------|--------------|---------|--------|
| Button | `h-8`/`h-9` (4px diff), gradient lacks focus ring | Proper size scale, fix focus ring | shadcn |
| Card | Flat default, glass low contrast light mode | Hover states, glass fix, Magic Card variant | shadcn + Magic UI |
| Badge | Fixed `h-5`, no sizes | Size variants `sm`/`md`/`lg` | shadcn |
| Dialog | Basic modal | Animated entry/exit | Magic UI |
| AlertDialog | **Missing** (uses `window.confirm`) | New component | shadcn |
| Empty state | 5 different patterns | Single reusable `EmptyState` | shadcn |
| Skeleton | Flat rectangles | Content-aware skeleton shapes | shadcn |
| Toast | Default Sonner | Custom styled with icons/actions | shadcn |

**New Magic UI Components:**

| Component | Source | Use Case | Target Pages |
|-----------|--------|----------|--------------|
| `BlurFade` | Magic UI | Page/section entrance animations | All pages |
| `AnimatedGradientText` | Magic UI | Hero headline, section titles | Landing |
| `Particles` | Magic UI | Hero background effect | Landing |
| `Grid Pattern` | Magic UI | Section backgrounds | Landing |
| `Dot Pattern` | Magic UI | Subtle section texture | Landing |
| `Bento Grid` | Magic UI | Feature showcase layout | Landing |
| `Number Ticker` | Magic UI | KPI counter animation | Dashboard |
| `Glare Hover` | Magic UI | Interactive card hover | Dashboard, Jobs |
| `Animated List` | Magic UI | Recent activity feed | Dashboard |
| `Marquee` | Magic UI | Logo/trust bar | Landing |
| `Shimmer Button` | Magic UI | Primary CTAs | Landing, Login |
| `Ripple Button` | Magic UI | Action buttons | Throughout |
| `Typing Animation` | Magic UI | Chat input indicator | Chat |
| `Progressive Blur` | Magic UI | Card depth effect | Landing |
| `Confetti` | Magic UI | Success celebration | Interview creation |
| `Meteors` | Magic UI | Hero background variant | Landing |

**shadcn Components to Install:**

```bash
npx shadcn@latest add alert-dialog
npx shadcn@latest add tooltip
npx shadcn@latest add dropdown-menu
npx shadcn@latest add tabs
npx shadcn@latest add progress
npx shadcn@latest add separator
npx shadcn@latest add avatar
npx shadcn@latest add switch
```

**Acceptance criteria:**
- [ ] Button component has proper size scale (sm/md/lg/xl)
- [ ] Button gradient variant has visible focus ring
- [ ] Card component has hover states
- [ ] Card glass variant has adequate light mode contrast
- [ ] Badge component has size variants (sm/md/lg)
- [ ] Dialog component has animated entry/exit
- [ ] AlertDialog component installed and functional
- [ ] EmptyState component created with variants
- [ ] Skeleton component has content-aware shapes
- [ ] Toast component has custom styling
- [ ] All Magic UI components installed and functional
- [ ] `make check` green

**Testing:**
```bash
make check
# Visual: verify all components render correctly
# Interaction: verify hover, focus, active states work
# Dark mode: verify all components have dark mode variants
```

**Dependencies:** Phase 0 (tokens) complete

**Effort:** 3-4 hours

---

### Week 2C: Competitive P0-9 — Request Human Interviewer

**Why:** Candidates feel anxious about AI-only interviews. A safety valve reduces anxiety.

**Files to create/modify:**
- `internal/interview/api/request_human_handler.go` — new: endpoint to flag interview for human review
- `internal/interview/application/request_human.go` — new: set flag + notify recruiter
- `notification/application/human_request_worker.go` — new: email/Slack notification to recruiter
- `frontend/src/pages/candidate/Chat.tsx` — add "Request Human Interviewer" button

**Implementation:**
1. Button placement: small text link below chat input ("Prefer a human interviewer?")
2. Click → confirmation modal: "Request a human interviewer? The AI interview will be paused."
3. On confirm:
   - Set `interview.human_requested = true` in DB
   - Send notification to recruiter (email + in-app)
   - Show candidate: "Your request has been noted. A team member will follow up."
   - Interview pauses (AI stops asking questions)
4. Recruiter can: (a) resume AI interview, (b) join manually, or (c) schedule human interview

**Acceptance criteria:**
- [ ] "Request Human Interviewer" link visible below chat input
- [ ] Confirmation modal appears on click
- [ ] On confirm: interview pauses, recruiter notified
- [ ] Candidate sees confirmation message
- [ ] `make check` green

**Testing:**
```bash
make check
# Manual: click "Request Human Interviewer" → verify confirmation → verify recruiter notification
```

**Dependencies:** P0-1 (magic link) should be done first (same auth flow)

**Effort:** 3-4 hours

---

## Week 3: Page Revamp — No Overlap

**Goal:** UI/UX phases for pages with no competitive task overlap

### Week 3A: UI/UX Phase 3.1 — Landing Page

**Files to modify:**
- `frontend/src/pages/Landing.tsx`

**Issues + Fixes:**

| Issue | Fix | Tool | Effort |
|-------|-----|------|--------|
| FAQ no ARIA | Add `role="button"`, `aria-expanded`, `tabIndex` | Manual | 30min |
| FAQ no animation | Add expand/collapse height transition | Magic UI `BlurFade` | 1hr |
| No scroll animations | Add `whileInView` stagger reveals per section | Magic UI | 2hr |
| Hero metrics flat | Number Ticker on each metric + visual hierarchy | Magic UI | 1hr |
| Section spacing `space-y-24` | Responsive `space-y-16 md:space-y-24` | Tailwind | 15min |
| No trust signals | Add Marquee logo bar | Magic UI | 30min |
| Hero background plain | Particles or Grid Pattern background | Magic UI | 1hr |
| CTA buttons plain | Shimmer Button for primary CTA | Magic UI | 30min |
| Feature cards flat | Bento Grid layout + Glare Hover | Magic UI | 2hr |
| No brand typeface | Add display font (DM Sans or Sora) | Font import | 30min |

**Acceptance criteria:**
- [ ] FAQ has proper ARIA attributes
- [ ] FAQ has expand/collapse animation
- [ ] All sections have scroll-triggered animations
- [ ] Hero metrics use Number Ticker animation
- [ ] Section spacing is responsive
- [ ] Trust signals (Marquee logo bar) present
- [ ] Hero background has visual effect (Particles/Grid)
- [ ] Primary CTA uses Shimmer Button
- [ ] Feature cards use Bento Grid + Glare Hover
- [ ] Brand typeface loaded and applied
- [ ] `make check` green

**Testing:**
```bash
make check
# Visual: verify all animations work
# Accessibility: verify FAQ has proper ARIA
# Performance: verify no layout shift from animations
```

**Dependencies:** Phase 0 (tokens) + Phase 2 (components) complete

**Effort:** 8-10 hours

---

### Week 3B: UI/UX Phase 3.2 — Dashboard

**Files to modify:**
- `frontend/src/pages/Dashboard.tsx`

**Issues + Fixes:**

| Issue | Fix | Tool | Effort |
|-------|-----|------|--------|
| KPI cards read-only | Wrap in `Link` to filtered views | Manual | 1hr |
| PipelineFunnel shows 0 during load | Skeleton until data ready | shadcn Skeleton | 30min |
| No KPI animation | Number Ticker on mount | Magic UI | 1hr |
| Empty state inconsistent | Standardized `EmptyState` component | shadcn | 30min |
| 5 simultaneous queries | Add `staleTime: 30_000`, tune `gcTime` | Manual | 30min |
| Screening Pass Rate misleading | Add per-role breakdown toggle or clarify label | Manual | 1hr |
| KPI cards flat | Glare Hover on hover | Magic UI | 30min |
| Activity feed plain | Animated List for recent items | Magic UI | 1hr |

**Acceptance criteria:**
- [ ] KPI cards are clickable links to filtered views
- [ ] PipelineFunnel shows skeleton during load
- [ ] KPI numbers animate on mount (Number Ticker)
- [ ] Empty states use standardized EmptyState component
- [ ] Queries have staleTime configured
- [ ] Screening Pass Rate label clarified or breakdown added
- [ ] KPI cards have Glare Hover effect
- [ ] Activity feed uses Animated List
- [ ] `make check` green

**Testing:**
```bash
make check
# Visual: verify all animations work
# Performance: verify no unnecessary re-renders
```

**Dependencies:** Phase 0 (tokens) + Phase 2 (components) complete

**Effort:** 5-6 hours

---

### Week 3C: UI/UX Phase 3.4 — Jobs Page

**Files to modify:**
- `frontend/src/pages/Jobs.tsx`

**Issues + Fixes:**

| Issue | Fix | Tool | Effort |
|-------|-----|------|--------|
| No confirmation on archive | Replace with `AlertDialog` | shadcn | 30min |
| Skills input raw text | Tag-style input with pills + keyboard support | Manual | 2hr |
| Step 2 confusing | Make pipeline stages configurable or simplify to info | Manual | 1hr |
| Native checkbox | Use shadcn `Checkbox` component | shadcn | 15min |
| No search/filter | Add search bar + status filter tabs | Manual | 1hr |

**Acceptance criteria:**
- [ ] Archive action uses AlertDialog confirmation
- [ ] Skills input shows tag pills
- [ ] Step 2 is either configurable or clearly informational
- [ ] Checkboxes use shadcn Checkbox component
- [ ] Search bar and status filter tabs present
- [ ] `make check` green

**Testing:**
```bash
make check
# Manual: verify archive confirmation works
# Manual: verify skills input shows pills
```

**Dependencies:** Phase 0 (tokens) + Phase 2 (components) complete

**Effort:** 5-6 hours

---

### Week 3D: UI/UX Phase 3.6 — CVs Page

**Files to modify:**
- `frontend/src/pages/CVs.tsx`

**Issues + Fixes:**

| Issue | Fix | Tool | Effort |
|-------|-----|------|--------|
| `window.confirm()` for delete | Replace with `AlertDialog` | shadcn | 30min |
| No drag-and-drop upload | Add drop zone with visual feedback | Magic UI + manual | 2hr |
| Upload card gradient conflict | Clean up gradient/tab styling | Manual | 30min |
| Bulk upload stale state | Normalize to state on file change | Manual | 15min |

**Acceptance criteria:**
- [ ] Delete action uses AlertDialog confirmation
- [ ] Drag-and-drop upload zone present
- [ ] Upload card styling cleaned up
- [ ] Bulk upload state normalized
- [ ] `make check` green

**Testing:**
```bash
make check
# Manual: verify drag-and-drop works
# Manual: verify delete confirmation works
```

**Dependencies:** Phase 0 (tokens) + Phase 2 (components) complete

**Effort:** 3-4 hours

---

### Week 3E: UI/UX Phase 3.9 — PublicLayout

**Files to modify:**
- `frontend/src/layouts/PublicLayout.tsx`

**Issues + Fixes:**

| Issue | Fix | Tool | Effort |
|-------|-----|------|--------|
| Footer links to `/#security` | Create real Privacy/Terms pages or remove links | Manual | 1hr |
| Footer nav data duplicated | Extract shared `NAV` data, reuse in footer | Manual | 30min |
| Dark mode toggle no animation | Add icon crossfade/rotation | Manual | 30min |
| Mobile drawer sticky conflict | Fix z-index stacking context | Manual | 30min |

**Acceptance criteria:**
- [ ] Footer links point to real pages (or removed)
- [ ] Footer nav data not duplicated
- [ ] Dark mode toggle has animation
- [ ] Mobile drawer z-index fixed
- [ ] `make check` green

**Testing:**
```bash
make check
# Visual: verify footer links work
# Dark mode: verify toggle animation
```

**Dependencies:** Phase 0 (tokens) + Phase 2 (components) complete

**Effort:** 2-3 hours

---

## Week 4: Page Revamp — With Competitive Merge

**Goal:** UI/UX phases that absorb competitive tasks

### Week 4A: UI/UX Phase 3.3 — Chat Interview (Absorbs P0-2, P0-3, P0-8)

**Files to modify:**
- `frontend/src/pages/candidate/Chat.tsx`
- `frontend/src/hooks/useWebSocket.ts`
- `frontend/src/components/ui/ConnectionBadge.tsx` (new)
- `frontend/src/hooks/useProctoring.ts`

**UI/UX Issues + Fixes:**

| Issue | Fix | Tool | Effort |
|-------|-----|------|--------|
| No scroll-to-bottom FAB | Floating button when scrolled up | Magic UI `Shimmer Button` | 1hr |
| Timer warnings missing | Show 30s/15s/5s warnings in chat area with color change | Manual | 1hr |
| Split view unusable on mobile | Stack vertically on `<lg`, full-width toggle | Tailwind | 1hr |
| Probes indistinguishable | Dashed border + distinct badge color for probes | Manual | 30min |
| No sent indicator | Checkmark icon on submitted answers | Manual | 30min |
| No typing indicator | Typing Animation when LLM is streaming | Magic UI | 1hr |
| Input placeholder changes confusing | Simplify to one consistent placeholder | Manual | 15min |

**Competitive Tasks Absorbed:**

| Task | What to Add | Effort |
|------|-------------|--------|
| P0-2 Mobile responsive | Responsive layout: full-width bubbles, sticky input, stacked scorecard | 2hr |
| P0-3 Hide proctoring | Remove all proctoring UI indicators from candidate view | 30min |
| P0-8 Connection indicator | WebSocket status badge (green/yellow/red dot + text) | 1hr |

**Combined Implementation:**
1. Responsive layout: test on iPhone SE (375px), iPhone 14 (390px), Pixel 7 (412px)
2. Chat messages: full-width bubbles, no horizontal scroll
3. Input field: fixed to bottom of viewport (sticky)
4. Touch targets: minimum 44px tap targets (WCAG 2.5.5)
5. Safe area padding for notch devices (iOS)
6. Remove proctoring UI indicators (keep telemetry collection)
7. Add WebSocket status badge (green/yellow/red)
8. Auto-reconnect: 3 attempts with 2s/4s/8s backoff
9. On reconnect: auto-resume from last question

**Acceptance criteria:**
- [ ] Chat works on iPhone SE (375px width)
- [ ] Chat works on iPhone 14 (390px width)
- [ ] Chat works on Pixel 7 (412px width)
- [ ] Input field stays visible when keyboard opens
- [ ] Candidate sees NO proctoring indicators during interview
- [ ] Telemetry still collected and stored (verify in DB)
- [ ] Green dot visible when WebSocket connected
- [ ] Yellow dot + text when reconnecting
- [ ] Red dot + text when failed
- [ ] Auto-reconnect attempts 3 times
- [ ] On reconnect: interview resumes from last question
- [ ] Scroll-to-bottom FAB present
- [ ] Timer warnings visible in chat area
- [ ] Split view stacked on mobile
- [ ] Probes visually distinct
- [ ] Sent indicator present
- [ ] Typing indicator present when LLM streaming
- [ ] `make check` green

**Testing:**
```bash
make check
# Playwright: viewport 375px, verify no layout breaks
# Manual: start interview → disable network → verify yellow dot appears
# Manual: verify no proctoring UI visible to candidate
```

**Dependencies:** Phase 0 (tokens) + Phase 2 (components) complete

**Effort:** 8-10 hours

---

### Week 4B: UI/UX Phase 3.5 — Candidates Page (Absorbs P0-2)

**Files to modify:**
- `frontend/src/pages/Candidates.tsx`

**UI/UX Issues + Fixes:**

| Issue | Fix | Tool | Effort |
|-------|-----|------|--------|
| Bulk actions overlap mobile nav | Reposition to top on mobile or stack | Manual | 1hr |
| Table not responsive | Card layout on `<sm` breakpoint | Tailwind | 2hr |
| No search debounce | Add 200ms debounce with `useDeferredValue` | Manual | 15min |
| Checkbox type cast hack | Refactor `toggleSelect` signature | Manual | 30min |

**Competitive Tasks Absorbed:**

| Task | What to Add | Effort |
|------|-------------|--------|
| P0-2 Mobile responsive | Responsive table → card layout on mobile | 1hr |

**Combined Implementation:**
1. Table → card layout on `<sm` breakpoint
2. Bulk actions repositioned to top on mobile
3. Search debounce added
4. Checkbox type cast fixed

**Acceptance criteria:**
- [ ] Candidates page works on iPhone SE (375px width)
- [ ] Table renders as cards on mobile
- [ ] Bulk actions don't overlap mobile nav
- [ ] Search has debounce
- [ ] Checkboxes use proper types
- [ ] `make check` green

**Testing:**
```bash
make check
# Playwright: viewport 375px, verify card layout
```

**Dependencies:** Phase 0 (tokens) + Phase 2 (components) complete

**Effort:** 4-5 hours

---

### Week 4C: UI/UX Phase 3.7 — Interview Result (Absorbs P0-6, P0-7)

**Files to modify:**
- `frontend/src/pages/interview/InterviewResult.tsx`
- `frontend/src/pages/interview/Scorecard.tsx` (new or refactor)
- `frontend/src/components/evaluation/DecisionOverride.tsx` (new)
- `frontend/src/components/evaluation/TranscriptViewer.tsx` (new)
- `internal/evaluation/api/report_handler.go`
- `internal/evaluation/application/update_decision.go`
- `pkg/db/migrations/0XX_recruiter_decision.up.sql`

**UI/UX Issues + Fixes:**

| Issue | Fix | Tool | Effort |
|-------|-----|------|--------|
| PDF blob URL leak | Add `useEffect` cleanup + `revokeObjectURL` | Manual | 15min |
| Dimension name `_` replace bug | Use `replace(/_/g, " ")` | Manual | 5min |
| `window.confirm()` for reject | Replace with `AlertDialog` | shadcn | 30min |
| No print styles | Add `@media print` CSS | Manual | 1hr |
| No share feature | Add share-by-email or copy-link button | Manual | 1hr |

**Competitive Tasks Absorbed:**

| Task | What to Add | Effort |
|------|-------------|--------|
| P0-6 Override AI recommendation | New endpoint + UI component for recruiter decision override | 3-4hr |
| P0-7 Full transcript on scorecard | New TranscriptViewer component showing Q&A pairs | 2-3hr |

**Combined Implementation:**
1. Fix PDF blob URL memory leak
2. Fix dimension name replace bug
3. Replace window.confirm with AlertDialog
4. Add print styles
5. Add share feature (copy link)
6. Add recruiter decision override:
   - New table: `recruiter_decisions`
   - API: `PUT /api/v1/interviews/:id/decision`
   - UI: Override button → modal with reason textarea
   - Audit trail: override logged with user + timestamp
7. Add full transcript viewer:
   - Scorecard layout: scores top, transcript bottom
   - Q&A pairs formatted as chat bubbles (AI left, candidate right)
   - Per-question scores visible inline
   - Transcript scrollable on mobile

**Acceptance criteria:**
- [ ] PDF blob URL revoked on unmount
- [ ] Dimension names display correctly (no underscores)
- [ ] Reject action uses AlertDialog confirmation
- [ ] Print styles present
- [ ] Share feature present (copy link)
- [ ] Recruiter can override AI recommendation on scorecard
- [ ] Override requires a reason (text input, non-empty)
- [ ] Scorecard shows both AI original + recruiter override
- [ ] Override logged in `recruiter_decisions` table
- [ ] Scorecard shows full transcript below scores
- [ ] Q&A pairs formatted as chat bubbles (AI left, candidate right)
- [ ] Per-question scores visible inline
- [ ] Transcript is scrollable on mobile
- [ ] `make check` green

**Testing:**
```bash
make check
make test-integration-dev
# Manual: view scorecard → click Override → enter reason → verify override saved
# Manual: view scorecard → scroll to transcript → verify all Q&A visible
```

**Dependencies:** Phase 0 (tokens) + Phase 2 (components) complete

**Effort:** 8-10 hours

---

### Week 4D: UI/UX Phase 3.8 — AppShell (Absorbs P0-2)

**Files to modify:**
- `frontend/src/layouts/AppShell.tsx`

**UI/UX Issues + Fixes:**

| Issue | Fix | Tool | Effort |
|-------|-----|------|--------|
| Mobile bottom nav 6 items overflow | Reduce to 4 primary + "More" overflow menu | Manual | 2hr |
| Workspace UUID display | Show org name/slug from JWT or API | Manual | 30min |
| No active route indicator on mobile | Add background pill or dot indicator | Manual | 30min |
| Brand gradient inconsistency | Unify gradient across breakpoints | Manual | 15min |
| No page transitions | Add `BlurFade` route transition wrapper | Magic UI | 1hr |
| No route loading boundary | Wrap `<Outlet />` in `Suspense` + skeleton | Manual | 30min |

**Competitive Tasks Absorbed:**

| Task | What to Add | Effort |
|------|-------------|--------|
| P0-2 Mobile responsive | Fix mobile nav overflow, reduce to 4 items + "More" | 1hr |

**Combined Implementation:**
1. Reduce mobile bottom nav to 4 primary items + "More" overflow menu
2. Show org name/slug instead of UUID
3. Add active route indicator (background pill or dot)
4. Unify brand gradient across breakpoints
5. Add BlurFade route transition wrapper
6. Add Suspense loading boundary with skeleton

**Acceptance criteria:**
- [ ] Mobile bottom nav has 4 items + "More" menu
- [ ] Workspace shows org name/slug (not UUID)
- [ ] Active route has visual indicator on mobile
- [ ] Brand gradient consistent across breakpoints
- [ ] Page transitions use BlurFade animation
- [ ] Route loading uses Suspense + skeleton
- [ ] `make check` green

**Testing:**
```bash
make check
# Playwright: viewport 375px, verify nav overflow fixed
# Manual: verify page transitions smooth
```

**Dependencies:** Phase 0 (tokens) + Phase 2 (components) complete

**Effort:** 4-5 hours

---

## Week 5: Cross-Cutting + Validation + P1 Tasks

**Goal:** Final polish + competitive P1 tasks

### Week 5A: UI/UX Phase 4 — Cross-Cutting Improvements

**Files to modify:**
- `frontend/src/App.tsx`
- `frontend/src/layouts/AppShell.tsx`
- `frontend/src/layouts/PublicLayout.tsx`
- Multiple pages

**Issues + Fixes:**

| Issue | Fix | Scope | Effort |
|-------|-----|-------|--------|
| No route loading boundaries | Wrap `<Outlet />` in `Suspense` + skeleton | `AppShell.tsx` | 30min |
| No page transitions | Add `BlurFade` route transition wrapper | `AppShell.tsx` | 1hr |
| No global error boundary | Add route-level error boundaries | `App.tsx` | 1hr |
| Inconsistent empty states | Create `EmptyState` component with variants | New component | 1hr |
| All `window.confirm()` | Replace with `AlertDialog` everywhere | CVs, InterviewResult, Jobs | 1hr |
| No keyboard shortcuts | Add `Cmd+K` command palette, `Cmd+N` new item | AppShell | 3hr |
| Dark mode toggle no animation | Add icon crossfade | PublicLayout, AppShell | 30min |
| Mobile bottom nav overflow | Reduce to 4 items + "More" | AppShell | 2hr |
| Workspace UUID display | Show org name/slug | AppShell | 30min |
| No print styles | Add `@media print` for interview results | InterviewResult | 1hr |

**Acceptance criteria:**
- [ ] All routes have loading boundaries
- [ ] All routes have page transitions
- [ ] Global error boundary present
- [ ] EmptyState component used across all pages
- [ ] No window.confirm() in production
- [ ] Keyboard shortcuts functional (Cmd+K, Cmd+N)
- [ ] Dark mode toggle animated
- [ ] Mobile nav overflow fixed
- [ ] Workspace shows org name
- [ ] Print styles present for interview results
- [ ] `make check` green

**Testing:**
```bash
make check
# Manual: verify all page transitions work
# Manual: verify keyboard shortcuts work
# Manual: verify no window.confirm() anywhere
```

**Dependencies:** All Phase 3 tasks complete

**Effort:** 10-12 hours

---

### Week 5B: Competitive P1-1 — SOC 2 Type I Preparation

**Why:** Enterprise HR requires SOC 2. Manatal and Hireflix are certified.

**Files to create:**
- `docs/compliance/soc2_readiness.md` — new: SOC 2 controls inventory
- `docs/compliance/access_control_policy.md` — new: access control documentation
- `docs/compliance/data_retention_policy.md` — new: data retention documentation
- `docs/compliance/incident_response_plan.md` — new: incident response documentation
- `internal/audit/access_log_middleware.go` — enhance: log all data access with user + action

**Implementation:**
1. Document SOC 2 Trust Service Criteria:
   - CC6.1: Logical access controls (JWT + RBAC + RLS — already implemented)
   - CC6.2: Authentication (bcrypt + JWT — already implemented)
   - CC6.3: Access removal (need: user deactivation endpoint)
   - CC7.1: Monitoring (need: enhanced audit logging)
   - CC7.2: Anomaly detection (need: rate limit alerts)
2. Create access control policy document
3. Create data retention policy document (90-day auto-delete, candidate consent)
4. Create incident response plan document
5. Enhance audit logging: log all `SELECT`, `INSERT`, `UPDATE`, `DELETE` on sensitive tables
6. Add user deactivation endpoint: `DELETE /api/v1/users/:id` (soft delete, revoke JWT)

**Acceptance criteria:**
- [ ] SOC 2 controls inventory document created
- [ ] Access control policy document created
- [ ] Data retention policy document created
- [ ] Incident response plan document created
- [ ] Audit logging captures all data access
- [ ] User deactivation endpoint works
- [ ] `make check` green

**Dependencies:** None

**Effort:** 1-2 weeks (documentation + implementation)

---

### Week 5C: Competitive P1-2 — DOCX Resume Parsing

**Why:** Many candidates submit DOCX files. Research doc already has the design.

**Files to modify:**
- `internal/cv/application/parse_worker.go` — add DOCX detection + parsing
- `internal/cv/infrastructure/docx/parser.go` — new: DOCX text extraction
- `pkg/db/migrations/0XX_cv_format.up.sql` — add `cv_format` column to candidates table

**Implementation:**
1. Detect file type from content bytes (not just extension):
   - PDF: starts with `%PDF`
   - DOCX: ZIP with `word/document.xml` inside
2. For DOCX: use `github.com/nguyenthenguyen/docx` (MIT, already in Research)
3. Add `cv_format` column: `pdf`, `docx`, `unknown`
4. Update candidate status flow: `new` → `parsing` → `extracting` → `extracted`
5. Add DOCX to bulk upload validation

**Acceptance criteria:**
- [ ] DOCX upload works end-to-end
- [ ] Text extraction from DOCX returns clean text
- [ ] `cv_format` column populated correctly
- [ ] Bulk upload accepts DOCX files
- [ ] `make check` green
- [ ] Integration test: upload DOCX → verify extraction

**Testing:**
```bash
make check
make test-integration-dev
# Manual: upload DOCX resume → verify extraction → verify candidate profile
```

**Dependencies:** None

**Effort:** 4-6 hours

---

### Week 5D: Competitive P1-3 — PDF Scorecard Export

**Why:** Recruiters expect downloadable PDF scorecards. JSON-only is friction.

**Files to modify:**
- `internal/evaluation/application/pdf.go` — already exists (Maroto), enhance for mobile-friendly layout
- `internal/evaluation/api/report_handler.go` — add `GET /interviews/:id/report/pdf` endpoint
- `frontend/src/pages/interview/Scorecard.tsx` — add "Download PDF" button

**Implementation:**
1. PDF layout (already implemented via Maroto):
   - Header: Company logo + candidate name + date
   - Overall score + recommendation badge
   - Dimension radar chart (if possible) or score bars
   - Verbatim quotes section
   - Proctoring summary (if available)
   - Footer: "Generated by Intivai" + page numbers
2. Mobile-friendly: A4 portrait, readable at 100% zoom
3. API endpoint: `GET /api/v1/interviews/:id/report/pdf` → returns PDF bytes
4. Frontend: "Download PDF" button on scorecard page

**Acceptance criteria:**
- [ ] PDF download works from scorecard page
- [ ] PDF contains all scorecard data (scores, quotes, transcript summary)
- [ ] PDF is readable on mobile (portrait A4)
- [ ] `make check` green

**Testing:**
```bash
make check
# Manual: view scorecard → click Download PDF → verify PDF opens correctly
```

**Dependencies:** None

**Effort:** 2-3 hours (Maroto already implemented)

---

### Week 5E: Competitive P1-4 — Encryption at Rest Documentation

**Why:** Security audit requirement. Competitors explicitly state encryption.

**Files to modify:**
- `PRD_SUMMARY.md` — add encryption at rest to NFR table
- `AI_Interviewer_Research.md` — section 7: add encryption details

**Implementation:**
1. PostgreSQL: enable `pgcrypto` extension for column-level encryption
2. MinIO: server-side encryption (SSE-S3) for CV files
3. Document: "AES-256 encryption at rest for all candidate data (CVs, transcripts, scores)"
4. Document: "TLS 1.3 in transit for all API and WebSocket connections"

**Acceptance criteria:**
- [ ] PRD NFR table includes encryption at rest
- [ ] MinIO bucket configured with SSE-S3
- [ ] `make check` green

**Dependencies:** None

**Effort:** 1 hour

---

### Week 5F: Competitive P1-5 — Add "Interview Credits" Pricing Tier

**Why:** CEO recommendation: usage-based pricing is Intivai's strongest card.

**Files to modify:**
- `PRD_SUMMARY.md` — add pricing tier
- `AI_Interviewer_Research.md` — section 6: update pricing model

**Implementation:**
1. New tier: "Interview Credits" — $99 for 500 credits (1 credit = 1 interview)
2. Credits don't expire (within 12 months)
3. Volume pricing: $0.20/credit for 1000+ credits
4. Update pricing table in PRD

**Acceptance criteria:**
- [ ] Pricing table updated with credits tier
- [ ] Volume pricing documented
- [ ] `make check` green

**Dependencies:** None

**Effort:** 30 minutes

---

## Week 6: Validation

**Goal:** Final validation across all changes

### Week 6A: UI/UX Phase 5 — Validation

**Accessibility Checks (ui-ux-pro-max):**

```bash
python3 scripts/search.py "animation accessibility z-index loading" --domain ux
```

| Check | Method | Scope |
|-------|--------|-------|
| Color contrast ≥4.5:1 | ui-ux-pro-max token audit | All text on all backgrounds |
| Touch targets ≥44pt | Manual measurement | All interactive elements |
| Dark mode contrast parity | ui-ux-pro-max `color-dark-mode` rules | All tokens in `.dark` |
| Keyboard navigation | Manual tab-through | All pages |
| Screen reader labels | Manual audit + `aria-*` check | All interactive elements |
| Reduced motion | `@media (prefers-reduced-motion: reduce)` | All animations |
| Focus visible indicators | Check `focus-visible:ring` on all buttons/inputs | All interactive |

**Layout Checks:**

| Check | Method | Scope |
|-------|--------|-------|
| 375px (small phone) | Playwright viewport | All pages |
| 768px (tablet) | Playwright viewport | All pages |
| 1024px (desktop) | Playwright viewport | All pages |
| 1440px (wide) | Playwright viewport | All pages |
| Landscape orientation | Playwright viewport | Chat, Interview |

**Performance Checks:**

| Check | Method |
|-------|--------|
| No layout shift from animations | Lighthouse CLS |
| Animations respect `prefers-reduced-motion` | Manual toggle |
| No memory leaks (blob URLs, timers) | Chrome DevTools |
| Bundle size impact from Magic UI | `npm run build` comparison |

**Acceptance criteria:**
- [ ] All accessibility checks pass
- [ ] All layout checks pass
- [ ] All performance checks pass
- [ ] Lighthouse Accessibility ≥90
- [ ] Lighthouse Performance ≥85
- [ ] Bundle size increase <15%
- [ ] `make check` green

**Testing:**
```bash
make check
# Playwright: all viewports
# Lighthouse: all pages
# Manual: keyboard navigation, screen reader
```

**Dependencies:** All Phase 4 tasks complete

**Effort:** 2-3 hours

---

## Dependency Graph

```
Week 1:
  UI/UX Phase 0 (tokens) ─────────────────────────────┐
  Competitive P0-10 (PRD update) ──────────────────────┤
  Competitive P0-1 (Magic link) ───────────────────────┤
  Competitive P0-4 (Webhooks) ─────────────────────────┤
  Competitive P0-5 (Data portal) ──────────────────────┤
                                                        │
Week 2:                                                 │
  UI/UX Phase 1 (Penpot) ───── depends on Phase 0 ────┤
  UI/UX Phase 2 (components) ── depends on Phase 0 ────┤
  Competitive P0-9 (Request human) ── depends on P0-1 ─┤
                                                        │
Week 3:                                                 │
  UI/UX Phase 3.1 (Landing) ──── depends on Phase 2 ──┤
  UI/UX Phase 3.2 (Dashboard) ── depends on Phase 2 ──┤
  UI/UX Phase 3.4 (Jobs) ─────── depends on Phase 2 ──┤
  UI/UX Phase 3.6 (CVs) ──────── depends on Phase 2 ──┤
  UI/UX Phase 3.9 (PublicLayout) depends on Phase 2 ──┤
                                                        │
Week 4:                                                 │
  UI/UX Phase 3.3 (Chat) ─────── depends on Phase 2 ──┤
    └── Absorbs: P0-2 mobile, P0-3 proctoring, P0-8 ──┤
  UI/UX Phase 3.5 (Candidates) ─ depends on Phase 2 ──┤
    └── Absorbs: P0-2 mobile ──────────────────────────┤
  UI/UX Phase 3.7 (InterviewResult) depends on Phase 2┤
    └── Absorbs: P0-6 override, P0-7 transcript ───────┤
  UI/UX Phase 3.8 (AppShell) ──── depends on Phase 2 ─┤
    └── Absorbs: P0-2 mobile nav ──────────────────────┤
                                                        │
Week 5:                                                 │
  UI/UX Phase 4 (cross-cutting) ─ depends on Phase 3 ─┤
  Competitive P1-1 (SOC 2) ────────────────────────────┤
  Competitive P1-2 (DOCX) ─────────────────────────────┤
  Competitive P1-3 (PDF export) ───────────────────────┤
  Competitive P1-4 (Encryption docs) ──────────────────┤
  Competitive P1-5 (Credits pricing) ──────────────────┤
                                                        │
Week 6:                                                 │
  UI/UX Phase 5 (validation) ──── depends on Phase 4 ─┘
```

---

## Execution Summary

| Week | Tasks | Total Effort | Target |
|------|-------|--------------|--------|
| **Week 1** | Phase 0 + P0-10 + P0-1 + P0-4 + P0-5 | ~18-23 hours | Foundation + backend |
| **Week 2** | Phase 1 + Phase 2 + P0-9 | ~10-14 hours | Design + components |
| **Week 3** | Phase 3.1 + 3.2 + 3.4 + 3.6 + 3.9 | ~23-29 hours | Pages without overlap |
| **Week 4** | Phase 3.3 + 3.5 + 3.7 + 3.8 | ~24-30 hours | Pages with competitive merge |
| **Week 5** | Phase 4 + P1-1 through P1-5 | ~18-25 hours | Cross-cutting + P1 |
| **Week 6** | Phase 5 validation | ~2-3 hours | Final validation |
| **Total** | 40+ tasks | ~95-124 hours | Full competitive + UI/UX resolution |

---

## Quick Win Sequence (First 48 Hours)

If you need to ship fast, do these first — highest impact, lowest effort:

1. **P0-10** (1 hour) — Update PRD title and positioning. Zero code, immediate clarity.
2. **Phase 0.4** (1 hour) — Add semantic design tokens. Foundation for all UI work.
3. **P0-3** (30min) — Hide proctoring from candidates. Remove UI elements.
4. **P0-7** (2-3 hours) — Show full transcript on scorecard. Add transcript section.
5. **P0-8** (1 hour) — Connection quality indicator. Add WebSocket status badge.
6. **P0-6** (3-4 hours) — Override AI recommendation. Add decision endpoint + UI.
7. **P1-5** (30 min) — Add credits pricing tier. Update docs.

**Total: ~8-10 hours for 7 tasks that address 6 of the top 10 competitive gaps + foundation for UI/UX revamp.**

---

## Success Metrics

| Metric | Before | Target |
|--------|--------|--------|
| Competitive P0 issues | 10 | 0 |
| Competitive P1 issues | 5 | 0 |
| UI/UX P1 issues | 8 | 0 |
| UI/UX P2 issues | 18 | 0 |
| UI/UX P3 issues | 14 | ≤5 |
| Lighthouse Accessibility | TBD | ≥90 |
| Lighthouse Performance | TBD | ≥85 |
| Bundle size impact | 0% | <15% increase |
| Page transition animations | 0 | All routes |
| Consistent empty states | 5 patterns | 1 component |
| `window.confirm()` in prod | 2 | 0 |
| Design tokens (semantic) | ~10 | 30+ |
