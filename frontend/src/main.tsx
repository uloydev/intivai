import { StrictMode } from "react"
import { createRoot } from "react-dom/client"
import "./index.css"
import * as Sentry from "@sentry/react";
import App from "./App"
import { ThemeProvider } from "@/lib/theme"
import { buildSentryConfig } from "@/lib/sentry"

const sentryConfig = buildSentryConfig(import.meta.env.VITE_SENTRY_DSN)
if (sentryConfig) {
  Sentry.init(sentryConfig);
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ThemeProvider>
      <App />
    </ThemeProvider>
  </StrictMode>,
)
