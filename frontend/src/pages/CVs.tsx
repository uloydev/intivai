import { useRef, useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { CloudArrowUp } from "@phosphor-icons/react"
import { api } from "@/lib/api"
import type { CVListItem } from "@/types/api"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Skeleton } from "@/components/ui/skeleton"
import { toast } from "sonner"

const POLL_STATUSES = new Set(["parsing", "extracting", "extracting_llm"])

function statusBadge(status: string) {
  if (status === "parsed" || status === "extracted") {
    return (
      <Badge className="bg-accent text-accent-foreground">{status}</Badge>
    )
  }
  if (status === "failed_ocr" || status === "failed_extract" || status === "failed_parse") {
    return <Badge variant="destructive">{status}</Badge>
  }
  return (
    <Badge variant="secondary" className="bg-amber-100 text-amber-800 dark:bg-amber-950 dark:text-amber-200">
      {status}
    </Badge>
  )
}

export function CVsPage() {
  const qc = useQueryClient()
  const { data: cvs, isLoading, error } = useQuery({
    queryKey: ["cvs"],
    queryFn: () => api.get<CVListItem[]>("/cvs"),
    // Poll while any CV is mid-pipeline (parse/extract workers).
    refetchInterval: (query) =>
      query.state.data?.some((c) => POLL_STATUSES.has(c.status)) ? 2000 : false,
  })

  const [name, setName] = useState("")
  const [email, setEmail] = useState("")
  const fileRef = useRef<HTMLInputElement>(null)
  const [uploading, setUploading] = useState(false)

  const upload = useMutation({
    mutationFn: async () => {
      const file = fileRef.current?.files?.[0]
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
      if (fileRef.current) fileRef.current.value = ""
      toast.success("CV uploaded — parsing started")
    },
    onError: (e) => {
      setUploading(false)
      toast.error(e instanceof Error ? e.message : "Upload failed")
    },
  })

  const reExtract = useMutation({
    mutationFn: (id: string) => api.post<CVListItem>(`/cvs/${id}/extract`, {}),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["cvs"] }),
    onError: (e) => toast.error(e instanceof Error ? e.message : "Retry failed"),
  })

  return (
    <div className="space-y-4">
      <h1 className="font-display text-2xl">CVs</h1>

      <Card className="p-4">
        <div className="grid gap-3 md:grid-cols-[1fr_1fr_1.5fr_auto]">
          <div className="space-y-1">
            <Label htmlFor="cv-name">Candidate name</Label>
            <Input id="cv-name" value={name} onChange={(e) => setName(e.target.value)} />
          </div>
          <div className="space-y-1">
            <Label htmlFor="cv-email">Email</Label>
            <Input id="cv-email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
          </div>
          <div className="space-y-1">
            <Label htmlFor="cv-file">PDF</Label>
            <Input id="cv-file" ref={fileRef} type="file" accept="application/pdf" />
          </div>
          <div className="flex items-end">
            <Button
              onClick={() => {
                setUploading(true)
                upload.mutate()
              }}
              disabled={uploading || !name.trim() || !email.trim()}
            >
              <CloudArrowUp className="mr-2 h-4 w-4" />
              {uploading ? "Uploading…" : "Upload"}
            </Button>
          </div>
        </div>
      </Card>

      {isLoading ? (
        <div className="space-y-2">
          <Skeleton className="h-14 w-full" />
          <Skeleton className="h-14 w-full" />
        </div>
      ) : error ? (
        <p className="text-destructive">{error instanceof Error ? error.message : "Failed to load CVs"}</p>
      ) : !cvs?.length ? (
        <p className="py-16 text-center text-sm text-muted-foreground">No CVs yet — upload the first one above</p>
      ) : (
        <div className="space-y-2">
          {cvs.map((cv) => (
            <div key={cv.id} className="flex items-center justify-between gap-3 rounded-md border border-border bg-card p-3 shadow-sm">
              <div className="min-w-0">
                <p className="truncate font-medium">{cv.name}</p>
                <p className="truncate text-xs text-muted-foreground">{cv.email}</p>
                {cv.error_message && <p className="mt-1 truncate text-xs text-destructive">{cv.error_message}</p>}
              </div>
              <div className="flex shrink-0 items-center gap-2">
                {statusBadge(cv.status)}
                {(cv.status === "failed_extract" || cv.status === "failed_ocr") && (
                  <Button variant="outline" size="sm" onClick={() => reExtract.mutate(cv.id)} disabled={reExtract.isPending}>
                    Retry
                  </Button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
