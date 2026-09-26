class MemoryStorage implements Storage {
  private store = new Map<string, string>()
  get length() { return this.store.size }
  clear() { this.store.clear() }
  getItem(key: string) { return this.store.get(key) ?? null }
  key(index: number) { return Array.from(this.store.keys())[index] ?? null }
  removeItem(key: string) { this.store.delete(key) }
  setItem(key: string, value: string) { this.store.set(key, String(value)) }
}

const memLocal = new MemoryStorage()
const memSession = new MemoryStorage()

Object.defineProperty(globalThis, "localStorage", {
  value: memLocal,
  configurable: true,
  writable: true,
})
Object.defineProperty(globalThis, "sessionStorage", {
  value: memSession,
  configurable: true,
  writable: true,
})
if (typeof window !== "undefined") {
  Object.defineProperty(window, "localStorage", {
    value: memLocal,
    configurable: true,
    writable: true,
  })
  Object.defineProperty(window, "sessionStorage", {
    value: memSession,
    configurable: true,
    writable: true,
  })
}
