// G2: the ws_ticket minted for /chat?t= expires after ~10 minutes while
// interviews run up to 30 minutes. The long-lived invitation_token is kept in
// sessionStorage (per interview) so a mid-session reconnect can re-mint a
// fresh ws_ticket exactly once. sessionStorage (not localStorage): the token
// must die with the tab, never persist across browser sessions.
const key = (interviewId: string) => `intivai_invitation_${interviewId}`

export function storeInvitationToken(interviewId: string, invitationToken: string): void {
  try {
    sessionStorage.setItem(key(interviewId), JSON.stringify({ invitation_token: invitationToken }))
  } catch {
    // Storage full/blocked — reconnect re-mint degrades to best effort.
  }
}

export function getStoredInvitationToken(interviewId: string): string | null {
  try {
    const raw = sessionStorage.getItem(key(interviewId))
    if (!raw) return null
    const parsed: unknown = JSON.parse(raw)
    if (
      parsed !== null &&
      typeof parsed === "object" &&
      "invitation_token" in parsed &&
      typeof (parsed as { invitation_token: unknown }).invitation_token === "string"
    ) {
      return (parsed as { invitation_token: string }).invitation_token
    }
    return null
  } catch {
    return null
  }
}
