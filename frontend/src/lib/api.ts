const API_BASE = import.meta.env.VITE_API_BASE ?? "/api/v1"

export class ApiError extends Error {
  code: string
  status: number

  constructor(status: number, code: string, message: string) {
    super(message)
    this.code = code
    this.status = status
  }
}

type ApiResponse<T> = { data: T }

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers)
  const token = localStorage.getItem("intivai_token")
  const candidateToken = localStorage.getItem("intivai_candidate_token")

  if (path.startsWith("/candidate/portal") && candidateToken) {
    headers.set("Authorization", `Bearer ${candidateToken}`)
  } else if (token) {
    headers.set("Authorization", `Bearer ${token}`)
  }

  if (init?.body && !(init.body instanceof FormData)) {
    headers.set("Content-Type", "application/json")
  }

  // Dead backend must not spin spinners forever.
  const signal = init?.signal ?? AbortSignal.timeout(15_000)
  const res = await fetch(`${API_BASE}${path}`, { ...init, headers, signal })
  const body = await res.json().catch(() => null)

  if (res.status === 401) {
    if (path.startsWith("/candidate/portal")) {
      localStorage.removeItem("intivai_candidate_token")
    } else if (
      !path.startsWith("/auth/") &&
      !path.startsWith("/public/") &&
      !path.startsWith("/candidate/")
    ) {
      // Expired/revoked recruiter session: force logout once, redirect to login.
      // Candidate endpoints are excluded — a candidate token failure must not
      // destroy a concurrently logged-in recruiter session.
      localStorage.removeItem("intivai_token")
      if (window.location.pathname !== "/login") {
        window.location.assign("/login")
      }
    }
  }

  if (!res.ok) {
    const err = body as { code?: string; error?: string } | null
    throw new ApiError(res.status, err?.code ?? "UNKNOWN", err?.error ?? `Request failed (${res.status})`)
  }
  return (body as ApiResponse<T>).data
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: "POST", body: body === undefined ? undefined : JSON.stringify(body) }),
  postForm: <T>(path: string, form: FormData) => request<T>(path, { method: "POST", body: form }),
  patch: <T>(path: string, body: unknown) =>
    request<T>(path, { method: "PATCH", body: JSON.stringify(body) }),
  put: <T>(path: string, body: unknown) =>
    request<T>(path, { method: "PUT", body: JSON.stringify(body) }),
  delete: <T = void>(path: string) => request<T>(path, { method: "DELETE" }),
  getBlob: async (path: string) => {
    const headers = new Headers()
    const token = localStorage.getItem("intivai_token")
    if (token) headers.set("Authorization", `Bearer ${token}`)
    const res = await fetch(`${API_BASE}${path}`, { headers })
    if (!res.ok) throw new Error(`Download failed (${res.status})`)
    return await res.blob()
  },
}
