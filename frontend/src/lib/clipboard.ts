import { toast } from "sonner"

export async function copyText(text: string, label?: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text)
    toast.success(`${label ?? "Text"} copied to clipboard`)
  } catch (err) {
    console.error("Clipboard write failed", err)
    toast.error("Failed to copy — please copy manually")
  }
}
