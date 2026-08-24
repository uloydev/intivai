-- 03_jobs.sql
-- Seed Core Engineering Job Requisitions with Rubric & Specs

-- Job 1: Senior Distributed Systems Engineer
INSERT INTO jobs (
    id, org_id, title, description, location, employment_type,
    salary_min, salary_max, currency, required_skills, min_experience,
    responsibilities, requirements, nice_to_haves, benefits,
    scoring_weights, min_score_to_proceed, status, created_at, updated_at,
    proctoring_mode, is_published, rubric
)
VALUES (
    'c3d4e5f6-a1b2-4c3d-8e4f-5a6b7c8d9e0f',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'Senior Distributed Systems Engineer',
    'Lead the design of high-throughput distributed microservices, event streaming pipelines, and fault-tolerant consensus mechanisms in Go and PostgreSQL.',
    'Remote (US / EU)',
    'Full-time',
    160000,
    210000,
    'USD',
    '["Go", "PostgreSQL", "Distributed Systems", "Docker", "Kubernetes", "Redis"]'::jsonb,
    5,
    '["Architect and maintain core distributed microservices handling millions of transactions.", "Design fault-tolerant event sourcing and async queue processing with Asynq and Redis.", "Optimize PostgreSQL queries, connection pooling, and multi-tenant RLS isolation.", "Lead technical design reviews and establish engineering standards across the team."]'::jsonb,
    '["5+ years of production experience building distributed systems in Go.", "Deep expertise in PostgreSQL (indexing, concurrency control, transaction isolation).", "Hands-on experience with containerization (Docker), orchestration (K8s), and CI/CD.", "Strong background in API design, gRPC, WebSockets, and distributed telemetry."]'::jsonb,
    '["Experience with WebRTC signaling or real-time audio streaming.", "Familiarity with pgvector and semantic search embeddings."]'::jsonb,
    '["Competitive salary & equity options", "100% remote flexibility with home office stipend", "Comprehensive health, dental, and vision insurance", "Unlimited PTO and annual learning budget ($3,000)"]'::jsonb,
    '{"skills_match": 0.35, "experience_years": 0.25, "semantic_match": 0.25, "education": 0.10, "certifications": 0.05}'::jsonb,
    60.0,
    'active',
    NOW() - INTERVAL '15 days',
    NOW() - INTERVAL '15 days',
    'optional',
    true,
    '{
        "summary": "Proven distributed systems engineer who excels at Go concurrency, transactional safety in PostgreSQL, and resilient async queue pipelines.",
        "dimensions": [
            {
                "name": "Technical Depth in Go",
                "description": "Deep understanding of Go runtime, goroutines, channels, atomic operations, and memory synchronization primitives.",
                "weight_percentage": 35,
                "criteria": ["5+ years Go experience", "Production race debugging", "Memory leak diagnostics"]
            },
            {
                "name": "Database & Transaction Isolation",
                "description": "Mastery of PostgreSQL indexing, multi-tenancy RLS, locking semantics, and query optimization.",
                "weight_percentage": 25,
                "criteria": ["PostgreSQL RLS architecture", "Connection pooling resilience", "Safe migration patterns"]
            },
            {
                "name": "Distributed System Design",
                "description": "Ability to design fault-tolerant consensus, message queues, and horizontal scaling strategies.",
                "weight_percentage": 25,
                "criteria": ["Event sourcing with Redis/Asynq", "Graceful degradation", "Telemetry and distributed tracing"]
            },
            {
                "name": "Engineering Leadership",
                "description": "Clarity of communication, code review rigor, and mentoring capability.",
                "weight_percentage": 15,
                "criteria": ["Clean architectural trade-off justification", "Cross-functional collaboration"]
            }
        ]
    }'::jsonb
)
ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    status = EXCLUDED.status,
    proctoring_mode = EXCLUDED.proctoring_mode,
    is_published = EXCLUDED.is_published,
    rubric = EXCLUDED.rubric;

-- Job 2: Staff Frontend Architect
INSERT INTO jobs (
    id, org_id, title, description, location, employment_type,
    salary_min, salary_max, currency, required_skills, min_experience,
    responsibilities, requirements, nice_to_haves, benefits,
    scoring_weights, min_score_to_proceed, status, created_at, updated_at,
    proctoring_mode, is_published, rubric
)
VALUES (
    'd4e5f6a1-b2c3-4d4e-8f5a-6b7c8d9e0f1a',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'Staff Frontend Architect',
    'Architect modern, ultra-responsive web applications using React 19, TypeScript, Tailwind CSS, and WebSockets with strict performance SLAs.',
    'San Francisco, CA / Remote',
    'Full-time',
    175000,
    230000,
    'USD',
    '["React", "TypeScript", "Tailwind CSS", "WebSockets", "Architecture"]'::jsonb,
    6,
    '["Architect component design systems and state management for real-time interview interfaces.", "Build live streaming token visualizers, WebRTC audio monitors, and Monaco code editors.", "Guarantee 60fps animations and sub-100ms UI interaction latencies across all browsers.", "Mentor frontend engineers and champion clean design systems."]'::jsonb,
    '["6+ years building complex, high-performance web applications with React and TypeScript.", "Deep understanding of browser DOM performance, WebSockets, and WebRTC streaming.", "Mastery of modern CSS (Tailwind, animations, responsive layouts, dark modes).", "Track record of creating reusable, accessible design systems."]'::jsonb,
    '["Experience with Monaco Editor integration or browser-based IDEs.", "Contributions to open-source UI libraries."]'::jsonb,
    '["Top-tier compensation package + high-growth equity", "Health, dental, vision coverage & 401(k) matching", "Flexible working hours and remote-first setup"]'::jsonb,
    '{"skills_match": 0.35, "experience_years": 0.30, "semantic_match": 0.25, "education": 0.10}'::jsonb,
    65.0,
    'active',
    NOW() - INTERVAL '12 days',
    NOW() - INTERVAL '12 days',
    'strict',
    true,
    '{
        "summary": "Frontend architect with exceptional design sensibilities, deep React 19 / TypeScript expertise, and real-time streaming proficiency.",
        "dimensions": [
            {
                "name": "Component & Design Systems",
                "description": "Design token architecture, accessibility (WCAG 2.1 AA), and composable component primitives.",
                "weight_percentage": 35,
                "criteria": ["Mastery of Radix/shadcn and Tailwind", "Design token hierarchy", "Strict type safety"]
            },
            {
                "name": "Real-Time & Streaming UI",
                "description": "Handling high-frequency WebSocket frames, token streaming reducers, and Monaco editor integration.",
                "weight_percentage": 30,
                "criteria": ["WebSocket state machines", "Smooth 60fps UI streaming", "Sub-100ms interaction latency"]
            },
            {
                "name": "Web Performance & Core Vitals",
                "description": "Code-splitting, memory leak prevention in long-lived sessions, and bundle optimization.",
                "weight_percentage": 20,
                "criteria": ["Route-level code splitting", "TanStack Query cache management", "DOM rendering optimization"]
            },
            {
                "name": "Engineering Leadership",
                "description": "Establishing frontend engineering standards, mentorship, and cross-discipline collaboration.",
                "weight_percentage": 15,
                "criteria": ["Clear technical documentation", "Reviewing frontend architecture RFCs"]
            }
        ]
    }'::jsonb
)
ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    status = EXCLUDED.status,
    proctoring_mode = EXCLUDED.proctoring_mode,
    is_published = EXCLUDED.is_published,
    rubric = EXCLUDED.rubric;

-- Job 3: Principal AI & ML Systems Engineer
INSERT INTO jobs (
    id, org_id, title, description, location, employment_type,
    salary_min, salary_max, currency, required_skills, min_experience,
    responsibilities, requirements, nice_to_haves, benefits,
    scoring_weights, min_score_to_proceed, status, created_at, updated_at,
    proctoring_mode, is_published, rubric
)
VALUES (
    'e5f6a1b2-c3d4-4e5f-8a6b-7c8d9e0f1a2b',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'Principal AI & ML Systems Engineer',
    'Build high-scale inference engines, real-time audio WebRTC pipelines, and vector retrieval infrastructure for intelligent voice agents.',
    'New York, NY / Remote',
    'Full-time',
    190000,
    250000,
    'USD',
    '["Python", "PyTorch", "WebRTC", "pgvector", "LLM", "Whisper"]'::jsonb,
    7,
    '["Design real-time voice-to-voice interview synthesis pipelines with sub-300ms latency.", "Train, fine-tune, and optimize Whisper STT and Kokoro TTS models for technical evaluation.", "Build vector semantic search infrastructure using PostgreSQL pgvector and HNSW indexing.", "Implement deterministic prompt injection guardrails and anti-cheating anomaly detection."]'::jsonb,
    '["7+ years engineering ML/AI systems in production with Python and PyTorch.", "Expertise in LLM inference, structured output generation, and prompt safety rails.", "Experience optimizing audio processing models (STT/TTS) for real-time WebRTC streams.", "Strong understanding of vector databases, embeddings, and cosine similarity ranking."]'::jsonb,
    '["Publications or open-source projects in speech processing or LLM evaluation.", "Experience with ONNX runtime, TensorRT, or CUDA kernel optimization."]'::jsonb,
    '["Top-of-market base salary + equity package", "High-end workstation hardware (M4 Max / RTX 4090)", "Comprehensive premium medical coverage"]'::jsonb,
    '{"skills_match": 0.40, "experience_years": 0.25, "semantic_match": 0.25, "education": 0.10}'::jsonb,
    70.0,
    'active',
    NOW() - INTERVAL '10 days',
    NOW() - INTERVAL '10 days',
    'none',
    false,
    '{
        "summary": "Principal ML engineer leading real-time speech processing, high-scale LLM evaluation pipelines, and vector memory retrieval.",
        "dimensions": [
            {
                "name": "LLM Systems & Structured Inference",
                "description": "Designing deterministic LLM evaluation pipelines, prompt safety rails, and structured JSON schemas.",
                "weight_percentage": 40,
                "criteria": ["JSON Schema enforcement", "Anti-injection defenses", "Token cost and latency budgeting"]
            },
            {
                "name": "Real-Time Audio & WebRTC",
                "description": "Low-latency audio streaming, Whisper STT optimization, and Voice Activity Detection (VAD).",
                "weight_percentage": 30,
                "criteria": ["Sub-300ms speech pipelines", "Audio anomaly detection", "WebRTC media pipelines"]
            },
            {
                "name": "Vector Retrieval & Memory",
                "description": "High-dimensional embedding recall, HNSW index tuning, and semantic similarity scoring in pgvector.",
                "weight_percentage": 20,
                "criteria": ["pgvector HNSW tuning", "FastEmbed embedding models", "Cosine distance ranking"]
            },
            {
                "name": "System Architecture & Scalability",
                "description": "Containerized ML deployment, GPU resource management, and resilience under high concurrent load.",
                "weight_percentage": 10,
                "criteria": ["Production ML serving", "Fault-tolerant architecture"]
            }
        ]
    }'::jsonb
)
ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    status = EXCLUDED.status,
    proctoring_mode = EXCLUDED.proctoring_mode,
    is_published = EXCLUDED.is_published,
    rubric = EXCLUDED.rubric;

-- Job 4: Senior Cloud Platform & SRE (London, GBP, Hybrid)
INSERT INTO jobs (
    id, org_id, title, description, location, employment_type,
    salary_min, salary_max, currency, required_skills, min_experience,
    responsibilities, requirements, nice_to_haves, benefits,
    scoring_weights, min_score_to_proceed, status, created_at, updated_at,
    proctoring_mode, is_published, rubric
)
VALUES (
    '11111111-2222-3333-4444-555555555501',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'Senior Cloud Platform & SRE',
    'Design, automate, and scale global multi-cluster Kubernetes infrastructure, Terraform IaC, and 99.99% SLA observability systems.',
    'London, UK / Hybrid',
    'Full-time',
    95000,
    130000,
    'GBP',
    '["Kubernetes", "Terraform", "AWS", "Prometheus", "Docker", "Golang", "CI/CD"]'::jsonb,
    5,
    '["Maintain multi-region AWS and bare-metal Kubernetes clusters with GitOps.", "Build automated failover, disaster recovery, and chaos engineering testing.", "Implement unified telemetry, alerting, and distributed tracing with Prometheus & Grafana.", "Partner with product engineering to optimize container build times and runtime isolation."]'::jsonb,
    '["5+ years managing high-availability cloud infrastructure in production.", "Deep expertise with Kubernetes orchestration, CNI, Helm, and GitOps (ArgoCD).", "Mastery of Terraform, AWS services (EKS, VPC, RDS), and Linux kernel internals.", "Proficiency writing operational automation tools in Go or Python."]'::jsonb,
    '["Certified Kubernetes Administrator (CKA) or Security Specialist (CKS).", "Experience with Cilium eBPF networking."]'::jsonb,
    '["Competitive base + UK pension match", "Private medical and dental coverage", "28 days paid holiday + bank holidays", "Hybrid flexibility (2 days office, 3 days home)"]'::jsonb,
    '{"skills_match": 0.35, "experience_years": 0.30, "semantic_match": 0.20, "certifications": 0.15}'::jsonb,
    65.0,
    'active',
    NOW() - INTERVAL '9 days',
    NOW() - INTERVAL '9 days',
    'optional',
    true,
    '{"summary": "Production-hardened SRE who excels at Kubernetes reliability, automated IaC, and high-uptime incident management.", "dimensions": [{"name": "Kubernetes & Cloud Infrastructure", "weight_percentage": 40, "description": "Cluster management, networking, and security."}, {"name": "Observability & Incident Response", "weight_percentage": 30, "description": "Prometheus metrics, SLOs, and root-cause postmortems."}, {"name": "Automation & Tooling", "weight_percentage": 30, "description": "Terraform, Go CLI tools, and CI/CD pipelines."}]}'::jsonb
)
ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    status = EXCLUDED.status,
    proctoring_mode = EXCLUDED.proctoring_mode,
    is_published = EXCLUDED.is_published,
    rubric = EXCLUDED.rubric;

-- Job 5: Staff Mobile Engineer - React Native & iOS (New York, USD, Hybrid)
INSERT INTO jobs (
    id, org_id, title, description, location, employment_type,
    salary_min, salary_max, currency, required_skills, min_experience,
    responsibilities, requirements, nice_to_haves, benefits,
    scoring_weights, min_score_to_proceed, status, created_at, updated_at,
    proctoring_mode, is_published, rubric
)
VALUES (
    '11111111-2222-3333-4444-555555555502',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'Staff Mobile Engineer - React Native & iOS',
    'Architect butter-smooth, offline-first mobile experiences and native bridge modules for iOS and Android interview apps.',
    'New York, NY / Hybrid',
    'Full-time',
    165000,
    215000,
    'USD',
    '["React Native", "TypeScript", "Swift", "iOS", "Android", "Performance", "Offline Sync"]'::jsonb,
    6,
    '["Lead architecture of real-time mobile candidate and proctoring client apps.", "Bridge native iOS Swift and Android Kotlin audio/video modules into React Native.", "Ensure 60fps animations, zero memory leaks, and sub-second cold starts on mobile.", "Establish mobile CI/CD automated test pipelines using Fastlane and Detox."]'::jsonb,
    '["6+ years building consumer or enterprise mobile applications.", "Deep expertise in React Native, TypeScript, and native iOS Swift development.", "Strong background in native thread bridging, WebRTC audio/video, and memory profiling.", "Proven track record publishing top-rated apps on App Store and Google Play."]'::jsonb,
    '["Experience with WebRTC audio streaming on iOS/Android.", "Familiarity with TurboModules and Fabric architecture."]'::jsonb,
    '["Top-tier salary + equity package", "Full medical, vision, dental coverage", "Commuter benefits & 401(k) match", "Annual $2,500 wellness & equipment stipend"]'::jsonb,
    '{"skills_match": 0.35, "experience_years": 0.30, "semantic_match": 0.25, "education": 0.10}'::jsonb,
    65.0,
    'active',
    NOW() - INTERVAL '8 days',
    NOW() - INTERVAL '8 days',
    'strict',
    true,
    '{"summary": "Staff mobile engineer with deep native iOS Swift and React Native bridging expertise.", "dimensions": [{"name": "Mobile Architecture & Bridging", "weight_percentage": 40, "description": "Native modules, React Native architecture, state management."}, {"name": "Performance & 60fps UI", "weight_percentage": 30, "description": "Memory leak debugging, frame drops, cold start optimization."}, {"name": "Native iOS & Android", "weight_percentage": 30, "description": "Swift, Kotlin, permissions, background execution."}]}'::jsonb
)
ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    status = EXCLUDED.status,
    proctoring_mode = EXCLUDED.proctoring_mode,
    is_published = EXCLUDED.is_published,
    rubric = EXCLUDED.rubric;

-- Job 6: Lead Full-Stack Product Engineer (Berlin, EUR, Remote EU)
INSERT INTO jobs (
    id, org_id, title, description, location, employment_type,
    salary_min, salary_max, currency, required_skills, min_experience,
    responsibilities, requirements, nice_to_haves, benefits,
    scoring_weights, min_score_to_proceed, status, created_at, updated_at,
    proctoring_mode, is_published, rubric
)
VALUES (
    '11111111-2222-3333-4444-555555555503',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'Lead Full-Stack Product Engineer',
    'Drive end-to-end recruiter workflows, candidate funnel interfaces, and high-performance server-rendered web platforms in Next.js and Go.',
    'Berlin, Germany / Remote (EU)',
    'Full-time',
    90000,
    125000,
    'EUR',
    '["Next.js", "Node.js", "TypeScript", "PostgreSQL", "GraphQL", "Tailwind CSS", "Go"]'::jsonb,
    6,
    '["Ship polished product features from database models to responsive frontend UI.", "Design GraphQL and REST APIs with sub-50ms p99 latency.", "Partner closely with design and product teams to refine user onboarding funnels.", "Mentor fullstack developers and champion strict type safety."]'::jsonb,
    '["6+ years of fullstack product engineering experience.", "Deep proficiency in TypeScript, Next.js, Node.js, and PostgreSQL.", "Track record of delivering revenue-generating SaaS features with great UX.", "Strong understanding of web security, CSRF/CORS, and state management."]'::jsonb,
    '["Experience with Tailwind CSS and Radix UI.", "Background in Golang microservices."]'::jsonb,
    '["Competitive European salary with equity participation", "30 days annual leave", "Work from anywhere in EU policy", "Urban Sports Club membership"]'::jsonb,
    '{"skills_match": 0.35, "experience_years": 0.25, "semantic_match": 0.25, "education": 0.15}'::jsonb,
    60.0,
    'active',
    NOW() - INTERVAL '7 days',
    NOW() - INTERVAL '7 days',
    'optional',
    true,
    '{"summary": "Product engineer who translates user empathy into fast, robust, beautifully crafted web software.", "dimensions": [{"name": "Fullstack Architecture", "weight_percentage": 40, "description": "Next.js, API design, PostgreSQL queries."}, {"name": "Product Polish & UX", "weight_percentage": 35, "description": "UI accessibility, responsive design, animations."}, {"name": "Code Rigor & Types", "weight_percentage": 25, "description": "Type safety, testing, maintainability."}]}'::jsonb
)
ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    status = EXCLUDED.status,
    proctoring_mode = EXCLUDED.proctoring_mode,
    is_published = EXCLUDED.is_published,
    rubric = EXCLUDED.rubric;

-- Job 7: Senior Data & Streaming Pipeline Engineer (Singapore, SGD, Hybrid)
INSERT INTO jobs (
    id, org_id, title, description, location, employment_type,
    salary_min, salary_max, currency, required_skills, min_experience,
    responsibilities, requirements, nice_to_haves, benefits,
    scoring_weights, min_score_to_proceed, status, created_at, updated_at,
    proctoring_mode, is_published, rubric
)
VALUES (
    '11111111-2222-3333-4444-555555555504',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'Senior Data & Streaming Pipeline Engineer',
    'Build real-time event streaming architectures, Kafka pipelines, and analytic warehouses processing millions of candidate telemetry data points.',
    'Singapore / Hybrid',
    'Full-time',
    130000,
    175000,
    'SGD',
    '["Apache Kafka", "Spark", "Python", "Snowflake", "dbt", "SQL", "Airflow"]'::jsonb,
    5,
    '["Design high-throughput Kafka streaming pipelines and CDC ingestion feeds.", "Build robust PySpark and dbt data transformations in Snowflake warehouse.", "Ensure data quality, lineage, and schema governance across recruitment analytics.", "Optimize query performance for executive dashboard reporting."]'::jsonb,
    '["5+ years engineering data pipelines and distributed data warehouses.", "Strong expertise with Apache Kafka, PySpark, Python, and SQL.", "Deep experience with Snowflake, dbt, Airflow, and dimensional data modeling.", "Solid understanding of streaming vs batch processing trade-offs."]'::jsonb,
    '["Experience with ClickHouse or Apache Iceberg.", "AWS Data Analytics certification."]'::jsonb,
    '["Competitive Singapore market salary + annual performance bonus", "Comprehensive health insurance", "Central CBD office with flexible hybrid schedule", "Learning and conference budget"]'::jsonb,
    '{"skills_match": 0.35, "experience_years": 0.30, "semantic_match": 0.25, "education": 0.10}'::jsonb,
    65.0,
    'active',
    NOW() - INTERVAL '6 days',
    NOW() - INTERVAL '6 days',
    'optional',
    true,
    '{"summary": "Data engineer who delivers dependable streaming pipelines, clean data models, and sub-second analytics.", "dimensions": [{"name": "Streaming & Event Ingestion", "weight_percentage": 40, "description": "Kafka, partition keys, CDC, consumer groups."}, {"name": "Data Modeling & SQL", "weight_percentage": 35, "description": "dbt, dimensional modeling, query tuning."}, {"name": "Data Quality & Reliability", "weight_percentage": 25, "description": "Testing, monitoring, schema drift resilience."}]}'::jsonb
)
ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    status = EXCLUDED.status,
    proctoring_mode = EXCLUDED.proctoring_mode,
    is_published = EXCLUDED.is_published,
    rubric = EXCLUDED.rubric;

-- Job 8: Lead Application Security & DevSecOps Engineer (Remote US/EU, USD)
INSERT INTO jobs (
    id, org_id, title, description, location, employment_type,
    salary_min, salary_max, currency, required_skills, min_experience,
    responsibilities, requirements, nice_to_haves, benefits,
    scoring_weights, min_score_to_proceed, status, created_at, updated_at,
    proctoring_mode, is_published, rubric
)
VALUES (
    '11111111-2222-3333-4444-555555555505',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'Lead Application Security & DevSecOps Engineer',
    'Enforce zero-trust security postures, conduct threat modeling, secure sandbox execution environments, and drive SOC 2 Type II compliance.',
    'Remote (US / EU)',
    'Full-time',
    170000,
    220000,
    'USD',
    '["AppSec", "OWASP", "Penetration Testing", "IAM", "SOC 2", "Python", "Go", "Docker Security"]'::jsonb,
    6,
    '["Audit application codebases, APIs, and WebSocket protocols for security vulnerabilities.", "Harden Docker code execution sandboxes against privilege escalation and breakout attacks.", "Implement automated SAST/DAST security scanning into CI/CD pipelines.", "Manage SOC 2 Type II compliance controls, penetration tests, and vendor security reviews."]'::jsonb,
    '["6+ years in application security, penetration testing, or DevSecOps.", "Deep knowledge of OWASP Top 10, cryptographic standards, and OAuth2/OIDC security.", "Hands-on experience securing container runtimes, Linux namespaces, and cgroups.", "Ability to code in Go or Python to build security automation tools."]'::jsonb,
    '["OSCP, CISSP, or GIAC security certifications.", "Experience with eBPF security auditing."]'::jsonb,
    '["High-tier compensation & equity options", "Comprehensive health/vision/dental plans", "Remote office setup budget", "Annual security conference and training stipend"]'::jsonb,
    '{"skills_match": 0.35, "experience_years": 0.25, "semantic_match": 0.20, "certifications": 0.20}'::jsonb,
    70.0,
    'active',
    NOW() - INTERVAL '5 days',
    NOW() - INTERVAL '5 days',
    'strict',
    true,
    '{"summary": "Security leader who protects multi-tenant cloud platforms, audits codebases, and hardens isolated execution sandboxes.", "dimensions": [{"name": "Application Security & Threat Modeling", "weight_percentage": 40, "description": "OWASP, vulnerability remediation, cryptographic defense."}, {"name": "Container & Sandbox Hardening", "weight_percentage": 35, "description": "Linux namespaces, cgroups, breakout prevention."}, {"name": "Compliance & Security Ops", "weight_percentage": 25, "description": "SOC 2, penetration testing, automated CI scanning."}]}'::jsonb
)
ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    status = EXCLUDED.status,
    proctoring_mode = EXCLUDED.proctoring_mode,
    is_published = EXCLUDED.is_published,
    rubric = EXCLUDED.rubric;

-- Job 9: Senior SDET & Test Automation Architect (Remote Worldwide, USD)
INSERT INTO jobs (
    id, org_id, title, description, location, employment_type,
    salary_min, salary_max, currency, required_skills, min_experience,
    responsibilities, requirements, nice_to_haves, benefits,
    scoring_weights, min_score_to_proceed, status, created_at, updated_at,
    proctoring_mode, is_published, rubric
)
VALUES (
    '11111111-2222-3333-4444-555555555506',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'Senior SDET & Test Automation Architect',
    'Architect automated E2E testing frameworks with Playwright, execute load tests with k6, and guarantee zero-regression CI/CD releases.',
    'Remote (Worldwide)',
    'Full-time',
    135000,
    175000,
    'USD',
    '["Playwright", "TypeScript", "CI/CD", "Docker", "Performance Testing", "Vitest", "k6"]'::jsonb,
    5,
    '["Architect end-to-end Playwright test suites covering complex realtime and WebRTC user journeys.", "Build automated load and stress testing pipelines with k6 and custom Go harnesses.", "Maintain hermetic, idempotent test database seeding and worker pipeline fixtures.", "Champion Test-Driven Development (TDD) across all backend and frontend teams."]'::jsonb,
    '["5+ years architecting test automation frameworks for web applications.", "Mastery of Playwright, TypeScript, and modern browser automation protocols.", "Strong background in CI/CD pipeline integration (GitHub Actions) and Docker test environments.", "Deep understanding of API contract testing, WebSocket load simulation, and race conditions."]'::jsonb,
    '["Experience testing real-time WebRTC or WebSocket streaming applications.", "Proficiency writing integration tests in Go."]'::jsonb,
    '["Global remote flexibility with competitive USD salary", "Full home office stipend", "Health insurance reimbursement", "Unlimited learning & book budget"]'::jsonb,
    '{"skills_match": 0.35, "experience_years": 0.30, "semantic_match": 0.25, "education": 0.10}'::jsonb,
    65.0,
    'active',
    NOW() - INTERVAL '4 days',
    NOW() - INTERVAL '4 days',
    'optional',
    true,
    '{"summary": "Test architect dedicated to rock-solid quality gates, fast flakiness-free E2E suites, and load testing.", "dimensions": [{"name": "E2E Framework Architecture", "weight_percentage": 40, "description": "Playwright, page objects, state isolation, flakiness eradication."}, {"name": "Performance & Load Testing", "weight_percentage": 30, "description": "k6, concurrency simulation, bottleneck diagnosis."}, {"name": "CI/CD & Developer Experience", "weight_percentage": 30, "description": "Fast feedback loops, hermetic fixtures, TDD culture."}]}'::jsonb
)
ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    status = EXCLUDED.status,
    proctoring_mode = EXCLUDED.proctoring_mode,
    is_published = EXCLUDED.is_published,
    rubric = EXCLUDED.rubric;

-- Job 10: Engineering Manager / Technical Lead (San Francisco, USD, Hybrid)
INSERT INTO jobs (
    id, org_id, title, description, location, employment_type,
    salary_min, salary_max, currency, required_skills, min_experience,
    responsibilities, requirements, nice_to_haves, benefits,
    scoring_weights, min_score_to_proceed, status, created_at, updated_at,
    proctoring_mode, is_published, rubric
)
VALUES (
    '11111111-2222-3333-4444-555555555507',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'Engineering Manager / Technical Lead',
    'Lead, coach, and grow high-performing engineering squads building autonomous AI recruitment technology while steering architectural decisions.',
    'San Francisco, CA / Hybrid',
    'Full-time',
    210000,
    265000,
    'USD',
    '["Engineering Management", "System Design", "Mentorship", "Agile", "Hiring", "Architecture"]'::jsonb,
    8,
    '["Manage, mentor, and foster career growth for a team of 8-12 software engineers.", "Collaborate with product and executive leadership to align technical roadmaps with business goals.", "Drive engineering excellence, robust code reviews, and high-velocity sprint cadence.", "Lead senior hiring interviews and cultivate an inclusive, high-trust engineering culture."]'::jsonb,
    '["8+ years total software engineering experience with 3+ years managing engineering teams.", "Strong technical foundation in distributed cloud systems, modern web, or AI pipelines.", "Demonstrated success recruiting, retaining, and developing top engineering talent.", "Exceptional communication, stakeholder alignment, and strategic planning skills."]'::jsonb,
    '["Experience leading teams at high-growth Series A-C SaaS startups.", "Background building AI-driven or real-time streaming products."]'::jsonb,
    '["Executive compensation package + significant equity grant", "Premium health, dental, and vision insurance", "401(k) matching & commuter benefits", "Executive coaching & leadership development budget"]'::jsonb,
    '{"experience_years": 0.40, "skills_match": 0.30, "semantic_match": 0.20, "education": 0.10}'::jsonb,
    70.0,
    'active',
    NOW() - INTERVAL '3 days',
    NOW() - INTERVAL '3 days',
    'none',
    true,
    '{"summary": "Engineering leader who builds autonomous, high-trust squads, drives technical vision, and develops people.", "dimensions": [{"name": "People Leadership & Coaching", "weight_percentage": 40, "description": "1-on-1s, retention, growth paths, conflict resolution."}, {"name": "Technical Strategy & Delivery", "weight_percentage": 35, "description": "System design reviews, roadmap execution, sprint velocity."}, {"name": "Hiring & Culture", "weight_percentage": 25, "description": "Bar raising, structured interviews, team empowerment."}]}'::jsonb
)
ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    status = EXCLUDED.status,
    proctoring_mode = EXCLUDED.proctoring_mode,
    is_published = EXCLUDED.is_published,
    rubric = EXCLUDED.rubric;

-- Job 11: Junior Fullstack Engineer (Remote Worldwide, USD, Contract)
INSERT INTO jobs (
    id, org_id, title, description, location, employment_type,
    salary_min, salary_max, currency, required_skills, min_experience,
    responsibilities, requirements, nice_to_haves, benefits,
    scoring_weights, min_score_to_proceed, status, created_at, updated_at,
    proctoring_mode, is_published, rubric
)
VALUES (
    '11111111-2222-3333-4444-555555555508',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'Junior Fullstack Engineer (Contract)',
    'Contribute to frontend UI components, bug fixes, and API integrations with close mentorship from senior architects.',
    'Remote (Worldwide)',
    'Contract',
    70000,
    90000,
    'USD',
    '["TypeScript", "React", "Node.js", "PostgreSQL", "Git"]'::jsonb,
    1,
    '["Build and test reusable React UI components under guidance from staff engineers.", "Implement REST API endpoints and database queries with TypeScript and PostgreSQL.", "Write unit tests for new features and assist with quality assurance verification.", "Participate in agile standups, sprint planning, and collaborative code reviews."]'::jsonb,
    '["1+ years experience building web applications with TypeScript, React, and Node.js.", "Solid understanding of JavaScript fundamentals, DOM APIs, and REST architecture.", "Familiarity with SQL databases, Git version control, and GitHub workflows.", "High curiosity, eager to learn modern software engineering best practices."]'::jsonb,
    '["Personal projects or open-source contributions on GitHub.", "Experience with Tailwind CSS."]'::jsonb,
    '["Competitive contract rate with opportunity to convert to full-time", "100% remote working flexibility", "Dedicated 1-on-1 mentorship from senior staff engineers"]'::jsonb,
    '{"skills_match": 0.40, "semantic_match": 0.30, "education": 0.20, "experience_years": 0.10}'::jsonb,
    55.0,
    'active',
    NOW() - INTERVAL '2 days',
    NOW() - INTERVAL '2 days',
    'optional',
    true,
    '{"summary": "Promising early-career engineer with strong TypeScript fundamentals, clean coding habits, and rapid learning capacity.", "dimensions": [{"name": "JavaScript & TypeScript Depth", "weight_percentage": 40, "description": "Language fundamentals, types, async/await."}, {"name": "React Component Building", "weight_percentage": 35, "description": "Hooks, props, state, clean markup."}, {"name": "Learning Agility", "weight_percentage": 25, "description": "Receptiveness to feedback, curiosity, problem solving."}]}'::jsonb
)
ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    status = EXCLUDED.status,
    proctoring_mode = EXCLUDED.proctoring_mode,
    is_published = EXCLUDED.is_published,
    rubric = EXCLUDED.rubric;

-- Job 12: AI Evaluation & Prompt Engineering Intern (San Francisco, USD, Internship)
INSERT INTO jobs (
    id, org_id, title, description, location, employment_type,
    salary_min, salary_max, currency, required_skills, min_experience,
    responsibilities, requirements, nice_to_haves, benefits,
    scoring_weights, min_score_to_proceed, status, created_at, updated_at,
    proctoring_mode, is_published, rubric
)
VALUES (
    '11111111-2222-3333-4444-555555555509',
    '968f66ef-91c6-4db3-8764-ceeffb753b1f',
    'AI Evaluation & Prompt Engineering Intern',
    'Conduct benchmark testing of LLM interview evaluation prompts, synthesize test datasets, and refine anti-hallucination rails.',
    'San Francisco, CA / On-site',
    'Internship',
    50000,
    65000,
    'USD',
    '["Python", "Prompt Engineering", "LLM Evaluation", "PyTorch", "NLP"]'::jsonb,
    0,
    '["Run automated evaluation benchmarks against LLM scoring and probing models.", "Curate synthetic CV and interview response datasets for edge-case evaluation testing.", "Analyze model hallucination rates and propose prompt refinement improvements.", "Work directly with AI research engineers on next-generation recruitment intelligence."]'::jsonb,
    '["Currently enrolled in or recent graduate of Computer Science, Data Science, or AI program.", "Strong Python programming skills and familiarity with LLM APIs (OpenAI, Anthropic, open-source models).", "Understanding of NLP metrics, prompt engineering techniques, and evaluation methods.", "Enthusiasm for building ethical, fair, and reliable AI systems."]'::jsonb,
    '["Coursework or research in Natural Language Processing or LLMs.", "Experience with LangChain, LlamaIndex, or Hugging Face."]'::jsonb,
    '["Competitive paid hourly internship rate ($50/hr)", "Hands-on experience deploying production AI systems", "Opportunity for full-time return offer upon graduation", "Daily catered lunches in downtown San Francisco office"]'::jsonb,
    '{"skills_match": 0.40, "education": 0.30, "semantic_match": 0.20, "experience_years": 0.10}'::jsonb,
    50.0,
    'active',
    NOW() - INTERVAL '1 day',
    NOW() - INTERVAL '1 day',
    'none',
    true,
    '{"summary": "Passionate AI/NLP student with strong analytical skills, Python scripting ability, and prompt evaluation rigor.", "dimensions": [{"name": "LLM & Prompt Understanding", "weight_percentage": 45, "description": "Prompt design, temperature, system constraints, token limits."}, {"name": "Python Scripting & Data Handling", "weight_percentage": 35, "description": "JSON parsing, automated evaluation scripts, pandas."}, {"name": "Analytical Rigor", "weight_percentage": 20, "description": "Experiment tracking, qualitative analysis, attention to detail."}]}'::jsonb
)
ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    status = EXCLUDED.status,
    proctoring_mode = EXCLUDED.proctoring_mode,
    is_published = EXCLUDED.is_published,
    rubric = EXCLUDED.rubric;

