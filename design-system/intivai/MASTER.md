# Design System Master File

> **LOGIC:** When building a specific page, first check `design-system/pages/[page-name].md`.
> If that file exists, its rules **override** this Master file.
> If not, strictly follow the rules below.

---

**Project:** Intivai — AI interview platform (SaaS, multi-tenant)
**Generated:** 2026-08-10 (revision 2 — corrected category, resolved contradictions)
**Category:** SaaS hiring/HR tool — corporate, trustworthy, accessible, professional
**Design Dials:** Variance 5/10 (Balanced) | Motion 2/10 (Subtle, app-grade) | Density 6/10 (Standard app; page overrides tune: dashboard 8, candidate chat 3)

---

## Global Rules

### Color Palette — Light

| Role | Hex | CSS Variable | Usage |
|------|-----|--------------|-------|
| Primary | `#2563EB` | `--color-primary` | Buttons, links, focus, active nav |
| On Primary | `#FFFFFF` | `--color-on-primary` | Text on primary |
| Accent | `#059669` | `--color-accent` | Positive actions, success, "passed" |
| On Accent | `#FFFFFF` | `--color-on-accent` | Text on accent |
| Background | `#F8FAFC` | `--color-background` | App background |
| Surface | `#FFFFFF` | `--color-surface` | Cards, modals, table rows |
| Foreground | `#0F172A` | `--color-foreground` | Primary text |
| Muted | `#64748B` | `--color-muted` | Secondary text, placeholders |
| Muted Surface | `#F1F5F9` | `--color-muted-surface` | Disabled bg, hover bg |
| Border | `#E2E8F0` | `--color-border` | All borders, dividers |
| Destructive | `#DC2626` | `--color-destructive` | Delete, reject, errors |
| Warning | `#D97706` | `--color-warning` | Time warnings, pending attention |
| Ring | `#2563EB` | `--color-ring` | Focus ring (same as primary) |

**Color Notes:** Primary blue = action/trust; accent green = positive outcomes only. Status colors MUST be paired with text labels (never color alone): `new` slate, `screening`/processing amber, `passed` green, `rejected` red, `interview` blue, `failed` red.

### Color Palette — Dark (opt-in toggle; light is default for recruiter workflow)

| Role | Hex | CSS Variable |
|------|-----|--------------|
| Primary | `#3B82F6` | `--color-primary` |
| Accent | `#10B981` | `--color-accent` |
| Background | `#0F172A` | `--color-background` |
| Surface | `#1E293B` | `--color-surface` |
| Foreground | `#F8FAFC` | `--color-foreground` |
| Muted | `#94A3B8` | `--color-muted` |
| Muted Surface | `#1E293B` | `--color-muted-surface` |
| Border | `#334155` | `--color-border` |
| Destructive | `#EF4444` | `--color-destructive` |
| Warning | `#F59E0B` | `--color-warning` |
| Ring | `#3B82F6` | `--color-ring` |

Dark mode: verify contrast independently (4.5:1 body) — do NOT inherit light values.

### Typography

- **Heading Font:** Space Grotesk — headings, KPI numbers, question text (600/700)
- **Body Font:** DM Sans — body, tables, labels, chat answers (400/500)
- **Mood:** corporate, trustworthy, readable, clean
- **Google Fonts:** [Space Grotesk + DM Sans](https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@500;600;700&family=DM+Sans:opsz,wght@9..40,400;9..40,500;9..40,600&display=swap)

**Font-size scale:**

| Token | Value | Usage |
|-------|-------|-------|
| `--text-xs` | 12px | Table cells, meta labels |
| `--text-sm` | 14px | Secondary UI, table headers |
| `--text-base` | 16px | Body, inputs |
| `--text-lg` | 18px | Card titles |
| `--text-xl` | 22px | Page titles, question text |
| `--text-2xl` | 28px | KPI numbers, hero numbers |
| `--text-3xl` | 36px | Empty states, splash |

**CSS Import:**
```css
@import url('https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@500;600;700&family=DM+Sans:opsz,wght@9..40,400;9..40,500;9..40,600&display=swap');
```

### Spacing Variables

| Token | Value | Usage |
|-------|-------|-------|
| `--space-xs` | 4px / 0.25rem | Tight gaps |
| `--space-sm` | 8px / 0.5rem | Icon gaps, inline |
| `--space-md` | 16px / 1rem | Component padding, form gaps |
| `--space-lg` | 24px / 1.5rem | Card padding, section gaps |
| `--space-xl` | 32px / 2rem | Section padding |
| `--space-2xl` | 48px / 3rem | Page padding, modals |
| `--space-3xl` | 64px / 4rem | Empty states, hero |

Density: page overrides MAY retune (dashboard 8px tiers, candidate chat 24px+). Rhythm must stay consistent within a page.

### Radius, Shadow, Z, Focus

| Token | Value |
|-------|-------|
| `--radius-sm` | 6px — inputs, tags |
| `--radius-md` | 8px — buttons, cards |
| `--radius-lg` | 12px — modals, dropdowns |
| `--radius-full` | 999px — pills, avatars |
| `--shadow-sm` | `0 1px 2px rgba(0,0,0,0.05)` |
| `--shadow-md` | `0 4px 6px rgba(0,0,0,0.1)` — cards |
| `--shadow-lg` | `0 10px 15px rgba(0,0,0,0.1)` — modals |
| `--z-sticky` | 100 |
| `--z-overlay` | 1000 |
| `--z-modal` | 1100 |
| `--z-toast` | 1200 |

Focus: `outline: none; box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-ring) 25%, transparent)` on all interactive elements; visible in BOTH themes.

---

## Icons

**Phosphor** (`@phosphor-icons/react`) — project convention (AGENTS.md). One style per hierarchy level (line for nav/actions, fill for active states). Sizes: `icon-sm` 16px, `icon-md` 24px (default), `icon-lg` 32px. **Never emojis as structural icons.** Stroke consistency within a surface.

---

## shadcn/ui Token Mapping (FE stack: React + Vite + Tailwind + shadcn)

| shadcn theme var | CSS variable | Light | Dark |
|------------------|--------------|-------|------|
| `--background` | `--color-background` | `#F8FAFC` | `#0F172A` |
| `--foreground` | `--color-foreground` | `#0F172A` | `#F8FAFC` |
| `--card` / `--popover` | `--color-surface` | `#FFFFFF` | `#1E293B` |
| `--card-foreground` | `--color-foreground` | `#0F172A` | `#F8FAFC` |
| `--primary` | `--color-primary` | `#2563EB` | `#3B82F6` |
| `--primary-foreground` | `--color-on-primary` | `#FFFFFF` | `#FFFFFF` |
| `--secondary` | `--color-muted-surface` | `#F1F5F9` | `#1E293B` |
| `--secondary-foreground` | `--color-foreground` | `#0F172A` | `#F8FAFC` |
| `--muted` | `--color-muted-surface` | `#F1F5F9` | `#1E293B` |
| `--muted-foreground` | `--color-muted` | `#64748B` | `#94A3B8` |
| `--accent` | `--color-accent` | `#059669` | `#10B981` |
| `--accent-foreground` | `--color-on-accent` | `#FFFFFF` | `#FFFFFF` |
| `--destructive` | `--color-destructive` | `#DC2626` | `#EF4444` |
| `--destructive-foreground` | `--color-on-primary` | `#FFFFFF` | `#FFFFFF` |
| `--border` | `--color-border` | `#E2E8F0` | `#334155` |
| `--input` | `--color-border` | `#E2E8F0` | `#334155` |
| `--ring` | `--color-ring` | `#2563EB` | `#3B82F6` |
| `--radius` | `--radius-md` | `0.5rem` | `0.5rem` |

Tailwind: extend `font-display: Space Grotesk`, `font-body: DM Sans`; semantic colors via CSS vars (dark mode = `.dark` class + `@apply` tokens, no per-screen hex).

---

## Component Specs

### Buttons

```css
.btn-primary { background: var(--color-primary); color: var(--color-on-primary);
  padding: 10px 20px; border-radius: var(--radius-md); font-weight: 600;
  transition: opacity 150ms ease, background 150ms ease; cursor: pointer; }
.btn-primary:hover { opacity: 0.9; }         /* no layout-shifting transforms */
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-secondary { background: transparent; color: var(--color-primary);
  border: 1px solid var(--color-border); border-radius: var(--radius-md);
  padding: 10px 20px; font-weight: 600; cursor: pointer; }
.btn-secondary:hover { background: var(--color-muted-surface); }
.btn-ghost { background: transparent; color: var(--color-foreground); padding: 10px 16px; cursor: pointer; }
.btn-destructive { background: var(--color-destructive); color: white; padding: 10px 20px; border-radius: var(--radius-md); cursor: pointer; }
```

Interaction: pressed feedback 80-150ms (opacity/bg, never scale); micro-interactions 150-300ms, exit faster than enter.

### Cards

```css
.card { background: var(--color-surface); border: 1px solid var(--color-border);
  border-radius: var(--radius-md); padding: var(--space-lg); box-shadow: var(--shadow-sm); }
```

### Inputs

```css
.input { padding: 10px 14px; border: 1px solid var(--color-border);
  border-radius: var(--radius-sm); font-size: var(--text-base); background: var(--color-surface);
  color: var(--color-foreground); transition: border-color 150ms ease; }
.input:focus { border-color: var(--color-primary); box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-ring) 25%, transparent); outline: none; }
.input:disabled { background: var(--color-muted-surface); cursor: not-allowed; }
.input-error { border-color: var(--color-destructive); }
```

### Modals

```css
.modal-overlay { background: rgba(15, 23, 42, 0.55); backdrop-filter: blur(4px); z-index: var(--z-modal); }
.modal { background: var(--color-surface); border-radius: var(--radius-lg); padding: var(--space-2xl);
  box-shadow: var(--shadow-lg); max-width: 520px; width: 92%; z-index: var(--z-modal); }
```

### Status Pills

```css
.pill { display: inline-flex; align-items: center; gap: 6px; padding: 2px 10px;
  border-radius: var(--radius-full); font-size: var(--text-xs); font-weight: 600; }
.pill-passed { background: color-mix(in srgb, var(--color-accent) 12%, transparent); color: var(--color-accent); }
.pill-rejected { background: color-mix(in srgb, var(--color-destructive) 12%, transparent); color: var(--color-destructive); }
.pill-processing { background: color-mix(in srgb, var(--color-warning) 14%, transparent); color: var(--color-warning); }
.pill-neutral { background: var(--color-muted-surface); color: var(--color-muted); }
/* pills always carry text — never color alone */
```

### Skeleton (async states)

```css
.skeleton { background: linear-gradient(90deg, var(--color-muted-surface) 25%, #E8EEF7 50%, var(--color-muted-surface) 75%);
  background-size: 200% 100%; animation: shimmer 1.4s infinite; border-radius: var(--radius-sm); }
@keyframes shimmer { to { background-position: -200% 0; } }
@media (prefers-reduced-motion: reduce) { .skeleton { animation: none; } }
```

### Empty / Error states

- Empty: centered, icon (Phosphor 48px, muted), title (`--text-xl`), one-line hint, primary CTA
- Error: inline message with `--color-destructive` text + retry action; never dead buttons
- Loading: skeletons (no layout shift) or progress pill; spinners only for filters/transforms

---

## Motion (app-grade)

- Micro-interactions 150-300ms, ease-out; exit 120-200ms (faster than enter)
- Reduced motion: `prefers-reduced-motion: reduce` → disable transforms/animations (fade opacity only or static)
- No scroll-reveal/choreography in app surfaces (GSAP marketing patterns are NOT for this product)

---

## Anti-Patterns (Do NOT Use)

- ❌ Emojis as icons — Phosphor SVG only
- ❌ Missing `cursor:pointer` on clickable elements
- ❌ Layout-shifting hovers (scale/translate that move neighbors)
- ❌ Low contrast: body ≥4.5:1, large UI glyphs ≥3:1, both themes
- ❌ Instant state changes — transitions 150-300ms
- ❌ Invisible focus states — ring on every interactive element
- ❌ Color-only status indicators — pair with text pills
- ❌ Hardcoded per-screen hex — semantic tokens only
- ❌ Mixed icon styles at one hierarchy level

---

## Pre-Delivery Checklist

- [ ] No emojis as icons; Phosphor only, consistent style per level
- [ ] `cursor-pointer` on all clickable elements
- [ ] Hover/press states 150-300ms, no layout shift
- [ ] Text contrast ≥4.5:1 (light AND dark verified independently)
- [ ] Focus rings visible (keyboard nav)
- [ ] `prefers-reduced-motion` respected
- [ ] Status never color-alone
- [ ] Responsive: 375px, 768px, 1024px, 1440px; no horizontal scroll on mobile
- [ ] No content behind fixed bars; touch targets ≥44px (48px for chat input)
- [ ] Semantic tokens only — no ad-hoc hex
