# Intivai Pricing

> Status: current · Last-reviewed: 2026-08-24 · Owner: EM · **Single source of truth** — other docs link here.
> Decision 2026-08-24: adopts the competitive-review adjustment ($29 Starter,
> Free tier at 10 interviews/mo). Supersedes the $49/3-interview table in
> `design-decisions.md` §6 (kept there as historical record only).
> Billing enforcement is out of code scope for beta (manual invoicing).

## Tiers

| Tier | Price | Limits | Target |
|------|-------|--------|--------|
| **Free** | $0 | 10 interviews/mo, chat only | Trial evaluation |
| **Starter** | $29/mo | 100 interviews/mo | Small teams |
| **Pro** | $199/mo | 1,000 interviews/mo | Growing companies |
| **Interview Credits** | $99 one-time | 500 credits (1 credit = 1 interview); credits expire after 12 months | Volume BPO without subscription |
| **Volume** | $0.50/interview ($0.20/credit at 1,000+ credits) | Usage-based | High-volume users |
| **Enterprise** | Custom | Self-hosted option, custom integrations | Data-sensitive orgs |

## Rationale

- Volume/usage pricing aligns with the beachhead BPO segment and undercuts flat-rate competitors at scale.
- $29 Starter lowers the trial barrier against full-ATS incumbents priced per seat; value story = transparent scoring + live conversational interviews + coding sandbox, not feature parity with an ATS.
- Credits give agencies a predictable budget line without a recurring commitment.

## Open questions (resolve with pilot data)

- Overage handling on Starter/Pro (hard stop vs metered).
- Whether voice minutes (post-ADR-0006) price separately or draw credits.
