import { useRef, useState } from "react"
import { Link } from "react-router-dom"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  CloudArrowUp,
  FilePdf,
  MagnifyingGlass,
  ArrowClockwise,
  CheckCircle,
  XCircle,
  Sparkle,
  Briefcase,
  Trash,
  Files,
  Clock,
} from "@phosphor-icons/react"
import { api } from "@/lib/api"
import type { CVListItem, Job } from "@/types/api"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Skeleton } from "@/components/ui/skeleton"
import { toast } from "sonner"

const POLL_STATUSES = new Set(["new", "parsing", "extracting", "pending_review"])
const QUEUE_STATUSES = new Set(["new", "pending_review"])

function statusBadge(status: string) {
  if (status === "parsed" || status === "extracted") {
    return (
      <Badge className="bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20 gap-1">
        <CheckCircle className="h-3 w-3" weight="fill" /> {status}
      </Badge>
    )
  }
  if (status === "failed_ocr" || status === "failed_extract" || status === "failed_parse") {
    return (
      <Badge variant="destructive" className="gap-1">
        <XCircle className="h-3 w-3" weight="fill" /> {status}
      </Badge>
    )
  }
  if (QUEUE_STATUSES.has(status)) {
    // Queued states can sit for a while (extraction backlog, candidate
    // review) — a static badge instead of an infinite pulsing spinner.
    return (
      <Badge variant="secondary" className="bg-sky-500/10 text-sky-600 dark:text-sky-400 border-sky-500/20 gap-1">
        <Clock className="h-3 w-3" /> {status === "pending_review" ? "Pending review" : "In queue"}
      </Badge>
    )
  }
  return (
    <Badge variant="secondary" className="bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20 animate-pulse gap-1">
      <ArrowClockwise className="h-3 w-3 animate-spin" /> {status}…
    </Badge>
  )
}

export function CVsPage() {
  const qc = useQueryClient()
  const { data: cvs, isLoading, error } = useQuery({
    queryKey: ["cvs"],
    queryFn: () => api.get<CVListItem[]>("/cvs"),
    refetchInterval: (query) =>
      query.state.data?.some((c) => POLL_STATUSES.has(c.status)) ? 2000 : false,
  })

  const { data: jobs } = useQuery({
    queryKey: ["jobs"],
    queryFn: () => api.get<Job[]>("/jobs"),
  })

  const [name, setName] = useState("")
  const [email, setEmail] = useState("")
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const fileRef = useRef<HTMLInputElement>(null)
  const [uploading, setUploading] = useState(false)
  const [search, setSearch] = useState("")

  const [uploadMode, setUploadMode] = useState<"single" | "bulk">("single")
  const [bulkFiles, setBulkFiles] = useState<File[]>([])
  const bulkFileRef = useRef<HTMLInputElement>(null)

  // Screen candidate dialog state
  const [screenCandidate, setScreenCandidate] = useState<CVListItem | null>(null)
  const [selectedJobId, setSelectedJobId] = useState<string>("")

  const upload = useMutation({
    mutationFn: async () => {
      const file = selectedFile || fileRef.current?.files?.[0]
      if (!file) throw new Error("Pick a PDF first")
      const form = new FormData()
      form.append("name", name)
      form.append("email", email)
      form.append("file", file)
      return api.postForm<{ id: string }>("/cvs", form)
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["cvs"] })
      setUploading(false)
      setName("")
      setEmail("")
      setSelectedFile(null)
      if (fileRef.current) fileRef.current.value = ""
      toast.success("CV uploaded — OCR & extraction pipeline started")
    },
    onError: (e) => {
      setUploading(false)
      toast.error(e instanceof Error ? e.message : "Upload failed")
    },
  })

  const bulkUpload = useMutation({
    mutationFn: async () => {
      if (!bulkFiles.length && !bulkFileRef.current?.files?.length) {
        throw new Error("Pick at least one PDF first")
      }
      const filesToUpload = bulkFiles.length ? bulkFiles : Array.from(bulkFileRef.current?.files || [])
      const form = new FormData()
      filesToUpload.forEach((f) => form.append("files", f))
      return api.postForm<{ batch_id: string }>("/cvs/bulk", form)
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["cvs"] })
      setUploading(false)
      setBulkFiles([])
      if (bulkFileRef.current) bulkFileRef.current.value = ""
      toast.success("Bulk CVs uploaded — processing pipeline started")
    },
    onError: (e) => {
      setUploading(false)
      toast.error(e instanceof Error ? e.message : "Bulk upload failed")
    },
  })

  const reExtract = useMutation({
    mutationFn: (id: string) => api.post<CVListItem>(`/cvs/${id}/extract`, {}),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["cvs"] })
      toast.success("Re-extraction enqueued")
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Retry failed"),
  })

  const deleteCV = useMutation({
    mutationFn: (id: string) => api.delete(`/cvs/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["cvs"] })
      qc.invalidateQueries({ queryKey: ["applications"] })
      toast.success("Candidate resume deleted")
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Delete failed"),
  })

  const screenMutation = useMutation({
    mutationFn: async () => {
      if (!screenCandidate || !selectedJobId) throw new Error("Select a job")
      return api.post("/screenings", {
        candidate_id: screenCandidate.id,
        job_id: selectedJobId,
      })
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["applications"] })
      setScreenCandidate(null)
      setSelectedJobId("")
      toast.success("Screening started! Check Candidates tab for score.")
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Screening failed"),
  })

  const filteredCVs = (cvs ?? []).filter((c) => {
    const name = (c.name || "").toLowerCase()
    const email = (c.email || "").toLowerCase()
    const status = (c.status || "").toLowerCase()
    const q = (search || "").toLowerCase()
    return name.includes(q) || email.includes(q) || status.includes(q)
  })

  return (
    <div className="space-y-6 animate-in fade-in duration-500">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="font-display text-3xl font-bold tracking-tight">CV Ingestion Hub</h1>
          <p className="text-sm text-muted-foreground">
            Automated PDF text extraction, OCR engine fallback, and structured skill profiling.
          </p>
        </div>
      </div>

      {/* Upload Card */}
      <Card className="glass border-primary/20 bg-gradient-to-b from-card via-card to-primary/5 shadow-md">
        <CardHeader className="pb-3 border-b border-border/50 mb-4">
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="font-display text-lg flex items-center gap-2">
                <CloudArrowUp className="h-5 w-5 text-primary" weight="bold" /> Upload Candidate Resume(s)
              </CardTitle>
              <CardDescription>
                PDF documents will be automatically parsed via poppler/tesseract and vectorized.
              </CardDescription>
            </div>
            <div className="flex bg-muted p-1 rounded-lg">
              <button
                className={`px-3 py-1 text-xs font-semibold rounded-md transition-colors ${uploadMode === "single" ? "bg-background shadow-sm text-foreground" : "text-muted-foreground hover:text-foreground"}`}
                onClick={() => setUploadMode("single")}
              >
                Single Candidate
              </button>
              <button
                className={`px-3 py-1 text-xs font-semibold rounded-md transition-colors ${uploadMode === "bulk" ? "bg-background shadow-sm text-foreground" : "text-muted-foreground hover:text-foreground"}`}
                onClick={() => setUploadMode("bulk")}
              >
                Bulk Upload
              </button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {uploadMode === "single" ? (
            <div className="grid gap-4 md:grid-cols-12 animate-in fade-in zoom-in-95">
              <div className="space-y-1.5 md:col-span-3">
                <Label htmlFor="cv-name" className="text-xs font-semibold">Candidate Full Name</Label>
                <Input
                  id="cv-name"
                  placeholder="e.g. Alex Morgan"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="bg-background/80"
                />
              </div>
              <div className="space-y-1.5 md:col-span-3">
                <Label htmlFor="cv-email" className="text-xs font-semibold">Candidate Email</Label>
                <Input
                  id="cv-email"
                  type="email"
                  placeholder="alex@example.com"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="bg-background/80"
                />
              </div>
              <div className="space-y-1.5 md:col-span-4">
                <Label htmlFor="cv-file" className="text-xs font-semibold">Resume File (PDF)</Label>
                <div className="flex items-center gap-2">
                  <Input
                    id="cv-file"
                    ref={fileRef}
                    type="file"
                    accept="application/pdf"
                    onChange={(e) => setSelectedFile(e.target.files?.[0] || null)}
                    className="bg-background/80 file:mr-2 file:rounded-md file:border-0 file:bg-primary/10 file:px-2 file:py-1 file:text-xs file:font-semibold file:text-primary"
                  />
                </div>
              </div>
              <div className="flex items-end md:col-span-2">
                <Button
                  className="w-full shadow-sm"
                  variant="gradient"
                  onClick={() => {
                    setUploading(true)
                    upload.mutate()
                  }}
                  disabled={uploading || !name.trim() || !email.trim() || (!selectedFile && !fileRef.current?.files?.[0])}
                >
                  <CloudArrowUp className="mr-1.5 h-4 w-4" weight="bold" />
                  {uploading ? "Ingesting…" : "Ingest CV"}
                </Button>
              </div>
            </div>
          ) : (
            <div className="flex flex-col gap-4 animate-in fade-in zoom-in-95">
              <div className="space-y-1.5 w-full">
                <Label htmlFor="cv-bulk" className="text-xs font-semibold">Select Multiple PDFs</Label>
                <div className="flex flex-col sm:flex-row items-start sm:items-center gap-4">
                  <Input
                    id="cv-bulk"
                    ref={bulkFileRef}
                    type="file"
                    accept="application/pdf"
                    multiple
                    onChange={(e) => setBulkFiles(Array.from(e.target.files || []))}
                    className="bg-background/80 flex-1 file:mr-2 file:rounded-md file:border-0 file:bg-primary/10 file:px-2 file:py-1 file:text-xs file:font-semibold file:text-primary"
                  />
                  <Button
                    className="shadow-sm shrink-0"
                    variant="gradient"
                    onClick={() => {
                      setUploading(true)
                      bulkUpload.mutate()
                    }}
                    disabled={uploading || (!bulkFiles.length && !bulkFileRef.current?.files?.length)}
                  >
                    <Files className="mr-1.5 h-4 w-4" weight="bold" />
                    {uploading ? "Ingesting Batch…" : `Ingest ${bulkFiles.length || bulkFileRef.current?.files?.length || 0} CVs`}
                  </Button>
                </div>
              </div>
              <p className="text-xs text-muted-foreground">
                <strong className="text-foreground">Note:</strong> Bulk uploaded CVs will use the PDF filename as the candidate name. HR or Candidates can review and correct the profile later.
              </p>
            </div>
          )}
        </CardContent>
      </Card>

      {/* CV Directory & Filter */}
      <div className="space-y-4">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="relative w-full max-w-sm">
            <MagnifyingGlass className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="Search candidate name, email, or status..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-9 bg-background/80"
            />
          </div>
          <span className="text-xs text-muted-foreground">
            Showing {filteredCVs.length} of {cvs?.length ?? 0} CVs
          </span>
        </div>

        {isLoading ? (
          <div className="space-y-3">
            <Skeleton className="h-16 w-full rounded-xl" />
            <Skeleton className="h-16 w-full rounded-xl" />
            <Skeleton className="h-16 w-full rounded-xl" />
          </div>
        ) : error ? (
          <div className="rounded-xl border border-destructive/30 bg-destructive/5 p-6 text-center text-sm text-destructive">
            {error instanceof Error ? error.message : "Failed to load CVs"}
          </div>
        ) : filteredCVs.length === 0 ? (
          <div className="rounded-2xl border border-dashed border-border/80 p-12 text-center">
            <FilePdf className="mx-auto h-12 w-12 text-muted-foreground/40 mb-3" />
            <p className="font-display font-semibold text-base">No resumes found</p>
            <p className="text-xs text-muted-foreground mt-1 max-w-sm mx-auto">
              Upload candidate resumes above. Once ingested, they will automatically appear here with extraction telemetry.
            </p>
          </div>
        ) : (
          <div className="grid gap-3">
            {filteredCVs.map((cv) => (
              <div
                key={cv.id}
                className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between rounded-xl border border-border/60 bg-card p-4 shadow-sm transition-all hover:border-primary/30 hover:shadow-md"
              >
                <div className="flex items-start gap-3 min-w-0">
                  <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                    <FilePdf className="h-5 w-5" weight="bold" />
                  </div>
                  <div className="min-w-0">
                    <p className="font-display font-semibold text-sm truncate">{cv.name}</p>
                    <p className="text-xs text-muted-foreground truncate">{cv.email}</p>
                    {cv.error_message && (
                      <p className="mt-1 text-xs text-destructive truncate max-w-md">Error: {cv.error_message}</p>
                    )}
                  </div>
                </div>

                <div className="flex flex-wrap items-center gap-2.5 shrink-0">
                  {statusBadge(cv.status)}

                  {/* Actions based on status */}
                  {(cv.status === "extracted" || cv.status === "parsed") && (
                    <>
                      <Button
                        variant="outline"
                        size="sm"
                        className="h-8 text-xs border-primary/30 text-primary hover:bg-primary/10 gap-1"
                        onClick={() => setScreenCandidate(cv)}
                      >
                        <Briefcase className="h-3.5 w-3.5" weight="bold" /> Screen for Role
                      </Button>
                      <Button
                        asChild
                        variant="ghost"
                        size="sm"
                        className="h-8 text-xs text-muted-foreground hover:text-foreground gap-1"
                      >
                        <Link to={`/candidates?candidate_id=${cv.id}`}>
                          Candidate 360 →
                        </Link>
                      </Button>
                    </>
                  )}

                  {(cv.status === "failed_extract" || cv.status === "failed_ocr" || cv.status === "failed_parse") && (
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-8 text-xs gap-1"
                      onClick={() => reExtract.mutate(cv.id)}
                      disabled={reExtract.isPending}
                    >
                      <ArrowClockwise className="h-3.5 w-3.5" /> Retry Extraction
                    </Button>
                  )}

                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-8 w-8 p-0 text-muted-foreground hover:text-destructive hover:bg-destructive/10"
                    title="Delete Candidate"
                    onClick={() => {
                      if (window.confirm(`Delete resume for ${cv.name}?`)) {
                        deleteCV.mutate(cv.id)
                      }
                    }}
                    disabled={deleteCV.isPending}
                  >
                    <Trash className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Screen Candidate Modal */}
      <Dialog open={screenCandidate !== null} onOpenChange={(o) => !o && setScreenCandidate(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle className="font-display text-lg flex items-center gap-2">
              <Sparkle className="h-5 w-5 text-primary" weight="fill" /> Screen Candidate against Role
            </DialogTitle>
            <DialogDescription>
              Select an active job role to run AI semantic matching and CV scoring for {screenCandidate?.name}.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label htmlFor="screen-job" className="text-xs font-semibold">Target Job Role</Label>
              <select
                id="screen-job"
                className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-1 focus:ring-primary"
                value={selectedJobId}
                onChange={(e) => setSelectedJobId(e.target.value)}
              >
                <option value="">-- Choose an open job --</option>
                {jobs?.map((j) => (
                  <option key={j.id} value={j.id}>
                    {j.title} ({j.min_experience}+ yrs exp)
                  </option>
                ))}
              </select>
            </div>
          </div>

          <DialogFooter>
            <Button variant="secondary" onClick={() => setScreenCandidate(null)}>
              Cancel
            </Button>
            <Button
              variant="gradient"
              onClick={() => screenMutation.mutate()}
              disabled={!selectedJobId || screenMutation.isPending}
            >
              {screenMutation.isPending ? "Screening…" : "Start Screening"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
