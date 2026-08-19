import { createContext, useContext, useEffect, useState } from "react"

type Theme = "light" | "dark"

const THEME_KEY = "intivai_theme"

function initialTheme(): Theme {
  const stored = localStorage.getItem(THEME_KEY)
  if (stored === "dark" || stored === "light") return stored
  // First visit: respect the OS preference; default to dark only when the OS
  // prefers dark, otherwise light. The manual toggle still overrides this.
  if (typeof window !== "undefined" && window.matchMedia("(prefers-color-scheme: dark)").matches) {
    return "dark"
  }
  return "light"
}

function applyThemeClass(theme: Theme) {
  document.documentElement.classList.toggle("dark", theme === "dark")
}

const ThemeContext = createContext<{ theme: Theme; toggle: () => void }>({
  theme: "light",
  toggle: () => undefined,
})

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setTheme] = useState<Theme>(() => {
    const initial = initialTheme()
    // Apply synchronously at mount so the class never waits for the effect
    // (the inline <head> script already handles the pre-paint case).
    applyThemeClass(initial)
    return initial
  })

  useEffect(() => {
    applyThemeClass(theme)
    localStorage.setItem(THEME_KEY, theme)
  }, [theme])

  return (
    <ThemeContext.Provider value={{ theme, toggle: () => setTheme((t) => (t === "dark" ? "light" : "dark")) }}>
      {children}
    </ThemeContext.Provider>
  )
}

export function useTheme() {
  return useContext(ThemeContext)
}
