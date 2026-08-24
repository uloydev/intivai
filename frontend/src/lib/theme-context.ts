import { createContext, useContext } from "react"

export type Theme = "light" | "dark"

export const ThemeContext = createContext<{ theme: Theme; toggle: () => void }>({
  theme: "light",
  toggle: () => undefined,
})

export function useTheme() {
  return useContext(ThemeContext)
}
