# Intivai — Design System & UI/UX Specification

> Brand direction, UX architecture, design tokens, and interaction rules for Intivai.
> Governed by `antislop` filter (rules R-01 through R-38).
>
> Dial: ENERGY 2 / RHYTHM 2 / MOTION 1

---

## 1. Product Identity & UX Philosophy

Intivai is an enterprise recruitment platform providing automated, AI-assisted video and audio interviews, CV scoring, and deep candidate evaluations.

### Core Persona
- **Tone:** Authoritative, objective, calm, and trustworthy.
- **Mood:** Structured enterprise software built for HR efficiency; eliminates recruiter fatigue without feeling sterile or intimidating to candidates.
- **HR-First Mindset:** Screens prioritize hiring decisions over flashy telemetry. Every layout element must answer: *"Does this make recruiter decision-making faster and fairer?"*

### Two Primary Workspaces
1. **Recruiter / Hiring Manager Portal:**
   - High information density, scannable scorecards, tabular figures.
   - Decision-first architecture: surface anomalies, flags, and score breakdowns immediately.
   - Asymmetric split-view: candidate roster on left, rich evaluation dossier on right.
2. **Candidate Interview Room:**
   - Empathic, distraction-free, low-anxiety interface.
   - High accessibility: crystal-clear system status (mic, camera, connection).
   - Explicit feedback: no ambiguous loading states or unconfirmed answer submissions.

---

## 2. Design Tokens

Derived from and synchronized with `frontend/src/index.css`.

### 2.1 Color Palette
Restrained 3-color core + 1 deliberate accent. No unanchored purple-blue gradients (R-01).

| Token | Light Value | Dark Value | Purpose / Usage |
|---|---|---|---|
| `--background` | `#F8FAFC` (Slate-50) | `#0F172A` (Slate-900) | Main viewport canvas |
| `--card` | `#FFFFFF` | `#1E293B` (Slate-800) | Surface panels, cards, dialogs |
| `--foreground` | `#0F172A` (Slate-900) | `#F8FAFC` (Slate-50) | High-contrast body text (meets WCAG AA) |
| `--primary` | `#1E3A5F` (Navy) | `#3B82F6` (Blue-500) | Primary branding, buttons, active nav |
| `--secondary` | `#2563EB` (Cobalt) | `#1E293B` (Slate-800) | Secondary actions, interactive badges |
| `--accent` | `#059669` (Emerald) | `#10B981` (Emerald-400)| **Single Deliberate Accent**: Passing scores, verified badges, positive hire signals |
| `--muted` | `#F1F3F5` | `#1E293B` | Neutral background fills, disabled states |
| `--muted-foreground` | `#64748B` | `#94A3B8` | Secondary labels, timestamps, metadata |
| `--border` | `#E4E7EB` | `#334155` | 1px crisp separation lines |

### 2.2 Semantic Status Colors
Strictly tied to domain state. Never used as decorative tinting (R-31).

- **Success / Passed:** `#059669` (Dark: `#10B981`) — candidate pass, completed interview.
- **Warning / Review Needed:** `#D97706` (Dark: `#F59E0B`) — flagged integrity check, borderline score.
- **Destructive / Rejected:** `#DC2626` (Dark: `#EF4444`) — candidate reject, failed pipeline, critical error.
- **Info / In Progress:** `#2563EB` (Dark: `#3B82F6`) — interview scheduled, processing analysis.

### 2.3 Typography Scale
- **Display & Headings:** `DM Sans`, fallback `Inter`, `sans-serif` (weight 600 or 700).
- **Body & Controls:** `Inter`, `sans-serif` (weight 400 for regular text, 500 for labels, 600 for semibold data).
- **Numeric & Scores:** `Inter` with CSS `font-variant-numeric: tabular-nums` (`tnum`) for aligned scorecard comparisons.
- **Code / Transcripts / Telemetry:** Monospace (`JetBrains Mono`, `ui-monospace`) used exclusively for sandbox code evaluation and timestamped audio transcripts.

### 2.4 Elevation & Radii
- **Base Radius:** `0.5rem` (8px). Inputs and cards use `--radius` (8px), buttons use `--radius` (8px). No pill-shaped cards or excessive rounded containers (R-11).
- **Shadow Dose Cap:** Flat by default. Single subtle shadow on floating modals and hovered cards (`0 1px 3px rgba(0,0,0,0.1)`). Zero heavy diffuse drop-shadows (R-12).
- **Glassmorphism:** Strictly prohibited across main surfaces. Only allowed on modal backdrops via subtle blur (`backdrop-blur-sm bg-slate-900/40`) (R-10).

---

## 3. UI Layout & Component Patterns

### 3.1 Recruiter Dashboard & Requisition Views
- **No Generic Stat Card Row:** Avoid 4 identical stat cards with fabricated percentages (R-17). Use actionable count metrics (e.g., `12 Pending Review`, `3 Live Today`) linked directly to filtered views.
- **Decision-First Table Layouts:**
  - Column 1: Candidate Name & Applied Role
  - Column 2: Overall Score & Recommendation Badge (Pass / Review / Fail)
  - Column 3: Integrity / Proctoring Indicator (Clean / Flagged)
  - Column 4: Interview Stage
  - Column 5: Key Actions (Review, Schedule, Export)
- **Scorecard Presentation:**
  - Multi-category radar or breakdown bar: Technical Competence, Communication, Problem Solving.
  - Scores always displayed with confidence range and direct transcript quote citations.

### 3.2 Candidate Interview Room
- **Layout:** Centered, distraction-free container with persistent peripheral status.
  - Top bar: Position title, question index (`Question 3 of 7`), overall time budget.
  - Main area: Video preview / audio waveform visualizer + prompt card.
  - Bottom controls: Mic toggle, camera toggle, "Submit Answer & Proceed" button.
- **Status Indicators:**
  - Recording indicator: solid red dot + label "Recording" when active; zero endless pulse when idle.
  - Network indicator: connection quality (Good / Fair / Reconnecting) with auto-reconnect fallback.

### 3.3 State Handling (R-27)
- **Empty States:** Never show a blank card or generic "No data found". Explain the reason and provide an immediate action:
  - *Example:* "No completed interviews for Senior Backend Engineer yet. Share the interview link with shortlisted candidates to begin screening." [Copy Link Button]
- **Loading States:** Skeleton screens matching exact target layout structure, or scoped spinners on buttons. Never block the entire screen with an opaque loading overlay.
- **Error States:** Actionable inline banners specifying the cause (network timeout, transcription failure) and a retry trigger.

---

## 4. Motion & Micro-Interactions (MOTION 1)

- **Purpose:** Purely utilitarian — state transitions and interactive feedback (R-19).
- **Duration Scale:**
  - Fast feedback (button hover, active tab switch): `150ms` (`ease-out`).
  - Container expansion (collapsible accordion, drawer slide): `250ms` (`cubic-bezier(0.4, 0, 0.2, 1)`).
- **Strict Bans:**
  - No infinite bouncing or pulsing icons.
  - No decorative background particle or gradient animations.
  - Respect `prefers-reduced-motion` across all transition utilities.

---

## 5. Accessibility & Contrast Gates (WCAG 2.1 AA)

- **Contrast Ratios:** Minimum 4.5:1 for body copy against cards/backgrounds; minimum 3:1 for large text and semantic indicators.
- **Keyboard Navigation:** Every actionable element (tables, rows, dropdowns, modal dismiss) must support `Tab`, `Enter`, `Escape`, and display a high-contrast focus ring (`outline: 2px solid var(--ring)`).
- **Screen Reader Support:** Semantic HTML tags (`<main>`, `<nav>`, `<aside>`, `<table>`, `<dialog>`). Live regions (`aria-live="polite"`) for incoming live transcription tokens.
