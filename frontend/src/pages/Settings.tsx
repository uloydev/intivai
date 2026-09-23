import { useState } from "react"
import { useMutation, useQuery } from "@tanstack/react-query"
import { ShieldCheck } from "@phosphor-icons/react"
import { api } from "@/lib/api"
import { getSession } from "@/lib/auth"
import { parseQaLimitInput } from "@/lib/qa-limit"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { toast } from "sonner"

const QA_LIMIT_PATH = (orgId: string) => `/orgs/${orgId}/settings/candidate-qa-limit`

interface QALimitSettings {
  candidate_qa_limit: number
}

export function SettingsPage() {
  const session = getSession()
  const isAdmin = session?.role === "admin"

  // J13: the backend exposes no GET for the current candidate Q&A limit —
  // only the admin PUT. We query it anyway so the input is pre-filled when a
  // GET is later published; until then the 404/error shows an honest notice
  // and the field stays blank (placeholder), never a fabricated default.
  const { isLoading, isError } = useQuery<QALimitSettings>({
    queryKey: ["org-settings", session?.orgId, "candidate-qa-limit"],
    queryFn: () => api.get(QA_LIMIT_PATH(session?.orgId ?? "")),
    enabled: isAdmin,
  })

  const [input, setInput] = useState("")
  const parsed = parseQaLimitInput(input)

  const updateMutation = useMutation({
    mutationFn: (value: number) => api.put<QALimitSettings>(QA_LIMIT_PATH(session?.orgId ?? ""), {
      candidate_qa_limit: value,
    }),
    onSuccess: (data) => {
      setInput(String(data.candidate_qa_limit))
      toast.success("Candidate Q&A limit saved.")
    },
    onError: (e: Error) => toast.error(e.message),
  })

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Settings</h1>
          <p className="text-sm text-muted-foreground">
            Configure workspace-level options.
          </p>
        </div>
      </div>

      {!isAdmin ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12 text-center">
            <ShieldCheck className="h-10 w-10 text-muted-foreground mb-3" />
            <p className="text-sm font-medium">Workspace settings are visible to organization admins only.</p>
            <p className="text-xs text-muted-foreground">Ask an admin for help with these options.</p>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardHeader>
            <CardTitle>Candidate Q&A limit</CardTitle>
            <CardDescription>
              The maximum number of questions a candidate can ask during an AI interview (0–50).
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="candidate-qa-limit">Candidate Q&A limit</Label>
              <Input
                id="candidate-qa-limit"
                type="number"
                min={0}
                max={50}
                step={1}
                value={input}
                onChange={(e) => setInput(e.target.value)}
                placeholder="10"
                disabled={isLoading || updateMutation.isPending}
              />
              <p className="text-xs text-muted-foreground">
                {isLoading
                  ? "Loading current limit…"
                  : isError
                    ? "The current limit is not available — set a new value below."
                    : "Integers from 0 to 50. Leaving a blank field uses the default."}
              </p>
              {input.length > 0 && parsed === null && (
                <p className="text-xs text-destructive">
                  Enter a whole number from 0–50.
                </p>
              )}
            </div>
            <Button
              onClick={() => parsed !== null && updateMutation.mutate(parsed)}
              disabled={parsed === null || isLoading || updateMutation.isPending}
            >
              {updateMutation.isPending ? "Saving…" : "Save limit"}
            </Button>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
