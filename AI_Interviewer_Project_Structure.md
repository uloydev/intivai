# AI Interviewer — DDD + Hexagonal Architecture

## Domain Model

### Core Domains (Bounded Contexts)

```
/backend
├── cmd/
│   └── server/                    # Entry point (wiring + startup)
│       └── main.go
│
├── internal/
│   │
│   ├── cv/                        # Bounded Context: CV Management
│   │   ├── domain/                # 🧠 CORE — Enterprise business rules
│   │   │   ├── cv.go              # CV entity (AR)
│   │   │   ├── skill.go           # Value Object
│   │   │   ├── experience.go      # Value Object
│   │   │   ├── education.go       # Value Object
│   │   │   └── cv_repository.go   # Port (interface)
│   │   │
│   │   ├── application/           # 🧠 CORE — Use cases
│   │   │   ├── upload_cv.go       # UploadCV use case
│   │   │   ├── parse_cv.go        # ParseCV use case  
│   │   │   ├── extract_cv.go      # ExtractStructuredData use case
│   │   │   └── dto.go             # Input/output DTOs
│   │   │
│   │   ├── infrastructure/        # 🌐 Adapters (Driven)
│   │   │   ├── persistence/       # Database adapter
│   │   │   │   └── postgres_cv_repo.go
│   │   │   ├── parser/            # File parser adapter
│   │   │   │   └── pdfcpu_parser.go
│   │   │   ├── ocr/               # OCR adapter
│   │   │   │   └── tesseract_ocr.go
│   │   │   └── storage/           # File storage adapter
│   │   │       └── s3_storage.go
│   │   │
│   │   └── api/                   # 🌐 Adapters (Driving) — HTTP handlers
│   │       ├── upload_handler.go
│   │       ├── parse_handler.go
│   │       └── router.go
│   │
│   ├── job/                       # Bounded Context: Job Management
│   │   ├── domain/
│   │   │   ├── job.go             # Job entity (AR)
│   │   │   ├── skill_requirement.go
│   │   │   └── job_repository.go  # Port
│   │   │
│   │   ├── application/
│   │   │   ├── create_job.go
│   │   │   ├── update_job.go
│   │   │   └── dto.go
│   │   │
│   │   ├── infrastructure/
│   │   │   └── persistence/
│   │   │       └── postgres_job_repo.go
│   │   │
│   │   └── api/
│   │       └── job_handler.go
│   │
│   ├── screening/                 # Bounded Context: CV Scoring
│   │   ├── domain/
│   │   │   ├── screening.go       # Screening entity (AR)
│   │   │   ├── score.go           # Value Object (score + breakdown)
│   │   │   ├── scoring_weights.go # Value Object (per-tenant config)
│   │   │   └── screening_repository.go
│   │   │
│   │   ├── application/
│   │   │   ├── score_cv.go        # ScoreCV use case
│   │   │   ├── batch_score.go     # BatchScore use case
│   │   │   └── dto.go
│   │   │
│   │   ├── infrastructure/
│   │   │   ├── persistence/
│   │   │   │   └── postgres_screening_repo.go
│   │   │   └── embedding/         # Embedding adapter
│   │   │       └── fastembed_embedder.go  # fastembed lokal (bge-small, 384 dims)
│   │   │
│   │   └── api/
│   │       └── screening_handler.go
│   │
│   ├── context/                   # Bounded Context: Company Context & Tenant Prompt
│   │   ├── domain/
│   │   │   ├── company_context.go # Context entity (AR, versioned)
│   │   │   ├── tenant_prompt.go   # Tenant prompt entity (versioned)
│   │   │   ├── context_version.go # Value Object
│   │   │   └── context_repository.go
│   │   │
│   │   ├── application/
│   │   │   ├── upload_context.go  # UploadContext use case
│   │   │   ├── index_context.go   # IndexContext use case (chunk+summarize+Mnemosyne)
│   │   │   ├── set_tenant_prompt.go
│   │   │   ├── resolve_prompt.go  # ResolvePrompt with fallback
│   │   │   └── dto.go
│   │   │
│   │   ├── infrastructure/
│   │   │   ├── persistence/
│   │   │   │   └── postgres_context_repo.go
│   │   │   ├── storage/           # Raw file storage adapter
│   │   │   │   └── s3_context_storage.go
│   │   │   └── memory/            # Mnemosyne bank adapter
│   │   │       └── mnemosyne_context_index.go
│   │   │
│   │   └── api/
│   │       └── context_handler.go # POST /orgs/:id/contexts, PUT /orgs/:id/prompt
│   │
│   ├── memory/                   # Bounded Context: Semantic Memory (Mnemosyne adapter)
│   │   ├── domain/
│   │   │   ├── memory_port.go    # Port: MemoryBank (Remember/Recall/Reflect/QueryGraph/Forget/Stats)
│   │   │   └── memory_hit.go     # Value Object
│   │   │
│   │   ├── infrastructure/
│   │   │   ├── mcp/
│   │   │   │   └── mnemosyne_mcp.go  # Option A: Mnemosyne MCP (stdio) + ForBank(orgID)
│   │   │   └── native/
│   │   │       └── native_memory.go  # Option B (recommended): SQLite + fastembed + BM25 + LLM reflect
│   │   │
│   │   └── application/
│   │       ├── sync_worker.go    # Sync candidate/interview ke bank tenant
│   │       └── reflect.go        # Cross-interview reflect use case
│   │
│   ├── interview/                 # Bounded Context: Interview Engine
│   │   ├── domain/
│   │   │   ├── interview.go       # Interview entity (AR)
│   │   │   ├── question.go        # Value Object
│   │   │   ├── answer.go          # Value Object
│   │   │   ├── evaluation.go      # Value Object
│   │   │   ├── transcript.go      # Value Object
│   │   │   └── interview_repository.go
│   │   │
│   │   ├── application/
│   │   │   ├── start_interview.go
│   │   │   ├── submit_answer.go
│   │   │   ├── generate_questions.go
│   │   │   ├── evaluate_answer.go
│   │   │   ├── evaluate_interview.go
│   │   │   └── dto.go
│   │   │
│   │   ├── domain/service/        # Domain services
│   │   │   ├── question_generator.go   # Question strategy
│   │   │   ├── prompt_composer.go      # Compose system prompt: default + tenant + context + safety rails
│   │   │   ├── bias_detector.go        # Bias prevention
│   │   │   └── scoring_calculator.go   # Score aggregation
│   │   │
│   │   ├── infrastructure/
│   │   │   ├── persistence/
│   │   │   │   └── postgres_interview_repo.go
│   │   ├── stt/               # Speech-to-text adapter
│   │   │   └── whisper_provider.go   # whisper.cpp (self-hosted, free)
│   │   ├── tts/               # Text-to-speech adapter
│   │   │   ├── edge_tts_provider.go  # Edge TTS (free, no auth)
│   │   │   └── piper_provider.go     # Piper (local, fallback)
│   │   └── webrtc/            # WebRTC adapter
│   │       └── pion_adapter.go
│   │   │
│   │   └── api/
│   │       ├── chat_handler.go        # WebSocket (chat)
│   │       ├── voice_handler.go       # WebRTC (voice)
│   │       └── interview_handler.go   # REST endpoints
│   │
│   ├── evaluation/                # Bounded Context: Evaluation & Reports
│   │   ├── domain/
│   │   │   ├── report.go          # Report entity (AR)
│   │   │   ├── evaluation_criteria.go
│   │   │   └── report_repository.go
│   │   │
│   │   ├── application/
│   │   │   ├── generate_report.go
│   │   │   └── dto.go
│   │   │
│   │   ├── infrastructure/
│   │   │   └── persistence/
│   │   │       └── postgres_report_repo.go
│   │   │
│   │   └── api/
│   │       └── report_handler.go
│   │
│   ├── iam/                       # Bounded Context: Identity & Access
│   │   ├── domain/
│   │   │   ├── org.go             # Org entity (AR)
│   │   │   ├── user.go            # User entity
│   │   │   ├── api_key.go         # Value Object
│   │   │   ├── role.go            # Value Object (RBAC)
│   │   │   └── iam_repository.go
│   │   │
│   │   ├── application/
│   │   │   ├── register_org.go
│   │   │   ├── invite_user.go
│   │   │   ├── authenticate.go
│   │   │   ├── authorize.go
│   │   │   └── dto.go
│   │   │
│   │   ├── infrastructure/
│   │   │   ├── persistence/
│   │   │   │   └── postgres_iam_repo.go
│   │   │   └── auth/
│   │   │       ├── jwt_provider.go     # JWT adapter
│   │   │       └── api_key_hasher.go
│   │   │
│   │   └── api/
│   │       ├── auth_middleware.go      # JWT middleware
│   │       ├── tenant_middleware.go    # Tenant resolver
│   │       └── auth_handler.go
│   │
│   ├── llm/                        # Shared: LLM abstraction — ONE port, every context uses it
│   │   ├── provider.go             # Port: Chat, ChatStream, StructuredOutput, Embed, CountTokens
│   │   ├── client.go               # Retry + exponential backoff + fallback provider + metrics
│   │   └── deepseek_provider.go    # DeepSeek API (model: deepseek-chat)
│   │
│   ├── shared/                    # Shared kernel
│   │   ├── domain/
│   │   │   ├── entity.go          # Base entity (UUID, timestamps)
│   │   │   ├── aggregate_root.go
│   │   │   ├── value_object.go
│   │   │   └── event.go           # Domain events
│   │   │
│   │   └── errors/
│   │       ├── domain_error.go
│   │       └── not_found_error.go
│   │
│   └── pkg/                       # Shared infrastructure
│       ├── config/
│       │   └── config.go          # Viper config loader
│       ├── logger/
│       │   └── zerolog.go         # Logger setup
│       ├── db/
│       │   ├── postgres.go        # Connection pool
│       │   └── migrations/        # SQL migration files
│       ├── queue/
│       │   └── asynq.go           # Queue client setup
│       └── observability/
│           ├── metrics.go         # Prometheus metrics
│           └── health.go          # Health check handlers
│
├── go.mod
└── go.sum
```

## Hexagonal Architecture (Per Bounded Context)

Each bounded context follows Ports & Adapters pattern:

```
                    ┌──────────────────────┐
                    │    Driving Adapters   │
                    │  (HTTP, gRPC, CLI)    │
                    │      api/             │
                    └──────┬───────────────┘
                           │
                           ▼
                    ┌──────────────────────┐
                    │   Application Layer   │
                    │     (Use Cases)       │
                    │    application/       │
                    └──────┬───────────────┘
                           │
                           ▼
                    ┌──────────────────────┐
                    │    Domain Layer       │ ◄── CORE (no deps)
                    │   domain/ + service/  │
                    └──────┬───────────────┘
                           │
                           ▼
                    ┌──────────────────────┐
                    │   Driven Adapters    │
                    │ (DB, LLM, STT, TTS,  │
                    │  File Storage, OCR)  │
                    │  infrastructure/     │
                    └──────────────────────┘
```

### Dependency Rule

```
domain/ → nothing (zero external deps)
application/ → domain/
infrastructure/ → domain/ + application/ (via ports)
api/ → application/ (via DTOs)
```

## Ports (Interfaces) Defined

```go
// Domain ports — interfaces live IN the domain layer

// cv/domain/cv_repository.go
type CVRepository interface {
    Save(ctx context.Context, cv *CV) error
    FindByID(ctx context.Context, id uuid.UUID) (*CV, error)
    FindByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*CV, error)
    Delete(ctx context.Context, id uuid.UUID) error
}

// interview/domain/interview_repository.go
type InterviewRepository interface {
    Save(ctx context.Context, interview *Interview) error
    FindByID(ctx context.Context, id uuid.UUID) (*Interview, error)
    FindPending(ctx context.Context, orgID uuid.UUID) ([]*Interview, error)
    UpdateStatus(ctx context.Context, id uuid.UUID, status InterviewStatus) error
    SaveAnswer(ctx context.Context, interviewID uuid.UUID, answer *Answer) error
    SaveEvaluation(ctx context.Context, interviewID uuid.UUID, eval *Evaluation) error
    SaveQuestion(ctx context.Context, q *Question) error   // persist generated question ke bank (reuse + audit)
}

// Application ports — interfaces in application layer (driven side)

// LLM: ONE shared port in internal/llm/provider.go — every context (cv extraction,
// interview chat, evaluation) uses the same port. Contexts do NOT define their own LLMProvider.
type LLMProvider interface {
    Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
    ChatStream(ctx context.Context, req ChatRequest) (<-chan string, error)
    StructuredOutput(ctx context.Context, req StructuredRequest) (any, error)
    Embed(ctx context.Context, text string) ([]float32, error)
    CountTokens(text string) int
}

// interview/application/stt_provider.go
type STTProvider interface {
    Transcribe(ctx context.Context, audio io.Reader) (string, error)
    TranscribeStream(ctx context.Context, audio <-chan []byte) (<-chan string, error)
}

// interview/application/tts_provider.go
type TTSProvider interface {
    Synthesize(ctx context.Context, text string) ([]byte, error)
    SynthesizeStream(ctx context.Context, text <-chan string) (<-chan []byte, error)
}

// screening/application/embedder.go
type Embedder interface {
    Embed(ctx context.Context, text string) ([]float32, error)
}

// cv/application/file_parser.go
type FileParser interface {
    Parse(ctx context.Context, reader io.Reader) (string, error)  // raw text
}

// cv/application/ocr.go
type OCRProvider interface {
    ExtractText(ctx context.Context, image io.Reader) (string, error)
}

// cv/application/storage.go
type FileStorage interface {
    Upload(ctx context.Context, path string, reader io.Reader) error
    Download(ctx context.Context, path string) (io.ReadCloser, error)
    Delete(ctx context.Context, path string) error
}
```

## Example Flow: Upload CV

```
POST /api/v1/cvs (multipart file)
  │
  ▼
cv/api/upload_handler.go
  │  Validates file type & size
  │  Creates UploadCV DTO
  │
  ▼
cv/application/upload_cv.go
  │ 1. Generate CV ID
  │ 2. Upload file to FileStorage
  │ 3. Enqueue ParseCV job (async)
  │ 4. Return CV ID + status "processing"
  │
  ▼
Queue (asynq)
  │
  ▼
cv/application/parse_cv.go (worker)
  │ 1. Download file from FileStorage
  │ 2. Parse via FileParser (pdfcpu)
  │ 3. If scanned → OCRProvider (Tesseract)
  │ 4. Save raw text to CVRepository
  │ 5. Enqueue ExtractStructuredData job
  │
  ▼
Queue (asynq)
  │
  ▼
cv/application/extract_cv.go (worker)
  │ 1. Get raw text from CVRepository
  │ 2. Call LLMProvider.StructuredOutput() → ResumeData
  │ 3. Call Embedder.Embed() → vector
  │ 4. Save structured data + embedding to CVRepository
  │ 5. Update status → "parsed"
```

## Wiring (Dependency Injection)

```go
// cmd/server/main.go
func main() {
    // Infrastructure
    db := postgres.NewPool(cfg)
    redis := redis.NewClient(cfg)
    queue := asynq.NewServer(redis)
    fileStorage := s3.NewStorage(cfg)
    llm := llm.NewClient(
        llmdeepseek.NewProvider(cfg.DeepSeekAPIKey), // primary: deepseek-chat
        llmfallback.NewProvider(cfg),                // secondary OpenAI-compatible (OpenRouter/dll)
    )
    stt := whisper.NewProvider()        // whisper.cpp self-hosted (free)
    tts := edgetts.NewProvider()        // fallback: piper
    embedder := fastembed.NewEmbedder() // lokal bge-small, 384 dims, $0
    memory := native.NewMemoryAdapter(cfg, embedder) // or mcp.NewMnemosyneAdapter(...) — same port
    fileParser := pdfcpu.NewParser()
    ocr := tesseract.NewOCR()

    // Repositories (driven adapters)
    cvRepo := postgrescvrepo.New(db)
    jobRepo := postgresjobrepo.New(db)
    screeningRepo := postgresscreeningrepo.New(db)
    interviewRepo := postgresinterviewrepo.New(db)
    iamRepo := postgresiamrepo.New(db)

    // Application services
    uploadCV := cvapp.NewUploadCV(cvRepo, fileStorage, queue)
    parseCV := cvapp.NewParseCV(cvRepo, fileStorage, fileParser, ocr, llm, embedder, queue)
    scoreCV := screeningapp.NewScoreCV(cvRepo, jobRepo, screeningRepo, embedder)
    startInterview := interviewapp.NewStartInterview(interviewRepo, jobRepo, cvRepo, llm)
    submitAnswer := interviewapp.NewSubmitAnswer(interviewRepo, llm)

    // HTTP handlers (driving adapters)
    cvHandler := cvapi.NewUploadHandler(uploadCV)
    interviewHandler := interviewapi.NewHandler(startInterview, submitAnswer)

    // Fiber app
    app := fiber.New()
    app.Post("/api/v1/cvs", authMiddleware, cvHandler.Upload)
    app.Get("/api/v1/interviews/:id", authMiddleware, interviewHandler.Get)
    app.Post("/api/v1/interviews", authMiddleware, interviewHandler.Start)
    // ...
}
```

## Key DDD Rules Enforced

| Rule | How |
|------|-----|
| **Domain has zero dependencies** | No imports from `infrastructure`, `api`, or external packages |
| **Entities have identity** | UUID, equality via `Equals()` |
| **Value Objects are immutable** | No setters, created via constructors |
| **Aggregate Root controls consistency** | All access to entities within aggregate goes through root |
| **Repositories return aggregates** | No leaking persistence models to domain |
| **Use cases are single-purpose** | One struct per use case, one `Execute()` method |
| **DTOs cross boundaries** | Domain models never exposed outside `application/` |
| **Ports are in domain or application** | Driven adapters implement interfaces defined in inner layers |
| **Domain events for cross-ctx communication** | `interview.completed` → `evaluation.generate_report`, `notification.send` (post-MVP) |
