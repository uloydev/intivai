# Intivai — UI/UX Remediation Plan (2026-08-19)

Covers: the reported **empty-tracker-after-apply issue** (investigated, with evidence) + the full **ui-ux-pro-max HR-lens review** (3 parallel audits: recruiter flows, candidate journey, system consistency).

Severity: 🔴 blocking · 🟡 important · 🟢 polish. Effort: S ≤0.5d · M ≤1d · L 2d+.

---

## Issue 01 — Candidate Portal tracker empty after public apply (REPRODUCED ANALYSIS)

**Report**: applicant applies via the public careers board, follows "Track in Candidate Portal", and the tracker is empty — even after refresh, and after applying to multiple jobs.

**Investigation (2026-08-19)**:
- ✅ **Backend verified healthy**: live repro (apply → OTP → verify → `GET /candidate/portal/applications`) returned the application immediately; `candidate_applications_lookup` returns all rows (tested with 4 real applications).
- ✅ No FE filtering of applications exists; loading/error/empty states are distinct.
- 🔴 **Root cause A — stale portal identity**: the dashboard step is gated only by `localStorage.intivai_candidate_token` (CandidatePortal.tsx:52-56). A browser that ever logged in as *another* email opens the portal straight to that old account's dashboard (old `intivai_candidate_email` in the query key, old JWT claims) — the just-applied applicant sees the **old account's** (possibly empty) list. Refresh repeats it. Applying to more jobs as the new identity never shows up. Exactly matches "empty even after refresh and multiple applications applied".
- 🔴 **Root cause B — login-wall handoff**: the apply success modal's "Track in Candidate Portal" (Careers.tsx:542) links to `/candidate/portal` with no session — the applicant hits an OTP login wall; if the OTP email stalls, they never see a tracker at all.

**Fixes**:
1. **Apply returns a portal magic token** (`portal_token` in the apply response — backend: create a candidate_otps magic-token row at apply time, no email needed) and the success modal navigates to `/candidate/portal?token=<portal_token>` — the existing `magicVerify` exchange (CandidatePortal.tsx:66-89) then lands the applicant **directly on their own dashboard**.
2. **Clear stale identity before exchange**: on apply success the FE drops any previous `intivai_candidate_token`/`intivai_candidate_email` (no silent cross-account dashboard).
3. **Trust the backend claims over localStorage**: the applications query key should use the email from the verified token response (already stored on login), and the portal should bounce to the email step when the token's email ≠ the stored email.
4. **Empty-state honesty** (CandidatePortal.tsx:338-345): when empty, add "Applications may take a few minutes to appear while your profile is processed" + one auto-refetch after 10s; never imply the email has no applications while a just-completed apply exists.
5. **Regression test**: integration — apply → verify with portal_token → list contains the applied job (extend the portal integration spec).

---

## Phase 1 — Fabricated confidence (trust = the product) 🔴

| # | Finding | Fix | Files | Effort |
|---|---|---|---|---|
| 1.1 | `RecommendationBadge.tsx:14` — missing/null recommendation renders green **PROCEED** | Render neutral "No evaluation" when recommendation is absent; never default to a positive verdict | RecommendationBadge.tsx, Candidate360Drawer.tsx:321 | S |
| 1.2 | `Dashboard.tsx:64` — "Strong Hire Recommendations" invented from `recommendation=="proceed" \|\| score>=75` | Show the AI recommendation only, or label the heuristic + threshold inline | Dashboard.tsx | S |
| 1.3 | `InterviewVoice.tsx:82,132-135` — voice page reads the **recruiter** token; WS never authenticates; `onerror` fakes "interview in progress" with mic streaming | Fail closed: stop mic, honest "couldn't connect — retry"; remove the voice page from candidate reach until it authenticates for real (D6 deferral) | InterviewVoice.tsx, App.tsx route guard | M |
| 1.4 | `Jobs.tsx:67-85` — Assessment Stage Pipeline toggles collected but never sent | Persist the stage config (backend `CreateJobCommand` + repo columns) or remove the dead controls — pick one, no silent theater | Jobs.tsx, job domain/handler | M |
| 1.5 | `Chat.tsx:184-188` — after 5 failed reconnects input stays enabled; answers append but are never delivered | Persistent "disconnected — answers are not being recorded" state; disable input; offer refresh/resume | Chat.tsx, useChatSession.ts | M |
| 1.6 | `Chat.tsx:116` — auto-submit writes third-person "Candidate did not submit…" into the candidate's transcript | First-person copy: "My time for this question ran out — nothing was submitted." | Chat.tsx | S |
| 1.7 | `ProctoringCard.tsx:16` — no telemetry shows red "Integrity: 0/100" + green "low Risk" | Show "No telemetry available" in place of both signals | ProctoringCard.tsx | S |
| 1.8 | `Candidate360Drawer.tsx:109` — invite link built without `?t=` when only `interview_id` exists | Re-issue a token or label the link "expired — regenerate" | Candidate360Drawer.tsx | S |

## Phase 2 — Candidate journey completion 🔴

| # | Finding | Fix | Files | Effort |
|---|---|---|---|---|
| 2.1 | `Chat.tsx:334-349` — post-interview dead end: raw "Score: 34/100", no next step (result page is recruiter-only) | Portal CTA + "the hiring team will contact you within N business days"; contextualize the score or defer it | Chat.tsx, InterviewResult (candidate view) | M |
| 2.2 | `TimerGate.tsx:60,155-162` — session budget hits 00:00 with no escalation; screen idles at zero | Warn at thresholds (e.g., 2m/30s), auto-end with explicit state + next-step CTA | TimerGate.tsx, Chat.tsx | M |
| 2.3 | `Invite.tsx:37-40` — expired invite only toasts; button re-enables with no reason | Persistent state: "this invitation has expired / already been used" + how to get a fresh one | Invite.tsx | S |
| 2.4 | `Chat.tsx:214` — sandbox squeezes chat to ~170px on mobile | Full-screen editor with back toggle below `sm` breakpoint | Chat.tsx, CodingSandbox.tsx | M |
| 2.5 | `Chat.tsx:141-143` — green "Real-Time AI Session" pulse stays green while disconnected | Drive the dot from the actual connection state | Chat.tsx, useChatSession.ts | S |
| 2.6 | `Invite.tsx:71,96` — "keyboard or dictation" promises dictation; consent lists "audio anomalies" for a text-only chat | Say "keyboard"; list only telemetry actually collected | Invite.tsx | S |
| 2.7 | `InterviewVoice.tsx:137-142` — socket close shows "Call completed" with no outcome/next action | Explicit outcome + portal redirect, or state clearly nothing was recorded | InterviewVoice.tsx | S |
| 2.8 | `CandidatePortal.tsx` — raw `applied/under_review` enums; static "10 minutes"; decimal scores; refresh blanks the list | Human labels ("Application received", "Under review"); live OTP countdown + "code expired" state; integer scores; keep list mounted with inline refresh indicator | CandidatePortal.tsx | M |
| 2.9 | `Careers.tsx:491` — PDF-only upload, .docx fails only at submit | Accept common formats or pre-validate with a clear hint; salary currency symbol fix (always "$") | Careers.tsx | S |
| 2.10 | `Careers.tsx:155` — "email updates as your application progresses" over-promises | Promise only what's sent: "we'll email you with each outcome" | Careers.tsx | S |
| 2.11 | `Chat.tsx:391-392` — textarea never refocuses on new question | autoFocus on question change | Chat.tsx | S |

## Phase 3 — Recruiter decision support 🟡

| # | Finding | Fix | Files | Effort |
|---|---|---|---|---|
| 3.1 | `InterviewResult.tsx:261` — "Reject Candidate" fires on one click (terminal) | Confirm step ("Reject Jane for X? This closes the application") | InterviewResult.tsx | S |
| 3.2 | `InterviewResult.tsx:162` — scorecard explains method, not meaning | Show rubric + dimension weights + how overall is composed; tie verdict to thresholds | InterviewResult.tsx | M |
| 3.3 | `ProctoringCard.tsx:27` — thresholds 85/60 unexplained; no human-review path | Document thresholds; add "Escalate for manual review" action | ProctoringCard.tsx, backend flag field | M |
| 3.4 | `Dashboard.tsx:303,62` — "Score: 87" no scale; Pass Rate mixes per-job thresholds, shows 0% pre-data | "87% Match" + threshold; per-job scope or label | Dashboard.tsx | S |
| 3.5 | `PipelineFunnel.tsx:67` — funnel links are dead-ends / never match counts | Wire each stage to the exact filtered view that produced its count | PipelineFunnel.tsx, Interviews.tsx (?status= param) | M |
| 3.6 | `Interviews.tsx:89` — "New Interview Session" disabled with no reason | "No candidates passed screening yet" reason text | Interviews.tsx | S |
| 3.7 | `Candidates.tsx:271` — no pagination/sorting | Sortable columns + pagination (server or client) | Candidates.tsx | M |
| 3.8 | `CVs.tsx:38` — raw extraction statuses as unexplained pills | "Profile ready / Processing / Needs attention" | CVs.tsx | S |
| 3.9 | `Candidate360Drawer.tsx:267` — dimension %s with no rationale/weights | Per-dimension explanation + weighting under "AI Screening Recommendation" | Candidate360Drawer.tsx | S |
| 3.10 | `Dashboard.tsx:318,354` — "Invite →" goes to generic list; two mode launchers, one list | Deep-link with preselected candidate; unify the two launchers | Dashboard.tsx, Interviews.tsx | S |
| 3.11 | `Jobs.tsx:239` — active/archived + Published/Internal pills unexplained | Explain the interaction or merge into one visibility state | Jobs.tsx | S |
| 3.12 | `CompanyContext.tsx:67,231,316` — no dirty guard; persona cards clobber edits silently; contexts cannot be removed | Dirty-state confirm-before-leave; persona buttons with confirm; delete/retire context | CompanyContext.tsx, backend delete endpoint | M |
| 3.13 | `Login.tsx:54,93` — demo autofill ships in prod; no password recovery | Gate demo box behind env flag; add "Forgot password?" | Login.tsx, backend reset flow | M |

## Phase 4 — System consistency & light theme 🟡

| # | Finding | Fix | Files | Effort |
|---|---|---|---|---|
| 4.1 | Light theme broken where statuses live: `*-400`/`-950` pills ≈1.6–2.7:1; theme defaults light with no OS detection | One token pass: `*-600 dark:*-400` everywhere; respect `prefers-color-scheme` on first visit | Careers.tsx, stages.ts, TimerGate.tsx, InterviewResult.tsx, Dashboard.tsx, theme.tsx | M |
| 4.2 | `Candidate360Drawer.tsx:118-123` — drawer: no focus trap, no Escape, no dialog semantics | Wrap in Radix `Dialog.Root` (focus trap + Escape + aria) | Candidate360Drawer.tsx | M |
| 4.3 | `TestCaseManager.tsx:53` — button inside button; `tr role="button"` with nested real button | Real tab semantics; drop row role or move action out | TestCaseManager.tsx, Candidates.tsx | S |
| 4.4 | `index.html:18` — `<title>frontend</title>` ships | "Intivai — AI Interview Platform" + meta description + OG | index.html | S |
| 4.5 | Icon families split 3 ways; both libs in one 206KB chunk | One family per surface (Phosphor pages, lucide primitives) or unify; remove emoji/glyph icons (`✦ ⚠ 🎉`); split chunks | vite.config.ts + all pages | M |
| 4.6 | ~97× `text-[10px]/[11px]` micro-text; pulse/ping/bounce with no reduced-motion guard | Consolidate to `text-xs` min; `motion-reduce:animate-none` + drop pulses on minute-long states | Global sweep | M |
| 4.7 | Token discipline: gradient button hardcodes hex; `--color-warning`/`--color-accent` unused; statuses bypass tokens | Use the defined tokens; audit `from-[#2563EB]` etc. | button.tsx, index.css, badges | S |
| 4.8 | Dead deps: `next-themes`, `@fontsource-variable/geist` never imported; themed `sonner.tsx` never mounted | Remove or wire; mount the themed Toaster | package.json, App.tsx | S |
| 4.9 | Global Suspense "Loading..." text on route swaps | Route-level skeletons | App.tsx + pages | S |
| 4.10 | Raw native selects/checkboxes beside Radix; `bg-neutral-750` dead class | Swap to ui components; fix dead classes | Candidates, CVs, Jobs, CodeEditor | S |
| 4.11 | Render-blocking Google Fonts @import, no preconnect | `preconnect` + non-blocking font loading | index.css, index.html | S |

## Phase 5 — Accessibility & interaction polish 🟢

| # | Finding | Fix | Files | Effort |
|---|---|---|---|---|
| 5.1 | Chat `aria-live="polite"` re-announces the whole transcript per token | Announce state changes only; `aria-busy` on streaming bubbles | Chat.tsx | S |
| 5.2 | Grace banner lacks `role="alert"`; login error lacks `role="alert"` | Add roles | TimerGate.tsx, Login.tsx | S |
| 5.3 | Icon-only buttons missing aria-labels (close/copy/delete) | Add aria-labels | Candidate360Drawer, Interviews, CVs | S |
| 5.4 | Segmented tabs are plain buttons, color-only active state | `role="tablist"/"tab"` + `aria-selected` + focus ring | Jobs, CompanyContext, Drawer, CVs, Interviews | M |
| 5.5 | Mobile bottom-nav ~40px hit areas | ≥44px | AppShell.tsx | S |
| 5.6 | PublicLayout mobile drawer: no aria-expanded/controls/Escape/focus | Disclosure semantics + focus return | PublicLayout.tsx | S |
| 5.7 | InterviewResult `text-[9px]` caption | ≥11px | InterviewResult.tsx | S |
| 5.8 | CVs single-upload forces name/email re-entry (OCR extracts them) | Make fields optional; correct after extraction | CVs.tsx | S |
| 5.9 | Export PDF double-click queues duplicates | Disable while generating | InterviewResult.tsx | S |
| 5.10 | Register: no confirm-password, slug discovered after submit | Confirm-password + inline slug validation | Register.tsx | S |

## Phase 6 — Verification & gates

1. **Gate per phase**: `make check` (0 lint, FE build, vitest 18/18) + `make test-integration-dev` for backend-touching items (1.4, 3.3, 3.12, 3.13, Issue-01)
2. **UI acceptance**: test on 375px; dark + light themes independently; keyboard-only pass over drawer/portal/interview; `prefers-reduced-motion` on
3. **Issue-01 regression**: apply → portal_token → dashboard shows the applied job immediately; second identity in the same browser does not leak into the first
4. **Candidate journey walk**: apply → confirm → invite → interview → result (portal CTA) — no dead ends, no fabricated states
5. **Re-run the ui-ux-pro-max pre-delivery checklist** (§1–§3 + light/dark + 44px targets) at the end

**Estimate**: ~10–12 focused sprint days. Order matters: Phase 1 (fabricated trust) and Issue-01 first — they are the two places a candidate or recruiter can be actively misled; Phases 2–3 complete the journeys; 4–5 are the consistency/a11y sweep.
