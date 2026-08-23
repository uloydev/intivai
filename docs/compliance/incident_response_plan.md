# Incident Response Plan

> Status: corrected 2026-08-24 · Owner: EM
> Solo-dev reality: the escalation roles below are **aspirational until a team
> exists** — during beta the "on-call engineer" is the founder. Keep the
> structure; be honest about who is actually paged.

## Severity Levels

| Level | Description | Response Time | Examples |
|-------|-------------|--------------|----------|
| P0 | Service down, data breach | Immediate | DB failure, auth bypass, data leak |
| P1 | Major feature broken | 4 hours | Interview creation fails, evaluation pipeline down |
| P2 | Minor feature degraded | 24 hours | Webhook delivery failures, slow queries |
| P3 | Cosmetic, non-urgent | Next sprint | UI glitches, minor bugs |

## Response Workflow

### 1. Detection
- Monitoring: Sentry error tracking, structured logs
- User reports: support channel, community forum
- Automated: health check failures, worker queue backlog

### 2. Triage
- Assign severity level
- Identify affected users and data scope
- Determine if security incident (breach, unauthorized access)

### 3. Containment
- P0: Feature flag to disable affected component
- P0/P1: Rotate affected credentials (API keys, JWT secrets)
- Data breach: Isolate affected systems, preserve evidence

### 4. Communication
- Internal: Incident channel, status page update
- Users: Email notification for P0/P1 within 24 hours
- Regulatory: DPA notification within 72 hours for data breaches (GDPR Art. 33)

### 5. Resolution
- Fix root cause
- Deploy with full test suite verification
- Verify no regressions

### 6. Post-Mortem
- Blameless post-mortem within 5 business days
- Action items tracked in issue tracker
- Update this document with lessons learned

## Contact Escalation

| Role | Responsibility |
|------|---------------|
| On-call engineer | First responder, initial triage |
| Engineering lead | Technical decisions, resource allocation |
| Product owner | User communication, business impact |
| Legal/compliance | Regulatory notification, legal review |

## Recovery Procedures

### Database Recovery
```bash
scripts/restore.sh <dump_file>
```

### Service Recovery
```bash
docker compose --env-file .env.prod up -d --force-recreate
```

### Data Verification
```bash
make smoke  # End-to-end API scenario
make test-integration-dev  # Full integration suite
```
