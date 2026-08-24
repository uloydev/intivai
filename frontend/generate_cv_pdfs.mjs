import fs from 'node:fs';
import path from 'node:path';
import { chromium } from 'playwright';

const candidates = [
  {
    filename: 'alex_rivera_cv.pdf',
    name: 'Alex Rivera',
    title: 'Staff Software Engineer & Distributed Systems Architect',
    email: 'alex.rivera@example.com',
    phone: '+1 (555) 234-5678',
    location: 'Seattle, WA (Remote)',
    linkedin: 'linkedin.com/in/alex-rivera-dist',
    github: 'github.com/alexrivera-go',
    summary: 'Seasoned distributed systems architect with 8+ years of production experience designing high-throughput microservices, event streaming architectures, and resilient databases using Go and PostgreSQL. Expert in Row-Level Security, concurrency optimization, and fault-tolerant background queues.',
    skills: [
      { category: 'Core Languages', items: ['Go (Golang)', 'SQL (PostgreSQL)', 'C/C++', 'Bash'] },
      { category: 'Distributed & Cloud', items: ['Kubernetes', 'Docker', 'Redis', 'Apache Kafka', 'gRPC', 'Asynq'] },
      { category: 'Database & Storage', items: ['PostgreSQL RLS', 'pgvector', 'Connection Pooling', 'Transaction Isolation'] },
      { category: 'Practices & Tooling', items: ['Distributed Tracing', 'CI/CD Pipelines', 'Race Condition Diagnostics', 'TDD'] },
    ],
    experience: [
      {
        role: 'Staff Software Engineer',
        company: 'CloudScale Technologies Inc.',
        period: '2021 – Present',
        location: 'Seattle, WA / Remote',
        highlights: [
          'Architected high-throughput distributed ingestion pipelines in Go processing 150k events/sec with sub-20ms p99 latency.',
          'Enforced multi-tenant PostgreSQL Row-Level Security (RLS) policies and transaction isolation across 45+ tenant databases.',
          'Designed fault-tolerant asynchronous queue workers using Redis and Asynq, cutting retry overhead by 40%.',
          'Eliminated race conditions and goroutine leaks across critical microservices, establishing strict concurrency linting in CI.'
        ]
      },
      {
        role: 'Senior Backend Engineer',
        company: 'Datastream Systems',
        period: '2018 – 2021',
        location: 'San Francisco, CA',
        highlights: [
          'Engineered gRPC microservices and RESTful HTTP APIs serving over 10 million daily active requests.',
          'Optimized PostgreSQL query plans, B-tree indexes, and connection pool sizing, improving database throughput by 65%.',
          'Authored hermetic integration testing harnesses for Go distributed transactions and Redis failover simulations.'
        ]
      }
    ],
    education: [
      { degree: 'B.S. in Computer Science', school: 'University of Washington', year: '2018', honors: 'Magna Cum Laude' }
    ],
    certifications: ['AWS Certified Solutions Architect – Professional']
  },
  {
    filename: 'elena_rostova_cv.pdf',
    name: 'Elena Rostova',
    title: 'Staff Frontend Architect & Design Systems Lead',
    email: 'elena.rostova@example.com',
    phone: '+1 (555) 345-6789',
    location: 'San Francisco, CA',
    linkedin: 'linkedin.com/in/elena-rostova-fe',
    github: 'github.com/erostova-ui',
    summary: 'Forward-thinking frontend architect with 7+ years of experience crafting ultra-responsive, accessible web applications and multi-brand design systems with React 19, TypeScript, and Tailwind CSS. Specialized in real-time streaming interfaces, WebSockets, and Monaco editor integration.',
    skills: [
      { category: 'Frontend Core', items: ['React 19', 'TypeScript', 'Next.js', 'Tailwind CSS', 'Vite', 'HTML5/CSS3'] },
      { category: 'State & Realtime', items: ['TanStack Query', 'WebSockets', 'WebRTC Audio/Video', 'Zustand', 'RxJS'] },
      { category: 'UI & Components', items: ['Radix UI Primitives', 'shadcn/ui', 'Monaco Code Editor', 'Design Tokens'] },
      { category: 'Quality & Perf', items: ['Vitest', 'Playwright', 'Core Web Vitals', 'Accessibility (WCAG 2.1 AA)'] }
    ],
    experience: [
      {
        role: 'Lead Frontend Architect',
        company: 'UI Labs Software',
        period: '2020 – Present',
        location: 'San Francisco, CA',
        highlights: [
          'Architected real-time AI interview and streaming telemetry interfaces with 60fps token streaming and sub-100ms latency.',
          'Built composable enterprise design system adopted across 8 product lines, reducing feature development time by 50%.',
          'Integrated browser-based Monaco code editor supporting multi-language syntax highlighting and live sandboxed output.',
          'Optimized bundle code-splitting and TanStack Query caching, improving Lighthouse Performance score from 68 to 98.'
        ]
      },
      {
        role: 'Senior UI Engineer',
        company: 'FlowInterface Studios',
        period: '2018 – 2020',
        location: 'New York, NY',
        highlights: [
          'Led migration of monolithic frontend to modular React and TypeScript architecture with 100% strict type safety.',
          'Implemented comprehensive WCAG 2.1 AA accessibility guidelines across core checkout and dashboard user journeys.'
        ]
      }
    ],
    education: [
      { degree: 'M.S. in Software Engineering', school: 'Carnegie Mellon University', year: '2019' },
      { degree: 'B.S. in Computer Science', school: 'University of Illinois Urbana-Champaign', year: '2017' }
    ],
    certifications: []
  },
  {
    filename: 'david_chen_cv.pdf',
    name: 'David Chen, Ph.D.',
    title: 'Principal AI & Machine Learning Systems Engineer',
    email: 'david.chen@example.com',
    phone: '+1 (555) 456-7890',
    location: 'New York, NY (Remote)',
    linkedin: 'linkedin.com/in/david-chen-ai',
    github: 'github.com/dchen-ml',
    summary: 'Distinguished ML systems engineer with 9+ years of research and production experience in real-time speech processing, high-scale LLM evaluation pipelines, and vector semantic retrieval. Expert in Whisper STT, Kokoro TTS, pgvector HNSW indexing, and anti-jailbreak prompt rails.',
    skills: [
      { category: 'AI & ML Frameworks', items: ['PyTorch', 'Transformers', 'ONNX Runtime', 'TensorRT', 'CUDA', 'Python'] },
      { category: 'Audio & Speech', items: ['Whisper STT', 'Kokoro TTS', 'WebRTC Media', 'Voice Activity Detection (VAD)'] },
      { category: 'Vector & LLM', items: ['pgvector (HNSW)', 'FastEmbed', 'Structured JSON Output', 'Prompt Injection Rails'] },
      { category: 'Infra & Systems', items: ['Docker ML Images', 'GPU Resource Pooling', 'Go/Python Interop', 'FastAPI'] }
    ],
    experience: [
      {
        role: 'Principal AI Engineer',
        company: 'Synthetix AI Systems',
        period: '2019 – Present',
        location: 'New York, NY',
        highlights: [
          'Designed sub-300ms real-time voice-to-voice interview streaming pipelines integrating Whisper STT and Kokoro TTS.',
          'Architected semantic vector search infrastructure in PostgreSQL using pgvector HNSW indexing across 2M+ candidate profiles.',
          'Engineered deterministic prompt injection defenses and structured JSON evaluation rails achieving 99.8% schema adherence.',
          'Authored CUDA kernel and ONNX optimizations reducing model inference latency by 3.5x on standard GPU instances.'
        ]
      },
      {
        role: 'Senior Machine Learning Researcher',
        company: 'DeepAcoustics Labs',
        period: '2016 – 2019',
        location: 'Palo Alto, CA',
        highlights: [
          'Published 4 peer-reviewed papers on acoustic feature representation and neural speech enhancement.',
          'Built distributed PyTorch training pipelines scaling acoustic models across 64-GPU clusters.'
        ]
      }
    ],
    education: [
      { degree: 'Ph.D. in Computer Science (AI & NLP)', school: 'Stanford University', year: '2017' },
      { degree: 'B.S. in Computer Engineering', school: 'Tsinghua University', year: '2013' }
    ],
    certifications: []
  },
  {
    filename: 'liam_oconnor_cv.pdf',
    name: 'Liam O\'Connor',
    title: 'Senior Cloud Platform & SRE Lead',
    email: 'liam.oconnor@example.com',
    phone: '+44 20 7946 0192',
    location: 'London, United Kingdom (Hybrid)',
    linkedin: 'linkedin.com/in/liam-oconnor-sre',
    github: 'github.com/liamoconnor-cloud',
    summary: 'Battle-tested Site Reliability Engineer with 6+ years of experience orchestrating global multi-region Kubernetes platforms, automated GitOps deployments, and high-uptime cloud infrastructure on AWS. Strong expertise in Terraform IaC, Prometheus metrics, and automated disaster recovery.',
    skills: [
      { category: 'Cloud & Orchestration', items: ['Kubernetes (EKS)', 'Docker', 'Terraform', 'AWS (VPC, RDS, IAM)', 'ArgoCD'] },
      { category: 'Observability & SRE', items: ['Prometheus', 'Grafana', 'OpenTelemetry', 'Alertmanager', 'SLO/SLI Dashboards'] },
      { category: 'Networking & Security', items: ['Cilium eBPF', 'mTLS', 'Cert-Manager', 'Vault', 'Cloudflare'] },
      { category: 'Languages & Automation', items: ['Golang', 'Python', 'Bash Scripting', 'GitHub Actions CI/CD'] }
    ],
    experience: [
      {
        role: 'Senior SRE Lead',
        company: 'FinTech Cloud UK Ltd.',
        period: '2021 – Present',
        location: 'London, UK',
        highlights: [
          'Managed 12 multi-region Kubernetes production clusters maintaining 99.99% service availability.',
          'Standardized infrastructure provisioning via Terraform and ArgoCD GitOps, reducing environment spin-up time to under 15 mins.',
          'Implemented unified telemetry and custom Prometheus exporters across 60+ microservices.',
          'Conducted quarterly chaos engineering exercises and architected automated multi-region failover routines.'
        ]
      },
      {
        role: 'Cloud Infrastructure Engineer',
        company: 'GlobalScale Networks',
        period: '2019 – 2021',
        location: 'London, UK',
        highlights: [
          'Migrated legacy on-premise application workloads into containerized AWS EKS clusters.',
          'Engineered automated zero-downtime rolling upgrades and PodDisruptionBudgets.'
        ]
      }
    ],
    education: [
      { degree: 'B.Sc. in Computer Networks & Systems', school: 'Imperial College London', year: '2019', honors: 'First Class Honours' }
    ],
    certifications: [
      'Certified Kubernetes Administrator (CKA)',
      'Certified Kubernetes Security Specialist (CKS)',
      'AWS Solutions Architect – Professional'
    ]
  },
  {
    filename: 'maya_patel_cv.pdf',
    name: 'Maya Patel',
    title: 'Staff Mobile Engineer (React Native & iOS)',
    email: 'maya.patel@example.com',
    phone: '+1 (555) 678-9012',
    location: 'New York, NY (Hybrid)',
    linkedin: 'linkedin.com/in/maya-patel-mobile',
    github: 'github.com/mayapatel-app',
    summary: 'Staff mobile architect with 7+ years of experience crafting high-performance, offline-first mobile applications for iOS and Android. Deep expertise in React Native, Swift native modules, WebRTC audio/video streaming, and 60fps frame rate guarantees.',
    skills: [
      { category: 'Mobile Frameworks', items: ['React Native', 'Swift (iOS)', 'Kotlin (Android)', 'Fabric & TurboModules'] },
      { category: 'State & Realtime', items: ['Redux Toolkit', 'SQLite Offline Sync', 'WebRTC Mobile', 'WebSockets'] },
      { category: 'Performance & Profiling', items: ['Xcode Instruments', 'Systrace', 'Memory Leak Diagnostics', 'Sub-1s Cold Starts'] },
      { category: 'Tooling & CI/CD', items: ['Fastlane', 'Detox E2E Testing', 'CocoaPods', 'App Store / Google Play Publishing'] }
    ],
    experience: [
      {
        role: 'Staff Mobile Architect',
        company: 'Nomad Health Technologies',
        period: '2020 – Present',
        location: 'New York, NY',
        highlights: [
          'Architected flagship mobile candidate app with 500k+ active users, maintaining 4.9-star rating on App Store.',
          'Engineered custom Swift C++ native audio bridge delivering real-time WebRTC audio recording with zero audio dropouts.',
          'Built offline-first sync engine using SQLite and background worker queues, ensuring zero data loss in poor connectivity.',
          'Eliminated JS thread bottlenecks and reduced memory consumption by 45% via TurboModules.'
        ]
      },
      {
        role: 'Lead iOS Developer',
        company: 'SwiftPulse Media',
        period: '2018 – 2020',
        location: 'San Francisco, CA',
        highlights: [
          'Developed native iOS video capturing and streaming components using AVFoundation and CoreAudio.',
          'Automated beta release pipelines using Fastlane, cutting deployment cycles from 4 hours to 10 minutes.'
        ]
      }
    ],
    education: [
      { degree: 'B.S. in Electrical Engineering & Computer Sciences', school: 'UC Berkeley', year: '2018' }
    ],
    certifications: []
  },
  {
    filename: 'henrik_lindqvist_cv.pdf',
    name: 'Henrik Lindqvist',
    title: 'Lead Full-Stack Product Engineer',
    email: 'henrik.lindqvist@example.com',
    phone: '+49 30 1234567',
    location: 'Berlin, Germany (Remote EU)',
    linkedin: 'linkedin.com/in/henrik-lindqvist-dev',
    github: 'github.com/hlindqvist-fullstack',
    summary: 'Versatile full-stack product engineer with 8+ years of experience delivering revenue-critical SaaS web platforms, rich UI workflows, and low-latency APIs. Master of Next.js, Node.js, TypeScript, PostgreSQL, and Go microservices.',
    skills: [
      { category: 'Fullstack Web', items: ['Next.js 15', 'React', 'Node.js', 'TypeScript', 'GraphQL', 'REST APIs'] },
      { category: 'Backend & DB', items: ['PostgreSQL', 'Golang', 'Prisma ORM', 'Redis Caching', 'Database Migrations'] },
      { category: 'UI & Styling', items: ['Tailwind CSS', 'Radix UI', 'Framer Motion', 'Responsive Web Design'] },
      { category: 'Engineering Ops', items: ['Docker', 'Vitest / Jest', 'GitHub Actions', 'Security (OWASP, CORS/CSRF)'] }
    ],
    experience: [
      {
        role: 'Principal Product Engineer',
        company: 'Klarna Ecosystem Ventures',
        period: '2019 – Present',
        location: 'Berlin, Germany',
        highlights: [
          'Led cross-functional team of 6 engineers shipping end-to-end recruitment and checkout workflows in Next.js and Go.',
          'Architected GraphQL and RESTful APIs serving 20k requests/minute with sub-50ms p99 response times.',
          'Optimized server-side rendering (SSR) and PostgreSQL indexing, driving 35% improvement in conversion funnels.',
          'Introduced strict TypeScript typing standards across fullstack monorepo, reducing production runtime exceptions by 80%.'
        ]
      },
      {
        role: 'Senior Fullstack Engineer',
        company: 'Spotify Web Experience',
        period: '2017 – 2019',
        location: 'Stockholm, Sweden',
        highlights: [
          'Developed high-traffic creator analytics dashboards and responsive playlist curation tools.',
          'Implemented comprehensive unit and integration test suites achieving 85%+ code coverage.'
        ]
      }
    ],
    education: [
      { degree: 'M.Sc. in Computer Science', school: 'KTH Royal Institute of Technology', year: '2017' },
      { degree: 'B.Sc. in Information Technology', school: 'Uppsala University', year: '2015' }
    ],
    certifications: []
  },
  {
    filename: 'priya_sharma_cv.pdf',
    name: 'Priya Sharma',
    title: 'Senior Data & Streaming Pipeline Engineer',
    email: 'priya.sharma@example.com',
    phone: '+65 6789 0123',
    location: 'Singapore (Hybrid)',
    linkedin: 'linkedin.com/in/priya-sharma-data',
    github: 'github.com/priyasharma-data',
    summary: 'Data platform engineer with 6+ years of expertise in building enterprise streaming pipelines, CDC event ingestion, and dimensional analytics warehouses with Apache Kafka, PySpark, Snowflake, and dbt. Passionate about data quality and lineage.',
    skills: [
      { category: 'Data Streaming', items: ['Apache Kafka', 'Kafka Connect', 'Debezium CDC', 'Spark Streaming', 'Flink'] },
      { category: 'Warehouse & Modeling', items: ['Snowflake', 'dbt', 'SQL (Advanced)', 'ClickHouse', 'Dimensional Modeling'] },
      { category: 'Orchestration & Code', items: ['Python (PySpark, Pandas)', 'Apache Airflow', 'DataHub Lineage', 'Go'] },
      { category: 'Cloud Infrastructure', items: ['AWS (S3, EMR, Redshift)', 'Docker', 'Terraform', 'Data Quality Testing'] }
    ],
    experience: [
      {
        role: 'Senior Data Platform Engineer',
        company: 'Grab Data Systems',
        period: '2020 – Present',
        location: 'Singapore',
        highlights: [
          'Architected enterprise Kafka event streaming bus processing over 25,000 telemetry messages/second.',
          'Engineered automated dbt transformation models in Snowflake, providing real-time recruiting analytics to 200+ HR stakeholders.',
          'Built robust CDC data ingestion pipelines with Debezium from transactional PostgreSQL to Snowflake warehouse.',
          'Enforced data contracts and automated schema anomaly testing with Airflow, reducing data pipeline outages by 90%.'
        ]
      },
      {
        role: 'Big Data Engineer',
        company: 'Shopee E-Commerce',
        period: '2018 – 2020',
        location: 'Singapore',
        highlights: [
          'Maintained large-scale PySpark ETL batch pipelines processing 5TB+ daily transaction logs on AWS EMR.',
          'Optimized SQL query performance and data partitioning strategies, reducing monthly cloud computing costs by $45,000.'
        ]
      }
    ],
    education: [
      { degree: 'B.Eng. in Computer Science', school: 'National University of Singapore (NUS)', year: '2019', honors: 'First Class Honours' }
    ],
    certifications: ['Snowflake Certified SnowPro Core', 'AWS Certified Data Analytics – Specialty']
  },
  {
    filename: 'zachary_taylor_cv.pdf',
    name: 'Zachary Taylor',
    title: 'Lead Application Security & DevSecOps Engineer',
    email: 'zachary.taylor@example.com',
    phone: '+1 (555) 789-0123',
    location: 'Atlanta, GA (Remote US/EU)',
    linkedin: 'linkedin.com/in/zachary-taylor-sec',
    github: 'github.com/ztaylor-appsec',
    summary: 'Lead security engineer with 7+ years of experience specializing in application security, penetration testing, container runtime hardening, and DevSecOps. Proven record securing multi-tenant cloud platforms, achieving SOC 2 Type II compliance, and preventing sandbox escapes.',
    skills: [
      { category: 'Application Security', items: ['OWASP Top 10', 'Threat Modeling', 'Code Auditing', 'OAuth2 / OIDC', 'mTLS'] },
      { category: 'Container & Sandbox', items: ['Docker Security Hardening', 'Linux Namespaces & cgroups', 'gVisor', 'Seccomp/AppArmor'] },
      { category: 'Offensive & Testing', items: ['Penetration Testing', 'Burp Suite Pro', 'SAST/DAST Automation', 'Vulnerability Assessment'] },
      { category: 'Languages & Compliance', items: ['Go', 'Python', 'SOC 2 Type II', 'ISO 27001', 'CI/CD Security Gates'] }
    ],
    experience: [
      {
        role: 'Head of Product Security',
        company: 'DefenseScale Cloud Security',
        period: '2021 – Present',
        location: 'Atlanta, GA / Remote',
        highlights: [
          'Hardened remote code execution Docker sandbox runners with strict cgroups, read-only root filesystems, and seccomp filters.',
          'Architected zero-trust IAM authentication and OAuth2 token verification layers for distributed microservices.',
          'Integrated automated Semgrep SAST and Trivy container vulnerability scanning into GitHub Actions CI pipeline.',
          'Led end-to-end audit and evidence collection to achieve SOC 2 Type II and ISO 27001 compliance with zero findings.'
        ]
      },
      {
        role: 'Senior Security Consultant',
        company: 'NCC Group',
        period: '2018 – 2021',
        location: 'Austin, TX',
        highlights: [
          'Conducted black-box and white-box penetration testing for 30+ enterprise web and API platforms.',
          'Discovered and remediated critical business logic flaws, SSRF vulnerabilities, and authentication bypasses.'
        ]
      }
    ],
    education: [
      { degree: 'B.S. in Cybersecurity & Network Systems', school: 'Georgia Institute of Technology', year: '2018' }
    ],
    certifications: [
      'Offensive Security Certified Professional (OSCP)',
      'Certified Information Systems Security Professional (CISSP)',
      'Certified Cloud Security Professional (CCSP)'
    ]
  },
  {
    filename: 'chloe_dubois_cv.pdf',
    name: 'Chloe Dubois',
    title: 'Senior SDET & Test Automation Architect',
    email: 'chloe.dubois@example.com',
    phone: '+1 (555) 890-1234',
    location: 'Montreal, QC (Remote Global)',
    linkedin: 'linkedin.com/in/chloe-dubois-sdet',
    github: 'github.com/cdubois-qa',
    summary: 'Quality engineering architect with 6+ years designing robust, flakiness-free automated test frameworks with Playwright, TypeScript, and Vitest. Specialized in hermetic test state isolation, parallel worker execution, and automated k6 performance load gating.',
    skills: [
      { category: 'E2E & Automation', items: ['Playwright', 'TypeScript', 'Cypress', 'Page Object Model', 'Parallel Isolation'] },
      { category: 'Load & Performance', items: ['k6 Load Testing', 'Artillery', 'WebSocket Concurrency', 'Resource Leak Profiling'] },
      { category: 'CI/CD & Fixtures', items: ['GitHub Actions CI', 'Docker Test Fixtures', 'Hermetic DB Seeding', 'Vitest'] },
      { category: 'Programming & API', items: ['TypeScript', 'Golang', 'Python', 'REST / WebSocket Contract Testing'] }
    ],
    experience: [
      {
        role: 'Principal Quality Engineer',
        company: 'Spotify Developer Tooling',
        period: '2021 – Present',
        location: 'Montreal / Remote',
        highlights: [
          'Architected company-wide Playwright test automation suite covering 150+ end-to-end user journeys with 99.7% pass stability.',
          'Designed isolated test database seeding harnesses, eliminating inter-test data contamination in parallel CI runs.',
          'Integrated automated k6 performance benchmarking into pull request CI gates, preventing latency regressions.',
          'Established Test-Driven Development (TDD) best practices across frontend and backend engineering teams.'
        ]
      },
      {
        role: 'Senior QA Automation Engineer',
        company: 'Deezer Music Services',
        period: '2018 – 2021',
        location: 'Paris, France',
        highlights: [
          'Built automated cross-browser testing pipelines for web player and mobile responsive layouts.',
          'Reduced regression testing execution time from 6 hours to 18 minutes using containerized parallel runners.'
        ]
      }
    ],
    education: [
      { degree: 'M.S. in Software Quality Engineering', school: 'Sorbonne Université', year: '2019' },
      { degree: 'B.S. in Computer Science', school: 'Université de Montréal', year: '2017' }
    ],
    certifications: ['ISTQB Certified Tester Advanced Level (CTAL-TA)']
  },
  {
    filename: 'samuel_okafor_cv.pdf',
    name: 'Samuel Okafor',
    title: 'Engineering Manager & Technical Lead',
    email: 'samuel.okafor@example.com',
    phone: '+1 (555) 901-2345',
    location: 'San Francisco, CA (Hybrid)',
    linkedin: 'linkedin.com/in/samuel-okafor-em',
    github: 'github.com/sokafor-lead',
    summary: 'Empathetic and results-driven engineering leader with 10+ years of software engineering experience (including 4+ years managing high-performance squads). Expert at steering technical strategy, architecting distributed cloud systems, scaling engineering culture, and mentoring staff engineers.',
    skills: [
      { category: 'Leadership & Strategy', items: ['People Management (1-on-1s)', 'Career Growth Paths', 'Engineering Hiring', 'OKRs'] },
      { category: 'Technical Governance', items: ['System Design Reviews', 'Architecture RFCs', 'Technical Debt Prioritization', 'Agile/Scrum'] },
      { category: 'Engineering Domains', items: ['Distributed Systems', 'Cloud Native (AWS/K8s)', 'Microservices in Go/Java', 'PostgreSQL'] },
      { category: 'Culture & Operations', items: ['High-Trust Engineering', 'Incident Retrospectives', 'Cross-Functional Collaboration'] }
    ],
    experience: [
      {
        role: 'Engineering Manager',
        company: 'Stripe Infrastructure Engineering',
        period: '2020 – Present',
        location: 'San Francisco, CA',
        highlights: [
          'Managed and mentored a high-performing distributed squad of 12 backend and infrastructure engineers.',
          'Delivered multi-region active-active database migration on schedule with zero downtime for payment processing.',
          'Revamped technical hiring rubrics and interview loops, improving team diversity and candidate offer acceptance to 88%.',
          'Achieved 0% voluntary turnover across 3 consecutive years while scaling team velocity and sprint deliverables.'
        ]
      },
      {
        role: 'Staff Software Engineer & Tech Lead',
        company: 'Square Financial Technologies',
        period: '2015 – 2020',
        location: 'San Francisco, CA',
        highlights: [
          'Led architecture of core settlement engine processing $10B+ annual merchant payment volumes.',
          'Mentored 8 mid-level engineers to senior and tech lead promotions.'
        ]
      }
    ],
    education: [
      { degree: 'M.S. in Engineering Management & CS', school: 'Massachusetts Institute of Technology (MIT)', year: '2015' },
      { degree: 'B.S. in Computer Science', school: 'University of Michigan', year: '2013' }
    ],
    certifications: []
  },
  {
    filename: 'jordan_brooks_cv.pdf',
    name: 'Jordan Brooks',
    title: 'Junior Fullstack Engineer',
    email: 'jordan.brooks@example.com',
    phone: '+1 (555) 012-3456',
    location: 'Austin, TX (Remote)',
    linkedin: 'linkedin.com/in/jordan-brooks-dev',
    github: 'github.com/jordanbrooks-code',
    summary: 'Passionate junior fullstack developer with 2 years of practical experience creating modern web applications using React, TypeScript, Node.js, and PostgreSQL. Eager learner with strong clean code fundamentals and active open-source contributions.',
    skills: [
      { category: 'Frontend', items: ['TypeScript', 'JavaScript (ES6+)', 'React', 'HTML5 & CSS3', 'Tailwind CSS'] },
      { category: 'Backend', items: ['Node.js', 'Express', 'PostgreSQL', 'Prisma ORM', 'REST APIs'] },
      { category: 'Tools & Workflow', items: ['Git & GitHub', 'Docker Basics', 'npm / pnpm', 'Postman', 'VS Code'] }
    ],
    experience: [
      {
        role: 'Junior Fullstack Developer',
        company: 'Venture Studio Inc.',
        period: '2023 – Present',
        location: 'Austin, TX / Remote',
        highlights: [
          'Built responsive React UI components and form validation workflows used by 15,000+ monthly active users.',
          'Developed Node.js and Express RESTful API endpoints connected to PostgreSQL database using Prisma ORM.',
          'Wrote automated unit tests with Vitest and participated in daily standups and peer code reviews.',
          'Fixed 40+ user-reported UI bugs and accessibility issues, improving user satisfaction ratings.'
        ]
      },
      {
        role: 'Web Development Intern',
        company: 'TechLab Solutions',
        period: '2022 – 2023',
        location: 'Austin, TX',
        highlights: [
          'Assisted senior engineers in refactoring JavaScript codebase to strict TypeScript.',
          'Designed landing pages using modern Tailwind CSS and mobile-first responsive layouts.'
        ]
      }
    ],
    education: [
      { degree: 'B.S. in Software Engineering', school: 'University of Texas at Austin', year: '2023' }
    ],
    certifications: []
  },
  {
    filename: 'sophia_zhang_cv.pdf',
    name: 'Sophia Zhang',
    title: 'AI Evaluation & Prompt Engineering Intern',
    email: 'sophia.zhang@example.com',
    phone: '+1 (555) 123-4567',
    location: 'Berkeley, CA (San Francisco Bay Area)',
    linkedin: 'linkedin.com/in/sophia-zhang-ai',
    github: 'github.com/sophiazhang-nlp',
    summary: 'Graduate AI researcher and prospective intern with strong background in automated prompt evaluation, LLM safety rails, and Python NLP workflows. Researching synthetic data generation, anti-jailbreak defenses, and structured JSON generation at UC Berkeley.',
    skills: [
      { category: 'AI & NLP', items: ['Python', 'PyTorch', 'Prompt Engineering', 'LLM Evaluation', 'Hugging Face Transformers'] },
      { category: 'Tools & Frameworks', items: ['LangChain', 'LlamaIndex', 'Pandas', 'NumPy', 'FastAPI', 'JSON Schema'] },
      { category: 'Evaluation Metrics', items: ['Semantic Similarity', 'BLEU/ROUGE', 'Hallucination Detection', 'Prompt Injection Testing'] }
    ],
    experience: [
      {
        role: 'AI Research Assistant',
        company: 'Berkeley AI Research (BAIR) Lab',
        period: '2024 – Present',
        location: 'Berkeley, CA',
        highlights: [
          'Developed benchmark evaluation suite testing LLM candidate assessment consistency across 1,000+ prompt permutations.',
          'Researched adversarial prompt injection vectors and formulated defensive system prompt rails with 99.4% mitigation rate.',
          'Synthesized diverse domain-specific resume test fixtures to evaluate CV extraction accuracy and rubric scoring models.'
        ]
      },
      {
        role: 'Machine Learning Intern',
        company: 'PromptTech Innovations',
        period: 'Summer 2024',
        location: 'San Francisco, CA',
        highlights: [
          'Built automated Python scripts to evaluate LLM structured JSON output compliance against complex Pydantic schemas.',
          'Curated training datasets and fine-tuned lightweight classification models for candidate skill tag extraction.'
        ]
      }
    ],
    education: [
      { degree: 'M.S. in Artificial Intelligence & Robotics', school: 'UC Berkeley', year: 'Expected 2025' },
      { degree: 'B.S. in Data Science & Mathematics', school: 'UCLA', year: '2023', honors: 'Summa Cum Laude' }
    ],
    certifications: []
  },
  {
    filename: 'marcus_vance_cv.pdf',
    name: 'Marcus Vance',
    title: 'Junior Web Developer',
    email: 'marcus.vance@example.com',
    phone: '+1 (555) 567-8901',
    location: 'Denver, CO',
    linkedin: 'linkedin.com/in/marcus-vance-web',
    github: 'github.com/marcusvance',
    summary: 'Aspiring web developer with 1 year of foundational experience in Python, HTML, CSS, and basic JavaScript. Passionate about learning modern software engineering methodologies.',
    skills: [
      { category: 'Web Basics', items: ['Python', 'HTML5', 'CSS3', 'Basic JavaScript', 'Git'] }
    ],
    experience: [
      {
        role: 'Junior Web Intern',
        company: 'Local Creative Agency',
        period: '2024 – Present',
        location: 'Denver, CO',
        highlights: [
          'Updated HTML/CSS templates for local business websites.',
          'Wrote simple Python automation scripts for data file conversion.'
        ]
      }
    ],
    education: [
      { degree: 'B.A. in Information Systems', school: 'State College', year: '2024' }
    ],
    certifications: []
  }
];

function generateHTML(candidate) {
  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>${candidate.name} - Resume</title>
  <style>
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
      color: #1e293b;
      background: #ffffff;
      line-height: 1.45;
      font-size: 10.5pt;
      padding: 36px 48px;
    }
    header {
      border-bottom: 2px solid #3b82f6;
      padding-bottom: 12px;
      margin-bottom: 16px;
    }
    h1 {
      font-size: 22pt;
      font-weight: 700;
      color: #0f172a;
      letter-spacing: -0.5px;
      margin-bottom: 3px;
    }
    .headline {
      font-size: 12pt;
      font-weight: 600;
      color: #2563eb;
      margin-bottom: 6px;
    }
    .contact-row {
      display: flex;
      flex-wrap: wrap;
      gap: 14px;
      font-size: 9pt;
      color: #64748b;
    }
    .contact-item { display: inline-flex; align-items: center; gap: 4px; }
    section { margin-bottom: 16px; }
    h2 {
      font-size: 11pt;
      font-weight: 700;
      text-transform: uppercase;
      letter-spacing: 0.8px;
      color: #1e3a8a;
      border-bottom: 1px solid #e2e8f0;
      padding-bottom: 3px;
      margin-bottom: 8px;
    }
    .summary-text {
      font-size: 10pt;
      color: #334155;
      line-height: 1.5;
    }
    .skills-grid {
      display: grid;
      grid-template-columns: 1fr;
      gap: 5px;
    }
    .skill-category {
      font-size: 9.5pt;
      color: #334155;
    }
    .skill-label {
      font-weight: 600;
      color: #0f172a;
    }
    .job-entry {
      margin-bottom: 12px;
    }
    .job-header {
      display: flex;
      justify-content: space-between;
      align-items: baseline;
      margin-bottom: 2px;
    }
    .job-title {
      font-size: 10.5pt;
      font-weight: 700;
      color: #0f172a;
    }
    .job-company {
      font-size: 10pt;
      font-weight: 600;
      color: #2563eb;
    }
    .job-meta {
      font-size: 9pt;
      color: #64748b;
      font-weight: 500;
    }
    ul.highlights {
      margin-left: 16px;
      margin-top: 4px;
      font-size: 9.5pt;
      color: #334155;
    }
    ul.highlights li {
      margin-bottom: 3px;
    }
    .edu-entry {
      display: flex;
      justify-content: space-between;
      align-items: baseline;
      margin-bottom: 4px;
      font-size: 9.5pt;
    }
    .edu-degree { font-weight: 700; color: #0f172a; }
    .edu-school { color: #475569; font-weight: 500; }
    .cert-item {
      font-size: 9.5pt;
      color: #334155;
      margin-bottom: 2px;
    }
  </style>
</head>
<body>
  <header>
    <h1>${candidate.name}</h1>
    <div class="headline">${candidate.title}</div>
    <div class="contact-row">
      <span class="contact-item">📧 ${candidate.email}</span>
      <span class="contact-item">📞 ${candidate.phone}</span>
      <span class="contact-item">📍 ${candidate.location}</span>
      ${candidate.linkedin ? `<span class="contact-item">🔗 ${candidate.linkedin}</span>` : ''}
      ${candidate.github ? `<span class="contact-item">💻 ${candidate.github}</span>` : ''}
    </div>
  </header>

  <section>
    <h2>Professional Summary</h2>
    <p class="summary-text">${candidate.summary}</p>
  </section>

  <section>
    <h2>Core Technical Competencies</h2>
    <div class="skills-grid">
      ${candidate.skills.map(s => `
        <div class="skill-category">
          <span class="skill-label">${s.category}:</span> ${s.items.join(' • ')}
        </div>
      `).join('')}
    </div>
  </section>

  <section>
    <h2>Professional Experience</h2>
    ${candidate.experience.map(exp => `
      <div class="job-entry">
        <div class="job-header">
          <div>
            <span class="job-title">${exp.role}</span> — <span class="job-company">${exp.company}</span>
          </div>
          <div class="job-meta">${exp.period} | ${exp.location}</div>
        </div>
        <ul class="highlights">
          ${exp.highlights.map(h => `<li>${h}</li>`).join('')}
        </ul>
      </div>
    `).join('')}
  </section>

  <section>
    <h2>Education</h2>
    ${candidate.education.map(edu => `
      <div class="edu-entry">
        <div>
          <span class="edu-degree">${edu.degree}</span> — <span class="edu-school">${edu.school}</span>
          ${edu.honors ? ` <em>(${edu.honors})</em>` : ''}
        </div>
        <div class="job-meta">${edu.year}</div>
      </div>
    `).join('')}
  </section>

  ${candidate.certifications && candidate.certifications.length > 0 ? `
    <section>
      <h2>Certifications & Credentials</h2>
      ${candidate.certifications.map(c => `
        <div class="cert-item">🏆 <strong>${c}</strong></div>
      `).join('')}
    </section>
  ` : ''}
</body>
</html>`;
}

async function main() {
  console.log('🚀 Starting PDF generation for all candidates...');
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();

  const outDirs = [
    path.resolve(import.meta.dirname, '..', 'fixtures', 'cvs'),
    path.resolve(import.meta.dirname, '..', 'scripts', 'seeds', 'demo', 'cvs')
  ];

  for (const dir of outDirs) {
    if (!fs.existsSync(dir)) {
      fs.mkdirSync(dir, { recursive: true });
    }
  }

  for (const candidate of candidates) {
    console.log(`📄 Generating: ${candidate.filename} (${candidate.name})...`);
    const html = generateHTML(candidate);
    await page.setContent(html, { waitUntil: 'load' });

    for (const dir of outDirs) {
      const outPath = path.join(dir, candidate.filename);
      await page.pdf({
        path: outPath,
        format: 'A4',
        printBackground: true,
        margin: { top: '20px', bottom: '20px', left: '20px', right: '20px' }
      });
    }
  }

  await browser.close();
  console.log(`✅ Successfully generated ${candidates.length} PDF resumes in fixtures/cvs and scripts/seeds/demo/cvs!`);
}

main().catch(err => {
  console.error('❌ Error generating PDFs:', err);
  process.exit(1);
});
