import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { createTrailingDebounce } from "./debounce"

describe("createTrailingDebounce", () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it("collapses a burst into one trailing call with the latest args", () => {
    const fn = vi.fn()
    const debounced = createTrailingDebounce(fn, 300)
    debounced("a")
    debounced("b")
    debounced("c")
    expect(fn).not.toHaveBeenCalled()
    vi.advanceTimersByTime(300)
    expect(fn).toHaveBeenCalledTimes(1)
    expect(fn).toHaveBeenCalledWith("c")
  })

  it("flush() invokes immediately with the latest args and cancels the timer", () => {
    const fn = vi.fn()
    const debounced = createTrailingDebounce(fn, 300)
    debounced("draft-1")
    debounced("draft-2")
    debounced.flush()
    expect(fn).toHaveBeenCalledTimes(1)
    expect(fn).toHaveBeenCalledWith("draft-2")
    vi.advanceTimersByTime(1000)
    expect(fn).toHaveBeenCalledTimes(1)
  })

  it("flush() is a no-op when nothing is pending", () => {
    const fn = vi.fn()
    const debounced = createTrailingDebounce(fn, 300)
    debounced.flush()
    expect(fn).not.toHaveBeenCalled()
  })

  it("cancel() drops the pending call", () => {
    const fn = vi.fn()
    const debounced = createTrailingDebounce(fn, 300)
    debounced("x")
    debounced.cancel()
    vi.advanceTimersByTime(1000)
    expect(fn).not.toHaveBeenCalled()
  })
})
