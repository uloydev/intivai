---
name: tech_lead_review
description: Enforces that all code reviews are deep, systemic, and adhere to production-grade standards.
trigger: always_on
---

# Tech Lead Review Mandate

When performing a codebase review or aggregating review findings:
1. **Never Sacrifice Quality for Speed**: Do not skim or accept surface-level findings. Perform deep, systemic analysis (checking transaction boundaries, foreign key constraints, connection pool exhaustion, rate-limiting edge cases).
2. **Verify All Findings**: Before presenting a review, manually verify the findings in the actual codebase (e.g., check the migration files, check the middleware order).
3. **Fail-Safe Mindset**: Ensure robust cleanup mechanisms are used to prevent stuck states like deadlocked database connections or hanging FK constraints.
4. **Follow AGENTS.md**: Adhere strictly to the project's internal rules, treating the repository as a revenue-generating production product where zero mistakes and no hallucinations are tolerated.
