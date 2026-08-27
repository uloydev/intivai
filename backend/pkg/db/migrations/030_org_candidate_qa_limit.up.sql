-- 030_org_candidate_qa_limit.up.sql — per-tenant candidate Q&A cap (B4/D3).
-- nil → code default 10. Added after the fact: org.go and the IAM repo already
-- select/insert this column, so the column must exist for any query to run.
ALTER TABLE orgs ADD COLUMN candidate_qa_limit INTEGER;
