export function chatInviteUrl(interviewId: string, token?: string | null): string {
  const base = `${window.location.origin}/invite/${interviewId}`
  return token ? `${base}?t=${encodeURIComponent(token)}` : base
}
