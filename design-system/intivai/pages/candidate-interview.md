# Candidate Interview Page Overrides

> **PROJECT:** Intivai
> **Page Type:** Focused task flow (candidate-facing, no account)

> ⚠️ **IMPORTANT:** Rules in this file **override** the Master file (`design-system/MASTER.md`).
> Only deviations from the Master are documented here. For all other rules, refer to the Master.

---

## Page-Specific Rules

### Layout Overrides

- **Max Width:** 720px content column (reading measure for chat); centered, single column
- **No product chrome:** no nav, no sidebar, no dashboard links — candidate opens via invitation token
- **Sections:** 1. Welcome (role title, question count, duration, consent notice), 2. Chat transcript, 3. Input (textarea + send), 4. Status bar (question idx, time remaining), 5. Completion → thank-you + what-happens-next
- Voice interview: recording state indicator + consent dialog (must be accepted before recording starts)

### Spacing Overrides

- **Content Density:** Low/spacious (24-96px scale, `--density 3`) — calm, non-pressure interview feel
- Chat message spacing: 16px between bubbles; input area 24px above safe bottom

### Typography Overrides

- Questions: Space Grotesk 600, 20-24px (distinct from answer text)
- Answers + messages: DM Sans 400, 16-17px (readability over density)
- No font family override — Master typography only

### Color Overrides

- **Trust strategy:** calm surfaces, no bright gradients. Background `#F8FAFC`; AI bubbles muted `#E8ECF1`; candidate bubbles accent-tinted with white text
- Destructive only for "end interview" + error states; amber for time warnings
- Live/evaluating status: pulsing dot (subtle, respects reduced-motion)

### Component Overrides

- **Avoid:** auto-scroll jank → smooth scroll with `behavior: smooth`; respect `prefers-reduced-motion` (jump-to-end instead)
- **Avoid:** fade transitions for content swap — use minimal opacity 80-120ms (Master spring set is for marketing surfaces; interview flow stays calm)
- Input: textarea auto-resize, Enter=send, Shift+Enter=newline; disabled state while AI streams
- Streaming indicator: small typing/streaming dots, not skeleton blocks
- Reconnect: WS drop banner (top, amber) + auto-resume from last question idx; never loses transcript
- Expired session: clear screen → "session expired" with recruiter contact, no dead buttons
- Recording (voice): animated level meter + timer + pause/stop; consent dialog first
- Touch targets ≥44px (mobile candidates); input 48px minimum

---

## Page-Specific Components

- **Question progress:** "Question 2 of 5" + time remaining — always visible, subtle (not a countdown timer alarm)
- **Consent dialog:** before any recording — text-only opt-out link
- **Transcript export notice:** candidate can download their own transcript after completion (GDPR access right)

---

## Recommendations

- Effects: streaming indicator dots; zero decorative animation; reduced-motion = static UI
- CTA Placement: single primary "Start Interview" / "Send"; secondary "End early"
- Accessibility: screen-reader-announce new question + AI message; focus lands on textarea after each question
