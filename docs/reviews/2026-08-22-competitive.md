# Intivai PRD — Deep Competitive Review

**Date:** 2026-08-22
**Competitors:** Manatal.com, Hireflix.com, Interviewer.ai
**Scope:** Feature parity, pricing, positioning gaps, and strategic risks

---

## 1. Executive Verdict

| Dimension | Intivai Strength | Critical Gap |
|---|---|---|
| **Live conversational AI interview** | ✅ Real-time streaming chat + voice | Competitors lack this — strongest moat |
| **CV→Score→Interview all-in-one** | ✅ End-to-end pipeline | Manatal has ATS + AI Interviewer add-on; Intivai has no ATS |
| **Technical coding sandbox** | ✅ Monaco IDE + sandboxed execution | **No competitor has this — killer differentiator for tech hiring** |
| **Proctoring & anti-cheat** | ✅ Advisory telemetry + verbatim quotes | Hireflix has zero proctoring; Manatal has none; Interviewer.ai has none |
| **Company context / tenant prompt** | ✅ Customizable interviewer persona | Interviewer.ai has structured interviews but not context-versioned prompts |
| **Pricing** | ⚠️ $49-199/mo estimated | Manatal at $15/user/mo; Hireflix at $75/mo; Interviewer.ai custom (est. $100+/mo) |
| **ATS features** | ❌ **Not an ATS** | Manatal is a full ATS + CRM — Intivai is interview-only |
| **Sourcing / Job boards** | ❌ None | Manatal has 2,500+ job boards; Intivai requires external sourcing |
| **Integrations ecosystem** | ❌ None | Manatal: Zapier, LinkedIn, Indeed; Hireflix: 50+ ATS integrations |
| **Mobile app** | ❌ None | Manatal has mobile app; Hireflix mobile-responsive |
| **SOC 2 compliance** | ⚠️ Planned P6b | Hireflix is SOC 2 certified; Manatal SOC 2 Type II certified |
| **Self-hosting / on-prem** | ✅ Architecture supports it | **No competitor offers this — strong enterprise differentiator** |
| **Global Talent Passport** | ✅ Portable verified credentials | **No competitor has this — novel concept** |
| **Pricing model flexibility** | ✅ Usage-based for volume BPO | Hireflix: flat rate; Manatal: per-user; Interviewer.ai: enterprise only |

---

## 2. Feature-by-Feature Comparison

### 2.1 Interview Delivery

| Feature | Intivai | Manatal AI Interviewer | Hireflix | Interviewer.ai |
|---|---|---|---|---|
| **Live chat interview** | ✅ Real-time WS streaming | ✅ Video (async) | ❌ One-way video (async) | ✅ Async video |
| **Live voice interview** | ✅ WebRTC (post-MVP) | ❌ Not offered | ❌ Not offered | ❌ Not offered |
| **Conversational AI (two-way)** | ✅ Dynamic follow-ups, probe on weakness | ❌ Pre-set questions | ❌ Pre-set questions | ⚠️ "Avatar" interviews (new product, unclear depth) |
| **Real-time interrupt** | ✅ Candidate can interrupt AI | ❌ | ❌ | ❌ |
| **CV-gap targeting** | ✅ Questions from JD↔CV gaps | ⚠️ Customizable but static questions | ❌ Static questions | ⚠️ Structured but not CV-adaptive |
| **Streaming tokens** | ✅ Token-by-token display | ❌ | ❌ | ❌ |
| **Multi-language** | ⚠️ LLM-dependent | ✅ EN/FR/DE/ES/IT/PT | ✅ 10+ languages | ✅ Multiple |
| **Reconnection / resume** | ✅ Session resume from last question | ❌ | ❌ | ❌ |

**Assessment:** Intivai's live conversational AI is a **category-defining advantage**. Hireflix is purely async one-shot. Manatal's new AI Interviewer is video-based but appears to be structured/static, not conversational. Interviewer.ai's "Avatar" is brand new and unproven. Intivai owns this space.

### 2.2 Technical Assessment

| Feature | Intivai | Manatal | Hireflix | Interviewer.ai |
|---|---|---|---|---|
| **Live coding sandbox** | ✅ Monaco IDE + Docker VMs (Go/Python/TS/Rust) | ❌ | ❌ | ❌ |
| **Code execution** | ✅ Sandboxed, mTLS gRPC | ❌ | ❌ | ❌ |
| **Multi-language support** | ✅ 4 languages | ❌ | ❌ | ❌ |
| **Proctoring telemetry** | ✅ Tab switch, paste ratio, away time | ❌ | ❌ | ❌ |
| **Anti-cheat advisory** | ✅ Tamper-evident audit trail | ❌ | ❌ | ❌ |
| **Verbatim quote grounding** | ✅ Every score cites candidate quotes | ❌ | ❌ | ❌ |

**Assessment:** This is Intivai's **strongest differentiator for tech hiring**. No competitor offers any form of live coding assessment. For engineering roles, this alone justifies choosing Intivai over alternatives.

### 2.3 CV & Screening

| Feature | Intivai | Manatal | Hireflix | Interviewer.ai |
|---|---|---|---|---|
| **CV upload** | ✅ PDF (OCR fallback) | ✅ PDF, DOCX, image | ❌ (ATS-dependent) | ✅ PDF, DOCX |
| **Bulk upload** | ✅ 50 concurrent | ✅ Mass upload + email forwarding | ❌ | ✅ Bulk |
| **AI extraction** | ✅ LLM structured output | ✅ AI enrichment from 20+ platforms | ❌ | ✅ AI parsing |
| **Semantic matching** | ✅ pgvector cosine similarity | ✅ AI Recommendations + scoring | ❌ | ✅ AI matching |
| **Scoring engine** | ✅ 5-dimension weighted (configurable) | ⚠️ AI scoring (opaque) | ❌ | ⚠️ AI scoring (opaque) |
| **Per-tenant weights** | ✅ org→job→global fallback | ❌ Not configurable | ❌ | ❌ |
| **Scanned PDF / OCR** | ✅ pdftoppm + Tesseract | ✅ | ❌ | ✅ |

**Assessment:** Intivai's scoring engine is **more transparent and configurable** than competitors. Manatal and Interviewer.ai use opaque AI scoring. Intivai's per-tenant weight configuration is unique and valuable for enterprises with different evaluation criteria.

### 2.4 Candidate Experience

| Feature | Intivai | Manatal | Hireflix | Interviewer.ai |
|---|---|---|---|---|
| **Public career board** | ✅ `/careers` with search | ✅ Branded career page | ❌ (ATS-dependent) | ❌ |
| **Passwordless access** | ✅ OTP + magic link | ❌ Account required | ✅ No login needed | ✅ No login |
| **Application stepper** | ✅ Visual stage tracking | ✅ Kanban pipeline | ❌ | ❌ |
| **Mobile responsive** | ⚠️ Not explicitly tested | ✅ Mobile app | ✅ All devices | ✅ |
| **Interview consent capture** | ✅ `consent_given` gate | ✅ | ⚠️ Implicit | ⚠️ Unclear |
| **Global Talent Passport** | ✅ Portable credentials | ❌ | ❌ | ❌ |
| **Candidate self-review** | ✅ Verify extracted data | ❌ | ❌ | ❌ |

**Assessment:** Intivai's candidate portal is well-designed but lacks mobile app support. Hireflix wins on candidate friction (zero login). The Global Talent Passport is a **novel innovation** with no market precedent — potential viral growth mechanism.

### 2.5 Recruiter Dashboard & Analytics

| Feature | Intivai | Manatal | Hireflix | Interviewer.ai |
|---|---|---|---|---|
| **Dashboard** | ✅ KPI summary + funnel | ✅ Full dashboard + reports | ❌ Minimal | ✅ Analytics |
| **Pipeline visualization** | ✅ Stage stepper + conversion | ✅ Kanban drag-and-drop | ❌ | ✅ |
| **ROI calculator** | ✅ Interactive model | ✅ ROI calculator on website | ❌ | ❌ |
| **Scorecard view** | ✅ 360° with verbatim quotes | ⚠️ Basic candidate profiles | ✅ Video review + comments | ✅ AI-generated reports |
| **Team collaboration** | ⚠️ Basic (recruiter/hiring mgr) | ✅ Messaging, activities, comments | ✅ Share via link | ✅ |
| **Report export** | ⚠️ JSON (PDF planned P4b) | ✅ Full reporting suite + custom builder | ✅ CSV export | ✅ PDF |
| **Email notifications** | ✅ Async worker | ✅ Mass email + templates | ✅ Automated messages | ✅ |

**Assessment:** Intivai's dashboard is functional but **significantly behind Manatal** in reporting depth. Manatal has a full reporting suite with custom builders. Intivai's PDF export is deferred (P4b). This is a known gap.

### 2.6 Platform & Integrations

| Feature | Intivai | Manatal | Hireflix | Interviewer.ai |
|---|---|---|---|---|
| **ATS integration** | ❌ None | ✅ Native ATS + CRM | ✅ 50+ ATS integrations | ✅ ATS integrations |
| **Job board posting** | ❌ None | ✅ 2,500+ boards | ❌ | ❌ |
| **Zapier/Make** | ❌ None | ✅ Zapier + n8n | ✅ Zapier + Make | ✅ |
| **Open API** | ❌ Not yet | ✅ Full REST API | ✅ GraphQL API | ✅ |
| **SSO/SAML** | ❌ Planned (post-MVP) | ✅ Enterprise Plus | ✅ SSO | ✅ |
| **LinkedIn integration** | ❌ None | ✅ Chrome extension + sourcing | ❌ | ❌ |
| **Email/Calendar sync** | ❌ None | ✅ G Suite + Outlook | ❌ | ✅ |

**Assessment:** **Intivai's integration story is its biggest weakness.** Manatal has 2,500+ job boards, Zapier, LinkedIn, G Suite, Outlook. Hireflix has 50+ ATS integrations. Intivai has zero. Enterprise buyers will reject Intivai on integration alone.

### 2.7 Security & Compliance

| Feature | Intivai | Manatal | Hireflix | Interviewer.ai |
|---|---|---|---|---|
| **GDPR** | ✅ Consent + deletion API | ✅ Full GDPR/CCPA/PDPA | ✅ GDPR | ✅ GDPR |
| **SOC 2** | ⚠️ Planned P6b | ✅ Type II certified | ✅ Certified | ⚠️ Unclear |
| **RLS / tenant isolation** | ✅ PostgreSQL RLS + FORCE | ✅ | ✅ | ✅ |
| **Encryption at rest** | ⚠️ Not explicitly stated | ✅ | ✅ | ✅ |
| **Audit logging** | ✅ Audit logs table | ✅ Full audit trail | ✅ Audit logs | ✅ |
| **Data residency** | ✅ Self-hosted option | ✅ | ✅ | ✅ |
| **Prompt injection defense** | ✅ `ContainsInjection` rails | ❌ | ❌ | ❌ |

**Assessment:** Intivai's RLS + prompt injection defense is unique. SOC 2 gap is significant for enterprise sales. Manatal is already SOC 2 Type II.

---

## 3. Pricing Comparison

### Manatal (Full ATS + AI)
| Plan | Price | Key Limits |
|---|---|---|
| Professional | $15/user/mo (annual) | 15 jobs, 10K candidates |
| Enterprise | $35/user/mo (annual) | Unlimited jobs/candidates |
| Enterprise Plus | $55/user/mo (annual) | + API, SSO, priority support |
| AI Interviewer | **Add-on** (price undisclosed) | Video screening, customizable |

### Hireflix (One-Way Video Only)
| Plan | Price | Key Limits |
|---|---|---|
| Small | $75/mo (annual) | Unlimited seats, positions, responses |
| Medium | $150/mo (annual) | 50-250 employees |
| Custom | Contact sales | High volume |

### Interviewer.ai (AI Video Interviews)
| Plan | Price | Key Limits |
|---|---|---|
| Business | Custom pricing | Enterprise sales only |
| University | Custom pricing | Admissions + career services |
| Free trial | Available | Limited |

### Intivai (Proposed)
| Tier | Price | Key Limits |
|---|---|---|
| Free | $0 | 3 interviews, chat only |
| Starter | $49/mo | 100 interviews |
| Pro | $199/mo | 1,000 interviews |
| Volume (BPO) | $0.5-1/interview | Usage-based |
| Enterprise | Custom | Self-hosted |

**Pricing Risk Assessment:**

1. **Manatal at $15/user/mo includes full ATS + CRM + 2,500 job boards + AI enrichment + career page + reporting.** Intivai at $49/mo offers interview-only. **Value perception risk: buyer sees "less for more."**

2. **Hireflix at $75/mo unlimited for small companies.** Intivai at $49/mo for 100 interviews is competitive, but Hireflix includes unlimited responses.

3. **Interviewer.ai custom pricing suggests $100-500+/mo range.** Intivai is cheaper but less established.

4. **The $0.5-1/interview volume pricing for BPO is Intivai's strongest pricing play** — it aligns with usage and is cheaper than Hireflix's flat rate for high-volume users.

---

## 4. Strategic Positioning Analysis

### 4.1 Where Intivai Wins

| Win Condition | Why |
|---|---|
| **Live conversational AI** | No competitor does real-time two-way AI chat. Hireflix is async one-shot. Manatal's AI Interviewer is video-based but not conversational. |
| **Technical coding sandbox** | Zero competitors offer this. For engineering roles, this is a category-of-one feature. |
| **Transparent, configurable scoring** | Competitors use opaque AI. Intivai's 5-dimension weighted system with per-tenant config is unique. |
| **Self-hosted / on-prem** | No competitor offers this. Strong for data-sensitive enterprises. |
| **Global Talent Passport** | Novel concept with viral potential. No market precedent. |
| **Cost per interview** | Near-zero infrastructure cost ($0.001/interview) enables aggressive volume pricing. |

### 4.2 Where Intivai Loses

| Loss Condition | Why |
|---|---|
| **Not an ATS** | Manatal is an all-in-one ATS+CRM. Intivai is interview-only. HR buyers want one platform. |
| **No integrations** | Zero ATS integrations, zero job board integrations, no Zapier. Enterprise deal-breaker. |
| **No sourcing** | Manatal has 2,500+ job boards + LinkedIn extension. Intivai requires external sourcing. |
| **Mobile experience** | Manatal has a mobile app. Intivai is desktop-only (Playwright tests). |
| **Reporting depth** | Manatal has full reporting suite with custom builders. Intivai has basic dashboard. |
| **SOC 2 gap** | Manatal and Hireflix are SOC 2 certified. Intivai is not. |
| **Brand trust** | Manatal: 10,000+ teams, 135+ countries. Hireflix: 434+ reviews. Intivai: zero customers. |

### 4.3 Positioning Recommendation

**Do NOT position as "AI recruitment platform"** — Manatal owns that position with 10K+ customers. Instead:

**Position as: "The AI Technical Interview Platform"**

- **Target:** Engineering teams, tech recruiters, BPOs doing volume tech hiring
- **Value:** "Stop wasting senior engineers' time on screening calls. Let AI conduct live technical interviews with coding assessments, then hand off only the top candidates."
- **Proof:** Coding sandbox + live conversational AI + proctoring + verbatim quotes

This avoids competing with Manatal (full ATS) and Hireflix (async video) on their turf, and instead creates a new category: **live AI technical interviewing**.

---

## 5. Critical Gaps & Risks

### 5.1 Must-Fix Before Beta

| Gap | Risk Level | Recommendation |
|---|---|---|
| **No ATS integrations** | 🔴 Critical | Build at least Greenhouse + Lever + Workable webhooks (outbound) before beta. HR teams won't switch ATS just for interviews. |
| **No mobile support** | 🔴 Critical | Candidates interview on phones. Must test responsive design on mobile. |
| **SOC 2 not achieved** | 🟡 High | Enterprise buyers require this. Start SOC 2 prep now (log retention, access controls documentation). |
| **No DOCX support** | 🟡 High | Many candidates submit DOCX. Add `go-docx` parsing (already in Research doc, just not implemented). |
| **PDF report deferred** | 🟡 High | Recruiters expect downloadable PDF scorecards. JSON-only is a friction point. Accelerate P4b. |

### 5.2 Strategic Risks

| Risk | Impact | Mitigation |
|---|---|---|
| **Manatal adds conversational AI** | High | They already have AI Interviewer. If they add real-time chat, Intivai's moat shrinks. Move fast on beta. |
| **Interviewer.ai improves avatar interviews** | Medium | Their "Avatar" product is brand new. If it becomes conversational, it competes directly. |
| **Hireflix adds AI features** | Low | Hireflix is deeply committed to async one-way video. Unlikely to pivot to live. |
| **Pricing pressure from Manatal** | High | At $15/user/mo for full ATS, Intivai at $49/mo for interview-only is hard to justify. Must prove ROI through time savings. |
| **LLM costs rise** | Medium | LLM is cheap now ($0.0001/1K tokens). If pricing changes, margin compresses. Lock in long-term API agreement. |

---

## 6. Recommendations

### 6.1 Product Priorities (Post-Beta)

| Priority | Action | Why |
|---|---|---|
| 1 | **ATS integration webhooks** | Without this, Intivai is a standalone tool, not part of the hiring workflow |
| 2 | **Mobile responsive candidate experience** | Candidates interview on phones — this is non-negotiable |
| 3 | **DOCX parsing** | Already designed in Research, just needs implementation |
| 4 | **PDF scorecard export** | Recruiters expect this; accelerate P4b |
| 5 | **SOC 2 Type I certification** | Required for enterprise sales within 6 months |

### 6.2 Positioning Adjustments

1. **Remove "Autonomous AI Recruitment Platform"** from PRD title. Replace with **"AI Technical Interview Platform"**
2. **Lead with coding sandbox** in marketing — this is the category-of-one feature
3. **Show ROI Calculator prominently** — the $38K/quarter savings is compelling
4. **Emphasize "Your data stays yours"** — self-hosted option is unique
5. **Position Global Talent Passport as candidate benefit** — "Never re-test your skills"

### 6.3 Pricing Adjustments

| Change | Why |
|---|---|
| Lower Starter to **$29/mo** | Compete with Manatal's $15/user/mo perception |
| Add **"Interview Credits" pack** at $99 for 500 | Volume discount for BPO without enterprise commitment |
| Keep volume at **$0.50/interview** | This is the strongest pricing play for beachhead market |
| Add **free tier: 10 interviews/mo** (not 3) | 3 is too low for meaningful trial; 10 allows real evaluation |

### 6.4 Go-to-Market Adjustment

| Current | Recommended |
|---|---|
| "AI recruitment platform" | "AI technical interview platform" |
| Target: all recruiters | Target: **tech recruiters + engineering hiring managers** |
| Beachhead: volume BPO | Beachhead: **tech startups hiring 5-20/mo** (higher willingness to pay for quality) |
| Channels: LinkedIn, JobStreet | Channels: **Dev communities (HN, Reddit r/cscareerquestions), tech recruiter Slack groups** |
| Pricing: subscription first | Pricing: **usage-based first** (lower barrier, prove value) |

---

## 7. Competitor Deep-Dive Notes

### Manatal
- **Strengths:** Full ATS+CRM, 2,500+ job boards, AI enrichment from 20+ social platforms, mobile app, SOC 2 Type II, 10K+ customers in 135+ countries
- **Weaknesses:** AI Interviewer is an add-on (not core), no live coding, no self-hosted option, no conversational AI
- **Intivai advantage over Manatal:** Live conversational AI + coding sandbox + self-hosted + transparent scoring
- **Manatal advantage over Intivai:** Everything else (ATS, integrations, sourcing, mobile, reporting, brand trust)

### Hireflix
- **Strengths:** Easiest UX (zero login, no app), unlimited responses at flat rate, 50+ ATS integrations, SOC 2, great candidate experience, strong in LATAM + Europe
- **Weaknesses:** One-way async only (not conversational), no AI analysis, no coding, no scoring, no ATS
- **Intivai advantage over Hireflix:** Live conversational AI + coding + scoring + proctoring
- **Hireflix advantage over Intivai:** Candidate UX (zero friction), integrations, brand trust, pricing simplicity

### Interviewer.ai
- **Strengths:** AI video interviews, explainable AI, structured interviews, university vertical, sustainability calculator, established brand
- **Weaknesses:** Async video (not live conversational), no coding, custom pricing (opaque), no self-hosted
- **Intivai advantage over Interviewer.ai:** Live conversational (not just async video) + coding sandbox + self-hosted + usage-based pricing
- **Interviewer.ai advantage over Intivai:** Established brand, university vertical, sustainability angle, more polished UX

---

## 8. Conclusion

Intivai has **two genuine category-defining differentiators**: live conversational AI interviews and technical coding sandbox. These features are not available in any competitor. The Global Talent Passport is a novel innovation.

However, Intivai has **critical gaps in ATS integration, mobile support, and brand trust** that will block enterprise adoption. The PRD should be updated to:

1. Acknowledge that Intivai is an **interview platform, not a recruitment platform** (avoid overclaiming)
2. Prioritize ATS integration webhooks as a **beta blocker** (not post-MVP)
3. Test mobile candidate experience before beta
4. Adjust pricing to be more competitive with Manatal's $15/user/mo baseline
5. Pivot positioning to **"AI Technical Interview Platform"** — own the niche, don't fight Manatal for the broad market

The beachhead market should shift from volume BPO to **tech startup hiring** — these buyers value quality over volume, are willing to pay for live AI interviews, and the coding sandbox is a decisive differentiator for engineering roles.

---

## 9. Multi-Perspective Deep Review

### 9.1 CEO Perspective — "Can this make money?"

**Overall assessment: Cautiously optimistic. Strong technology, weak go-to-market readiness.**

The CEO looks at three things: market size, defensibility, and unit economics.

**Market:** AI recruitment is $2.22B in 2026, projected $6.4B by 2030 (30.3% CAGR). The AI interview sub-segment is the fastest growing. There is money here. But the market is also getting crowded — Manatal has 10,000+ customers, Interviewer.ai is well-funded, and Hireflix owns the async video niche. Intivai is entering as a zero-customer startup with no brand recognition.

**Defensibility:** The coding sandbox is genuinely defensible — it requires significant engineering (Docker isolation, mTLS gRPC, multi-language runtimes) that competitors can't replicate quickly. The live conversational AI is also strong, but LLM technology is commoditizing fast — today's moat is tomorrow's commodity. The Global Talent Passport is clever but unproven — nobody knows if candidates will value portable credentials.

**Unit economics:**
- Cost per interview: ~$0.001 (LLM API + self-hosted STT/TTS)
- Revenue per interview: $0.50-1.00 (volume pricing)
- **Gross margin: 99.8%** — exceptional if volume materializes
- But: infrastructure cost (VPS, PostgreSQL, Redis, MinIO) is ~$200-500/mo for a single server
- Break-even at ~500-1,000 interviews/mo on Starter plan

**CEO's concern:** The PRD says "80% reduction in first-round screening time" and "$38,000+/quarter in engineering hours saved." These are strong claims. But Manatal already makes similar claims with 10,000+ customers as social proof. Intivai has zero customers. **The PRD needs a customer validation section — not just features, but evidence that buyers will pay for these features.**

**CEO's recommendation:**
1. Ship beta with 5 pilot customers as fast as possible — **revenue validation before feature completeness**
2. Price at $29/mo Starter (not $49) to lower the trial barrier
3. Add "Interview Credits" pack ($99/500) for volume users — usage-based pricing is Intivai's strongest card
4. **Kill the "Global Talent Passport" from MVP scope** — it's innovative but unproven, and adds complexity. Ship it post-beta when you have evidence candidates want it.
5. Focus beta on **3 tech startups in Jakarta** — validate the coding sandbox + live AI interview workflow end-to-end

---

### 9.2 Head of Product Perspective — "Is this the right product?"

**Overall assessment: Over-scoped for MVP. The core is strong but the surrounding features dilute focus.**

The Head of Product looks at scope clarity, user workflows, and what NOT to build.

**What's right:**
- The core pipeline (CV→Score→Interview→Report) is clean and well-defined
- CV-gap targeting is a genuinely smart approach — questions from JD↔CV gaps, not generic banks
- The coding sandbox is a differentiator that should be the hero feature
- Per-tenant configurable scoring is a smart enterprise feature
- Company context + tenant prompt is a strong personalization layer

**What's wrong:**

1. **The PRD calls this an "Autonomous AI Recruitment Platform"** — it's not. It doesn't do sourcing, job posting, CRM, onboarding, or offer management. It's an **AI Interview Platform**. This mislabeling will confuse buyers and set wrong expectations.

2. **Too many epics for MVP.** 10 epics is a lot. The PRD should be ruthlessly prioritized:
   - **Must have for beta:** Epic 1 (Jobs), Epic 2 (CV), Epic 3 (Screening), Epic 4 (Candidate Portal), Epic 5 (Chat Interview), Epic 7 (Scorecard)
   - **Should have:** Epic 9 (Company Context) — already built
   - **Could have:** Epic 6 (Proctoring) — advisory only, low effort
   - **Won't have (MVP):** Epic 8 (Global Talent Passport), Epic 10 (Analytics beyond basic dashboard)

3. **The proctoring "advisory only" posture (ADR-0004) is a product risk.** If proctoring data is displayed but doesn't affect scores, recruiters will ignore it. Either make it actionable (flag suspicious interviews for review) or remove it from the MVP. Advisory-only proctoring is a feature that adds UI complexity without value.

4. **Mobile candidate experience is not addressed in the PRD.** The NFR section mentions "public pages load in < 500ms" but nothing about mobile viewport testing. This is a critical oversight — most candidates will interview on phones.

5. **The evaluation schema is over-engineered for MVP.** 4 dimensions + per-question scores + strengths/weaknesses + recommendation. Simplify to: overall score (0-100) + recommendation (proceed/hold/reject) + top 3 strengths + top 3 weaknesses. Detailed breakdown comes post-beta.

**Head of Product recommendation:**
1. Rename the PRD: **"Intivai — AI Technical Interview Platform"**
2. Cut Epic 8 (Global Talent Passport) from MVP — ship post-beta
3. Simplify evaluation to 4 fields: score, recommendation, strengths, weaknesses
4. Add mobile responsive testing to NFRs (mandatory before beta)
5. Make proctoring either actionable or remove it — advisory-only is dead weight
6. The PRD has 178 lines of "Executive Summary" before getting to features. **HR buyers won't read past page 2.** Restructure: Problem → Solution → How it works → Pricing → Get started. Move technical architecture to appendix.

---

### 9.3 HR Specialist Perspective — "Will this actually help me hire?"

**Overall assessment: Impressive technology, but I'm not sure it fits my workflow.**

The HR Specialist (recruiter or hiring manager) looks at: daily workflow impact, ease of use, candidate quality, and ROI.

**What excites me:**
- **"80% reduction in first-round screening time"** — if this is real, it's game-changing. I spend 2-3 hours/day on phone screens. If AI can do 80% of that, I can focus on closing candidates.
- **CV-gap targeting** — finally, an interviewer that asks relevant questions. Most AI interview tools ask generic "tell me about yourself" questions. If Intivai actually reads the CV and asks about specific gaps, that's valuable.
- **Coding sandbox for technical roles** — I currently send take-home coding tests, then worry about cheating. A proctored live coding environment solves both problems.
- **Scorecard with verbatim quotes** — I hate when AI gives me a score but I can't see why. If every score cites what the candidate actually said, I can trust the evaluation.

**What worries me:**

1. **"I already use Manatal for everything."** This is the biggest objection. Manatal is my ATS — I post jobs, track candidates, collaborate with my team, run reports. If Intivai is a separate tool, that's another login, another dashboard, another thing to learn. **Why wouldn't I just use Manatal's AI Interviewer add-on?**

2. **"How do I get candidates into Intivai?"** The PRD mentions CV upload, but in practice I'm sourcing from LinkedIn, Indeed, and employee referrals. If I have to manually export CSVs from Manatal and import into Intivai, that's a deal-breaker. I need a one-click integration.

3. **"My candidates hate new tools."** Every time I introduce a new tool, candidates complain about downloads, logins, technical issues. Hireflix wins here — zero login, zero download, works on any device. Intivai requires a career portal visit, OTP authentication, and WebSocket connection. That's three failure points before the interview starts.

4. **"I need to see this on my phone."** I review candidates while commuting, between meetings, over lunch. If Intivai's dashboard is desktop-only, I won't use it. Manatal has a mobile app. Hireflix is mobile-responsive. Intivai needs to be too.

5. **"What happens when the AI makes a bad recommendation?"** The PRD mentions "Strong Hire / Hire / Consider / Reject" badges. But what if the AI rejects a great candidate because they didn't word their answer perfectly? I need to see the full transcript, not just a score. The "verbatim quotes" feature helps, but I also need to override the AI's recommendation without feeling like I'm fighting the system.

6. **"SOC 2 compliance."** My company requires SOC 2 for any vendor handling candidate data. If Intivai isn't SOC 2 certified, I can't use it. Period. Manatal and Hireflix are certified. This is a non-negotiable for enterprise HR.

**HR Specialist recommendation:**
1. **Build a Manatal integration first** — not just webhooks, but a native "Send to Intivai" button in Manatal's candidate view. This removes the #1 workflow friction.
2. **Make the candidate experience as frictionless as Hireflix** — no login if possible (magic link only), mobile-first design, no downloads.
3. **Add an "Override AI Recommendation" button** on the scorecard — recruiters must feel in control, not overridden.
4. **Show the full transcript alongside the score** — don't hide the evidence behind a summary.
5. **Add SOC 2 certification timeline to the PRD** — even if it's 6 months out, HR needs to see a commitment.

---

### 9.4 Candidate Perspective — "What's this like to actually use?"

**Overall assessment: The concept is cool but the experience needs to be invisible.**

The candidate looks at: friction to start, interview experience, and fairness.

**What works:**
- **"AI reads my CV and asks relevant questions"** — this feels personalized, not generic. Much better than "tell me about yourself" for the 10th time.
- **Coding sandbox in the browser** — no setup, no downloads, no VPN. I just open a link and start coding. This is better than HackerRank/LeetCode's clunky interfaces.
- **Real-time chat feels natural** — streaming tokens feel like talking to a human, not waiting for a wall of text. The interrupt feature is nice — if I know the answer quickly, I can jump in.
- **"Your data stays yours"** — self-hosted option means my CV isn't floating around in some SaaS dashboard. Privacy matters.

**What frustrates me:**

1. **"Why do I need an OTP code?"** Hireflix sends me a link, I click it, I'm in. Intivai wants me to enter a 6-digit code, wait for it to arrive, type it in. That's 30 seconds of friction before I even start. **Just use a magic link — one click, done.**

2. **"Is this interview公平?"** The proctoring section mentions tab switching, paste detection, away time. As a candidate, this feels like surveillance. Even if it's "advisory," knowing that every tab switch is being logged makes me anxious. **Don't show the proctoring UI to candidates.** Let recruiters see the audit trail, but don't make candidates feel watched.

3. **"What if my WiFi drops?"** The PRD mentions reconnection, but does it actually work? If I lose connection mid-answer, do I have to start the question over? Hireflix doesn't have this problem because it's pre-recorded — I record once, upload, done. Live interviews have a real failure mode here.

4. **"I'm on my phone."** Most candidates will check the interview link on their phone first. If the chat interface doesn't work on mobile, I'll abandon the interview. The coding sandbox especially needs to work on tablets at minimum.

5. **"What happens to my data after?"** The PRD mentions 90-day auto-delete, but as a candidate, I want to know: can I see my own data? Can I delete it myself? The "Global Talent Passport" sounds cool, but I didn't ask for it — is it opt-in or opt-out?

6. **"The AI is judging me."** There's something心理ally different about being interviewed by a human vs. an AI. The streaming tokens make it feel conversational, but I know it's an algorithm scoring my answers. **The PRD should include a "candidate transparency" section** — tell candidates upfront: "This is an AI interview. Your responses are evaluated by AI with human review. You can request a human interviewer at any time."

**Candidate recommendation:**
1. **Magic link only — kill the OTP flow for MVP.** One click to start. Reduce friction to zero.
2. **Hide proctoring from candidates.** Show it in the recruiter dashboard, not in the interview UI.
3. **Add "Request human interviewer" option** — even if it's just a button that notifies the recruiter, it gives candidates a safety valve.
4. **Mobile-first candidate UI** — test on iPhone SE, Android mid-range. Not just desktop.
5. **Add candidate data portal** — "View my data" / "Delete my data" link in the interview invitation email. GDPR compliance + candidate trust.
6. **Add connection quality indicator** — show candidates their WebSocket connection status. If it drops, auto-save and show "Reconnecting..." not a blank screen.

---

## 10. Cross-Perspective Synthesis

### Where all 4 perspectives agree:

| Agreement | Evidence |
|---|---|
| **Mobile is non-negotiable** | CEO: "most candidates use phones"; Head of Product: "not in NFRs"; HR: "I review on my phone"; Candidate: "I'll abandon on desktop-only" |
| **Integrations are critical** | CEO: "enterprise deal-breaker"; Head of Product: "mislabeling as recruitment platform"; HR: "another login = another tool I won't use" |
| **Candidate friction must be minimal** | CEO: "Hireflix wins on UX"; Head of Product: "OTP adds failure points"; HR: "candidates hate new tools"; Candidate: "just give me a magic link" |
| **SOC 2 is required for enterprise** | CEO: "blocks enterprise sales"; Head of Product: "not in PRD"; HR: "non-negotiable for my company"; Candidate: "privacy matters" |
| **Pricing needs adjustment** | CEO: "Manatal at $15 is hard to compete with"; Head of Product: "usage-based is strongest play"; HR: "show ROI or I won't justify the purchase"; Candidate: N/A |

### Where perspectives diverge:

| Divergence | CEO says | HR says | Candidate says |
|---|---|---|---|
| **Global Talent Passport** | "Innovative but unproven — cut from MVP" | "Not relevant to my hiring workflow" | "I didn't ask for this — is it opt-in?" |
| **Proctoring** | "Advisory-only is dead weight — make it actionable or remove" | "I want to see it but don't show it to candidates" | "Don't surveil me — hide the proctoring UI" |
| **Evaluation complexity** | "Keep it simple for MVP" | "I need to see the full transcript, not just a score" | "The AI is judging me — add transparency" |
| **Company context** | "Strong differentiator" | "I already have my interview process documented" | "The AI should feel like my company, not a generic bot" |

### Final Priority Matrix (all perspectives weighted):

| Priority | Action | Who Benefits | Effort |
|---|---|---|---|
| **P0 (Beta blocker)** | Mobile candidate experience | Candidate + HR + CEO | Medium |
| **P0 (Beta blocker)** | Magic link only (kill OTP) | Candidate + Head of Product | Low |
| **P0 (Beta blocker)** | ATS integration (at least 1) | HR + CEO | Medium |
| **P1 (Before general availability)** | SOC 2 Type I prep | HR + CEO | High |
| **P1 (Before GA)** | Candidate data portal (view/delete) | Candidate + CEO | Low |
| **P1 (Before GA)** | Hide proctoring from candidates | Candidate + HR | Low |
| **P2 (Post-beta)** | PDF scorecard export | HR + Head of Product | Medium |
| **P2 (Post-beta)** | Global Talent Passport | CEO + Head of Product | High |
| **P3 (Post-GA)** | Full reporting suite | HR + Head of Product | High |
| **P3 (Post-GA)** | Mobile recruiter app | HR + CEO | High |
