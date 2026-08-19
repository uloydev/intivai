import { useState, useRef, useEffect } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  Brain,
  Sparkle,
  FloppyDisk,
  UploadSimple,
  FileText,
  CheckCircle,
  ShieldCheck,
  Lightning,
  TreeStructure,
  Trash,
} from "@phosphor-icons/react"
import { api } from "@/lib/api"
import { getSession } from "@/lib/auth"
import type { CompanyContextItem, TenantPromptResult } from "@/types/api"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Textarea } from "@/components/ui/textarea"
import { Label } from "@/components/ui/label"
import { Skeleton } from "@/components/ui/skeleton"
import { toast } from "sonner"

const PROMPT_PRESETS = [
  {
    name: "Engineering Excellence",
    prompt:
      "You are Intivai, an expert AI Technical Interviewer evaluating senior software engineers. You probe deeply for system design trade-offs, concurrency bugs, scalability patterns, and algorithmic clarity while maintaining a professional and encouraging demeanor.",
  },
  {
    name: "Fast-Paced Startup Architect",
    prompt:
      "You are Intivai, evaluating full-stack and backend talent for high-growth tech ventures. Emphasize pragmatic decision-making, speed vs reliability trade-offs, microservice resiliency, and proactive problem solving under ambiguity.",
  },
  {
    name: "Empathetic & Rigorous Behavioral",
    prompt:
      "You are Intivai, evaluating cross-functional communication, conflict resolution, mentorship, and ownership. Use the STAR methodology (Situation, Task, Action, Result) to guide candidates into structured, quantifiable explanations.",
  },
]

export function CompanyContextPage() {
  const qc = useQueryClient()
  const session = getSession()
  const orgId = session?.orgId || ""

  const [activeTab, setActiveTab] = useState<"prompt" | "knowledge">("prompt")
  const [promptText, setPromptText] = useState("")
  const [contextText, setContextText] = useState("")
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const promptTouchedRef = useRef(false)
  const dirtyRef = useRef(false)

  // Unsaved-changes guard: warn before unload and before discarding edits via
  // a tab switch. Reset the flag only after a successful save/upload.
  useEffect(() => {
    const handler = (e: BeforeUnloadEvent) => {
      if (!dirtyRef.current) return
      e.preventDefault()
      e.returnValue = ""
    }
    window.addEventListener("beforeunload", handler)
    return () => window.removeEventListener("beforeunload", handler)
  }, [])

  const switchTab = (tab: "prompt" | "knowledge") => {
    if (tab === activeTab) return
    if (dirtyRef.current && !window.confirm("You have unsaved changes. Discard them and switch tabs?")) return
    setActiveTab(tab)
  }

  // Fetch Tenant Prompt — the textarea is DERIVED from query data; syncing
  // state inside queryFn would overwrite a prompt the user just edited on
  // every refetch (e.g. window-focus).
  const { data: promptData, isLoading: loadingPrompt } = useQuery({
    queryKey: ["tenant-prompt", orgId],
    queryFn: async () => {
      if (!orgId) return null
      const res = await api.get<TenantPromptResult>(`/orgs/${orgId}/prompt`)
      return res
    },
    enabled: Boolean(orgId),
  })

  useEffect(() => {
    if (promptData?.system_prompt && promptText === "" && !promptTouchedRef.current) {
      setPromptText(promptData.system_prompt)
    }
  }, [promptData, promptText])

  // Fetch Vectorized Contexts
  const { data: contexts, isLoading: loadingContexts } = useQuery({
    queryKey: ["company-contexts", orgId],
    queryFn: () => (orgId ? api.get<CompanyContextItem[]>(`/orgs/${orgId}/contexts`) : []),
    enabled: Boolean(orgId),
  })

  // Set Prompt Mutation
  const savePrompt = useMutation({
    mutationFn: () =>
      api.put<TenantPromptResult>(`/orgs/${orgId}/prompt`, {
        system_prompt: promptText,
      }),
    onSuccess: (res) => {
      qc.invalidateQueries({ queryKey: ["tenant-prompt", orgId] })
      dirtyRef.current = false
      promptTouchedRef.current = false
      toast.success(`AI system prompt updated (Version ${res.version})`)
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed to update prompt"),
  })

  // Upload Context Mutation
  const uploadContext = useMutation({
    mutationFn: async () => {
      if (selectedFile) {
        const form = new FormData()
        form.append("file", selectedFile)
        return api.postForm<CompanyContextItem>(`/orgs/${orgId}/contexts`, form)
      }
      if (!contextText.trim()) throw new Error("Enter context text or select a file")
      return api.post<CompanyContextItem>(`/orgs/${orgId}/contexts`, {
        content: contextText,
      })
    },
    onSuccess: (res) => {
      qc.invalidateQueries({ queryKey: ["company-contexts", orgId] })
      setContextText("")
      setSelectedFile(null)
      dirtyRef.current = false
      toast.success(`Company context vectorized and synced (Version ${res.version})`)
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Upload failed"),
  })

  // Delete one vectorized context artifact (tenant-pinned, admin/recruiter).
  const deleteContext = useMutation({
    mutationFn: (id: string) => api.delete(`/orgs/${orgId}/contexts/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["company-contexts", orgId] })
      toast.success("Company context deleted")
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Delete failed"),
  })

  return (
    <div className="space-y-6 animate-in fade-in duration-500">
      {/* Header */}
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between border-b border-border/60 pb-4">
        <div>
          <div className="flex items-center gap-2">
            <h1 className="font-display text-2xl font-bold tracking-tight text-foreground">
              Company Intelligence & AI Interview Rails
            </h1>
            <Badge variant="outline" className="gap-1 border-primary/30 bg-primary/5 text-primary text-xs py-0.5">
              <Brain className="h-3.5 w-3.5" weight="fill" /> Mnemosyne Memory Bank
            </Badge>
          </div>
          <p className="text-xs text-muted-foreground mt-1">
            Provide organization memory, engineering blueprints, and custom AI interviewer persona rails.
          </p>
        </div>
      </div>

      {/* Tabs */}
      <div role="tablist" aria-label="Company intelligence sections" className="flex border-b border-border">
        <button
          role="tab"
          aria-selected={activeTab === "prompt"}
          tabIndex={activeTab === "prompt" ? 0 : -1}
          onClick={() => switchTab("prompt")}
          className={`flex items-center gap-2 border-b-2 py-2.5 px-4 text-xs font-semibold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/60 ${
            activeTab === "prompt"
              ? "border-primary text-primary"
              : "border-transparent text-muted-foreground hover:text-foreground"
          }`}
        >
          <Sparkle className="h-4 w-4" />
          <span>AI Persona & Prompt Rails</span>
        </button>
        <button
          role="tab"
          aria-selected={activeTab === "knowledge"}
          tabIndex={activeTab === "knowledge" ? 0 : -1}
          onClick={() => switchTab("knowledge")}
          className={`flex items-center gap-2 border-b-2 py-2.5 px-4 text-xs font-semibold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/60 ${
            activeTab === "knowledge"
              ? "border-primary text-primary"
              : "border-transparent text-muted-foreground hover:text-foreground"
          }`}
        >
          <TreeStructure className="h-4 w-4" />
          <span>Company Knowledge & Vector Contexts ({contexts?.length ?? 0})</span>
        </button>
      </div>

      {/* TAB 1: AI Persona & Prompt Rails */}
      {activeTab === "prompt" && (
        <div className="grid gap-6 lg:grid-cols-3">
          {/* Main Prompt Editor */}
          <Card className="glass border-border/60 lg:col-span-2 shadow-sm">
            <CardHeader className="flex flex-row items-center justify-between pb-3">
              <div>
                <CardTitle className="font-display text-base font-bold flex items-center gap-2">
                  <Sparkle className="h-4 w-4 text-primary" weight="fill" />
                  Tenant System Prompt
                </CardTitle>
                <CardDescription className="text-xs mt-0.5">
                  The primary persona instructions injected into all AI interview sessions for your organization.
                </CardDescription>
              </div>
              {promptData && (
                <Badge variant="secondary" className="text-xs">
                  Version {promptData.version || 1}
                </Badge>
              )}
            </CardHeader>

            <CardContent className="space-y-4">
              {loadingPrompt ? (
                <Skeleton className="h-48 w-full" />
              ) : (
                <div className="space-y-2">
                  <Textarea
                    value={promptText || promptData?.system_prompt || ""}
                    onChange={(e) => {
                      promptTouchedRef.current = true
                      dirtyRef.current = true
                      setPromptText(e.target.value)
                    }}
                    placeholder="Enter custom AI interviewer instructions, evaluation standards, or persona rails..."
                    rows={9}
                    className="font-mono text-xs leading-relaxed bg-background/60 focus-visible:ring-primary p-3"
                  />
                  <div className="flex items-center justify-between text-xs text-muted-foreground pt-1">
                    <span className="flex items-center gap-1 text-emerald-600 dark:text-emerald-400">
                      <ShieldCheck className="h-3.5 w-3.5" /> Injection Rails Protected
                    </span>
                    <span>{(promptText || promptData?.system_prompt || "").length} characters</span>
                  </div>
                </div>
              )}

              <Button
                onClick={() => savePrompt.mutate()}
                disabled={savePrompt.isPending || !promptText.trim()}
                variant="gradient"
                className="w-full gap-2 text-xs font-bold shadow-md shadow-primary/20"
              >
                <FloppyDisk className="h-4 w-4" weight="bold" />
                {savePrompt.isPending ? "Saving Rails..." : "Save AI System Prompt"}
              </Button>
            </CardContent>
          </Card>

          {/* Persona Presets */}
          <div className="space-y-4">
            <Card className="glass border-border/60 shadow-sm">
              <CardHeader className="pb-3">
                <CardTitle className="font-display text-sm font-bold flex items-center gap-1.5">
                  <Lightning className="h-4 w-4 text-amber-600 dark:text-amber-400" weight="fill" />
                  Interviewer Persona Presets
                </CardTitle>
                <CardDescription className="text-xs">
                  Click a preset to quickly apply standard interview configurations.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-2.5">
                {PROMPT_PRESETS.map((preset, idx) => (
                  <button
                    type="button"
                    key={idx}
                    onClick={() => {
                      if (
                        promptTouchedRef.current &&
                        !window.confirm(
                          `Replace the current prompt with the "${preset.name}" template? Unsaved edits will be lost.`
                        )
                      ) {
                        return
                      }
                      setPromptText(preset.prompt)
                      dirtyRef.current = true
                      toast.info(`Loaded "${preset.name}" preset template`)
                    }}
                    className="group w-full text-left cursor-pointer rounded-xl border border-border/60 bg-background/50 p-3 transition-all hover:border-primary/40 hover:bg-card hover:shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/60"
                  >
                    <p className="font-display text-xs font-bold text-foreground group-hover:text-primary transition-colors">
                      {preset.name}
                    </p>
                    <p className="text-xs text-muted-foreground line-clamp-2 mt-1 leading-relaxed">
                      {preset.prompt}
                    </p>
                  </button>
                ))}
              </CardContent>
            </Card>
          </div>
        </div>
      )}

      {/* TAB 2: Company Knowledge & Vector Contexts */}
      {activeTab === "knowledge" && (
        <div className="grid gap-6 lg:grid-cols-3">
          {/* Upload New Context */}
          <Card className="glass border-border/60 shadow-sm">
            <CardHeader>
              <CardTitle className="font-display text-base font-bold flex items-center gap-2">
                <UploadSimple className="h-4 w-4 text-primary" weight="bold" />
                Ingest Company Context
              </CardTitle>
              <CardDescription className="text-xs">
                Upload architecture guidelines, core values, or handbooks to vectorize into Mnemosyne.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label className="text-xs font-semibold">Paste Text or Documentation</Label>
                <Textarea
                  value={contextText}
                  onChange={(e) => {
                    dirtyRef.current = true
                    setContextText(e.target.value)
                  }}
                  placeholder="e.g. Our engineering stack relies on Go microservices with PostgreSQL RLS. We value pragmatic simplicity, clean concurrency, and high test coverage..."
                  rows={6}
                  className="text-xs bg-background/60 leading-relaxed p-3"
                />
              </div>

              <div className="space-y-2">
                <Label className="text-xs font-semibold">Or Attach Markdown / Text File</Label>
                <input
                  type="file"
                  accept=".txt,.md,.json,.pdf"
                  onChange={(e) => setSelectedFile(e.target.files?.[0] || null)}
                  className="w-full text-xs file:mr-3 file:py-1.5 file:px-3 file:rounded-lg file:border-0 file:text-xs file:font-semibold file:bg-primary/10 file:text-primary hover:file:bg-primary/20 cursor-pointer"
                />
              </div>

              <Button
                onClick={() => uploadContext.mutate()}
                disabled={uploadContext.isPending || (!contextText.trim() && !selectedFile)}
                variant="gradient"
                className="w-full gap-2 text-xs font-bold"
              >
                <Brain className="h-4 w-4" weight="fill" />
                {uploadContext.isPending ? "Vectorizing & Syncing..." : "Ingest & Vectorize Context"}
              </Button>
            </CardContent>
          </Card>

          {/* Active Context Versions */}
          <Card className="glass border-border/60 lg:col-span-2 shadow-sm">
            <CardHeader>
              <CardTitle className="font-display text-base font-bold flex items-center gap-2">
                <TreeStructure className="h-4 w-4 text-primary" weight="bold" />
                Vectorized Memory Bank Artifacts
              </CardTitle>
              <CardDescription className="text-xs">
                Active tenant context versions pinned and referenced by the AI interviewer during probing.
              </CardDescription>
            </CardHeader>
            <CardContent>
              {loadingContexts ? (
                <Skeleton className="h-36 w-full" />
              ) : !contexts?.length ? (
                <div className="rounded-xl border border-dashed border-border p-8 text-center space-y-2">
                  <Brain className="mx-auto h-8 w-8 text-muted-foreground/40" />
                  <p className="text-xs font-medium text-foreground">No custom company context ingested yet</p>
                  <p className="text-xs text-muted-foreground max-w-sm mx-auto">
                    The interviewer uses standard domain knowledge. Ingest your company engineering context on the left to customize probing.
                  </p>
                </div>
              ) : (
                <div className="space-y-2.5">
                  {contexts.map((ctx) => (
                    <div
                      key={ctx.id}
                      className="flex items-center justify-between rounded-xl border border-border/60 bg-background/50 p-3.5 transition-all hover:bg-muted/40"
                    >
                      <div className="flex items-center gap-3">
                        <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
                          <FileText className="h-4 w-4" weight="bold" />
                        </div>
                        <div>
                          <div className="flex items-center gap-2">
                            <span className="font-display text-xs font-bold text-foreground">
                              Company Context Version {ctx.version}
                            </span>
                            <Badge variant="outline" className="text-xs uppercase">
                              {ctx.type}
                            </Badge>
                          </div>
                          <p className="font-mono text-xs text-muted-foreground truncate max-w-md mt-0.5">
                            Hash: {ctx.content_hash}
                          </p>
                        </div>
                      </div>

                      <div className="flex items-center gap-2">
                        <div className="flex items-center gap-2 text-xs text-muted-foreground">
                          <CheckCircle className="h-3.5 w-3.5 text-emerald-600 dark:text-emerald-400" weight="fill" />
                          <span>Indexed</span>
                        </div>
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          className="text-muted-foreground hover:text-destructive hover:bg-destructive/10"
                          aria-label={`Delete company context version ${ctx.version}`}
                          title="Delete context"
                          onClick={() => {
                            if (
                              window.confirm(
                                `Delete company context version ${ctx.version}? This permanently removes it from the memory bank.`
                              )
                            ) {
                              deleteContext.mutate(ctx.id)
                            }
                          }}
                          disabled={deleteContext.isPending}
                        >
                          <Trash className="h-4 w-4" />
                        </Button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  )
}
