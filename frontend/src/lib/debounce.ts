// Trailing-edge debounce with an explicit flush — used for WS code-change
// telemetry so keystroke storms do not flood the socket, while submit can
// flush the latest editor state before the answer frame leaves.
export interface TrailingDebounce<A extends unknown[]> {
  (...args: A): void
  flush(): void
  cancel(): void
}

export function createTrailingDebounce<A extends unknown[]>(
  fn: (...args: A) => void,
  waitMs: number,
): TrailingDebounce<A> {
  let timer: ReturnType<typeof setTimeout> | null = null
  let pendingArgs: A | null = null

  const run = () => {
    timer = null
    const args = pendingArgs
    pendingArgs = null
    if (args) fn(...args)
  }

  const debounced = (...args: A): void => {
    pendingArgs = args
    if (timer) clearTimeout(timer)
    timer = setTimeout(run, waitMs)
  }

  debounced.flush = (): void => {
    if (timer) {
      clearTimeout(timer)
      timer = null
    }
    run()
  }

  debounced.cancel = (): void => {
    if (timer) {
      clearTimeout(timer)
      timer = null
    }
    pendingArgs = null
  }

  return debounced
}
