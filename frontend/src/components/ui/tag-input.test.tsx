import { render, screen, fireEvent, cleanup } from "@testing-library/react"
import { useState } from "react"
import { describe, expect, it, afterEach, vi } from "vitest"
import { TagInput } from "./tag-input"

function Harness({ initial, max, label, hint }: { initial: string[]; max?: number; label?: string; hint?: string }) {
  const [value, setValue] = useState(initial)
  return <TagInput value={value} onChange={setValue} max={max} label={label} hint={hint} />
}

describe("TagInput", () => {
  afterEach(() => {
    cleanup()
  })

  it("renders chips from the initial value list", () => {
    render(<Harness initial={["Go", "Postgres"]} label="Skills" />)
    expect(screen.getByText("Go")).toBeDefined()
    expect(screen.getByText("Postgres")).toBeDefined()
    expect(screen.getByLabelText("Skills")).toBeDefined()
  })

  it("adds a chip on Enter, trimmed, case-insensitive dedupe", () => {
    const onChange = vi.fn()
    render(<TagInput value={["Go"]} onChange={onChange} label="Skills" />)
    const input = screen.getByLabelText("Skills") as HTMLInputElement
    fireEvent.change(input, { target: { value: "  react  " } })
    fireEvent.keyDown(input, { key: "Enter" })
    expect(onChange).toHaveBeenCalledWith(["Go", "react"])
    cleanup()
    render(<TagInput value={["Go", "react"]} onChange={onChange} label="Skills" />)
    const input2 = screen.getByLabelText("Skills") as HTMLInputElement
    fireEvent.change(input2, { target: { value: "REACT" } })
    fireEvent.keyDown(input2, { key: "Enter" })
    expect(onChange).toHaveBeenCalledTimes(1)
  })

  it("adds a chip on comma press", () => {
    const onChange = vi.fn()
    render(<TagInput value={[]} onChange={onChange} label="Certifications" />)
    const input = screen.getByLabelText("Certifications") as HTMLInputElement
    fireEvent.change(input, { target: { value: "AWS" } })
    fireEvent.keyDown(input, { key: "," })
    expect(onChange).toHaveBeenCalledWith(["AWS"])
  })

  it("removes last chip on Backspace when the input is empty", () => {
    const onChange = vi.fn()
    render(<TagInput value={["Go", "React"]} onChange={onChange} label="Skills" />)
    const input = screen.getByLabelText("Skills") as HTMLInputElement
    fireEvent.keyDown(input, { key: "Backspace" })
    expect(onChange).toHaveBeenCalledWith(["Go"])
  })

  it("removes a chip via its X remove button", () => {
    const onChange = vi.fn()
    render(<TagInput value={["Go", "React"]} onChange={onChange} label="Skills" />)
    const removeButtons = screen.getAllByRole("button", { name: /remove/i })
    expect(removeButtons.length).toBe(2)
    fireEvent.click(removeButtons[0])
    expect(onChange).toHaveBeenCalledWith(["React"])
  })

  it("ignores adds at the max cap", () => {
    const onChange = vi.fn()
    render(<TagInput value={["a", "b"]} onChange={onChange} label="Skills" max={2} />)
    const input = screen.getByLabelText("Skills") as HTMLInputElement
    fireEvent.change(input, { target: { value: "c" } })
    fireEvent.keyDown(input, { key: "Enter" })
    expect(onChange).not.toHaveBeenCalled()
  })

  it("shows a hint when provided", () => {
    render(<Harness initial={[]} label="Skills" hint="Press Enter or comma to add a skill" />)
    expect(screen.getByText("Press Enter or comma to add a skill")).toBeDefined()
  })
})
