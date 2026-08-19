import { api } from "./api"
import type { LoginResult } from "@/types/api"

const TOKEN_KEY = "intivai_token"

export interface Session {
  token: string
  userId: string
  orgId: string
  role: string
}

// base64url → binary string. JWTs strip '=' padding; atob REQUIRES it —
// decoding without padding intermittently throws (payload length % 4 != 0),
// which would log out users with perfectly valid tokens.
export function decodePayload(payload: string): Record<string, unknown> | null {
  const b64 = payload.replace(/-/g, "+").replace(/_/g, "/")
  const padded = b64 + "=".repeat((4 - (b64.length % 4)) % 4)
  try {
    return JSON.parse(atob(padded)) as Record<string, unknown>
  } catch {
    return null
  }
}

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function getSession(): Session | null {
  const token = getToken()
  if (!token) return null
  const payload = token.split(".")[1]
  if (!payload) return null
  const claims = decodePayload(payload)
  if (!claims) return null
  // Expired tokens are NOT sessions — RequireAuth must send the user to
  // login instead of letting dead pages render.
  if (typeof claims.exp === "number" && claims.exp * 1000 < Date.now()) {
    localStorage.removeItem(TOKEN_KEY)
    return null
  }
  return {
    token,
    userId: String(claims.sub ?? ""),
    orgId: String(claims.org_id ?? ""),
    role: String(claims.role ?? ""),
  }
}

export async function login(orgSlug: string, email: string, password: string): Promise<Session> {
  const result = await api.post<LoginResult>("/auth/login", { org_slug: orgSlug, email, password })
  localStorage.setItem(TOKEN_KEY, result.token)
  return {
    token: result.token,
    userId: result.user.user_id,
    orgId: result.user.org_id,
    role: result.user.role,
  }
}

export function logout(): void {
  localStorage.removeItem(TOKEN_KEY)
}
