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
  required_skills: string[]
  min_experience: number
  scoring_weights?: Record<string, number>
  min_score_to_proceed?: number
  status: string
  created_at: string
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

export interface Application {
  id: string
  candidate_id: string
  candidate_name: string
  candidate_email: string
  job_id: string
  job_title: string
  status: string
  cv_score?: number
  passed_screening?: boolean
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

export interface InterviewDetail {
  interview_id: string
  status: string
  context_version: number
  total_questions: number
  questions: QuestionDTO[]
  answers: AnswerDTO[]
  evaluation: EvaluationReport | null
  candidate?: { id: string; name: string; email: string }
  job?: { id: string; title: string }
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
  job_title: string
  evaluation: EvaluationReport | null
  created_at: string
}
