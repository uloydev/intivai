import { Navigate, Outlet, useLocation } from "react-router-dom"
import { getSession } from "./auth"

// Route guard: no valid session → /login. A 401 during any call forces
// logout (handled in api.ts consumers via onUnauthorized).
export function RequireAuth() {
  const session = getSession()
  const location = useLocation()
  if (!session) {
    return <Navigate to="/login" state={{ from: location.pathname }} replace />
  }
  return <Outlet context={{ session }} />
}
