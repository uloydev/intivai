import { useState } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { api } from "@/lib/api"
import type { WebhookConfig, WebhookDelivery } from "@/types/api"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Badge } from "@/components/ui/badge"
import { Checkbox } from "@/components/ui/checkbox"
import { toast } from "sonner"
import { Plus, Trash, Globe, CheckCircle, XCircle, Clock } from "@phosphor-icons/react"

const AVAILABLE_EVENTS = [
  { value: "interview.completed", label: "Interview Completed" },
  { value: "candidate.screened", label: "Candidate Screened" },
  { value: "candidate.advanced", label: "Candidate Advanced" },
]

export function IntegrationsPage() {
  const queryClient = useQueryClient()
  const [showForm, setShowForm] = useState(false)
  const [url, setUrl] = useState("")
  const [secret, setSecret] = useState("")
  const [selectedEvents, setSelectedEvents] = useState<string[]>([])
  const [selectedWebhook, setSelectedWebhook] = useState<string | null>(null)

  const { data: webhooks = [] } = useQuery<WebhookConfig[]>({
    queryKey: ["webhooks"],
    queryFn: () => api.get("/webhooks"),
  })

  const { data: deliveries = [] } = useQuery<WebhookDelivery[]>({
    queryKey: ["webhook-deliveries", selectedWebhook],
    queryFn: () => api.get(`/webhooks/${selectedWebhook}/deliveries`),
    enabled: !!selectedWebhook,
  })

  const createMutation = useMutation({
    mutationFn: () => api.post("/webhooks", { url, events: selectedEvents, secret }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["webhooks"] })
      setShowForm(false)
      setUrl("")
      setSecret("")
      setSelectedEvents([])
      toast.success("Webhook created")
    },
    onError: (e: Error) => toast.error(e.message),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/webhooks/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["webhooks"] })
      toast.success("Webhook deleted")
    },
    onError: (e: Error) => toast.error(e.message),
  })

  const toggleEvent = (event: string) => {
    setSelectedEvents((prev) =>
      prev.includes(event) ? prev.filter((e) => e !== event) : [...prev, event]
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Integrations</h1>
          <p className="text-sm text-muted-foreground">Configure webhooks to push data to your ATS or custom endpoints.</p>
        </div>
        <Button onClick={() => setShowForm(!showForm)} variant="gradient">
          <Plus className="mr-2 h-4 w-4" /> Add Webhook
        </Button>
      </div>

      {showForm && (
        <Card>
          <CardHeader>
            <CardTitle>New Webhook</CardTitle>
            <CardDescription>POST event payloads to your endpoint with HMAC-SHA256 signature verification.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label>Endpoint URL</Label>
              <Input placeholder="https://your-ats.com/webhooks/intivai" value={url} onChange={(e) => setUrl(e.target.value)} />
            </div>
            <div className="space-y-2">
              <Label>Secret (for HMAC-SHA256 signature)</Label>
              <Input type="password" placeholder="whsec_..." value={secret} onChange={(e) => setSecret(e.target.value)} />
            </div>
            <div className="space-y-2">
              <Label>Events</Label>
              <div className="flex flex-wrap gap-3">
                {AVAILABLE_EVENTS.map((event) => (
                  <label key={event.value} className="flex items-center gap-2 text-sm cursor-pointer">
                    <Checkbox
                      checked={selectedEvents.includes(event.value)}
                      onCheckedChange={() => toggleEvent(event.value)}
                    />
                    {event.label}
                  </label>
                ))}
              </div>
            </div>
            <div className="flex gap-2">
              <Button onClick={() => createMutation.mutate()} disabled={!url || selectedEvents.length === 0}>
                Create Webhook
              </Button>
              <Button variant="outline" onClick={() => setShowForm(false)}>Cancel</Button>
            </div>
          </CardContent>
        </Card>
      )}

      <div className="space-y-3">
        {webhooks.length === 0 && !showForm && (
          <Card>
            <CardContent className="flex flex-col items-center justify-center py-12 text-center">
              <Globe className="h-10 w-10 text-muted-foreground mb-3" />
              <p className="text-sm font-medium">No webhooks configured</p>
              <p className="text-xs text-muted-foreground">Add a webhook to push interview results to your ATS or custom endpoint.</p>
            </CardContent>
          </Card>
        )}

        {webhooks.map((wh) => (
          <Card key={wh.id} className={selectedWebhook === wh.id ? "ring-2 ring-primary" : ""}>
            <CardContent className="flex items-center justify-between py-4">
              <div className="space-y-1 flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <p className="text-sm font-medium truncate">{wh.url}</p>
                  <Badge variant={wh.active ? "success" : "secondary"} size="sm">
                    {wh.active ? "Active" : "Inactive"}
                  </Badge>
                </div>
                <div className="flex flex-wrap gap-1">
                  {wh.events.map((e) => (
                    <Badge key={e} variant="info" size="sm">{e}</Badge>
                  ))}
                </div>
              </div>
              <div className="flex items-center gap-2 ml-4">
                <Button variant="outline" size="sm" onClick={() => setSelectedWebhook(selectedWebhook === wh.id ? null : wh.id)}>
                  Logs
                </Button>
                <Button variant="destructive" size="icon-sm" onClick={() => {
                  if (window.confirm("Delete this webhook and its delivery history?")) deleteMutation.mutate(wh.id)
                }}>
                  <Trash className="h-4 w-4" />
                </Button>
              </div>
            </CardContent>
            {selectedWebhook === wh.id && deliveries.length > 0 && (
              <div className="border-t px-4 py-3 space-y-2">
                <p className="text-xs font-medium text-muted-foreground">Recent Deliveries</p>
                {deliveries.map((d) => (
                  <div key={d.id} className="flex items-center gap-3 text-xs">
                    {d.final_status === "delivered" ? (
                      <CheckCircle className="h-3.5 w-3.5 text-success shrink-0" />
                    ) : d.final_status === "failed" ? (
                      <XCircle className="h-3.5 w-3.5 text-destructive shrink-0" />
                    ) : (
                      <Clock className="h-3.5 w-3.5 text-warning shrink-0" />
                    )}
                    <span className="font-mono">{d.event}</span>
                    <span className="text-muted-foreground">HTTP {d.status_code}</span>
                    <span className="text-muted-foreground">{d.attempts} attempt(s)</span>
                    <span className="text-muted-foreground ml-auto">{new Date(d.created_at).toLocaleString()}</span>
                  </div>
                ))}
              </div>
            )}
            {selectedWebhook === wh.id && deliveries.length === 0 && (
              <div className="border-t px-4 py-3">
                <p className="text-xs text-muted-foreground">No deliveries yet.</p>
              </div>
            )}
          </Card>
        ))}
      </div>
    </div>
  )
}
