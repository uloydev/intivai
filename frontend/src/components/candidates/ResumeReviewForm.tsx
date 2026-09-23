import * as React from "react"
import { ArrowRight } from "@phosphor-icons/react"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { TagInput } from "@/components/ui/tag-input"
import type { CVDraftProfile, ResumeData } from "@/types/api"

const NAME_MAX = 200
const EMAIL_MAX = 254
const EDUCATION_MAX = 200
const SUMMARY_MAX = 2000
const SKILLS_MAX = 50
const CERTS_MAX = 25
const EXPERIENCE_MIN = 0
const EXPERIENCE_MAX = 50
const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

type Draft = ResumeData & { name: string; email: string }

interface ResumeReviewFormProps {
  initial: Draft
  onSubmit: (draft: CVDraftProfile) => void
  submitLabel?: string
  disabled?: boolean
}

function ResumeReviewForm({ initial, onSubmit, submitLabel, disabled = false }: ResumeReviewFormProps) {
  const [name, setName] = React.useState(initial.name)
  const [email, setEmail] = React.useState(initial.email)
  const [skills, setSkills] = React.useState(initial.skills)
  const [experienceYears, setExperienceYears] = React.useState(String(initial.experience_years))
  const [education, setEducation] = React.useState(initial.education)
  const [certifications, setCertifications] = React.useState(initial.certifications)
  const [summary, setSummary] = React.useState(initial.summary)

  const nameError = name.trim().length === 0 ? "Your name is required." : null
  const emailError =
    email.trim().length > 0 && !EMAIL_RE.test(email.trim()) ? "Enter a valid email address." : null
  const expNum = Number(experienceYears)
  const expError =
    !experienceYears || Number.isNaN(expNum) || expNum < EXPERIENCE_MIN || expNum > EXPERIENCE_MAX
      ? `Experience must be a number between ${EXPERIENCE_MIN} and ${EXPERIENCE_MAX}.`
      : null

  const invalid = Boolean(nameError || emailError || expError)

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (invalid || disabled) return
    onSubmit({
      name: name.trim(),
      email: email.trim(),
      skills,
      experience_years: Number(experienceYears),
      education,
      certifications,
      summary,
    })
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-8">
      <section className="space-y-4">
        <h3 className="text-base font-semibold text-foreground">Profile</h3>
        <div className="space-y-1.5">
          <Label htmlFor="review-name">Name</Label>
          <Input
            id="review-name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            maxLength={NAME_MAX}
            required
            data-testid="resume-form-name"
          />
          {nameError ? <p className="text-xs text-destructive">{nameError}</p> : null}
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="review-email">Email</Label>
          <Input
            id="review-email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            maxLength={EMAIL_MAX}
            data-testid="resume-form-email"
          />
          {emailError ? <p className="text-xs text-destructive">{emailError}</p> : null}
          <p className="text-xs text-muted-foreground">Leave empty if you have no email.</p>
        </div>
      </section>

      <section className="space-y-4">
        <h3 className="text-base font-semibold text-foreground">Experience &amp; Education</h3>
        <div className="space-y-1.5">
          <Label htmlFor="review-exp">Experience Years</Label>
          <Input
            id="review-exp"
            type="number"
            min={EXPERIENCE_MIN}
            max={EXPERIENCE_MAX}
            step={0.5}
            value={experienceYears}
            onChange={(e) => setExperienceYears(e.target.value)}
            data-testid="resume-form-exp"
          />
          {expError ? <p className="text-xs text-destructive">{expError}</p> : null}
          <p className="text-xs text-muted-foreground">From 0 to 50 years, in half-year increments.</p>
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="review-education">Education</Label>
          <Input
            id="review-education"
            value={education}
            onChange={(e) => setEducation(e.target.value)}
            maxLength={EDUCATION_MAX}
            data-testid="resume-form-education"
          />
        </div>
      </section>

      <section className="space-y-4">
        <h3 className="text-base font-semibold text-foreground">Skills</h3>
        <TagInput
          value={skills}
          onChange={setSkills}
          max={SKILLS_MAX}
          label="Skills"
          hint="Press Enter or comma to add a skill."
        />
      </section>

      <section className="space-y-4">
        <h3 className="text-base font-semibold text-foreground">Certifications</h3>
        <TagInput
          value={certifications}
          onChange={setCertifications}
          max={CERTS_MAX}
          label="Certifications"
        />
      </section>

      <section className="space-y-4">
        <h3 className="text-base font-semibold text-foreground">Professional Summary</h3>
        <div className="space-y-1.5">
          <Label htmlFor="review-summary">Professional Summary</Label>
          <Textarea
            id="review-summary"
            rows={4}
            value={summary}
            onChange={(e) => setSummary(e.target.value)}
            maxLength={SUMMARY_MAX}
            data-testid="resume-form-summary"
          />
          <p className="text-xs text-muted-foreground">{summary.length}/{SUMMARY_MAX} characters</p>
        </div>
      </section>

      <div className="flex justify-end pt-2">
        <Button type="submit" size="lg" disabled={disabled || invalid} className="w-full sm:w-auto">
          {submitLabel ?? "Confirm & Continue to Screening"}
          <ArrowRight className="ml-2 h-4 w-4" />
        </Button>
      </div>
    </form>
  )
}

export { ResumeReviewForm }
