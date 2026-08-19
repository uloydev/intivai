import { useRef, useState } from "react"
import { useMutation, useQuery } from "@tanstack/react-query"
import { Link } from "react-router-dom"
import {
  Briefcase,
  MagnifyingGlass,
  CheckCircle,
  Sparkle,
  ArrowRight,
  CloudArrowUp,
  Clock,
  Robot,
  MapPin,
  CurrencyDollar,
  Buildings,
  Check,
  Star,
  Gift,
  Eye,
} from "@phosphor-icons/react"
import { api } from "@/lib/api"
import type { PublicJob } from "@/types/api"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { toast } from "sonner"


export function CareersPage() {
  const [search, setSearch] = useState("")
  const [selectedSkill, setSelectedSkill] = useState<string>("all")
  const [selectedJob, setSelectedJob] = useState<PublicJob | null>(null)
  const [detailModalOpen, setDetailModalOpen] = useState(false)
  const [applyModalOpen, setApplyModalOpen] = useState(false)
  const [submitted, setSubmitted] = useState(false)

  // Form fields
  const [name, setName] = useState("")
  const [email, setEmail] = useState("")
  const [file, setFile] = useState<File | null>(null)
  const fileRef = useRef<HTMLInputElement>(null)
  const [submitting, setSubmitting] = useState(false)

  // Query public active jobs from backend
  const { data: serverJobs, isLoading } = useQuery({
    queryKey: ["public-jobs"],
    queryFn: async () => {
      try {
        const res = await api.get<PublicJob[]>("/public/jobs")
        return res || []
      } catch (err) {
        console.error("Failed to load jobs", err)
        throw err
      }
    },
  })

  const jobs = serverJobs || []
  const activeJobs = jobs.filter((j) => j.status === "active")

  // Extract all unique skills
  const allSkills = Array.from(new Set(activeJobs.flatMap((j) => j.required_skills ?? [])))

  const filteredJobs = activeJobs.filter((j) => {
    const skills = j.required_skills ?? []
    const matchesSearch =
      j.title.toLowerCase().includes(search.toLowerCase()) ||
      j.description.toLowerCase().includes(search.toLowerCase()) ||
      (j.org_name && j.org_name.toLowerCase().includes(search.toLowerCase())) ||
      (j.location && j.location.toLowerCase().includes(search.toLowerCase())) ||
      skills.some((s) => s.toLowerCase().includes(search.toLowerCase()))
    const matchesSkill = selectedSkill === "all" || skills.includes(selectedSkill)
    return matchesSearch && matchesSkill
  })

  const applyMutation = useMutation({
    mutationFn: async () => {
      if (!name || !email || !file) {
        throw new Error("Please provide your name, email, and resume PDF")
      }
      if (!selectedJob) {
        throw new Error("No job selected")
      }
      const form = new FormData()
      form.append("name", name)
      form.append("email", email)
      form.append("file", file)

      return await api.postForm<{ candidate_id: string }>(`/public/jobs/${selectedJob.id}/apply`, form)
    },
    onSuccess: () => {
      setSubmitting(false)
      setSubmitted(true)
      localStorage.setItem("intivai_candidate_email", email.trim().toLowerCase())
      toast.success("Application successfully submitted and queued for AI screening!")
    },
    onError: (e) => {
      setSubmitting(false)
      toast.error(e instanceof Error ? e.message : "Submission error")
    },
  })

  function handleApplyClick(job: PublicJob) {
    setSelectedJob(job)
    setSubmitted(false)
    setName("")
    setEmail(localStorage.getItem("intivai_candidate_email") || "")
    setFile(null)
    setDetailModalOpen(false)
    setApplyModalOpen(true)
  }

  function handleViewDetails(job: PublicJob) {
    setSelectedJob(job)
    setDetailModalOpen(true)
  }

  function formatSalary(min?: number | null, max?: number | null, cur?: string) {
    if (!min && !max) return null
    const c = cur || "USD"
    if (min && max) return `$${(min / 1000).toFixed(0)}k – $${(max / 1000).toFixed(0)}k ${c}`
    if (min) return `From $${(min / 1000).toFixed(0)}k ${c}`
    if (max) return `Up to $${(max / 1000).toFixed(0)}k ${c}`
    return null
  }

  return (
    <div className="space-y-12 py-10 px-6 max-w-6xl mx-auto animate-in fade-in duration-500">
      {/* Header Banner */}
      <div className="text-center space-y-4 max-w-3xl mx-auto">
        <Badge variant="outline" className="text-primary border-primary/30 bg-primary/5 text-xs py-1 px-3">
          <Sparkle className="mr-1.5 h-3.5 w-3.5" weight="fill" /> Public Career Board
        </Badge>
        <h1 className="font-display text-4xl sm:text-5xl font-extrabold tracking-tight">
          Join High-Growth Engineering Teams
        </h1>
        <p className="text-sm sm:text-base text-muted-foreground leading-relaxed">
          Browse verified technical openings with transparent salary ranges, detailed engineering requirements, and instant AI screening feedback.
        </p>

        <div className="pt-2 flex items-center justify-center gap-3">
          <Link
            to="/candidate/portal"
            className="inline-flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-semibold bg-primary/10 border border-primary/20 text-primary hover:bg-primary/20 transition-colors"
          >
            <span>Already applied? Track your status in the Candidate Portal</span> →
          </Link>
        </div>
      </div>

      {/* Search & Skill Filters */}
      <div className="space-y-4">
        <div className="flex flex-col sm:flex-row gap-3 items-center justify-between">
          <div className="relative w-full max-w-md">
            <MagnifyingGlass className="absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="Search by role, company, skills, or location..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-10 h-11 bg-card/80 text-sm"
            />
          </div>
          <p className="text-xs text-muted-foreground">
            Showing <strong className="text-foreground">{filteredJobs.length}</strong> open positions
          </p>
        </div>

        {/* Skill Filter Tags */}
        <div className="flex flex-wrap items-center gap-1.5 pt-1">
          <button
            type="button"
            onClick={() => setSelectedSkill("all")}
            className={`text-xs rounded-full px-3 py-1 font-medium transition-all ${
              selectedSkill === "all"
                ? "bg-primary text-primary-foreground shadow-sm"
                : "bg-muted hover:bg-muted/80 text-muted-foreground"
            }`}
          >
            All Disciplines
          </button>
          {allSkills.map((skill) => (
            <button
              key={skill}
              type="button"
              onClick={() => setSelectedSkill(skill)}
              className={`text-xs rounded-full px-3 py-1 font-medium transition-all ${
                selectedSkill === skill
                  ? "bg-primary text-primary-foreground shadow-sm"
                  : "bg-muted hover:bg-muted/80 text-muted-foreground"
              }`}
            >
              {skill}
            </button>
          ))}
        </div>
      </div>

      {/* Jobs Grid */}
      {isLoading ? (
        <div className="grid gap-6 md:grid-cols-2">
          <Skeleton className="h-64 w-full rounded-2xl" />
          <Skeleton className="h-64 w-full rounded-2xl" />
        </div>
      ) : filteredJobs.length === 0 ? (
        <div className="rounded-2xl border border-dashed border-border p-16 text-center space-y-3">
          <Briefcase className="mx-auto h-12 w-12 text-muted-foreground/40" />
          <h3 className="font-display font-semibold text-lg">No matching roles found</h3>
          <p className="text-xs text-muted-foreground max-w-sm mx-auto">
            Try adjusting your search criteria or clear your skill filter to view all open opportunities.
          </p>
          <Button variant="outline" size="sm" onClick={() => { setSearch(""); setSelectedSkill("all"); }}>
            Clear Filters
          </Button>
        </div>
      ) : (
        <div className="grid gap-6 md:grid-cols-2">
          {filteredJobs.map((job) => {
            const salary = formatSalary(job.salary_min, job.salary_max, job.currency)

            return (
              <Card
                key={job.id}
                className="glass border-border/60 p-6 flex flex-col justify-between hover:border-primary/40 hover:shadow-xl hover:shadow-primary/5 transition-all duration-300 rounded-2xl group"
              >
                <div className="space-y-3.5">
                  {/* Top Company & Hiring Meta */}
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <div className="flex items-center gap-2 mb-1.5 flex-wrap">
                        <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-primary/10 text-primary border border-primary/20">
                          <Buildings className="h-3 w-3" /> {job.org_name || "Hiring Company"}
                        </span>
                        <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-muted text-muted-foreground">
                          <MapPin className="h-3 w-3" /> {job.location || "Remote"}
                        </span>
                        {job.employment_type && (
                          <span className="px-2 py-0.5 rounded-full text-[11px] font-medium bg-muted/80 text-muted-foreground">
                            {job.employment_type}
                          </span>
                        )}
                      </div>
                      <h2 className="font-display text-xl font-bold tracking-tight text-foreground group-hover:text-primary transition-colors">
                        {job.title}
                      </h2>
                    </div>

                    <Badge className="bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20 text-[10px] shrink-0">
                      Actively Hiring
                    </Badge>
                  </div>

                  {/* Salary Bracket Banner */}
                  {salary && (
                    <div className="inline-flex items-center gap-1.5 px-3 py-1 rounded-xl bg-emerald-950/30 border border-emerald-800/40 text-emerald-300 font-semibold text-xs">
                      <CurrencyDollar className="h-3.5 w-3.5 text-emerald-400" />
                      <span>{salary}</span>
                    </div>
                  )}

                  <p className="text-xs text-muted-foreground leading-relaxed line-clamp-3">
                    {job.description}
                  </p>

                  {/* Skill Chips */}
                  <div className="flex flex-wrap gap-1.5 pt-1">
                    {(job.required_skills ?? []).map((skill) => (
                      <span
                        key={skill}
                        className="rounded-lg bg-primary/5 border border-primary/15 px-2 py-0.5 text-[11px] font-medium text-foreground"
                      >
                        {skill}
                      </span>
                    ))}
                  </div>
                </div>

                {/* Footer Actions */}
                <div className="pt-6 mt-4 border-t border-border/40 flex items-center justify-between gap-3">
                  <button
                    type="button"
                    onClick={() => handleViewDetails(job)}
                    className="text-xs font-semibold text-muted-foreground hover:text-primary transition-colors flex items-center gap-1"
                  >
                    <Eye className="h-3.5 w-3.5" /> Details & Specs
                  </button>

                  <Button
                    variant="gradient"
                    size="sm"
                    className="shadow-md shadow-primary/20 rounded-xl"
                    onClick={() => handleApplyClick(job)}
                  >
                    Apply Now <ArrowRight className="ml-1.5 h-3.5 w-3.5" />
                  </Button>
                </div>
              </Card>
            )
          })}
        </div>
      )}

      {/* Rich Job Details Modal */}
      <Dialog open={detailModalOpen} onOpenChange={setDetailModalOpen}>
        <DialogContent className="max-w-2xl max-h-[85vh] overflow-y-auto">
          {selectedJob && (
            <div className="space-y-6 py-2">
              <DialogHeader>
                <div className="flex items-center gap-2 mb-2 flex-wrap">
                  <Badge variant="outline" className="text-primary border-primary/30 bg-primary/5 text-xs">
                    {selectedJob.org_name || "Company Role"}
                  </Badge>
                  <span className="text-xs text-muted-foreground flex items-center gap-1">
                    <MapPin className="h-3.5 w-3.5" /> {selectedJob.location || "Remote"}
                  </span>
                  <span className="text-xs text-muted-foreground flex items-center gap-1">
                    <Clock className="h-3.5 w-3.5" /> {selectedJob.employment_type || "Full-time"}
                  </span>
                </div>
                <DialogTitle className="font-display text-2xl font-bold">
                  {selectedJob.title}
                </DialogTitle>
                {formatSalary(selectedJob.salary_min, selectedJob.salary_max, selectedJob.currency) && (
                  <div className="pt-2">
                    <span className="px-3 py-1 rounded-xl bg-emerald-950/40 border border-emerald-800/50 text-emerald-300 font-bold text-xs">
                      💰 {formatSalary(selectedJob.salary_min, selectedJob.salary_max, selectedJob.currency)}
                    </span>
                  </div>
                )}
              </DialogHeader>

              {/* Role Overview */}
              <div className="space-y-2">
                <h4 className="text-xs font-bold uppercase tracking-wider text-primary">Role Overview</h4>
                <p className="text-xs text-muted-foreground leading-relaxed whitespace-pre-line">
                  {selectedJob.description}
                </p>
              </div>

              {/* Key Responsibilities */}
              {selectedJob.responsibilities && selectedJob.responsibilities.length > 0 && (
                <div className="space-y-2">
                  <h4 className="text-xs font-bold uppercase tracking-wider text-primary flex items-center gap-1.5">
                    <Check className="h-3.5 w-3.5" /> Key Responsibilities
                  </h4>
                  <ul className="space-y-1.5">
                    {selectedJob.responsibilities.map((resp, idx) => (
                      <li key={idx} className="text-xs text-muted-foreground flex items-start gap-2">
                        <span className="text-primary font-bold mt-0.5">•</span>
                        <span>{resp}</span>
                      </li>
                    ))}
                  </ul>
                </div>
              )}

              {/* Requirements & Qualifications */}
              {selectedJob.requirements && selectedJob.requirements.length > 0 && (
                <div className="space-y-2">
                  <h4 className="text-xs font-bold uppercase tracking-wider text-primary flex items-center gap-1.5">
                    <Star className="h-3.5 w-3.5" /> Required Qualifications
                  </h4>
                  <ul className="space-y-1.5">
                    {selectedJob.requirements.map((req, idx) => (
                      <li key={idx} className="text-xs text-muted-foreground flex items-start gap-2">
                        <span className="text-primary font-bold mt-0.5">•</span>
                        <span>{req}</span>
                      </li>
                    ))}
                  </ul>
                </div>
              )}

              {/* Nice to Haves */}
              {selectedJob.nice_to_haves && selectedJob.nice_to_haves.length > 0 && (
                <div className="space-y-2">
                  <h4 className="text-xs font-bold uppercase tracking-wider text-muted-foreground">
                    Nice to Haves & Bonus Experience
                  </h4>
                  <ul className="space-y-1.5">
                    {selectedJob.nice_to_haves.map((nice, idx) => (
                      <li key={idx} className="text-xs text-muted-foreground flex items-start gap-2">
                        <span className="text-muted-foreground font-bold mt-0.5">◦</span>
                        <span>{nice}</span>
                      </li>
                    ))}
                  </ul>
                </div>
              )}

              {/* Benefits & Perks */}
              {selectedJob.benefits && selectedJob.benefits.length > 0 && (
                <div className="space-y-2 p-4 rounded-xl bg-card border border-border/80">
                  <h4 className="text-xs font-bold uppercase tracking-wider text-emerald-400 flex items-center gap-1.5">
                    <Gift className="h-3.5 w-3.5" /> Benefits & Compensation
                  </h4>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 pt-1">
                    {selectedJob.benefits.map((ben, idx) => (
                      <div key={idx} className="text-xs text-muted-foreground flex items-center gap-2">
                        <span className="text-emerald-400 font-bold">✓</span>
                        <span>{ben}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              <DialogFooter className="pt-4 border-t border-border">
                <Button variant="secondary" onClick={() => setDetailModalOpen(false)}>
                  Close
                </Button>
                <Button variant="gradient" onClick={() => handleApplyClick(selectedJob)}>
                  Apply for this Role <ArrowRight className="ml-1.5 h-3.5 w-3.5" />
                </Button>
              </DialogFooter>
            </div>
          )}
        </DialogContent>
      </Dialog>

      {/* Application Modal */}
      <Dialog open={applyModalOpen} onOpenChange={setApplyModalOpen}>
        <DialogContent className="max-w-lg">
          {!submitted ? (
            <>
              <DialogHeader>
                <div className="flex items-center gap-2 mb-1">
                  <Badge variant="outline" className="text-primary border-primary/30 bg-primary/5 text-[10px]">
                    Direct Application
                  </Badge>
                  {selectedJob?.org_name && (
                    <span className="text-xs text-muted-foreground">• {selectedJob.org_name}</span>
                  )}
                </div>
                <DialogTitle className="font-display text-xl">
                  Apply for {selectedJob?.title}
                </DialogTitle>
                <DialogDescription className="text-xs">
                  Submit your details and PDF resume. Intivai's AI will parse your technical background and schedule your interview.
                </DialogDescription>
              </DialogHeader>

              <div className="space-y-4 py-2">
                <div className="space-y-1.5">
                  <Label htmlFor="app-name" className="text-xs font-semibold">Full Name</Label>
                  <Input
                    id="app-name"
                    placeholder="e.g. Jane Doe"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    className="bg-background/80"
                  />
                </div>

                <div className="space-y-1.5">
                  <Label htmlFor="app-email" className="text-xs font-semibold">Email Address</Label>
                  <Input
                    id="app-email"
                    type="email"
                    placeholder="jane.doe@example.com"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    className="bg-background/80"
                  />
                </div>

                <div className="space-y-1.5">
                  <Label htmlFor="app-file" className="text-xs font-semibold">Resume / CV (PDF format)</Label>
                  <Input
                    id="app-file"
                    ref={fileRef}
                    type="file"
                    accept="application/pdf"
                    onChange={(e) => setFile(e.target.files?.[0] || null)}
                    className="bg-background/80 file:mr-2 file:rounded-md file:border-0 file:bg-primary/10 file:px-2 file:py-1 file:text-xs file:font-semibold file:text-primary"
                  />
                  <p className="text-[11px] text-muted-foreground">
                    PDF text and OCR will be processed to analyze your skills for this role.
                  </p>
                </div>
              </div>

              <DialogFooter>
                <Button variant="secondary" onClick={() => setApplyModalOpen(false)}>
                  Cancel
                </Button>
                <Button
                  variant="gradient"
                  onClick={() => {
                    setSubmitting(true)
                    applyMutation.mutate()
                  }}
                  disabled={!name.trim() || !email.trim() || !file || submitting}
                >
                  <CloudArrowUp className="mr-1.5 h-4 w-4" weight="bold" />
                  {submitting ? "Analyzing & Submitting…" : "Submit Application"}
                </Button>
              </DialogFooter>
            </>
          ) : (
            <div className="py-6 text-center space-y-4">
              <div className="flex h-16 w-16 mx-auto items-center justify-center rounded-2xl bg-emerald-500/10 text-emerald-500">
                <CheckCircle className="h-10 w-10" weight="fill" />
              </div>
              <div>
                <h3 className="font-display text-2xl font-bold">Application Received!</h3>
                <p className="text-xs text-muted-foreground max-w-sm mx-auto mt-1.5 leading-relaxed">
                  Thank you, <strong className="text-foreground">{name}</strong>. Your resume has been uploaded and queued for automated AI screening against the <strong className="text-foreground">{selectedJob?.title}</strong> role requirements.
                </p>
              </div>

              <div className="rounded-xl border border-primary/20 bg-primary/5 p-4 text-xs text-muted-foreground text-left space-y-2">
                <p className="font-semibold text-foreground flex items-center gap-1.5">
                  <Robot className="h-4 w-4 text-primary" weight="fill" /> What happens next:
                </p>
                <ul className="space-y-1 pl-1">
                  <li>• Semantic engine extracts your technical skills and calculates match compatibility.</li>
                  <li>• You will receive email notifications and can track live status in the Candidate Portal.</li>
                </ul>
              </div>

              <div className="pt-2 flex flex-col sm:flex-row gap-2 justify-center">
                <Button asChild variant="gradient" className="shadow-md shadow-primary/20">
                  <Link to="/candidate/portal">
                    Track in Candidate Portal <ArrowRight className="ml-1.5 h-3.5 w-3.5" />
                  </Link>
                </Button>
                <Button
                  variant="outline"
                  onClick={() => setApplyModalOpen(false)}
                >
                  Browse More Jobs
                </Button>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
