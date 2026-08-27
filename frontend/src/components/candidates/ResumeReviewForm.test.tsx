import { render, screen, fireEvent, cleanup } from "@testing-library/react"
import { describe, expect, it, afterEach, vi } from "vitest"
import type { CVDraftProfile, ResumeData } from "@/types/api"
import { ResumeReviewForm } from "./ResumeReviewForm"

const INITIAL: ResumeData & { name: string; email: string } = {
  name: "Alex Dev",
  email: "alex@dev.io",
  skills: ["Go", "Postgres"],
  experience_years: 5,
  education: "BS Computer Science",
  certifications: ["AWS"],
  summary: "Backend engineer with 5 years of experience.",
}

function renderForm({
  initial = INITIAL,
  onSubmit,
  submitLabel = "Confirm & Continue to Screening",
  disabled = false,
}: {
  initial?: ResumeData & { name: string; email: string }
  onSubmit: (draft: CVDraftProfile) => void
  submitLabel?: string
  disabled?: boolean
}) {
  return render(
    <ResumeReviewForm initial={initial} onSubmit={onSubmit} submitLabel={submitLabel} disabled={disabled} />,
  )
}

describe("ResumeReviewForm", () => {
  afterEach(() => {
    cleanup()
  })

  it("renders all 7 inputs prefilled with plain-language labels", () => {
    renderForm({ onSubmit: () => {} })
    expect((screen.getByLabelText(/name/i) as HTMLInputElement).value).toBe("Alex Dev")
    expect((screen.getByLabelText(/email/i) as HTMLInputElement).value).toBe("alex@dev.io")
    expect(screen.getByText("Go")).toBeDefined()
    expect(screen.getByText("Postgres")).toBeDefined()
    expect((screen.getByLabelText(/experience years/i) as HTMLInputElement).value).toBe("5")
    expect((screen.getByLabelText(/education/i) as HTMLInputElement).value).toBe("BS Computer Science")
    expect(screen.getByText("AWS")).toBeDefined()
    expect((screen.getByLabelText(/professional summary/i) as HTMLTextAreaElement).value).toContain("Backend engineer")
  })

  it("submits the full 7-field payload", () => {
    const onSubmit = vi.fn()
    renderForm({ onSubmit })
    fireEvent.click(screen.getByRole("button", { name: /confirm & continue/i }))
    expect(onSubmit).toHaveBeenCalledWith({
      name: "Alex Dev",
      email: "alex@dev.io",
      skills: ["Go", "Postgres"],
      experience_years: 5,
      education: "BS Computer Science",
      certifications: ["AWS"],
      summary: "Backend engineer with 5 years of experience.",
    })
  })

  it("disables submit with a red hint when experience is out of range", () => {
    const onSubmit = vi.fn()
    const { container } = renderForm({ onSubmit })
    const exp = screen.getByLabelText(/experience years/i) as HTMLInputElement
    fireEvent.change(exp, { target: { value: "-1" } })
    const button = screen.getByRole("button", { name: /confirm & continue/i }) as HTMLButtonElement
    expect(button.disabled).toBe(true)
    expect(container.textContent).toContain("between 0 and 50")
    fireEvent.change(exp, { target: { value: "51" } })
    expect((screen.getByRole("button", { name: /confirm & continue/i }) as HTMLButtonElement).disabled).toBe(true)
    expect(container.textContent).toContain("between 0 and 50")
  })

  it("requires a name and validates email format when provided", () => {
    const onSubmit = vi.fn()
    const { container } = renderForm({ onSubmit })
    const name = screen.getByLabelText(/name/i) as HTMLInputElement
    fireEvent.change(name, { target: { value: "" } })
    expect((screen.getByRole("button", { name: /confirm & continue/i }) as HTMLButtonElement).disabled).toBe(true)

    fireEvent.change(name, { target: { value: "Alex Dev" } })
    const email = screen.getByLabelText(/email/i) as HTMLInputElement
    fireEvent.change(email, { target: { value: "not-an-email" } })
    expect((screen.getByRole("button", { name: /confirm & continue/i }) as HTMLButtonElement).disabled).toBe(true)
    expect(container.textContent).toContain("valid email")
    fireEvent.change(email, { target: { value: "" } })
    expect((screen.getByRole("button", { name: /confirm & continue/i }) as HTMLButtonElement).disabled).toBe(false)
  })

  it("caps text fields to backend limits", () => {
    renderForm({ onSubmit: () => {} })
    const summary = screen.getByLabelText(/professional summary/i) as HTMLTextAreaElement
    expect(summary.maxLength).toBe(2000)
    const name = screen.getByLabelText(/name/i) as HTMLInputElement
    expect(name.maxLength).toBe(200)
    const email = screen.getByLabelText(/email/i) as HTMLInputElement
    expect(email.maxLength).toBe(254)
    const education = screen.getByLabelText(/education/i) as HTMLInputElement
    expect(education.maxLength).toBe(200)
  })

  it("respects max cap on skills and certifications chips", () => {
    renderForm({ onSubmit: () => {} })
    const skills = screen.getByLabelText(/skills/i) as HTMLInputElement
    const certs = screen.getByLabelText(/certifications/i) as HTMLInputElement
    fireEvent.change(skills, { target: { value: "AWS" } })
    fireEvent.keyDown(skills, { key: "Enter" })
    expect(screen.getAllByText("AWS").length).toBeGreaterThan(0)
    fireEvent.change(certs, { target: { value: "GCP" } })
    fireEvent.keyDown(certs, { key: "Enter" })
    expect(screen.getAllByText("GCP").length).toBeGreaterThan(0)
  })

  it("reflects edited draft values in the submitted payload", () => {
    const onSubmit = vi.fn()
    renderForm({ onSubmit })
    const name = screen.getByLabelText(/name/i) as HTMLInputElement
    fireEvent.change(name, { target: { value: "Alex New" } })
    const skills = screen.getByLabelText(/skills/i) as HTMLInputElement
    fireEvent.change(skills, { target: { value: "Python" } })
    fireEvent.keyDown(skills, { key: "Enter" })
    fireEvent.click(screen.getByRole("button", { name: /confirm & continue/i }))
    const draft = onSubmit.mock.calls[0][0] as CVDraftProfile
    expect(draft.name).toBe("Alex New")
    expect(draft.skills).toEqual(["Go", "Postgres", "Python"])
  })
})
