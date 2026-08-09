import { api } from "./api"
import type { LoginResult } from "@/types/api"

const TOKEN_KEY = "intivai_token"

export interface Session {
  token: string
  userId: string
  orgId: string
  role: string
}

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function getSession(): Session | null {
  const token = getToken()
  if (!token) return null
  const payload = token.split(".")[1]
  if (!payload) return null
  try {
    const claims = JSON.parse(atob(payload.replace(/-/g, "+").replace(/_/g, "/")))
    return {
      token,
      userId: claims.sub,
      orgId: claims.org_id,
      role: claims.role,
    }
  } catch {
    return null
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
