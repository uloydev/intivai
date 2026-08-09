-- Pin the company-context version at interview creation (audit traceability:
-- which context the interviewer saw, regardless of later context uploads).

ALTER TABLE interviews ADD COLUMN context_version INT NOT NULL DEFAULT 0;
