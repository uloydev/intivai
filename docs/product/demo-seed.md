# Intivai Demo Seed Data & Example Candidate Resumes (`EXAMPLES.md`)

This guide explains the comprehensive demo scenario seeded into Intivai for organization **Demo Corp** (`slug: demo`). It outlines all **12 engineering job requisitions**, their corresponding **passing candidate CV profiles & PDF fixtures**, screening rubrics, evaluation scorecards, and interactive portal authentication links.

---

## 1. Quick Reference: Seeded Jobs & Passing CV Matrix

| # | Job Requisition | Seniority | Location & Policy | Type | Salary Band | Passing Candidate | Score | Funnel Stage | PDF Resume File |
|---|---|---|---|---|---|---|---|---|---|
| **1** | **Senior Distributed Systems Engineer** | Senior | Remote (US/EU) | Full-time | $160k–$210k USD | **Alex Rivera** | **92.5%** | `interview_completed` | [`fixtures/cvs/alex_rivera_cv.pdf`](fixtures/cvs/alex_rivera_cv.pdf) |
| **2** | **Staff Frontend Architect** | Staff | San Francisco / Remote | Full-time | $175k–$230k USD | **Elena Rostova** | **94.0%** | `offer_extended` | [`fixtures/cvs/elena_rostova_cv.pdf`](fixtures/cvs/elena_rostova_cv.pdf) |
| **3** | **Principal AI & ML Systems Engineer** | Principal | New York / Remote | Full-time | $190k–$250k USD | **David Chen, Ph.D.** | **89.0%** | `interview_invited` | [`fixtures/cvs/david_chen_cv.pdf`](fixtures/cvs/david_chen_cv.pdf) |
| **4** | **Senior Cloud Platform & SRE** | Senior | London, UK (Hybrid) | Full-time | £95k–£130k GBP | **Liam O'Connor** | **91.0%** | `interview_invited` | [`fixtures/cvs/liam_oconnor_cv.pdf`](fixtures/cvs/liam_oconnor_cv.pdf) |
| **5** | **Staff Mobile Engineer (React Native/iOS)** | Staff | New York, NY (Hybrid) | Full-time | $165k–$215k USD | **Maya Patel** | **93.5%** | `screening_passed` | [`fixtures/cvs/maya_patel_cv.pdf`](fixtures/cvs/maya_patel_cv.pdf) |
| **6** | **Lead Full-Stack Product Engineer** | Lead | Berlin, DE (Remote EU) | Full-time | €90k–€125k EUR | **Henrik Lindqvist** | **90.5%** | `interview_invited` | [`fixtures/cvs/henrik_lindqvist_cv.pdf`](fixtures/cvs/henrik_lindqvist_cv.pdf) |
| **7** | **Senior Data & Streaming Pipeline** | Senior | Singapore (Hybrid) | Full-time | S$130k–S$175k SGD | **Priya Sharma** | **88.5%** | `screening_passed` | [`fixtures/cvs/priya_sharma_cv.pdf`](fixtures/cvs/priya_sharma_cv.pdf) |
| **8** | **Lead AppSec & DevSecOps Engineer** | Lead | Remote (US/EU) | Full-time | $170k–$220k USD | **Zachary Taylor** | **95.0%** | `offer_extended` | [`fixtures/cvs/zachary_taylor_cv.pdf`](fixtures/cvs/zachary_taylor_cv.pdf) |
| **9** | **Senior SDET & Test Automation Architect** | Senior | Remote (Worldwide) | Full-time | $135k–$175k USD | **Chloe Dubois** | **92.0%** | `interview_completed` | [`fixtures/cvs/chloe_dubois_cv.pdf`](fixtures/cvs/chloe_dubois_cv.pdf) |
| **10**| **Engineering Manager / Technical Lead** | Lead/EM | San Francisco (Hybrid) | Full-time | $210k–$265k USD | **Samuel Okafor** | **96.0%** | `hired` | [`fixtures/cvs/samuel_okafor_cv.pdf`](fixtures/cvs/samuel_okafor_cv.pdf) |
| **11**| **Junior Fullstack Engineer** | Junior | Remote (Worldwide) | Contract | $70k–$90k USD | **Jordan Brooks** | **85.0%** | `screening_passed` | [`fixtures/cvs/jordan_brooks_cv.pdf`](fixtures/cvs/jordan_brooks_cv.pdf) |
| **12**| **AI Evaluation & Prompt Intern** | Intern | San Francisco (On-site) | Internship | $50k–$65k USD ($50/h)| **Sophia Zhang** | **88.0%** | `interview_invited` | [`fixtures/cvs/sophia_zhang_cv.pdf`](fixtures/cvs/sophia_zhang_cv.pdf) |
| *Ctrl*| *Junior Web Developer (Failed Control)* | Junior | Remote | Full-time | — | *Marcus Vance* | *38.0%* | `screening_failed` | [`fixtures/cvs/marcus_vance_cv.pdf`](fixtures/cvs/marcus_vance_cv.pdf) |

---

## 2. Detailed Job Requisitions & Candidate Profiles

### 1. Senior Distributed Systems Engineer
- **Job ID:** `c3d4e5f6-a1b2-4c3d-8e4f-5a6b7c8d9e0f`
- **Domain & Tech:** Golang, PostgreSQL RLS, Distributed Queues (Asynq/Redis), Docker, Kubernetes, gRPC, Kafka.
- **Requirements:** 5+ years experience building fault-tolerant distributed systems, transaction safety, and connection pooling.
- **Scoring Weights:** Skills: 35%, Experience: 25%, Semantic Match: 25%, Education: 10%, Certifications: 5%. Threshold: 60.0%.
- **Matching Candidate:** **Alex Rivera** (`alex.rivera@example.com`)
  - **Resume Highlights:** 8 years experience at CloudScale Inc; led Go microservices handling 150k events/sec; implemented PostgreSQL RLS multi-tenancy.
  - **Screening Result:** **92.5%** (Skills: 0.95, Experience: 0.90, Semantic: 0.92, Education: 0.85, Certifications: 1.0) ➔ **Passed**.
  - **Application Stage:** `interview_completed`
  - **Completed Assessment Score:** **92.0 / 100** (Technical Concurrency: 95%, DB Isolation: 92%).
  - **Candidate Portal Magic Link:** `http://localhost:5173/candidate/portal?token=demo-magic-token-alex-2026`
  - **PDF Resume File:** `fixtures/cvs/alex_rivera_cv.pdf`

---

### 2. Staff Frontend Architect
- **Job ID:** `d4e5f6a1-b2c3-4d4e-8f5a-6b7c8d9e0f1a`
- **Domain & Tech:** React 19, TypeScript, Tailwind CSS, WebSockets, Radix UI Primitives, Design Token Systems.
- **Requirements:** 6+ years experience in frontend architecture, WCAG 2.1 AA accessibility, and real-time streaming UI.
- **Scoring Weights:** Skills: 35%, Experience: 30%, Semantic Match: 25%, Education: 10%. Threshold: 65.0%.
- **Matching Candidate:** **Elena Rostova** (`elena.rostova@example.com`)
  - **Resume Highlights:** 7 years experience at UI Labs; built company-wide Radix/Tailwind design system; sub-100ms WebSocket streaming latency.
  - **Screening Result:** **94.0%** (Skills: 0.96, Experience: 0.92, Semantic: 0.95, Education: 0.90) ➔ **Passed**.
  - **Application Stage:** `offer_extended`
  - **Candidate Portal Magic Link:** `http://localhost:5173/candidate/portal?token=demo-magic-token-elena-2026`
  - **PDF Resume File:** `fixtures/cvs/elena_rostova_cv.pdf`

---

### 3. Principal AI & ML Systems Engineer
- **Job ID:** `e5f6a1b2-c3d4-4e5f-8a6b-7c8d9e0f1a2b`
- **Domain & Tech:** Python, PyTorch, Whisper STT, Kokoro TTS, pgvector (HNSW), WebRTC Audio, Prompt Rails.
- **Requirements:** 7+ years experience in ML inference, low-latency audio pipelines, and vector semantic similarity.
- **Scoring Weights:** Skills: 40%, Experience: 25%, Semantic Match: 25%, Education: 10%. Threshold: 70.0%.
- **Matching Candidate:** **David Chen, Ph.D.** (`david.chen@example.com`)
  - **Resume Highlights:** 9 years experience at Synthetix AI; Ph.D. Stanford University; authored sub-300ms speech pipelines and pgvector search.
  - **Screening Result:** **89.0%** (Skills: 0.90, Experience: 0.88, Semantic: 0.88, Education: 0.95) ➔ **Passed**.
  - **Application Stage:** `interview_invited`
  - **Live Interview Token:** `demo-invitation-token-david-chen-2026`
  - **Candidate Portal Magic Link:** `http://localhost:5173/candidate/portal?token=demo-magic-token-david-2026`
  - **PDF Resume File:** `fixtures/cvs/david_chen_cv.pdf`

---

### 4. Senior Cloud Platform & SRE
- **Job ID:** `11111111-2222-3333-4444-555555555501`
- **Domain & Tech:** Kubernetes (EKS), Terraform, AWS, Prometheus, Docker, Golang, CI/CD, GitOps (ArgoCD).
- **Requirements:** 5+ years managing high-availability cloud infrastructure in production; CKA/CKS certified preferred.
- **Scoring Weights:** Skills: 35%, Experience: 30%, Semantic: 20%, Certifications: 15%. Threshold: 65.0%.
- **Matching Candidate:** **Liam O'Connor** (`liam.oconnor@example.com`)
  - **Resume Highlights:** 6 years SRE lead at FinTech Cloud UK; CKA & AWS Pro certified; 99.99% uptime for multi-region Kubernetes.
  - **Screening Result:** **91.0%** (Skills: 0.95, Experience: 0.90, Semantic: 0.88, Certifications: 0.95) ➔ **Passed**.
  - **Application Stage:** `interview_invited`
  - **Live Interview Token:** `demo-invitation-token-liam-oconnor-2026`
  - **Candidate Portal Magic Link:** `http://localhost:5173/candidate/portal?token=demo-magic-token-liam-2026`
  - **PDF Resume File:** `fixtures/cvs/liam_oconnor_cv.pdf`

---

### 5. Staff Mobile Engineer - React Native & iOS
- **Job ID:** `11111111-2222-3333-4444-555555555502`
- **Domain & Tech:** React Native, TypeScript, Swift (iOS), Kotlin (Android), WebRTC Audio, Offline SQLite Sync.
- **Requirements:** 6+ years building mobile apps, custom native bridges, 60fps frame rate guarantees, and cold-start optimization.
- **Scoring Weights:** Skills: 35%, Experience: 30%, Semantic Match: 25%, Education: 10%. Threshold: 65.0%.
- **Matching Candidate:** **Maya Patel** (`maya.patel@example.com`)
  - **Resume Highlights:** 7 years mobile architect at Nomad Health; built custom Swift C++ audio bridges and offline sync engine.
  - **Screening Result:** **93.5%** (Skills: 0.96, Experience: 0.92, Semantic: 0.94, Education: 0.90) ➔ **Passed**.
  - **Application Stage:** `screening_passed`
  - **Candidate Portal Magic Link:** `http://localhost:5173/candidate/portal?token=demo-magic-token-maya-2026`
  - **PDF Resume File:** `fixtures/cvs/maya_patel_cv.pdf`

---

### 6. Lead Full-Stack Product Engineer
- **Job ID:** `11111111-2222-3333-4444-555555555503`
- **Domain & Tech:** Next.js 15, Node.js, TypeScript, PostgreSQL, GraphQL, Tailwind CSS, Go.
- **Requirements:** 6+ years fullstack product engineering, SaaS conversion funnels, sub-50ms API response times.
- **Scoring Weights:** Skills: 35%, Experience: 25%, Semantic Match: 25%, Education: 15%. Threshold: 60.0%.
- **Matching Candidate:** **Henrik Lindqvist** (`henrik.lindqvist@example.com`)
  - **Resume Highlights:** 8 years fullstack lead at Klarna ecosystem; M.Sc. KTH; built high-throughput Next.js and Go systems.
  - **Screening Result:** **90.5%** (Skills: 0.92, Experience: 0.90, Semantic: 0.90, Education: 0.90) ➔ **Passed**.
  - **Application Stage:** `interview_invited`
  - **Candidate Portal Magic Link:** `http://localhost:5173/candidate/portal?token=demo-magic-token-henrik-2026`
  - **PDF Resume File:** `fixtures/cvs/henrik_lindqvist_cv.pdf`

---

### 7. Senior Data & Streaming Pipeline Engineer
- **Job ID:** `11111111-2222-3333-4444-555555555504`
- **Domain & Tech:** Apache Kafka, PySpark, Python, Snowflake, dbt, SQL, Airflow, Debezium CDC.
- **Requirements:** 5+ years engineering streaming pipelines, dimensional data modeling, and schema anomaly governance.
- **Scoring Weights:** Skills: 35%, Experience: 30%, Semantic Match: 25%, Education: 10%. Threshold: 65.0%.
- **Matching Candidate:** **Priya Sharma** (`priya.sharma@example.com`)
  - **Resume Highlights:** 6 years senior data engineer at Grab; built Kafka streaming pipelines processing 25k msg/sec into Snowflake.
  - **Screening Result:** **88.5%** (Skills: 0.90, Experience: 0.88, Semantic: 0.88, Education: 0.85) ➔ **Passed**.
  - **Application Stage:** `screening_passed`
  - **Candidate Portal Magic Link:** `http://localhost:5173/candidate/portal?token=demo-magic-token-priya-2026`
  - **PDF Resume File:** `fixtures/cvs/priya_sharma_cv.pdf`

---

### 8. Lead Application Security & DevSecOps Engineer
- **Job ID:** `11111111-2222-3333-4444-555555555505`
- **Domain & Tech:** AppSec, OWASP, Penetration Testing, IAM, SOC 2 Type II, Python, Go, Docker Hardening.
- **Requirements:** 6+ years in application security, container sandboxing defenses, zero-trust architectures; OSCP/CISSP certified.
- **Scoring Weights:** Skills: 35%, Experience: 25%, Semantic: 20%, Certifications: 20%. Threshold: 70.0%.
- **Matching Candidate:** **Zachary Taylor** (`zachary.taylor@example.com`)
  - **Resume Highlights:** 7 years security lead at DefenseScale; OSCP & CISSP; hardened code execution sandboxes and achieved SOC 2 Type II.
  - **Screening Result:** **95.0%** (Skills: 0.98, Experience: 0.92, Semantic: 0.92, Certifications: 1.0) ➔ **Passed**.
  - **Application Stage:** `offer_extended`
  - **Candidate Portal Magic Link:** `http://localhost:5173/candidate/portal?token=demo-magic-token-zachary-2026`
  - **PDF Resume File:** `fixtures/cvs/zachary_taylor_cv.pdf`

---

### 9. Senior SDET & Test Automation Architect
- **Job ID:** `11111111-2222-3333-4444-555555555506`
- **Domain & Tech:** Playwright, TypeScript, CI/CD, Docker, Performance Testing (k6), Vitest.
- **Requirements:** 5+ years architecting test automation frameworks, parallel worker state isolation, and load testing gates.
- **Scoring Weights:** Skills: 35%, Experience: 30%, Semantic Match: 25%, Education: 10%. Threshold: 65.0%.
- **Matching Candidate:** **Chloe Dubois** (`chloe.dubois@example.com`)
  - **Resume Highlights:** 6 years principal quality engineer at Spotify tooling; author of hermetic Playwright CI test harness.
  - **Screening Result:** **92.0%** (Skills: 0.95, Experience: 0.90, Semantic: 0.90, Education: 0.90) ➔ **Passed**.
  - **Application Stage:** `interview_completed`
  - **Completed Assessment Score:** **94.0 / 100** (Test Isolation: 96%, CI/CD Gating: 94%).
  - **Candidate Portal Magic Link:** `http://localhost:5173/candidate/portal?token=demo-magic-token-chloe-2026`
  - **PDF Resume File:** `fixtures/cvs/chloe_dubois_cv.pdf`

---

### 10. Engineering Manager / Technical Lead
- **Job ID:** `11111111-2222-3333-4444-555555555507`
- **Domain & Tech:** Engineering Management, System Design, 1-on-1 Coaching, Agile, Technical Hiring, Architecture RFCs.
- **Requirements:** 8+ years total software engineering with 3+ years managing squads; high-trust team leadership.
- **Scoring Weights:** Experience: 40%, Skills: 30%, Semantic Match: 20%, Education: 10%. Threshold: 70.0%.
- **Matching Candidate:** **Samuel Okafor** (`samuel.okafor@example.com`)
  - **Resume Highlights:** 10 years experience; Engineering Manager at Stripe Infrastructure; M.S. MIT; 0% voluntary attrition over 3 years.
  - **Screening Result:** **96.0%** (Experience: 0.98, Skills: 0.95, Semantic: 0.94, Education: 0.95) ➔ **Passed**.
  - **Application Stage:** `hired`
  - **Candidate Portal Magic Link:** `http://localhost:5173/candidate/portal?token=demo-magic-token-samuel-2026`
  - **PDF Resume File:** `fixtures/cvs/samuel_okafor_cv.pdf`

---

### 11. Junior Fullstack Engineer (Contract)
- **Job ID:** `11111111-2222-3333-4444-555555555508`
- **Domain & Tech:** TypeScript, React, Node.js, PostgreSQL, Git, Tailwind CSS.
- **Requirements:** 1+ years experience creating modern web applications, clean code habits, and fast learning agility.
- **Scoring Weights:** Skills: 40%, Semantic Match: 30%, Education: 20%, Experience: 10%. Threshold: 55.0%.
- **Matching Candidate:** **Jordan Brooks** (`jordan.brooks@example.com`)
  - **Resume Highlights:** 2 years junior developer at Venture Studio; B.S. UT Austin; built React UI components and Express/Postgres APIs.
  - **Screening Result:** **85.0%** (Skills: 0.88, Semantic: 0.85, Education: 0.85, Experience: 0.80) ➔ **Passed**.
  - **Application Stage:** `screening_passed`
  - **Candidate Portal Magic Link:** `http://localhost:5173/candidate/portal?token=demo-magic-token-jordan-2026`
  - **PDF Resume File:** `fixtures/cvs/jordan_brooks_cv.pdf`

---

### 12. AI Evaluation & Prompt Engineering Intern (Internship)
- **Job ID:** `11111111-2222-3333-4444-555555555509`
- **Domain & Tech:** Python, Prompt Engineering, LLM Evaluation, PyTorch, NLP, LangChain, Hugging Face.
- **Requirements:** Current CS/AI student or recent graduate; strong Python scripting and prompt evaluation rigor.
- **Scoring Weights:** Skills: 40%, Education: 30%, Semantic Match: 20%, Experience: 10%. Threshold: 50.0%.
- **Matching Candidate:** **Sophia Zhang** (`sophia.zhang@example.com`)
  - **Resume Highlights:** M.S. AI candidate at UC Berkeley; AI research assistant at BAIR; built prompt evaluation benchmark suites.
  - **Screening Result:** **88.0%** (Skills: 0.90, Education: 0.92, Semantic: 0.85, Experience: 0.75) ➔ **Passed**.
  - **Application Stage:** `interview_invited`
  - **Candidate Portal Magic Link:** `http://localhost:5173/candidate/portal?token=demo-magic-token-sophia-2026`
  - **PDF Resume File:** `fixtures/cvs/sophia_zhang_cv.pdf`

---

## 3. PDF Generation & Fixture Locations

All 13 resume PDFs are generated with ISO 32000-1 searchable text, professional layout hierarchy, and high-fidelity typography.

- **Primary Fixture Directory:** `fixtures/cvs/`
- **Seed Mirror Directory:** `scripts/seeds/demo/cvs/`
- **Automated Generator Script:** `frontend/generate_cv_pdfs.mjs`

To regenerate all PDF resumes at any time:
```bash
cd frontend && node generate_cv_pdfs.mjs
```

---

## 4. How to Run & Verify

### A. Apply Seed Data
```bash
# From repository root
make seed
# Or for a completely clean slate with fresh migrations:
make seed-fresh
```

### B. Access Recruiter Portal
- **URL:** `http://localhost:5173/login`
- **Organization Slug:** `demo`
- **Email:** `admin@demo.io` (or `recruiter@demo.io`)
- **Password:** `password123`
- **Candidates Screening Pool:** `http://localhost:5173/candidates`
- **Company Intelligence & Rails:** `http://localhost:5173/company-context`

### C. Access Candidate Portal (Any Seeded Candidate)
Candidates authenticate seamlessly via Email OTP or Magic Link:
- **Candidate Portal Login:** `http://localhost:5173/candidate/portal`
- **Demo OTP Code (for all seeded candidates):** `123456`
- **Direct Magic Link Example:**
  - Alex Rivera: `http://localhost:5173/candidate/portal?token=demo-magic-token-alex-2026`
  - Elena Rostova: `http://localhost:5173/candidate/portal?token=demo-magic-token-elena-2026`
  - David Chen: `http://localhost:5173/candidate/portal?token=demo-magic-token-david-2026`
