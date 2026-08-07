# AI Interviewer SaaS — Production-Grade Research

## Table of Contents
1. CV Parsing & Scoring
   - 1.1 PDF Text Extraction
   - 1.2 Scanned/Image-Based PDFs
   - 1.3 LLM-Based Structured Extraction
   - 1.4 Scoring Algorithm
   - 1.5 Configurable Scoring — Per-Tenant dengan Default Fallback
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

**For MVP, PDF only.** DOCX can be added later after MVP.

### ⚠️ Scanned / Image-Based PDFs

pdfcpu **cannot** extract text from scanned documents. For these, you need OCR:

| OCR Option | Approach | Pros | Cons |
|------------|----------|------|------|
| **Tesseract CLI** | `os/exec` tesseract | Free, open source, well-known | Needs tesseract installed, slower |
| **LLM vision (via Ollama)** | `llama3.2-vision` or similar | No OCR infra, multi-language | Requires GPU, slower |

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
type ResumeData struct {
    Name        string       `json:"name"`
    Email       string       `json:"email"`
    Skills      []Skill      `json:"skills"`
    Experience  []Experience `json:"experience"`
    Education   []Education  `json:"education"`
    TotalYears  float64      `json:"years_of_exp"`
}

// Use OpenAI JSON mode for structured output
resp, _ := client.CreateChatCompletion(ctx, ChatCompletionRequest{
    Model: "deepseek-v4-flash",  // murah untuk extraction
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
type ScoreResult struct {
    TotalScore     float64            `json:"total_score"`
    Breakdown      map[string]float64 `json:"breakdown"`
    MaxScore       float64            `json:"max_score"`
    Passed         bool               `json:"passed"`
}

type ScoringWeights struct {
    SkillsMatch      float64 `json:"skills_match"`      // default 0.35
    ExperienceYears  float64 `json:"experience_years"`  // default 0.20
    SemanticMatch    float64 `json:"semantic_match"`    // default 0.25
    Education        float64 `json:"education"`         // default 0.10
    Certifications   float64 `json:"certifications"`    // default 0.10
    // Per-tenant configurable
}

const MinScoreToProceed = 50.0

func ScoreResume(cv ResumeData, jd JobDescription, weights ScoringWeights) ScoreResult {
    // 1. Keyword match with normalization
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
        Passed:     total >= MinScoreToProceed,
    }
}
```

**Key scoring rules:**
- Experience capped at 100% (10yr when 5yr required = 100%, not 200%)
- Per-tenant configurable weights
- Minimum threshold to proceed (default: 50/100)
- Skills normalization: partial matches via embedding similarity

### Configurable Scoring — Per-Tenant dengan Default Fallback

Setiap tenant boleh override scoring weights, tapi semua nilai punya default global yang dipakai kalau tenant belum set / set parsial / set invalid.

**Hierarki resolusi (dari yang paling spesifik):**

```
tenant_scoring_weights (orgs.scoring_weights JSONB)
        │  kalau field ada & valid → pakai
        ▼
job_scoring_weights (jobs.scoring_weights JSONB, opsional per-JD)
        │  kalau field ada & valid → pakai
        ▼
tenant defaults (dibuat pas org register, copy dari global defaults)
        │
        ▼
GLOBAL DEFAULTS (code constant — selalu ada, ga bisa hilang)
```

**Aturan fallback:**
1. Tenant ga set apa-apa → pakai **global defaults** (0.35/0.20/0.25/0.10/0.10, threshold 50)
2. Tenant set sebagian (misal cuma SkillsMatch) → field yang ga di-set fallback ke global default
3. Tenant set nilai invalid (negatif, NaN, >1) → validasi tolak & fallback ke global default buat field itu
4. Threshold (`min_score_to_proceed`) juga per-tenant: default 50, bisa di-override per org maupun per job

**Implementasi:**

```go
// defaults.go — satu-satunya sumber default global
var GlobalScoringDefaults = ScoringWeights{
    SkillsMatch:     0.35,
    ExperienceYears: 0.20,
    SemanticMatch:   0.25,
    Education:       0.10,
    Certifications:  0.10,
}
const GlobalMinScoreToProceed = 50.0

// resolve.go — resolver dengan fallback
func (s *Server) ResolveScoringWeights(ctx context.Context, orgID, jobID uuid.UUID) (ScoringWeights, float64, error) {
    w := GlobalScoringDefaults          // 1. mulai dari global
    threshold := GlobalMinScoreToProceed

    org, err := s.repo.GetOrg(ctx, orgID)      // 2. overlay tenant
    if err != nil { return w, threshold, err }
    if org.ScoringWeights != nil {
        applyValidOverrides(&w, org.ScoringWeights) // field valid doang, sisanya default
    }
    if org.MinScoreToProceed != nil {
        threshold = *org.MinScoreToProceed
    }

    if jobID != uuid.Nil {                       // 3. overlay per-job (opsional)
        job, err := s.repo.GetJob(ctx, jobID)
        if err == nil && job.ScoringWeights != nil {
            applyValidOverrides(&w, job.ScoringWeights)
        }
        if err == nil && job.MinScoreToProceed != nil {
            threshold = *job.MinScoreToProceed
        }
    }
    return w, threshold, nil
}

// applyValidOverrides — cuma field valid yang di-override; NaN/negatif/>1 ditolak
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

**Normalisasi jumlah bobot:** kalau tenant set weights yang totalnya ≠ 1 (misal 0.5+0.5 = 1.0, atau 0.6+0.2 = 0.8), **NORMALISASI di resolver** — bagi tiap weight dengan total sum. Ini menjamin max achievable score SELALU 100, dan skor antar tenant bisa dibandingkan. Tanpa normalisasi, tenant dengan total weight 0.8 punya max score 80 — cap 100 jadi meaningless (mudah tembus) dan skor nggak konsisten.

```go
// normalizeWeights — normalisasi di resolver, BUKAN di input
func normalizeWeights(w ScoringWeights) ScoringWeights {
    total := w.SkillsMatch + w.ExperienceYears + w.SemanticMatch +
             w.Education + w.Certifications
    if total <= 0 || math.Abs(total-1.0) < 1e-9 {
        return w // total 1 atau invalid → biarkan (fallback default)
    }
    w.SkillsMatch /= total
    w.ExperienceYears /= total
    w.SemanticMatch /= total
    w.Education /= total
    w.Certifications /= total
    return w
}
```

Panggil di akhir `ResolveScoringWeights` sebelum return. Max score = 100 selalu.

**Alasan desain ini:**
- **Integritas:** fallback ke global default menjamin skor selalu valid, ga ada field kosong/NaN
- **Sederhana buat tenant:** org tinggal set 1-2 field yang mau diubah, sisanya otomatis default
- **Migrasi:** tenant baru langsung jalan dengan defaults; override hanya saat dibutuhkan
- **Audit:** breakdown menyimpan weights yang DIPAKAI per scoring (bukan yang dikonfigurasi) → traceable kenapa skor segitu

**Embeddings for semantic matching:**
- Gunakan **embedding lokal (fastembed, bge-small-en-v1.5)** — $0, konsisten sama layer Mnemosyne. Alternatif: DeepSeek embedding API kalau butuh kualitas lebih.
- Pre-compute JD embeddings on create
- Compute CV embedding on upload
- Cosine similarity between CV and JD (catches "Go" ↔ "Golang" ↔ "Go programming")

### Company Context & Tenant System Prompt

**Masalah:** AI interviewer tanpa konteks perusahaan = jawaban generik. Tenant butuh cara ngasih tau interviewer soal budaya, tech stack, values, dan preferensi pertanyaan mereka — biar interview relevan ke kebutuhan spesifik perusahaan.

**Solusi: 2 mekanisme, keduanya per-tenant, keduanya versioned, keduanya punya default fallback.**

#### A. Company Context (file atau text)

Tenant upload file (PDF/MD/TXT) atau paste text berisi konteks perusahaan:
- Values & budaya kerja
- Tech stack, product, architecture
- Role-specific requirements
- FAQ / expected answers

**Flow:**

```
tenant upload file/text (POST /orgs/:id/contexts)
  → validasi tipe + ukuran (10MB max)
  → simpan mentah di MinIO (file) / Postgres (text), hash untuk dedup
  → version bump (v1, v2, v3...)
  → queue: index_context
    → parse + chunk (file)
    → LLM summarize → ringkasan semantic
    → index ke banks/<org_id>/mnemosyne.db (per-tenant bank)
  → interview baru otomatis pake context versi terbaru
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
    version, err := s.repo.NextVersion(ctx, cmd.OrgID) // version bump
    if err != nil { return ContextVersion{}, err }

    // simpan mentah (source of truth di Postgres/MinIO)
    if err := s.storage.Save(ctx, cmd.OrgID, version, cmd.Content); err != nil {
        return ContextVersion{}, err
    }

    // async: chunk + summarize + index ke Mnemosyne bank tenant
    s.queue.Enqueue(JobTypeIndexContext, IndexContextPayload{
        OrgID: cmd.OrgID, Version: version, Hash: string(hash[:]),
    })
    return ContextVersion{Version: version}, nil
}
```

**Retrieval pas interview:** interview start → recall top-K chunk relevan dari bank tenant → inject ke system prompt interview (bounded budget, misal max 2-3K tokens). Interviewer bisa recall lagi on-demand kalau butuh detail.

#### B. Tenant System Prompt

Tenant bisa set custom system prompt buat interviewer mereka. Ini yang ngontrol *gaya* interview.

**Hierarki resolusi (sama kayak scoring):**

```
tenant_prompts (versi aktif) → kalau ada & valid → pakai
        ▼ (ga set / invalid)
GLOBAL DEFAULT INTERVIEW PROMPT (code constant)
```

**Safety rails (Wajib — jangan di-override tenant):**

| Boleh tenant set | DILOCK (pinned, ga bisa diubah) |
|---|---|
| Tone & gaya interview | Anti-bias rules |
| Fokus skill / prioritas pertanyaan | Prohibited questions |
| Company values, budaya, tech stack | GDPR/consent flow |
| Role-specific instruction | Bot detection, max duration, emergency stop |

Implementasi: global default prompt jadi **anchor**; tenant prompt di-append/inject di posisi yang aman, safety rails tetap hard-coded setelah tenant prompt. Jangan pernah tenant prompt replace seluruh system prompt.

```go
// internal/interview/domain/service/prompt_composer.go
func ComposeInterviewSystemPrompt(
    tenantPrompt *string,     // opsional, tenant override
    companyContext []string,  // recall dari Mnemosyne bank
    safetyRails string,       // HARD-CODED, tidak bisa diubah
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

**Kenapa safety rails di paling akhir:** LLM cenderung nurutin instruksi yang paling dekat dengan akhir prompt. Posisi paling bawah = paling kuat. Tenant prompt di tengah = bisa override default tapi ga bisa nutup safety rails.

**Aturan validasi tenant prompt:**
- Max length (misal 4K chars) — cegah prompt injection via panjang
- Max context budget per interview (misal 3K tokens company context) — cegah overshoot token
- Tenant prompt ga boleh mengandung instruksi yang ngerusak integrity (deteksi keyword: "pass all", "ignore safety", "always hire")

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

**Audit:** interview simpan `context_version` + `prompt_version` yang dipakai → traceable: "interview ini pake company context v3, tenant prompt v2, safety rails v1".

---

## 2. AI Interview (Chat)

### Architecture
```
Browser ──WebSocket──▶ Go Server ──WebSocket──▶ LLM API (streaming)
                           │
                      Save to PostgreSQL
                      (messages, evaluations)
```

**Single protocol: WebSocket for both directions.** No SSE. Keeps it simple.

### WebSocket Chat Flow
```
Client connects → Server upgrades to WS
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
  "technical_score": 82,
  "communication_score": 75,
  "culture_fit_score": 70,
  "strengths": ["Go expertise", "System design"],
  "weaknesses": ["Limited cloud experience"],
  "recommendation": "proceed"
}
```

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

**MVP approach:** Use **whisper.cpp** via `os/exec` in Go. Start with `tiny` model (fast, decent accuracy), upgrade to `small` or `base` later.

```go
func transcribeAudio(audioPath string) (string, error) {
    cmd := exec.Command("whisper.cpp", "--model", "tiny", "--output-txt", audioPath)
    output, err := cmd.Output()
    return string(output), err
}
```

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
| **Deepgram + DeepSeek + ElevenLabs** | ~$0.28 | 3 paid APIs (opsi premium, bukan MVP) |
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

**Recommendation:** Use **LiveKit** as your WebRTC layer. It handles signaling, SFU, TURN, recording out of the box. The LiveKit server is Go. You write the AI agent as a separate service that subscribes to the audio stream.

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

### Candidate Access & Interview Link (Auth Flow Kandidat)

Kandidat itu **bukan user internal** — mereka ga punya akun. Flow akses interview:

```sql
-- Interview invitation token (short-lived, 1x pakai)
CREATE TABLE interview_tokens (
    id UUID PRIMARY KEY,
    interview_id UUID REFERENCES interviews(id),
    token TEXT UNIQUE NOT NULL,      -- random 32-char, high entropy
    expires_at TIMESTAMPTZ NOT NULL, -- default +7 hari
    used_at TIMESTAMPTZ,             -- 1x pakai
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

```go
// Flow:
// 1. Recruiter create interview → generate token
// 2. Email ke kandidat: https://app.intivai.com/i/<token>
// 3. Candidate klik → token divalidasi (exists, belum used, belum expire)
// 4. Interview dimulai → token di-mark used
// 5. Reconnect: session_id + token (bukan re-use token baru)
```

**Rules:**
- Token: crypto/rand 32-byte, URL-safe — ga bisa ditebak
- Expire: 7 hari dari invite, 1x pakai
- Reconnect pakai session_id yang beda dari token
- Rate limit per-IP auth attempts (10/min)
- Revoke: recruiter bisa revoke token sebelum dipakai

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
│   │   ├── evaluator.go   # LLM evaluation
│   │   ├── scheduler.go   # Interview scheduling
│   │   └── session.go     # Timeout, heartbeat, reconnection
│   ├── job/           # Job description management
│   ├── user/          # Auth, orgs, roles (multi-tenant)
│   ├── api/           # HTTP handlers (Fiber)
│   │   ├── middleware/
│   │   │   ├── auth.go        # JWT + tenant context
│   │   │   ├── ratelimit.go   # Per-tenant token bucket
│   │   │   ├── cors.go        # CORS configuration
│   │   │   └── audit.go       # Request audit logging
│   │   ├── handler/
│   │   └── ws/            # WebSocket hub + heartbeat
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
    scoring_weights JSONB,  -- per-tenant configurable weights (partial override; fallback ke global defaults)
    min_score_to_proceed DOUBLE PRECISION,  -- per-tenant threshold override (default 50 via code)
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Users (⚠️ UNIQUE per org_id, not global)
CREATE TABLE users (
    id UUID PRIMARY KEY,
    org_id UUID REFERENCES orgs(id),
    email TEXT NOT NULL,
    role TEXT DEFAULT 'member',  -- admin, recruiter, interviewer
    password_hash TEXT NOT NULL,
    UNIQUE(org_id, email)  -- same email allowed in different orgs
);

-- Interview Questions (reusable bank)
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
    scoring_weights JSONB,  -- per-job override (opsional; fallback ke org → global)
    min_score_to_proceed DOUBLE PRECISION DEFAULT 50,
    embedding VECTOR(1536),
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
    cv_embedding VECTOR(1536),
    cv_ocr_method TEXT,  -- pdfcpu, tesseract, gpt4o-vision
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
-- Wajib: aktifkan di SEMUA tabel yang punya org_id
-- ═══════════════════════════════════════════════════════
-- Setiap request set app.org_id via middleware tenant:
--   SELECT set_config('app.org_id', $1, true);  -- per-transaction

-- Contoh lengkap (ulangi per tabel: candidates, jobs, applications, interviews, audit_logs, company_contexts, tenant_prompts, questions)
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
-- interviews ga punya org_id langsung → via join application → job
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

-- users: admin bisa liat semua user org-nya
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_users ON users
    USING (org_id = NULLIF(current_setting('app.org_id', true), '')::uuid);

-- ⚠️ orgs: special case — perlu akses org sendiri buat set weights.
-- Pake policy yang mengizinkan read org yang match app.org_id.
ALTER TABLE orgs ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_orgs ON orgs
    USING (id = NULLIF(current_setting('app.org_id', true), '')::uuid);
```

### Hybrid Memory DB — Semantic Recall (Mnemosyne per-tenant)

**Masalah:** pgvector di Postgres bagus buat similarity search sederhana (CV ↔ job), tapi lemah buat *semantic recall lintas data*: "kandidat mana yang jawabannya menunjukkan pola skill yang sama dengan kandidat yang LOLOS kemarin?" atau "dari 50 interview, pertanyaan mana yang paling sering gagal?" — itu butuh memory layer dengan entity resolution + knowledge graph + refleksi lintas-interview.

**Solusi: hybrid 2 layer.**

```
┌──────────────────────────────────────────────────────────┐
│  PostgreSQL  (SOURCE OF TRUTH — integritas data)          │
│  • orgs, users, candidates, jobs, interviews              │
│  • ACID, FK, RLS (tenant isolation), transactions, audit  │
│  • pgvector utk embedding primitif (cv ↔ job match)       │
└───────────────┬──────────────────────────────────────────┘
                │ sync via Go worker (outbox / event bus)
┌───────────────▼──────────────────────────────────────────┐
│  Mnemosyne  (SEMANTIC LAYER — 1 bank per tenant)         │
│  • banks/<org_id>/mnemosyne.db  (SQLite terisolasi)      │
│  • entity resolution + knowledge graph                   │
│  • recall semantik multi-strategy                        │
│  • reflect: sintesis lintas-memori                        │
└──────────────────────────────────────────────────────────┘
```

#### Kenapa hybrid (bukan Mnemosyne jadi satu-satunya DB)

| Kebutuhan | Postgres | Mnemosyne |
|-----------|----------|-----------|
| **Integritas data** (ACID, FK, constraints) | ✅ | ❌ (memory layer, bukan relational DB) |
| **Query relasional kompleks** (JOIN, reporting) | ✅ | ❌ |
| **Audit & billing** (immutable, transactional) | ✅ | ❌ |
| **Semantic recall** ("kandidat mirip yang lolos") | 🟡 pgvector dangkal | ✅ vector + BM25 + entity |
| **Knowledge graph** (kandidat↔skill↔interview) | ❌ | ✅ |
| **Refleksi lintas-interview** (pola, insight) | ❌ | ✅ `reflect` |
| **Tenant isolation** | ✅ RLS | ✅ bank per tenant (level DB) |

**Data integrity:**
- Postgres = source of truth. Semua data struktural (candidate, interview, score, payment) TETAP di Postgres.
- Mnemosyne cuma nyimpen **embedding + entity + reference** (bukan transcript mentah).
- Kalau Mnemosyne hilang/down → di-rebuild dari Postgres (cache, bukan sumber).
- PII sensitif (transcript, CV) tidak dikirim ke LLM untuk extraction di tiap write — extraction cuma di layer semantic index.

#### Tenant Isolation — Bank per Tenant

Mnemosyne punya **Memory Bank Isolation** native:

```
~/.hermes/mnemosyne/data/banks/<org_id>/mnemosyne.db
```

- **1 tenant = 1 SQLite bank terpisah** — isolasi di level file/database (bukan sekadar `WHERE tenant_id`).
- **⚠️ Mnemosyne itu Python package — dari Go backend, panggil lewat MCP server (stdio) atau HTTP API, BUKAN SDK langsung.** Definisikan port Go sendiri:

```go
// internal/memory/domain/memory_port.go
// Port (Go) — implementasi adapter ke Mnemosyne via MCP/HTTP
type MemoryBank interface {
    Remember(ctx context.Context, entityType, summary string, importance float64) error
    Recall(ctx context.Context, query string, budget string) ([]MemoryHit, error)
    Reflect(ctx context.Context, question string) (string, error)
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
// internal/memory/infrastructure/mcp/mnemosyne_mcp.go
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

- **Dobel isolasi:** Postgres RLS (row-level) + Mnemosyne bank (file-level). Data tenant A ga mungkin ke-recall dari tenant B.

#### Semantic Indexing Pipeline (Go worker)

Setelah event di Postgres, worker sinkronisasi index ke Mnemosyne:

```go
// Event: candidate scored / interview completed
type SyncEvent struct {
    OrgID       string `json:"org_id"`
    EntityType  string `json:"entity_type"` // candidate, interview, job
    EntityID    string `json:"entity_id"`
    Summary     string `json:"summary"`     // ringkasan semantic (bukan PII mentah)
    Importance  float64 `json:"importance"`
}

func (s *Server) syncToMnemosyne(ctx context.Context, ev SyncEvent) error {
    mn := s.memory.ForBank(ev.OrgID) // adapter MCP/HTTP ke banks/<org_id>/
    return mn.Remember(ctx, ev.EntityType, ev.Summary, ev.Importance)
}
```

Event yang di-index:
- `candidate_profile` — ringkasan CV (nama, skill, ex, catatan) → importance tinggi
- `interview_summary` — ringkasan interview (jawaban inti, kesan) → importance sedang
- `interview_reflection` — refleksi pasca-interview → pemicu `reflect`
- `job_requirements` — skill wajib buat matching

#### Recall & Reflect (use case konkret)

```go
// 1. Semantic search kandidat (bukan keyword) — lewat port Go
res, _ := mn.Recall(ctx, "kandidat kuat di Go + pernah fintech payment", "high")

// 2. Knowledge graph query: skill yang dimiliki kandidat yang lolos
// (via MCP tool mnemosyne_graph_query)
graph, _ := mn.EntityQuery(ctx, "candidate", "passed_screening=true")

// 3. Refleksi lintas-interview: pola pertanyaan yang gagal
insight, _ := mn.Reflect(ctx, "dari 50 interview terakhir, pertanyaan mana yang paling sering gagal dan kenapa?")
```

#### Keuntungan buat produk

| Fitur produk | Implementasi |
|--------------|--------------|
| "Cari kandidat mirip yang pernah lolos" | recall semantik + entity |
| "Skill gap analysis" | graph query (skills of passing candidates) |
| "Pola interview" | reflect lintas interview |
| "Perbaiki pertanyaan bank" | reflect: pertanyaan dengan fail-rate tinggi |
| Compliance | bank terpisah + hapus bank = hapus semua memori tenant |

**Catatan cost:** Mnemosyne pakai embedding lokal (fastembed, bge-small) = **$0, ga ada LLM call di index-time** (kecuali extraction optional). Ini lebih murah dari pgvector + external embedding API, dan cocok buat server 1 core.

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

### Backup & Disaster Recovery (WAJIB — survival)

Solo founder + 1 server = 1 bad day dari kehilangan semua. Minimal:

```bash
# 1. PostgreSQL — daily dump ke MinIO (cron)
# /etc/cron.d/pg-backup
0 3 * * * pg_dump -Fc $DATABASE_URL | s3cmd put - s3://backups/pg-$(date +\%F).dump

# 2. MinIO replication / offsite
# rclone sync ke object storage eksternal (Backblaze B2 ~$6/TB)
0 4 * * * rclone sync s3://backups b2:hermes-backups --fast-list

# 3. Mnemosyne banks — SQLite file, backup bersama
# banks/<org_id>/mnemosyne.db → rebuildable dari Postgres (cache), tapi backup juga murah
0 5 * * * tar czf - ~/.hermes/mnemosyne/data/banks | s3cmd put - s3://backups/mnemosyne-$(date +\%F).tar.gz

# 4. Config & .env — git (tanpa secret) + encrypted copy
```

**Recovery target:**
- RPO ≤ 24 jam (daily backup)
- RTO ≤ 1 jam (restore dump + rebuild Mnemosyne dari Postgres)
- Test restore bulanan — backup yang ga pernah di-restore = ga ada

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
| **LLM** | DeepSeek Flash (via API) | Cheap ($0.0001/1K tokens), fast, high quality |
| **TTS** | Piper TTS / Edge TTS | Free, self-hosted or free API |
| **Embeddings** | fastembed lokal (bge-small) | $0, konsisten sama Mnemosyne layer |
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
    Embed(ctx context.Context, text string) ([]float64, error)
    CountTokens(text string) int
}

type LLMClient struct {
    primary    LLMProvider  // DeepSeek Flash (cheap, fast)
    fallback   LLMProvider  // Secondary OpenAI-compatible provider (e.g. OpenRouter / Neuralwatt) — BUKAN Anthropic (mahal, kontradiksi strategi cost)
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
type InterviewEvaluation struct {
    OverallScore      int      `json:"overall_score" jsonschema:"minimum=0,maximum=100"`
    TechnicalScore    int      `json:"technical_score"`
    CommunicationScore int    `json:"communication_score"`
    Strengths         []string `json:"strengths"`
    Weaknesses        []string `json:"weaknesses"`
    Recommendation    string   `json:"recommendation" jsonschema:"enum=proceed,enum=hold,enum=reject"`
}

func (e *Evaluator) Evaluate(transcript []Message) (*InterviewEvaluation, error) {
    resp, err := e.llm.ChatWithRetry(ctx, ChatRequest{
        Model: "deepseek-v4-flash",  // murah untuk evaluasi
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

    // 4. Content moderation
    if hasToxicContent(eval.Feedback) {
        return nil, fmt.Errorf("response failed moderation")
    }

    return &eval, nil
}
```

### Cost Optimization

| Strategy | Savings | Implementation |
|----------|---------|---------------|
| **Prompt caching** (DeepSeek prefix caching) | Up to 50% on long contexts | Structure prompts with JD + CV as prefix |
| **Semantic caching** | 20-40% on repeat queries | Redis + pgvector similarity (>0.95) |
| **Model tiering** | 30-60% | DeepSeek Flash untuk extraction/evaluation, DeepSeek Flash (reasoning) untuk complex questions |
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

**Frame ke outcome buyer, bukan fitur engineering.** Recruiter/HR nggak peduli backend kita Go — mereka peduli: dapet kandidat bagus lebih cepet, nggak ketinggalan talent bagus, evaluasi konsisten & adil.

| # | Buyer outcome | Fitur pendukung |
|---|---------------|-----------------|
| 1 | **Dapet kandidat yang cocok 2x lebih cepet** | CV-gap questioning: pertanyaan digenerate dari gap JD↔CV, bukan question bank generik |
| 2 | **All-in-one pipeline, bayar sekali** | CV screening → chat interview → voice interview → report. Kompetitor charge per module |
| 3 | **Evaluasi konsisten & adil (anti bias lawsuit)** | Bias prevention, prohibited questions, scoring transparan, audit log |
| 4 | **Interview relevan ke budaya perusahaan** | Company context + tenant system prompt (differentiator baru, ga ada kompetitor yang punya) |
| 5 | **Live, bukan rekaman** | Live conversational AI (kebanyakan kompetitor async/recorded) |
| 6 | **Data di tangan sendiri** | Self-hostable (enterprise) + on-prem option |

**Bukan differentiator:** "Go-native stack", "real-time scoring" (fitur teknis — jangan dijual, cuma di-internal).

### Beachhead Market (Target Pasar Pertama)

**Jangan mulai dari "semua recruiter" — mulai dari satu segmen yang paling sakit:**

| Segmen | Kenapa cocok | Kenapa duluan |
|--------|--------------|---------------|
| **Volume hiring SEA (BPO, call center, retail, agency recruiter)** | Hire ratusan/bulan, screening cost paling sakit, budget sensitif | 🥇 Target pertama |
| Tech startup hiring (5-20/bln) | Butuh quality, mau bayar | 🥈 Kedua |
| Enterprise (100+/bln) | Budget gede, tapi butuh SOC 2 + integrasi | 🥉 Nanti (post-MVP) |

**Go-to-market volume hiring:**
- Pilot: 3-5 agency recruiter / BPO di Jakarta-Bandung (network Hyperscal/OCBC)
- Channel: LinkedIn agency network, JobStreet/Glints partnership, referral dari pilot
- Paket: per-interview pricing (bukan subscription) buat volume — $0.5-1/interview
- KPI pilot: interview completion rate + waktu screening turun

### Pricing Model Recommendation (revisi — usage-based buat volume, subscription buat enterprise)

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
| `github.com/jackc/pgx/v5` | PostgreSQL driver | ✅ |
| `github.com/pgvector/pgvector-go` | pgvector for Go | ✅ |
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
| ✅ **Replaced OpenAI Realtime WebRTC dengan Whisper + DeepSeek Flash + Piper/Edge TTS** | Stack voice self-hosted, $0.001/interview |
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
