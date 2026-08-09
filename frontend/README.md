# Intivai Frontend

React SPA for the Intivai AI interview platform: recruiter workspace
(dashboard-lite) + candidate interview flow.

Stack: React 18 + Vite + TypeScript (strict) + Tailwind CSS v4 + shadcn/ui +
Phosphor icons + TanStack Query v5 + react-router v6. Tests: Vitest (unit) +
Playwright (E2E).

## Commands

```bash
npm ci
npm run dev        # http://localhost:5173 — proxies /api (+WS) to :8081
npm run build      # tsc + vite build → dist/
npm run test       # vitest run (lib/api, lib/ws)
npm run e2e        # playwright test (needs stack + INTIVAI_DEEPSEEK_API_KEY)
```

## Design tokens

`src/index.css` holds the shadcn theme vars mapped from
`../design-system/intivai/MASTER.md` (rev 2): light + dark palettes, fonts
(Space Grotesk display / DM Sans body), radius. Tailwind exposes them as
`bg-background`, `text-foreground`, `bg-primary`, `border-border`, etc.
Page-specific rules: `design-system/intivai/pages/` (candidate-interview,
dashboard). Dark mode is an opt-in toggle (recruiter shell only).

## Structure

```
src/
  lib/api.ts         fetch wrapper — bearer token, typed ApiError{code}, 401→logout,
                     15s timeout, FormData support
  lib/auth.ts        JWT session (padded base64url decode + exp check), login/logout
  lib/ws.ts          candidate chat WS client — typed frames, single-close notify,
                     isOpen/send-result, reconnect managed by the page
  lib/theme.tsx      dark mode provider (.dark class, persisted)
  pages/             Login, Register, Jobs, CVs, Candidates, Interviews,
                     InterviewResult, Invite, Chat
  types/api.ts       DTO types mirroring ../api/openapi.yaml (snake_case)
  e2e/               Playwright: full candidate journey (step-logged), smoke
```

## Key contracts

- **Auth**: recruiters log in with org slug + email + password → JWT in
  localStorage. Candidates never have accounts — they use the invite link
  (`/invite/:id?t=<invitation_token>`), consent, exchange for a 10-min WS
  ticket, then chat.
- **Chat protocol**: frames from `api/openapi.yaml` (`interview.start`,
  `question`, `token`, `response`, `evaluation{status: complete|pending}`,
  `error{code,message}`). Ticket travels as `?ticket=` (browsers cannot set
  WS headers). Server pings are auto-ponged by the browser.
- **Backend 401** on any authed call → token cleared + redirect to `/login`.

## Playwright E2E

`e2e/happy-path.spec.ts` walks the full journey against the live stack:
register → job → CV upload → DeepSeek extraction (polled, logged) → passed →
interview → invite → consent → WS chat → streamed reply. Requires the dev
stack (`cd backend && make dev`), the DeepSeek key in the stack, and
`/tmp/kilo/cv.pdf` (any text PDF). Runs are step-logged to stdout for
long-run observability.
