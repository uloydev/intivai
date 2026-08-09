# AI Interviewer SaaS — Production-Grade Research

## Table of Contents
1. CV Parsing & Scoring
   - 1.1 PDF Text Extraction
   - 1.2 Scanned/Image-Based PDFs
   - 1.3 LLM-Based Structured Extraction
   - 1.4 Scoring Algorithm
   - 1.5 Configurable Scoring — Per-Tenant with Global Default Fallback
   - 1.6 Company Context & Tenant System Prompt
2. AI Interview (Chat)
3. AI Interview (Voice/Video)
4. Production Architecture
   - 4.1 Modular Monolith
   - 4.2 Multi-Tenant Data Model
   - 4.3 Hybrid Memory DB — Semantic Recall (Mnemosyne per-tenant)
   - 4.4 Async Processing with Queue
   - 4.5 Database Migrations
   - 4.6 Observability
   - 4.7 CORS & Security
   - 4.8 Rate Limiting
   - 4.9 Tech Stack Summary
5. LLM Integration Patterns
6. Competition & Differentiation
7. Compliance, Privacy & Legal

---

## 1. CV Parsing & Scoring

### PDF Text Extraction (Go Options) — MVP: PDF Only

| Library | Type | Extraction | Pros | Cons |
|---------|------|------------|------|------|
| **pdfcpu** | Free (Apache 2.0) | Text content | Pure Go, no deps, mature | Extracts raw text, no layout; fails on scanned/image PDFs |
| **sajari/docconv** | Free (MIT) | Text via pdftotext | Handles PDF well | Requires poppler-utils installed |

> **Implementation note:** `ledongthuc/pdf` (MIT, pure Go) was chosen — simpler extraction API for raw text; pdfcpu remains a viable alternative. See `internal/cv/application/parse_worker.go`.

**For MVP, PDF only.** DOCX can be added later after MVP.

### ⚠️ Scanned / Image-Based PDFs

pdfcpu **cannot** extract text from scanned documents. For these, you need OCR:

| OCR Option | Approach | Pros | Cons |
|------------|----------|------|------|
| **Tesseract CLI** | `os/exec` tesseract | Free, open source, well-known | Needs tesseract installed, slower |
| **LLM vision (via Ollama)** | `llama3.2-vision` or similar | No OCR infra, multi-language | Requires GPU, slower |

> **Implementation note:** Alpine `tesseract` cannot read PDF input directly (leptonica lacks rasterization) — `internal/cv/infrastructure/ocr` rasterizes pages with `pdftoppm` (poppler-utils, 200dpi PNG) then runs tesseract per page. Image deps: `tesseract-ocr tesseract-ocr-data-eng poppler-utils ghostscript` (Dockerfile — do not remove).

### DOCX Parsing (MIT-licensed, no AGPL trap)

```go
import "github.com/nguyenthenguyen/docx"

func extractDocx(path string) (string, error) {
    doc, err := docx.ReadDocxFile(path)
    if err != nil {
        return "", err
    }
    return doc.Editable().GetContent(), nil
}
```

⚠️ Do NOT use `unidoc/unioffice` without a commercial license — it's AGPL. Use `go-docx` (MIT) for basic extraction.

### LLM-Based Structured Extraction

Instead of building regex rules (fragile), use LLM to extract:

```
System: Extract structured data from this resume in JSON format.
Fields: name, email, phone, skills[], experience[], education[], years_of_exp

Resume text: [extracted text]

Return valid JSON only.
```

```go
// Implemented schema (internal/cv/application/dto.go) — scoring-oriented:
// name/email/phone live on the candidates table; experience flattened to years.
type ResumeData struct {
    Skills          []string `json:"skills"`
    ExperienceYears float64  `json:"experience_years"`
    Education       string   `json:"education"`
    Certifications  []string `json:"certifications"`
    Summary         string   `json:"summary"`
}

// Use OpenAI JSON mode for structured output
resp, _ := client.CreateChatCompletion(ctx, ChatCompletionRequest{
    Model: "deepseek-chat",  // cheap for extraction
    Messages: []Message{
        {Role: "system", Content: prompt},
        {Role: "user", Content: resumeText},
    },
    ResponseFormat: &ResponseFormat{Type: "json_object"},
})
```

**Cost optimization:** Cache parsed CV data. Only re-parse when CV is updated. Extraction is one-time cost.

### Scoring Algorithm

```go
// Implemented (internal/screening/domain/scoring.go): Total normalized to
// 0-100 (weights sum to 1.0; threshold default 50, job > org > default).
type ScoreResult struct {
    Total      float64            `json:"total"`
    Breakdown  map[string]float64 `json:"breakdown"`
    Passed     bool               `json:"passed"`
}

type ScoringWeights struct {
    SkillsMatch      float64 `json:"skills_match"`      // default 0.35
    ExperienceYears  float64 `json:"experience_years"`  // default 0.20
    SemanticMatch    float64 `json:"semantic_match"`    // default 0.25
    Education        float64 `json:"education"`         // default 0.10
    Certifications   float64 `json:"certifications"`    // default 0.10
    // Per-tenant configurable
}

func ScoreResume(cv ResumeData, jd JobDescription, weights ScoringWeights, minScore float64) ScoreResult {
    // 1. Keyword match + synonym map (Go↔Golang); NOT embeddings — avoid double-count with semanticMatch
    skillsScore := matchSkills(cv.Skills, jd.RequiredSkills) * weights.SkillsMatch

    // 2. Experience years (capped at 100% of requirement)
    expRatio := math.Min(cv.TotalYears / jd.MinYears, 1.0)
    expScore := expRatio * weights.ExperienceYears

    // 3. Semantic similarity (embedding-based)
    semanticScore := cosineSimilarity(cvEmbedding, jdEmbedding) * weights.SemanticMatch

    // 4. Education match
    eduScore := matchEducation(cv.Education, jd.Requirements) * weights.Education

    // 5. Certifications / bonuses
    certScore := matchCertifications(cv, jd) * weights.Certifications

    total := skillsScore + expScore + semanticScore + eduScore + certScore

    return ScoreResult{
        TotalScore: total,
        Breakdown:  {...},
        MaxScore:   100,
        Passed:     total >= minScore,
    }
}
```

**Key scoring rules:**
- Experience capped at 100% (10yr when 5yr required = 100%, not 200%)
- Per-tenant configurable weights
- Minimum threshold to proceed (default: 50/100)
- Skills normalization: synonym map (Go↔Golang) + keyword match; partial match does NOT use embeddings — avoid double-counting with semanticMatch

### Configurable Scoring — Per-Tenant with Global Default Fallback

Each tenant may override scoring weights, but every value has a global default that is used when the tenant has not set it, set it partially, or set an invalid value.

**Resolution hierarchy (most specific first):**

```
tenant_scoring_weights (orgs.scoring_weights JSONB)
        │  if field exists & valid → use
        ▼
job_scoring_weights (jobs.scoring_weights JSONB, optional per-JD)
        │  if field exists & valid → use
        ▼
tenant defaults (created at org registration, copied from global defaults)
        │
        ▼
GLOBAL DEFAULTS (code constant — always present, can't disappear)
```

**Fallback rules:**
1. Tenant sets nothing → **global defaults** (0.35/0.20/0.25/0.10/0.10, threshold 50)
2. Tenant sets partial (e.g. only SkillsMatch) → unset fields fall back to the global default
3. Tenant sets invalid value (negative, NaN, >1) → validation rejects & falls back to the global default for that field
4. Threshold (`min_score_to_proceed`) is per-tenant too: default 50, overridable per org or per job

**Implementasi:**

```go
// defaults.go — single source of global defaults
var GlobalScoringDefaults = ScoringWeights{
    SkillsMatch:     0.35,
    ExperienceYears: 0.20,
    SemanticMatch:   0.25,
    Education:       0.10,
    Certifications:  0.10,
}
const GlobalMinScoreToProceed = 50.0

// resolve.go — resolver with fallback
func (s *Server) ResolveScoringWeights(ctx context.Context, orgID, jobID uuid.UUID) (ScoringWeights, float64, error) {
    w := GlobalScoringDefaults          // 1. start from global defaults
    threshold := GlobalMinScoreToProceed

    org, err := s.repo.GetOrg(ctx, orgID)      // 2. overlay tenant
    if err != nil { return w, threshold, err }
    if org.ScoringWeights != nil {
        applyValidOverrides(&w, org.ScoringWeights) // field valid doang, sisanya default
    }
    if org.MinScoreToProceed != nil {
        threshold = *org.MinScoreToProceed
    }

    if jobID != uuid.Nil {                       // 3. overlay per-job (optional)
        job, err := s.repo.GetJob(ctx, jobID)
        if err == nil && job.ScoringWeights != nil {
            applyValidOverrides(&w, job.ScoringWeights)
        }
        if err == nil && job.MinScoreToProceed != nil {
            threshold = *job.MinScoreToProceed
        }
    }
    w = normalizeWeights(w)              // normalize → max score ALWAYS 100
    return w, threshold, nil
}

// applyValidOverrides — only valid fields override; NaN/negative/>1 rejected
func applyValidOverrides(w *ScoringWeights, o map[string]float64) {
    if v, ok := o["skills_match"]; ok && validWeight(v) { w.SkillsMatch = v }
    if v, ok := o["experience_years"]; ok && validWeight(v) { w.ExperienceYears = v }
    if v, ok := o["semantic_match"]; ok && validWeight(v) { w.SemanticMatch = v }
    if v, ok := o["education"]; ok && validWeight(v) { w.Education = v }
    if v, ok := o["certifications"]; ok && validWeight(v) { w.Certifications = v }
}

func validWeight(v float64) bool {
    return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0 && v <= 1
}
```

**Weight normalization:** if a tenant sets weights whose total ≠ 1 (e.g. 0.5+0.5 = 1.0, or 0.6+0.2 = 0.8), **NORMALIZE in the resolver** — divide each weight by the total sum. This guarantees the max achievable score is ALWAYS 100, and scores are comparable across tenants. Without normalization, a tenant with total weight 0.8 has a max score of 80 — the 100 cap becomes meaningless (easy to reach) and scores are inconsistent.

```go
// normalizeWeights — normalize in the resolver, NOT at input
func normalizeWeights(w ScoringWeights) ScoringWeights {
    total := w.SkillsMatch + w.ExperienceYears + w.SemanticMatch +
             w.Education + w.Certifications
    if math.Abs(total-1.0) < 1e-9 {
        return w // already sums to 1
    }
    if total <= 0 {
        return GlobalScoringDefaults // all 0 / invalid → global fallback
    }
    w.SkillsMatch /= total
    w.ExperienceYears /= total
    w.SemanticMatch /= total
    w.Education /= total
    w.Certifications /= total
    return w
}
```

Call at the end of `ResolveScoringWeights` before returning — **required, do not skip**. Max score is always 100.

**Design rationale:**
- **Integrity:** fallback to global defaults guarantees scores are always valid, no empty/NaN fields
- **Simple for tenants:** an org sets only the 1-2 fields it wants changed; the rest default automatically
- **Migration:** new tenants run on defaults immediately; overrides only when needed
- **Audit:** the breakdown stores the weights USED per scoring (not the configured ones) → traceable why a score is what it is

**Embeddings for semantic matching:**
- Use **local embedding (fastembed, bge-small-en-v1.5 = 384 dims)** — $0, consistent with the Mnemosyne layer. DeepSeek has no public embedding API — do not use. If you change the embedding model, update `VECTOR(dim)` + migration.
  - **Status:** deferred (M2.5). Columns `VECTOR(384)` + HNSW index exist (migration 002); `SemanticMatch` currently uses keyword overlap (`internal/screening/domain/semantic.go`); pgvector bank adapter stores NULL embeddings until fastembed lands.
- Pre-compute JD embeddings on create
- Compute CV embedding on upload
- Cosine similarity between CV and JD (catches "Go" ↔ "Golang" ↔ "Go programming")

### Company Context & Tenant System Prompt

**Problem:** AI interviewer without company context = generic answers. Tenants need a way to tell the interviewer about their culture, tech stack, values, and question preferences — so interviews are relevant to the company's specific needs.

**Solution: 2 mechanisms, both per-tenant, both versioned, both with default fallback.**

#### A. Company Context (file or text)

Tenant uploads a file (PDF/MD/TXT) or pastes text containing company context:
- Values & budaya kerja
- Tech stack, product, architecture
- Role-specific requirements
- FAQ / expected answers

**Flow:**

```
tenant upload file/text (POST /orgs/:id/contexts)
  → validate type + size (10MB max)
  → store raw in MinIO (file) / Postgres (text), hash for dedup
  → version bump (v1, v2, v3...)
  → queue: index_context
    → parse + chunk (file)
    → LLM summarize → semantic summary
    → index to banks/<org_id>/mnemosyne.db (per-tenant bank)
  → new interviews automatically use the latest context version
```

```go
// internal/context/application/upload_context.go
type UploadContextCommand struct {
    OrgID   uuid.UUID
    Type    string // file | text
    Content []byte // raw content
    Name    string
}

func (s *Service) UploadContext(ctx context.Context, cmd UploadContextCommand) (ContextVersion, error) {
    hash := sha256.Sum256(cmd.Content)
    version, err := s.repo.NextVersion(ctx, cmd.OrgID) // version bump — use a transaction + SELECT ... FOR UPDATE (2 concurrent uploads → collide UNIQUE(org_id, version))
    if err != nil { return ContextVersion{}, err }

    // store raw (source of truth in Postgres/MinIO)
    if err := s.storage.Save(ctx, cmd.OrgID, version, cmd.Content); err != nil {
        return ContextVersion{}, err
    }

    // async: chunk + summarize + index into the tenant Mnemosyne bank
    s.queue.Enqueue(JobTypeIndexContext, IndexContextPayload{
        OrgID: cmd.OrgID, Version: version, Hash: string(hash[:]),
    })
    return ContextVersion{Version: version}, nil
}
```

**Retrieval during interview:** interview start → recall top-K relevant chunks from the tenant bank → inject into the interview system prompt (bounded budget, e.g. max 2-3K tokens). The interviewer can recall again on-demand if it needs detail.

#### B. Tenant System Prompt

Tenants can set a custom system prompt for their interviewer. This controls the interview *style*.

**Resolution hierarchy (same as scoring):**

```
tenant_prompts (active version) → if present & valid → use
        ▼ (not set / invalid)
GLOBAL DEFAULT INTERVIEW PROMPT (code constant)
```

**Safety rails (Mandatory — cannot be overridden by tenants):**

| Tenant may set | LOCKED (pinned, cannot be changed) |
|---|---|
| Tone & interview style | Anti-bias rules |
| Skill focus / question priorities | Prohibited questions |
| Company values, culture, tech stack | GDPR/consent flow |
| Role-specific instructions | Bot detection, max duration, emergency stop |

Implementation: the global default prompt is the **anchor**; the tenant prompt is appended/injected in a safe position; safety rails stay hard-coded AFTER the tenant prompt. The tenant prompt must never replace the whole system prompt.

```go
// internal/interview/domain/service/prompt_composer.go
func ComposeInterviewSystemPrompt(
    tenantPrompt *string,     // optional, tenant override
    companyContext []string,  // recall from the Mnemosyne bank
    safetyRails string,       // HARD-CODED, cannot be changed
) string {
    var b strings.Builder
    b.WriteString(GlobalDefaultPrompt)   // 1. anchor
    if tenantPrompt != nil && *tenantPrompt != "" {
        b.WriteString("\n\n[TENANT INSTRUCTION]\n")
        b.WriteString(*tenantPrompt)     // 2. tenant override (bounded)
    }
    if len(companyContext) > 0 {
        b.WriteString("\n\n[COMPANY CONTEXT]\n")
        for _, chunk := range companyContext { // 3. company context (bounded budget)
            b.WriteString("- " + chunk + "\n")
        }
    }
    b.WriteString("\n\n[SAFETY RAILS — MUST ALWAYS FOLLOW]\n")
    b.WriteString(safetyRails)           // 4. safety rails pinned LAST (highest priority)
    return b.String()
}
```

**Why safety rails go last:** LLMs tend to follow instructions closest to the end of the prompt. Bottom position = strongest. Tenant prompt in the middle = can override the default but cannot override safety rails.

**Tenant prompt validation rules:**
- Max length (e.g. 4K chars) — prevent prompt injection via length
- Max context budget per interview (e.g. 3K tokens of company context) — prevent token overshoot
- Tenant prompt must not contain integrity-breaking instructions (keyword detection: "pass all", "ignore safety", "always hire")

**Data model:**

```sql
-- Company context (versioned)
CREATE TABLE company_contexts (
    id UUID PRIMARY KEY,
    org_id UUID REFERENCES orgs(id),
    type TEXT NOT NULL,          -- file | text
    content_hash TEXT NOT NULL,  -- dedup + change detection
    version INT NOT NULL DEFAULT 1,
    storage_path TEXT,           -- MinIO path (file) / NULL (text)
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(org_id, version)
);

-- Tenant system prompt (versioned)
CREATE TABLE tenant_prompts (
    id UUID PRIMARY KEY,
    org_id UUID REFERENCES orgs(id),
    system_prompt TEXT NOT NULL,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(org_id, version)
);
```

**Audit:** the interview stores the `context_version` + `prompt_version` used → traceable: "this interview used company context v3, tenant prompt v2, safety rails v1".

---

## 2. AI Interview (Chat)

### Architecture
```
Browser ──WebSocket──▶ Go Server ──SSE──▶ LLM API (streaming)   // DeepSeek streaming = SSE, not WebSocket
                           │
                      Save to PostgreSQL
                      (messages, evaluations)
```

**Single protocol: WebSocket for both directions.** No SSE. Keeps it simple.

### WebSocket Chat Flow
```
Client connects → Server upgrades to WS (Authorization: WS ticket — lihat §3 Candidate Access)
  ↓
Server sends: {"type": "interview.start", 
               "session_id": "iv_abc123", 
               "total_questions": 5,
               "max_duration_min": 30}
  ↓
Client sends: {"type": "answer", "content": "...", "question_idx": 1}
  ↓
Server streams to LLM with context:
  [System prompt + JD + CV + conversation history]
  ↓
Server streams tokens back via WebSocket:
  {"type": "token", "content": "Great"}
  {"type": "token", "content": " answer"}
  ...
  {"type": "response_end", "evaluation": {...}}
  ↓
On complete → LLM generates final evaluation:
  {"type": "evaluation", "scores": {...}, "feedback": "..."}
```

### Critical: Heartbeat, Timeout & Reconnection

```go
// Server-side config
const (
    InterviewMaxDuration = 30 * time.Minute
    IdleTimeout         = 5 * time.Minute
    PerQuestionTimeout  = 3 * time.Minute
    PingInterval        = 30 * time.Second
    PongWait            = 10 * time.Second
)

// WebSocket upgrader with ping/pong
var upgrader = websocket.Upgrader{
    HandshakeTimeout: 10 * time.Second,
}

// In handler:
conn.SetPingHandler(func(appData string) error {
    conn.WriteMessage(websocket.PongMessage, nil)
    return nil
})
conn.SetReadDeadline(time.Now().Add(PongWait))
```

**Reconnection strategy:**
1. Server stores last completed question index in PostgreSQL
2. On reconnect, client sends `session_id`
3. Server returns `{"type": "resume", "last_question_idx": 3, "history": [...]}`
4. Interview continues from last unanswered question
5. If session expired (>30min), return `{"type": "expired"}`

### Context Management

```go
type InterviewContext struct {
    SystemPrompt      string      // Interviewer role, scoring rubric
    JobDescription    string      // Parsed JD
    CV                ResumeData  // Parsed + scored CV
    Conversation      []Message   // Sliding window: last 10 Q&A
    TotalTokenBudget  int         // Max context tokens (e.g., 8000)
    CurrentTokenCount int         // Tracked via tiktoken-go
}

func buildPrompt(ctx InterviewContext) ([]Message, error) {
    systemPrompt := fmt.Sprintf(`
You are a professional interviewer for the position: %s

Job Description:
%s

Candidate's CV:
Name: %s
Skills: %v
Experience: %v

Rules:
- Ask one question at a time
- Adapt follow-ups based on answers
- Evaluate after each answer: technical_depth (1-10), communication (1-10)
- After all questions, provide final evaluation
- Do not ask about: age, marital status, religion, political affiliation
`, ctx.JobDescription, ctx.JD,
       ctx.CV.Name, ctx.CV.Skills, ctx.CV.Experience)

    // Count tokens before sending
    promptTokens := tiktoken.Count(systemPrompt)
    if promptTokens > ctx.TotalTokenBudget {
        return nil, fmt.Errorf("prompt exceeds token budget")
    }

    // Sliding window: keep last 10 messages
    window := ctx.Conversation
    if len(window) > 10 {
        window = window[len(window)-10:]
    }

    return append([]Message{{Role: "system", Content: systemPrompt}}, window...), nil
}
```

### Question Generation Strategy

```
1. Parse JD → extract: required skills, nice-to-haves, responsibilities
2. Parse CV → identify: matching skills, gaps, weak areas
3. Score each potential question by:
   a. Gap relevance (how important is this missing skill?)
   b. Depth verification (verify claimed expertise)
   c. Diversity (don't ask 2 similar questions)
4. Pick highest-scoring unanswered question
5. After answer, re-evaluate remaining questions
6. If candidate shows weakness → probe deeper (follow-up)
7. If candidate shows strength → move to next topic
8. Selected questions are persisted to the `questions` bank — reuse + audit trail; the bank also seeds the next generation
```

### Bias Prevention
- System prompt explicitly prohibits: age, marital status, religion, political questions
- LLM response filter: scan for protected-class keywords, reject if found
- Evaluation criteria focus on job-relevant skills only
- Store bias audit log for compliance

### Language Handling
- Detect language from candidate's first answer
- If mismatch with JD language, flag for recruiter
- Continue in candidate's language if supported by LLM

### Evaluation
After interview, LLM generates structured evaluation via function calling:
```json
{
  "overall_score": 78,
  "dimensions": {
    "technical": { "score": 82, "weight": 0.4 },
    "communication": { "score": 75, "weight": 0.2 },
    "problem_solving": { "score": 80, "weight": 0.25 },
    "culture_fit": { "score": 70, "weight": 0.15 }
  },
  "per_question": [
    {
      "question_idx": 1,
      "score": 85,
      "rationale": "Strong understanding of distributed systems",
      "strengths": ["Clear explanation", "Used real examples"],
      "weaknesses": []
    }
  ],
  "strengths": ["Go expertise", "System design"],
  "weaknesses": ["Limited cloud experience"],
  "recommendation": "proceed"
}
```

**Canonical schema** — the evaluator struct (§5) and Phase 4 must follow this; don't create new schemas. Dimension weights must sum to ≈ 1.

---

## 3. AI Interview (Voice/Video)

### ⚠️ Recommendation: Open Source / Self-Hosted (MVP)

**For MVP, use all open source, zero API cost.** No Deepgram, no OpenAI, no ElevenLabs. Run everything locally:

| Component | Open Source Solution | Cost | Go Integration |
|-----------|--------------------|------|----------------|
| **STT** | **Whisper** via `whisper.cpp` | Free (self-hosted) | `os/exec` whisper.cpp CLI, or CGo bindings |
| **LLM** | **DeepSeek Flash** (via API) | $0.0001/1K tokens | REST API from Go |
| **TTS** | **Piper TTS** or **Edge TTS** (free API) | Free | `os/exec` piper, or REST for Edge TTS |
| **WebRTC** | **Pion** + **coturn** (STUN/TURN) | Free | Pure Go WebRTC |

**Total voice cost per interview: ~$0.001** (LLM only). STT + TTS = FREE.

### Architecture (Open Source Stack)

```
Browser (getUserMedia)
  │  WebRTC audio
  │
  ├──▶ Go Server (Pion + coturn)
  │       │
  │       ├──▶ VAD (silero-vad) — segmentasi utterance
  │       │    └─ batas jawaban → trigger STT
  │       ├──▶ Whisper (STT)
  │       │    └─ via whisper.cpp subprocess
  │       │
  │       ├──▶ DeepSeek Flash (LLM)
  │       │    └─ via API: https://api.deepseek.com/v1/chat/completions
  │       │
  │       ├──▶ Piper / Edge TTS
  │       │    └─ via piper subprocess or Edge TTS REST
  │       │
  │       └──▶ Audio back to browser via WebRTC
  │
  └──▶ PostgreSQL (transcripts, scores)
```

### Component Details

**STT — Whisper (self-hosted)**

| Option | Method | Quality | Latency |
|--------|--------|---------|---------|
| **whisper.cpp** | CLI subprocess from Go | Good (tiny-large models) | Real-time on GPU, ~2-3x on CPU |
| **faster-whisper** | Python (CTranslate2) | Better, faster | Real-time on CPU |
| **Whisper via Ollama** | `ollama run whisper` | Good | Depends on hardware |

**MVP approach:** Use **whisper.cpp** via `os/exec` in Go. `tiny` is fine for dev; production uses `small`/`large-v3` — `tiny` accuracy is poor for Indonesian.

```go
func transcribeAudio(audioPath string) (string, error) {
    cmd := exec.Command("whisper.cpp", "--model", "tiny", "--output-txt", audioPath)
    output, err := cmd.Output()
    return string(output), err
}
```

**VAD (Voice Activity Detection) — MANDATORY before STT**

Without VAD the server can't tell when an answer ends → audio gets cut or runs together across sentences.

| Option | Approach |
|------|-----------|
| **silero-vad** | ONNX model, high accuracy, CPU-only; Go bindings available |
| **Energy-based** | Simple RMS threshold; enough for MVP, false positives in noisy environments |

Flow: WebRTC audio → jitter buffer → VAD → segment → Whisper STT (once per segment).

**LLM — DeepSeek Flash (API)**

DeepSeek Flash is the cheapest production-grade LLM at $0.0001/1K tokens (~$0.001 per interview response). Has OpenAI-compatible API so Go SDK works directly.

```go
// DeepSeek Flash via OpenAI-compatible API
func deepSeekChat(messages []Message) (string, error) {
    client := openai.NewClient(os.Getenv("DEEPSEEK_API_KEY"))
    client.SetBaseURL("https://api.deepseek.com/v1")
    
    resp, err := client.CreateChatCompletion(ctx, ChatCompletionRequest{
        Model: "deepseek-chat",  // Flash model
        Messages: messages,
        Stream: true,
    })
    return resp.Choices[0].Message.Content, err
}
```

**TTS — Piper or Edge TTS**

| Option | Quality | Cost | Implementation |
|--------|---------|------|---------------|
| **Piper TTS** | Good (many voices) | Free, local | `os/exec piper` |
| **Edge TTS** | Very good (natural) | Free API | `http.Get` (no auth needed) |
| **Ollama TTS** (future) | TBD | Free | — |

⚠️ `api.edge-tts.com` is a community endpoint (unofficial, reverse-engineered Edge Read Aloud) — can change/go offline without notice. Fallback chain: Edge TTS → Piper (local, safe).

```go
// Edge TTS — completely free, no API key
func edgeTTS(text string) ([]byte, error) {
    resp, err := http.Post("https://api.edge-tts.com/v1/tts", "application/json", 
        json.RawMessage(`{"text": "`+text+`", "voice": "id-ID-GadisNeural"}`))
    // Returns audio bytes
}
```

**For Indonesian voice:** Edge TTS supports `id-ID-GadisNeural` and `id-ID-ArdiNeural` — perfect for local market.

### Pricing Comparison

| Stack | Cost per interview (15min) | Dependency |
|-------|---------------------------|------------|
| **OpenAI Realtime API** | ~$0.90 | Paid API, vendor lock-in |
| **Deepgram + DeepSeek + ElevenLabs** | ~$0.28 | 3 paid APIs (premium option, not MVP) |
| **Whisper + DeepSeek Flash + Piper/Edge TTS** | **~$0.001** | Whisper/Piper free, DeepSeek ~$0.0001/1K tokens |

**Recommendation:** Start with Whisper + DeepSeek Flash + Piper/Edge TTS. Near-zero API cost ($0.001/interview), DeepSeek API is cheap enough that self-hosting hardware costs more. Upgrade to paid services only when you need higher accuracy or scale.

### Hardware Requirements

| Setup | RAM | GPU | Concurrent Users |
|-------|-----|-----|------------------|
| **Minimal** | 16GB RAM | CPU only (slow) | 1 |
| **Standard** | 32GB RAM | 8GB VRAM | 2-3 |
| **Production** | 64GB RAM | 24GB VRAM | 5-10 |

For MVP, a single machine with 32GB RAM + RTX 3060/4060 can handle 2-3 concurrent interviews.

### WebRTC Complexity (Read This Before Starting)

WebRTC in Go requires:
1. **Signaling server** — Exchange SDP offers/answers via WebSocket
2. **STUN server** — NAT traversal discovery (free: `stun:stun.l.google.com:19302`)
3. **TURN server** — Relay when UDP blocked (⚠️ Many corporate networks block UDP. You need TURN.)
4. **Audio pipeline** — Opus encode/decode in Go using Pion
5. **Jitter buffer** — Handle network jitter
6. **ICE restart** — Handle network changes mid-call

**Running TURN server:** `coturn` is the reference implementation. Must deploy alongside your app.

**Decision (MVP): Pion + coturn** — self-hosted, $0, consistent with the cost strategy (voice itself is post-MVP). **LiveKit = upgrade option** when voice becomes paid + needs scale: signaling, SFU, TURN, recording out of the box (Go server; the AI agent becomes a separate service that subscribes to the audio stream).

### Recording & Compliance
- **Must** get candidate consent before recording
- Store recordings encrypted in S3 with retention policy
- Auto-delete after configurable period (default: 90 days)
- Transcribe and store transcript only (not raw audio) for evaluation
- GDPR right-to-deletion: delete recording + transcript on request

### Bot Detection
- Track response timing: human typing has natural delay, AI-generated responses are instant
- Flag suspicious patterns: copy-paste, too-perfect answers, code formatting
- Send flagged interviews for human review

### Candidate Access & Interview Link (Candidate Auth Flow)

Candidates are **not internal users** — they have no account. Interview access flow:

```sql
-- Interview invitation token — invitation credential (1x START), not "single-use raw"
CREATE TABLE interview_tokens (
    id UUID PRIMARY KEY,
    org_id UUID REFERENCES orgs(id),        -- tenant: RLS + revoke
    interview_id UUID REFERENCES interviews(id),
    token TEXT UNIQUE NOT NULL,             -- random 32-char, high entropy
    expires_at TIMESTAMPTZ NOT NULL,        -- default +7 days
    used_at TIMESTAMPTZ,                    -- set when the interview FIRST starts
    revoked_at TIMESTAMPTZ,                 -- recruiter revoke → all access denied
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

```go
// Flow:
// 1. Recruiter creates interview → generate token
// 2. Deliver token SECURELY: email body / copy-paste. NOT in URL — leaks via referrer/log
// 3. Candidate validates → token checked (exists, not expired, not revoked)
// 4. FIRST start → used_at set (the same token stays valid for reconnect)
// 5. Reconnect: session_id + the same token (still valid)
// 6. WS upgrade: token → server issues a WS ticket (short-lived JWT, 10 min,
//    bound to session_id + interview_id) → upgrade uses Authorization header
// 7. Starting a DIFFERENT interview with the same token → rejected (used_at != NULL)
```

**Rules:**
- Token: crypto/rand 32-byte, URL-safe — unguessable
- Token = invitation credential (1x START); session_id = resume credential
- Expiry: 7 days from invite; revoked_at → all access dies
- WS ticket: short-lived JWT (10 min), not a session replacement
- Rate limit per-IP auth attempts (10/min)
- Revoke: recruiter revokes the token before use → access denied immediately
- Candidate validation goes through the security-definer function `validate_interview_token(token)` — candidates must never access tables directly

---

## 4. Production Architecture

### Modular Monolith (Recommended for MVP → Scale)

```
/backend
├── cmd/
│   └── server/        # Single binary entry point
├── internal/
│   ├── cv/            # CV parsing + scoring (bounded context)
│   │   ├── parser.go      # pdfcpu + LLM extraction + OCR fallback
│   │   ├── scorer.go      # Weighted configurable scoring
│   │   └── store.go       # PostgreSQL CRUD
│   ├── interview/     # Interview engine (bounded context)
│   │   ├── chat/          # Chat interview (WebSocket)
│   │   ├── voice/         # Voice interview (Whisper + LLM + Edge TTS)
│   │   ├── ws/            # WebSocket hub + heartbeat (chat & voice)
│   │   ├── evaluator.go   # LLM evaluation
│   │   ├── scheduler.go   # Interview scheduling
│   │   └── session.go     # Timeout, heartbeat, reconnection
│   ├── job/           # Job description management
│   ├── user/          # Auth, orgs, roles (multi-tenant)
│   ├── api/           # HTTP handlers (Fiber)
│   │   ├── middleware/
│   │   │   ├── auth.go        # JWT + tenant context
│   │   │   ├── ratelimit.go   # Per-tenant sliding window (Redis)
│   │   │   ├── cors.go        # CORS configuration
│   │   │   └── audit.go       # Request audit logging
│   │   └── handler/
│   ├── queue/         # Async job processing (asynq)
│   ├── storage/       # S3 file storage (CVs, recordings)
│   ├── llm/           # LLM provider abstraction + retry + fallback
│   ├── audit/         # Audit log for compliance
│   └── observability/ # Health checks, metrics, graceful shutdown
└── pkg/
    └── types/         # Shared DTOs
```

### Multi-Tenant Data Model

```sql
-- Organizations
CREATE TABLE orgs (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    plan TEXT DEFAULT 'free',
    scoring_weights JSONB,  -- per-tenant configurable weights (partial override; falls back to global defaults)
    min_score_to_proceed DOUBLE PRECISION,  -- per-tenant threshold override (default 50 via code)
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Users (⚠️ UNIQUE per org_id, not global)
CREATE TABLE users (
    id UUID PRIMARY KEY,
    org_id UUID REFERENCES orgs(id),
    email TEXT NOT NULL,
    role TEXT DEFAULT 'member',  -- admin, recruiter, interviewer
    password_hash TEXT,  -- NULL if OAuth-only (Google)
    auth_provider TEXT DEFAULT 'password',  -- password | google
    UNIQUE(org_id, email)  -- same email allowed in different orgs
);

-- Question bank: generated questions persisted automatically (reuse + audit; seeds the next generation)
CREATE TABLE questions (
    id UUID PRIMARY KEY,
    org_id UUID REFERENCES orgs(id),
    category TEXT NOT NULL,  -- technical, behavioral, situational
    difficulty TEXT DEFAULT 'medium',  -- junior, mid, senior
    body TEXT NOT NULL,
    skills TEXT[],  -- related skills
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Job Descriptions
CREATE TABLE jobs (
    id UUID PRIMARY KEY,
    org_id UUID REFERENCES orgs(id),
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    required_skills JSONB,
    min_experience INT,
    scoring_weights JSONB,  -- per-job override (optional; falls back to org → global)
    min_score_to_proceed DOUBLE PRECISION DEFAULT 50,
    embedding VECTOR(384),  -- fastembed bge-small = 384 dims
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Candidates
CREATE TABLE candidates (
    id UUID PRIMARY KEY,
    org_id UUID REFERENCES orgs(id),
    name TEXT NOT NULL,
    email TEXT,
    cv_path TEXT,
    cv_raw_text TEXT,
    cv_structured JSONB,
    cv_embedding VECTOR(384),
    cv_ocr_method TEXT,  -- pdfcpu, tesseract, ollama-vision
    status TEXT DEFAULT 'new',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Applications
CREATE TABLE applications (
    id UUID PRIMARY KEY,
    candidate_id UUID REFERENCES candidates(id),
    job_id UUID REFERENCES jobs(id),
    cv_score DOUBLE PRECISION,
    score_breakdown JSONB,
    passed_screening BOOLEAN,
    status TEXT DEFAULT 'screening',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Interview Sessions
CREATE TABLE interviews (
    id UUID PRIMARY KEY,
    application_id UUID REFERENCES applications(id),
    type TEXT DEFAULT 'chat',
    status TEXT DEFAULT 'pending',
    transcript JSONB,
    evaluation JSONB,
    recording_url TEXT,
    consent_given BOOLEAN DEFAULT false,
    last_question_idx INT DEFAULT 0,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,  -- max duration deadline
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Audit Log
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY,
    org_id UUID REFERENCES orgs(id),
    user_id UUID REFERENCES users(id),
    action TEXT NOT NULL,     -- view_candidate, start_interview, export_data
    resource_type TEXT,
    resource_id UUID,
    ip_address INET,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ═══════════════════════════════════════════════════════
-- ROW-LEVEL SECURITY (RLS) — multi-tenant isolation
-- Mandatory: enable on EVERY table with org_id
-- ═══════════════════════════════════════════════════════
-- Each request sets app.org_id via the tenant middleware:
--   SELECT set_config('app.org_id', $1, true);  -- per-transaction

-- Full example (repeat per table: candidates, jobs, applications, interviews, audit_logs, company_contexts, tenant_prompts, questions)
ALTER TABLE candidates ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_candidates ON candidates
    USING (org_id = NULLIF(current_setting('app.org_id', true), '')::uuid);

ALTER TABLE jobs ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_jobs ON jobs
    USING (org_id = NULLIF(current_setting('app.org_id', true), '')::uuid);

ALTER TABLE applications ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_applications ON applications
    USING (org_id = NULLIF(current_setting('app.org_id', true), '')::uuid);

ALTER TABLE interviews ENABLE ROW LEVEL SECURITY;
-- interviews have no direct org_id → via join application → job
CREATE POLICY tenant_isolation_interviews ON interviews
    USING (EXISTS (
        SELECT 1 FROM applications a
        JOIN jobs j ON j.id = a.job_id
        WHERE a.id = interviews.application_id
          AND j.org_id = NULLIF(current_setting('app.org_id', true), '')::uuid
    ));

ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_audit ON audit_logs
    USING (org_id = NULLIF(current_setting('app.org_id', true), '')::uuid);

ALTER TABLE company_contexts ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_contexts ON company_contexts
    USING (org_id = NULLIF(current_setting('app.org_id', true), '')::uuid);

ALTER TABLE tenant_prompts ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_prompts ON tenant_prompts
    USING (org_id = NULLIF(current_setting('app.org_id', true), '')::uuid);

ALTER TABLE questions ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_questions ON questions
    USING (org_id = NULLIF(current_setting('app.org_id', true), '')::uuid);

ALTER TABLE interview_tokens ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_tokens ON interview_tokens
    USING (org_id = NULLIF(current_setting('app.org_id', true), '')::uuid);
-- Candidate token validation (no auth): via security-definer function validate_interview_token(token),
-- returns valid/expired/used/revoked — NOT direct table access

-- users: admins can see all users in their org
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_users ON users
    USING (org_id = NULLIF(current_setting('app.org_id', true), '')::uuid);

-- ⚠️ orgs: special case — needs access to its own org to set weights.
-- Use a policy that allows reading the org that matches app.org_id.
ALTER TABLE orgs ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_orgs ON orgs
    USING (id = NULLIF(current_setting('app.org_id', true), '')::uuid);
```

### Hybrid Memory DB — Semantic Recall (Mnemosyne per-tenant)

**Problem:** pgvector in Postgres is good for simple similarity search (CV ↔ job), but weak for *cross-data semantic recall*: "which candidate's answers show the same skill pattern as the candidate who PASSED yesterday?" or "out of 50 interviews, which question fails most often?" — that needs a memory layer with entity resolution + knowledge graph + cross-interview reflection.

**Solution: hybrid 2 layers.**

```
┌──────────────────────────────────────────────────────────┐
│  PostgreSQL  (SOURCE OF TRUTH — data integrity)            │
│  • orgs, users, candidates, jobs, interviews               │
│  • ACID, FK, RLS (tenant isolation), transactions, audit   │
│  • pgvector for primitive embeddings (cv ↔ job match)      │
└───────────────┬──────────────────────────────────────────┘
                │ sync via Go worker (outbox / event bus)
┌───────────────▼──────────────────────────────────────────┐
│  Mnemosyne  (SEMANTIC LAYER — 1 bank per tenant)         │
│  • banks/<org_id>/mnemosyne.db  (isolated SQLite)        │
│  • entity resolution + knowledge graph                   │
│  • multi-strategy semantic recall                        │
│  • reflect: cross-memory synthesis                        │
└──────────────────────────────────────────────────────────┘
```

#### Why hybrid (not Mnemosyne as the only DB)

| Requirement | Postgres | Mnemosyne |
|-----------|----------|-----------|
| **Data integrity** (ACID, FK, constraints) | ✅ | ❌ (memory layer, not a relational DB) |
| **Complex relational queries** (JOIN, reporting) | ✅ | ❌ |
| **Audit & billing** (immutable, transactional) | ✅ | ❌ |
| **Semantic recall** ("similar candidates who passed") | 🟡 shallow pgvector | ✅ vector + BM25 + entity |
| **Knowledge graph** (candidate↔skill↔interview) | ❌ | ✅ |
| **Cross-interview reflection** (patterns, insights) | ❌ | ✅ `reflect` |
| **Tenant isolation** | ✅ RLS | ✅ bank per tenant (DB level) |

**Data integrity:**
- Postgres = source of truth. All structural data (candidate, interview, score, payment) STAYS in Postgres.
- Mnemosyne only stores **embedding + entity + reference** (not raw transcripts).
- If Mnemosyne is lost/down → rebuilt from Postgres (cache, not a source).
- Sensitive PII (transcript, CV) is not sent to the LLM for extraction on every write — extraction only happens in the semantic index layer.

#### Tenant Isolation — Bank per Tenant

Mnemosyne has native **Memory Bank Isolation**:

```
~/.hermes/mnemosyne/data/banks/<org_id>/mnemosyne.db
```

- **1 tenant = 1 separate SQLite bank** — isolation at the file/database level (not just `WHERE tenant_id`).
- **⚠️ Mnemosyne is a Python package (MCP server). Two adapter options — ONE Go port, decision can be deferred:**
  - **Option A — Mnemosyne MCP (stdio/HTTP):** use the existing plugin; graph/reflect features limited to what the plugin provides.
  - **Option B — Native Go (recommended):** SQLite + fastembed (bge-small) + BM25 + LLM for reflect; full control, 1 binary, no Python runtime — fits the 1-server strategy.
  - The Go port stays the same for both options — swap the adapter without touching use cases.

```go
// internal/memory/domain/memory_port.go
// Port (Go) — adapter implementation for the memory layer via MCP/HTTP or native
type MemoryBank interface {
    Remember(ctx context.Context, entityType, summary string, importance float64) error
    Recall(ctx context.Context, query string, budget string) ([]MemoryHit, error)
    Reflect(ctx context.Context, question string) (string, error)
    QueryGraph(ctx context.Context, entityType, filter string) ([]MemoryHit, error)
    Forget(ctx context.Context, memoryID string) error
    Stats(ctx context.Context) (MemoryStats, error)
}

type MemoryHit struct {
    ID      string
    Content string
    Score   float64
}
```

```go
// internal/memory/infrastructure/mcp/mnemosyne_mcp.go  (Option A)
// Adapter — Mnemosyne MCP server via stdio
func (a *MCPAdapter) Remember(ctx context.Context, entityType, summary string, importance float64) error {
    // Mnemosyne MCP tool: mnemosyne_remember(content=..., importance=...)
    return a.client.Call("mnemosyne_remember", map[string]any{
        "content":    summary,
        "importance": importance,
    })
}

// Set bank per tenant via MCP: Mnemosyne(bank="<org_id>")
func (a *MCPAdapter) ForBank(orgID string) MemoryBank {
    return &MCPAdapter{client: a.client.WithParam("bank", orgID)}
}
```

- **Double isolation:** Postgres RLS (row-level) + Mnemosyne bank (file-level). Tenant A's data can never be recalled from tenant B.

#### Semantic Indexing Pipeline (Go worker)

After an event lands in Postgres, a sync worker indexes it to Mnemosyne:

```go
// Event: candidate scored / interview completed
type SyncEvent struct {
    OrgID       string `json:"org_id"`
    EntityType  string `json:"entity_type"` // candidate, interview, job
    EntityID    string `json:"entity_id"`
    Summary     string    `json:"summary"`     // semantic summary (not raw PII)
    Importance  float64 `json:"importance"`
}

func (s *Server) syncToMnemosyne(ctx context.Context, ev SyncEvent) error {
    mn := s.memory.ForBank(ev.OrgID) // MCP/HTTP adapter to banks/<org_id>/
    return mn.Remember(ctx, ev.EntityType, ev.Summary, ev.Importance)
}
```

Events indexed:
- `candidate_profile` — CV summary (name, skills, experience, notes) → high importance
- `interview_summary` — interview summary (key answers, impressions) → medium importance
- `interview_reflection` — post-interview reflection → triggers `reflect`
- `job_requirements` — required skills for matching

#### Recall & Reflect (concrete use cases)

```go
// 1. Semantic candidate search (not keyword) — via the Go port
res, _ := mn.Recall(ctx, "strong Go candidate with fintech payment experience", "high")

// 2. Knowledge graph query: skills of candidates who passed
// (port method — per-adapter implementation: MCP tool or SQLite query)
graph, _ := mn.QueryGraph(ctx, "candidate", "passed_screening=true")

// 3. Cross-interview reflection: patterns of failing questions
insight, _ := mn.Reflect(ctx, "of the last 50 interviews, which question fails most often and why?")
```

#### Product benefits

| Product feature | Implementation |
|--------------|--------------|
| "Find similar candidates who passed before" | semantic recall + entity |
| "Skill gap analysis" | graph query (skills of passing candidates) |
| "Interview patterns" | cross-interview reflect |
| "Improve question bank" | reflect: questions with high fail rates |
| Compliance | separate bank + delete bank = delete all tenant memory |

**Cost note:** Index-time = **$0** (local fastembed bge-small embeddings — no LLM calls, except optional extraction). **Reflect = small LLM call at query-time** (not $0) — short prompt, per-reflect cost negligible. Cheaper than pgvector + external embedding API, and fits a 1-core server.

### Async Processing with Queue

```go
const (
    JobTypeParseCV       = "cv.parse"
    JobTypeScoreCV       = "cv.score"
    JobTypeEvaluateInterview = "interview.evaluate"
    JobTypeTranscribeRecording = "recording.transcribe"
    JobTypeGenerateReport    = "report.generate"
)

func (s *Server) handleCVUpload(c *fiber.Ctx) error {
    // Validate file type & size
    file, _ := c.FormFile("cv")
    if file.Size > 10*1024*1024 { // 10MB max
        return c.Status(413).JSON(fiber.Map{"error": "file too large"})
    }

    cvID := uuid.New()

    // 1. Save to S3
    s.storage.Put(ctx, fmt.Sprintf("cvs/%s.pdf", cvID), file)

    // 2. Enqueue parsing job (async, non-blocking)
    task, _ := NewTask(JobTypeParseCV, map[string]any{
        "cv_id": cvID,
        "path":  fmt.Sprintf("cvs/%s.pdf", cvID),
    })
    s.asynqClient.Enqueue(task, asynq.MaxRetry(3), asynq.Timeout(5*time.Minute))

    return c.Status(202).JSON(fiber.Map{"cv_id": cvID, "status": "processing"})
}
```

**Job monitoring:** asynq has a built-in web UI at `/asynq/monitoring`. Enable it in dev.

### Database Migrations

Use `golang-migrate/migrate` or `pressly/goose`:

```bash
migrate create -ext sql -dir migrations create_interviews_table
migrate up
```

Migrations committed to git, run as part of deploy pipeline.

### Observability

```go
// Health checks
http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    // Check DB, Redis, S3 connectivity
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
})

http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
    // Check if service can accept traffic
})

// Prometheus metrics
// - llm_requests_total, llm_request_duration_ms
// - interviews_active, interviews_completed
// - cv_parse_duration, cv_parse_errors
// - websocket_connections_active

// Graceful shutdown
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

<-ctx.Done()
log.Println("shutting down...")
server.Shutdown(ctx)
wsHub.Close()
llmClient.Close()
```

### Backup & Disaster Recovery (MANDATORY — survival)

Solo founder + 1 server = 1 bad day away from losing everything. Minimum:

```bash
# 1. PostgreSQL — daily dump to MinIO (cron)
# /etc/cron.d/pg-backup
0 3 * * * pg_dump -Fc $DATABASE_URL | s3cmd put - s3://backups/pg-$(date +\%F).dump

# 2. MinIO replication / offsite
# rclone sync to external object storage (Backblaze B2 ~$6/TB)
0 4 * * * rclone sync s3://backups b2:hermes-backups --fast-list

# 3. Mnemosyne banks — SQLite file, back up together
# banks/<org_id>/mnemosyne.db → rebuildable from Postgres (cache), but cheap to back up too
0 5 * * * tar czf - ~/.hermes/mnemosyne/data/banks | s3cmd put - s3://backups/mnemosyne-$(date +\%F).tar.gz

# 4. Config & .env — git (no secrets) + encrypted copy
```

**Recovery target:**
- RPO ≤ 24h (daily backup)
- RTO ≤ 1h (restore dump + rebuild Mnemosyne from Postgres)
- Monthly restore test — a backup never restored = no backup

### CORS & Security

```go
app.Use(cors.New(cors.Config{
    AllowOrigins:     "https://*.yourapp.com",
    AllowMethods:     "GET,POST,PUT,DELETE",
    AllowHeaders:     "Origin, Content-Type, Authorization",
    AllowCredentials: true,
}))
```

### Rate Limiting

| Scope | Limit | Implementation |
|-------|-------|---------------|
| Per-tenant API calls | 1000 req/min | Redis sliding window |
| Per-user API calls | 100 req/min | Redis sliding window |
| Per-tenant LLM tokens | 100K TPM | Track via tiktoken + Redis |
| Per-IP auth attempts | 10 req/min | Prevent brute force |

### Tech Stack Summary

| Layer | Technology | Rationale |
|-------|-----------|-----------|
| **Backend** | Go + Fiber | Long-term value, concurrency, single binary |
| **Database** | PostgreSQL + pgvector | JSONB for flexible data, vector embeddings |
| **Queue** | asynq (Redis) | Go-native, built-in monitoring |
| **Cache** | Redis | Sessions, rate limits, LLM token tracking |
| **Storage** | S3/MinIO | CV files, interview recordings |
| **WebSocket** | gorilla/websocket | Chat interview, signaling |
| **WebRTC** | Pion + coturn | Voice/video (STUN/TURN required) |
| **STT** | Whisper via whisper.cpp | Free, self-hosted, `os/exec` integration |
| **LLM** | DeepSeek Flash = `deepseek-chat` (via API) | Cheap ($0.0001/1K tokens), fast, high quality |
| **TTS** | Piper TTS / Edge TTS | Free, self-hosted or free API |
| **Embeddings** | local fastembed (bge-small) | $0, consistent with the Mnemosyne layer |
| **Migrations** | golang-migrate | Versioned DB migrations |
| **Frontend** | Next.js + React | SSR dashboard, real-time UI |
| **Auth** | JWT + per-tenant RBAC | Multi-tenant ready |
| **Observability** | Prometheus + health endpoints | Production monitoring |
| **Deploy** | Docker Compose → K8s | MVP to production |

---

## 5. LLM Integration Patterns

### Provider Abstraction with Fallback

```go
type LLMProvider interface {
    Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
    ChatStream(ctx context.Context, req ChatRequest) (<-chan string, error)
    StructuredOutput(ctx context.Context, req StructuredRequest) (any, error)
    Embed(ctx context.Context, text string) ([]float64, error)
    CountTokens(text string) int
}

type LLMClient struct {
    primary    LLMProvider  // DeepSeek Flash (cheap, fast)
    fallback   LLMProvider  // Secondary OpenAI-compatible provider (e.g. OpenRouter / Neuralwatt) — NOT Anthropic (expensive, contradicts cost strategy)
    metrics    *MetricsRecorder
}

func (c *LLMClient) ChatWithRetry(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
    for attempt := 0; attempt < 3; attempt++ {
        resp, err := c.primary.Chat(ctx, req)
        if err == nil {
            c.metrics.RecordSuccess("primary")
            return resp, nil
        }
        // Check if rate limited (429)
        if isRateLimited(err) {
            backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
            select {
            case <-time.After(backoff):
                continue
            case <-ctx.Done():
                return nil, ctx.Err()
            }
        }
        c.metrics.RecordError("primary", err)
        // Fallback to secondary provider
        if c.fallback != nil {
            return c.fallback.Chat(ctx, req)
        }
    }
    return nil, fmt.Errorf("all providers failed after retries")
}
```

### Token Counting (Critical)

```go
import "github.com/pkoukk/tiktoken-go"

func CountTokens(text string) int {
    encoding := "cl100k_base"
    tke, err := tiktoken.GetEncoding(encoding)
    if err != nil {
        return len(strings.Fields(text)) // fallback estimate
    }
    return len(tke.Encode(text, nil, nil))
}

// Track tokens before sending to LLM
func buildSafePrompt(ctx InterviewContext) ([]Message, error) {
    // Count all tokens
    totalTokens := CountTokens(systemPrompt) + CountTokens(transcript)
    if totalTokens > ctx.TotalTokenBudget {
        // Force truncation to oldest messages
        transcript = truncateToTokenBudget(transcript, ctx.TotalTokenBudget/2)
    }
    // ...
}
```

**Note:** `cl100k_base` ≈ DeepSeek tokenizer (5-10% drift) — good enough for a budget guard; use the provider's tokenizer if you need precision.

### Rate Limit Handling (429)

```go
func isRateLimited(err error) bool {
    var apiErr *openai.APIError
    if errors.As(err, &apiErr) {
        return apiErr.HTTPStatusCode == 429
    }
    return false
}

// Token bucket per-tenant
func (rl *RateLimiter) Allow(tenantID string, tokens int) bool {
    key := fmt.Sprintf("llm:tokens:%s:%s", tenantID, time.Now().Format("2006-01-02-15-04"))
    current, _ := rl.redis.IncrBy(ctx, key, int64(tokens))
    rl.redis.Expire(ctx, key, time.Minute)
    return current <= rl.maxTPM
}
```

### Structured Output (Function Calling)

```go
// Canonical — single source of truth (synced with schema §2 + Phase 4)
type InterviewEvaluation struct {
    OverallScore   int                  `json:"overall_score" jsonschema:"minimum=0,maximum=100"`
    Dimensions     map[string]Dimension `json:"dimensions"` // technical, communication, problem_solving, culture_fit
    PerQuestion    []PerQuestionScore   `json:"per_question"`
    Strengths      []string             `json:"strengths"`
    Weaknesses     []string             `json:"weaknesses"`
    Recommendation string               `json:"recommendation" jsonschema:"enum=proceed,enum=hold,enum=reject"`
}

type Dimension struct {
    Score  int     `json:"score"`
    Weight float64 `json:"weight"`
}

type PerQuestionScore struct {
    QuestionIdx int      `json:"question_idx"`
    Score       int      `json:"score"`
    Rationale   string   `json:"rationale"`
    Strengths   []string `json:"strengths"`
    Weaknesses  []string `json:"weaknesses"`
}

func (e *Evaluator) Evaluate(transcript []Message) (*InterviewEvaluation, error) {
    resp, err := e.llm.ChatWithRetry(ctx, ChatRequest{
        Model: "deepseek-chat",  // cheap for evaluation
        Messages: []Message{
            {Role: "system", Content: evaluationPrompt},
            {Role: "user", Content: formatTranscript(transcript)},
        },
        ResponseFormat: &ResponseFormat{Type: "json_object"},
    })
    // Validate response
    if err != nil || resp == nil {
        return nil, fmt.Errorf("evaluation failed: %w", err)
    }
    var eval InterviewEvaluation
    if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &eval); err != nil {
        return nil, fmt.Errorf("evaluation parse failed: %w", err)
    }
    // Validate fields
    if eval.OverallScore < 0 || eval.OverallScore > 100 {
        eval.OverallScore = clamp(eval.OverallScore, 0, 100)
    }
    return &eval, nil
}
```

### Response Validation Layer

```go
func (v *Validator) ValidateResponse(raw string) (*InterviewEvaluation, error) {
    // 1. Parse JSON
    var eval InterviewEvaluation
    if err := json.Unmarshal([]byte(raw), &eval); err != nil {
        // Retry with strict prompt
        return v.retryWithStrictPrompt(raw)
    }

    // 2. Validate ranges
    if eval.OverallScore < 0 || eval.OverallScore > 100 {
        eval.OverallScore = clamp(eval.OverallScore, 0, 100)
    }

    // 3. Validate enum
    validRecs := map[string]bool{"proceed": true, "hold": true, "reject": true}
    if !validRecs[eval.Recommendation] {
        eval.Recommendation = "hold" // default safe
    }

    // 3b. Dimensions: weight sum must be ≈ 1 — if not, normalize so overall_score stays consistent
    normalizeDimensionWeights(&eval)

    // 4. Content moderation
    if hasToxicContent(eval.Feedback) {
        return nil, fmt.Errorf("response failed moderation")
    }

    return &eval, nil
}

// normalizeDimensionWeights — weight sum ≈ 1 so overall_score stays consistent across tenants
func normalizeDimensionWeights(e *InterviewEvaluation) {
    var sum float64
    for _, d := range e.Dimensions {
        sum += d.Weight
    }
    if sum <= 0 || math.Abs(sum-1) < 1e-9 {
        return
    }
    for k, d := range e.Dimensions {
        d.Weight /= sum
        e.Dimensions[k] = d
    }
}
```

### Cost Optimization

| Strategy | Savings | Implementation |
|----------|---------|---------------|
| **Prompt caching** (DeepSeek prefix caching) | Up to 50% on long contexts | Structure prompts with JD + CV as prefix |
| **Semantic caching** | 20-40% on repeat queries | Redis + pgvector similarity (>0.95) |
| **Model tiering** | 30-60% | DeepSeek Flash for extraction/evaluation, DeepSeek Flash (reasoning) for complex questions |
| **Response caching** | 10-20% | Cache common interview questions |
| **Token optimization** | 10-20% | Sliding window, trim whitespace, concise prompts |
| **Streaming vs non-streaming** | Same cost | Always stream (better UX, same tokens) |
| **Count tokens before sending** | Avoids waste | Truncate early, not after paying for overflow |

---

## 6. Competition & Differentiation

### Existing AI Interview Platforms

| Platform | Chat | Voice | CV Scoring | Pricing | Go Stack | Self-Host |
|----------|------|-------|-----------|---------|----------|-----------|
| **HireVue** | ❌ | ✅ Video | ✅ | Enterprise | ❌ | ❌ |
| **Interviewer.AI** | ✅ | ✅ | ✅ | $100/mo | ❌ | ❌ |
| **Metaview** | ❌ | ✅ Notes | ❌ | $50/mo | ❌ | ❌ |
| **Testlify** | ❌ | ❌ | ✅ | $80/mo | ❌ | ❌ |
| **Willo** | ❌ | ✅ Video | ❌ | $40/mo | ❌ | ❌ |
| **Vervoe** | ✅ | ❌ | ✅ | Enterprise | ❌ | ❌ |
| **BarRaiser** | ❌ | ✅ Live | ❌ | $200/mo | ❌ | ❌ |
| **Mokka** | ✅ | ✅ | ❌ | $3-4/interview | ❌ | ❌ |
| **InterviewFlowAI** | ❌ | ✅ | ✅ | $0.99/interview | ❌ | ❌ |

### Market Size
- AI recruitment market: **$2.22B in 2026**, projected $6.4B by 2030 (30.3% CAGR)
- AI interview intelligence sub-segment is the fastest growing

### Differentiation

**Frame to the buyer outcome, not engineering features.** Recruiters/HR don't care that our backend is Go — they care about: getting good candidates faster, not missing good talent, consistent & fair evaluations.

| # | Buyer outcome | Fitur pendukung |
|---|---------------|-----------------|
| 1 | **Get matching candidates 2x faster** | CV-gap questioning: questions generated from JD↔CV gaps, not a generic question bank |
| 2 | **All-in-one pipeline, pay once** | CV screening → chat interview → voice interview → report. Competitors charge per module |
| 3 | **Consistent & fair evaluations (anti-bias lawsuit)** | Bias prevention, prohibited questions, transparent scoring, audit log |
| 4 | **Interviews relevant to company culture** | Company context + tenant system prompt (new differentiator, no competitor has it) |
| 5 | **Live, not recorded** | Live conversational AI (most competitors are async/recorded) |
| 6 | **Your data in your hands** | Self-hostable (enterprise) + on-prem option |

**Not differentiators:** "Go-native stack", "real-time scoring" (technical features — don't sell them, keep internal).

### Beachhead Market (First Target Market)

**Don't start with "all recruiters" — start with the single most painful segment:**

| Segment | Why it fits | Why first |
|--------|--------------|---------------|
| **Volume hiring SEA (BPO, call center, retail, agency recruiter)** | Hiring hundreds/month, screening cost hurts most, budget-sensitive | 🥇 First target |
| Tech startup hiring (5-20/mo) | Need quality, willing to pay | 🥈 Second |
| Enterprise (100+/mo) | Big budget, but needs SOC 2 + integrations | 🥉 Later (post-MVP) |

**Volume hiring go-to-market:**
- Pilot: 3-5 agency recruiters / BPOs in Jakarta-Bandung (Hyperscal/OCBC network)
- Channels: LinkedIn agency network, JobStreet/Glints partnership, referral from pilots
- Package: per-interview pricing (not subscription) for volume — $0.5-1/interview
- Pilot KPIs: interview completion rate + reduced screening time

### Pricing Model Recommendation (revised — usage-based for volume, subscription for enterprise)

| Tier | Price | Features |
|------|-------|----------|
| **Free** | $0 | 3 interviews, chat only (bikin ngerasain → upsell) |
| **Starter** | $49/mo | 100 interviews, chat + voice, company context |
| **Pro** | $199/mo | 1000 interviews, all features, tenant prompt, reflect |
| **Volume (BPO/agency)** | $0.5-1/interview | Usage-based, target beachhead market |
| **Enterprise** | Custom | Self-hosted, SOC 2 (post-MVP), custom integrations |

### Compliance Requirements (Enterprise Readiness)

| Requirement | Implementation |
|------------|---------------|
| **GDPR** | Data deletion API, interview consent recording, 90-day auto-delete |
| **SOC 2** | Audit logging, access controls, encryption at rest |
| **Bias detection** | Prohibited question filter, demographic fairness analysis |
| **Data sovereignty** | Self-hosted option, EU region hosting |
| **Encryption** | AES-256 at rest, TLS 1.3 in transit |
| **SSO/SAML** | Enterprise auth (Okta, Azure AD, Google Workspace) |

---

## 7. Compliance, Privacy & Legal

### Interview Recording Consent
- **Must** show consent dialog before any recording starts
- Store consent timestamp + candidate acknowledgment
- Provide opt-out (text-only interview)
- Recording retention policy (configurable, default 90 days)

### GDPR Rights
- Right to access: export all candidate data as JSON
- Right to deletion: delete candidate + all associated data
- Data Processing Agreement (DPA) for enterprise clients
- Data residency: choose region for data storage

### Bias & Fairness
- Prohibited question categories: age, religion, marital status, political affiliation, disability
- LLM prompt includes bias prevention instructions
- Post-interview bias audit: flag questions that may target protected characteristics
- Demographic fairness report (aggregate, not per-candidate)

### Data Security
- CV files encrypted at rest (AES-256)
- API keys stored in env vars, not database
- API rate limiting per key
- Audit log for all data access
- Row-Level Security (RLS) in PostgreSQL for multi-tenant isolation

---

## Go Packages Quick Reference

| Package | Purpose | Production Ready |
|---------|---------|------------------|
| `github.com/pdfcpu/pdfcpu` | PDF text extraction | ✅ |
| `github.com/gofiber/fiber/v2` | HTTP framework | ✅ |
| `github.com/gorilla/websocket` | WebSocket (chat) | ✅ |
| `github.com/pion/webrtc/v4` | WebRTC (voice) | ✅ |
| `github.com/pion/turn/v3` | TURN server for WebRTC | ✅ |
| `github.com/hibiken/asynq` | Async queue (Redis) | ✅ |
| `github.com/jackc/pgx/v5` | PostgreSQL driver (via GORM) | ✅ |
| `gorm.io/gorm` + `gorm.io/driver/postgres` | ORM — all app DB access | ✅ |
| `github.com/ledongthuc/pdf` | PDF text extraction (chosen over pdfcpu) | ✅ |
| `github.com/pgvector/pgvector-go` | pgvector for Go (raw SQL used today) | ⏳ |
| `github.com/redis/go-redis/v9` | Redis client | ✅ |
| `github.com/golang-jwt/jwt/v5` | JWT auth | ✅ |
| `github.com/pkoukk/tiktoken-go` | LLM token counting | ✅ |
| `github.com/golang-migrate/migrate` | DB migrations | ✅ |
| `github.com/rs/zerolog` | Structured logging | ✅ |
| `github.com/minio/minio-go/v7` | S3 client | ✅ |
| `github.com/getsentry/sentry-go` | Error tracking | ✅ |
| `github.com/prometheus/client_golang` | Metrics | ✅ |

| External Tool | Purpose | Go Integration |
|---------------|---------|----------------|
| **DeepSeek Flash** | LLM (cheap, fast) | OpenAI-compatible API `api.deepseek.com/v1` |
| **whisper.cpp** | Self-hosted STT | `os/exec` CLI |
| **Piper TTS** | Self-hosted TTS | `os/exec` CLI |
| **Edge TTS** | Free TTS API | `http.Get` (no API key) |
| **coturn** | TURN server (WebRTC) | Network dependency |

---

## Changes from v1

| Change | Reason |
|--------|--------|
| ✅ Added OCR section for scanned/image PDFs | pdfcpu can't handle image-only PDFs |
| ✅ Fixed unioffice license warning (AGPL) | Legal risk for commercial SaaS |
| ✅ Normalized scoring: experience capped at 100% | Prevent over-scoring overqualified canddates |
| ✅ Added configurable per-tenant weights | Each client has unique priorities |
| ✅ Added minimum score threshold | Prevent auto-advancing low scorers |
| ✅ Replaced SSE with WebSocket only | Single protocol, less complexity |
| ✅ Added heartbeat, timeout, reconnection | Production WebSocket reliability |
| ✅ Added bias prevention + prohibited questions | Legal compliance for HR SaaS |
| ✅ Added language detection | Handle multi-language candidates |
| ✅ **Replaced OpenAI Realtime WebRTC with Whisper + DeepSeek Flash + Piper/Edge TTS** | Voice stack self-hosted, $0.001/interview |
| ✅ Added WebRTC complexity warning + TURN requirement | Realistic architecture assessment |
| ✅ Added recording consent + GDPR compliance | Legal requirement |
| ✅ Added bot detection | Prevent AI-cheating |
| ✅ Fixed `UNIQUE(email)` → `UNIQUE(org_id, email)` | Multi-tenant correctness |
| ✅ Added audit log table | HR compliance requirement |
| ✅ Added interview question bank table | Reusable questions |
| ✅ Added migrations tooling | Schema versioning |
| ✅ Added health checks, metrics, graceful shutdown | Production observability |
| ✅ Added rate limit retry + exponential backoff | LLM 429 handling |
| ✅ Added token counting with tiktoken-go | Prevent context overflow |
| ✅ Added response validation layer | Handle LLM malformed output |
| ✅ Added pricing model + tiers | Go-to-market ready |
| ✅ Added enterprise compliance requirements | SOC 2, GDPR, SSO |
