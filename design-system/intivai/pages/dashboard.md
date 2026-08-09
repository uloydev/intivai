# Dashboard Page Overrides

> **PROJECT:** Intivai
> **Page Type:** Dashboard / Data View (recruiter-facing)

> ⚠️ **IMPORTANT:** Rules in this file **override** the Master file (`design-system/MASTER.md`).
> Only deviations from the Master are documented here. For all other rules, refer to the Master.

---

## Page-Specific Rules

### Layout Overrides

- **Max Width:** 1400px, full-width layout (Master default 1200px is too narrow for pipeline tables)
- **Grid:** 12-column grid; candidate pipeline table spans 12, sidebar widgets span 4-8
- **Sections:** 1. Header (org switcher + plan + CTA), 2. KPI row (active interviews, pass rate, time-to-screen, LLM spend), 3. Candidate pipeline table (status: new → screening → passed → interview → hired/rejected), 4. Candidate detail drawer, 5. Interview/evaluation views, 6. Question bank & analytics
- **Status colors:** new=slate, screening=amber, passed=green, rejected=red, interview=blue — status must never rely on color alone (add status pill text)

### Spacing Overrides

- **Content Density:** High (8-32px scale, `--density 8`). Table rows 40-48px, KPI cards gap 16px
- Vertical rhythm tiers: 8/16/24/32 (dashboard) instead of Master's 16/24/32/48

### Typography Overrides

- KPI numbers: Space Grotesk 600 (Master heading font), tabular-nums (stable column alignment)
- Table + labels: DM Sans 400/500, 13-14px
- No overrides to font family — Master typography only (Space Grotesk + DM Sans)

### Color Overrides

- Surfaces stay light (`--color-background #F8FAFC`) — light mode is default for recruiter workflow
- Accent `--color-accent #059669` (Master) reserved for positive outcomes; focus ring = primary `#2563EB`; destructive `#DC2626` for reject/delete
- No dark-mode-by-default (Master avoid rule applies); dark theme = opt-in toggle only

### Component Overrides

- **Avoid:** wide tables breaking layout → horizontal scroll container or card layout under 768px
- **Avoid:** single-row actions only → row hover reveals actions, plus bulk multi-select (batch status change)
- **Avoid:** auto-play video/audio → interview recordings are click-to-play
- Async states: CV parsing / scoring / transcription are queued jobs → skeleton rows + progress pill (processing) instead of blank cells
- Data loading: spinners for filters/charts, hover tooltips on scores (show breakdown tooltip), chart zoom on click
- Export: CSV export for filtered candidate list (audit + offline review)

---

## Page-Specific Components

- **Score breakdown popover:** hover/press on CV score → weighted breakdown (skills/experience/semantic/education/certs) + weights used (audit traceability)
- **Interview list:** per-candidate chat/voice interviews with status, duration, evaluation recommendation (proceed/hold/reject)
- **Question bank manager:** category/difficulty filters, fail-rate column (from cross-interview reflect)

---

## Recommendations

- Effects: hover tooltips, row highlight on hover, smooth filter transitions, skeleton loading (no layout shift)
- Responsive: table → cards under 768px; filters collapsible on mobile
- Data Entry: multi-select + bulk edit; inline status dropdown per row
- CTA Placement: primary CTA in nav ("New Interview") + after KPI row ("Invite Candidate")
