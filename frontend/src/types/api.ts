export interface LoginResult {
  token: string
  expires_at: string
  user: { user_id: string; org_id: string; role: string }
}

export interface RegisterResult {
  org_id: string
  user_id: string
  slug: string
  plan: string
}

export interface Job {
  id: string
  title: string
  description: string
  location?: string
  employment_type?: string
  salary_min?: number | null
  salary_max?: number | null
  currency?: string
  required_skills?: string[] | null
  min_experience: number
  responsibilities?: string[] | null
  requirements?: string[] | null
  nice_to_haves?: string[] | null
  benefits?: string[] | null
  scoring_weights?: Record<string, number>
  min_score_to_proceed?: number
  status: string
  proctoring_mode?: string
  is_published?: boolean
  created_at: string
}

export interface CompanyContextItem {
  id: string
  type: "file" | "text"
  version: number
  content_hash: string
  created_at: string
}

export interface TenantPromptResult {
  system_prompt: string
  version: number
}

export interface PublicJob extends Job {
  org_id: string
  org_name: string
  org_slug: string
}

export interface CandidateApplicationItem {
  application_id: string
  org_id: string
  org_name: string
  org_slug: string
  job_id: string
  job_title: string
  job_location: string
  job_employment_type: string
  candidate_id: string
  candidate_name: string
  candidate_email: string
  cv_score?: number | null
  passed_screening?: boolean | null
  application_status: string
  applied_at: string
  interview_id?: string | null
  interview_status?: string | null
  interview_type?: string | null
  invitation_token?: string | null
  overall_score?: number | null
  recommendation?: string | null
}

export interface CandidateOTPResponse {
  message: string
  expires_in: number
}

export interface CandidateVerifyResponse {
  token: string
  email: string
  expires_at: string
}

export interface CVListItem {
  id: string
  name: string
  email: string
  status: string
  cv_ocr_method?: string
  error_message?: string
  created_at: string
}

export interface CVDetail extends CVListItem {
  cv_path: string
  cv_raw_text?: string
  cv_structured?: unknown
}

export interface BulkUploadResponse {
  batch_id: string
}

export interface CandidatePassport {
  id: string
  email: string
  verified_profile?: unknown
  global_score?: number
  created_at: string
  updated_at: string
}

export type CandidateLifecycleStage =
  | "applied"
  | "screening_passed"
  | "screening_failed"
  | "interview_invited"
  | "interview_completed"
  | "offer_extended"
  | "hired"
  | "rejected"

export interface Application {
  id: string
  candidate_id: string
  candidate_name: string
  candidate_email: string
  job_id: string
  job_title: string
  status: string
  stage?: CandidateLifecycleStage
  cv_score?: number
  passed_screening?: boolean
  years_experience?: number
  matched_skills?: string[]
  missing_skills?: string[]
  screening_rationale?: string
  cv_path?: string
  recruiter_notes?: string
  interview_id?: string
  interview_status?: string
  interview_score?: number
  score_breakdown?: any
  recommendation?: string
  integrity_score?: number
  applied_at?: string
}

export interface CreateInterviewResult {
  interview_id: string
  invitation_token: string
  expires_at: string
  context_version: number
}

export interface QuestionDTO {
  idx: number
  content: string
  category: string
  skill?: string
}

export interface AnswerDTO {
  idx: number
  content: string
  answered_at: string
}

export interface EvaluationReport {
  overall_score: number
  dimensions: Record<string, { score: number; weight: number }>
  per_question: {
    question_idx: number
    category: string
    score: number
    rationale?: string
    strengths: string[]
    weaknesses: string[]
  }[]
  strengths: string[]
  weaknesses: string[]
  recommendation: string
}

export interface ProctoringEvent {
  type: string
  timestamp: string
  question_idx?: number
  details?: Record<string, unknown>
}

export interface ProctoringSummary {
  integrity_score: number
  risk_level: "low" | "medium" | "high"
  tab_switch_count: number
  total_away_duration_sec: number
  paste_event_count: number
  suspicious_paste_count: number
  audio_anomaly_count: number
  flags: string[]
}

export type SandboxLanguage = "go" | "python" | "typescript" | "javascript"

export interface SandboxTestCase {
  id: string
  input: string
  expected_output: string
  hidden?: boolean
}

export interface SandboxTestCaseResult {
  test_case: SandboxTestCase
  actual_output: string
  passed: boolean
  duration_ms: number
  error?: string
}

export interface SandboxExecutionResult {
  stdout: string
  stderr: string
  exit_code: number
  duration_ms: number
  memory_kb?: number
  all_passed: boolean
  test_results?: SandboxTestCaseResult[]
  error?: string
}

export interface AICodeReview {
  time_complexity: string
  space_complexity: string
  quality_score: number
  summary: string
  strengths: string[]
  improvements: string[]
}

export interface CodingSession {
  question_idx: number
  language: SandboxLanguage
  code: string
  final_result?: SandboxExecutionResult
  ai_code_review?: AICodeReview
  submitted_at: string
}

export interface InterviewDetail {
  interview_id: string
  application_id: string
  status: string
  context_version: number
  total_questions: number
  questions: QuestionDTO[]
  answers: AnswerDTO[]
  evaluation: EvaluationReport | null
  candidate?: { id: string; name: string; email: string }
  job?: { id: string; title: string }
  proctoring_events?: ProctoringEvent[]
  proctoring_summary?: ProctoringSummary
  coding_sessions?: CodingSession[]
  created_at: string
  completed_at?: string
}

export interface CandidateReport {
  candidate: { id: string; name: string; email: string }
  interviews: {
    interview_id: string
    status: string
    evaluation: EvaluationReport | null
    completed_at?: string
  }[]
}

export interface ConsentResult {
  consent_given: boolean
}

export interface InterviewListItem {
  interview_id: string
  status: string
  candidate_id: string
  candidate_name: string
  candidate_email?: string
  job_id?: string
  job_title: string
  cv_score?: number
  evaluation: EvaluationReport | null
  overall_score?: number
  recommendation?: string
  integrity_score?: number
  invitation_token?: string
  created_at: string
  completed_at?: string
}
