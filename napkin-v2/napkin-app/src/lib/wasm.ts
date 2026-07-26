export interface WasmResult {
  ok: boolean
  value?: string
  error?: string
}

export interface NapkinAPI {
  version(): string
  echo(value: string): string
}

declare global {
  var Go: {
    new (): {
      importObject: WebAssembly.Imports
      run(instance: WebAssembly.Instance): Promise<void>
    }
  }

  var napkin: NapkinAPI | undefined
}

let loadPromise: Promise<NapkinAPI> | undefined

function loadRuntime(): Promise<void> {
  if (globalThis.Go) return Promise.resolve()

  return new Promise((resolve, reject) => {
    const script = document.createElement('script')
    script.src = '/wasm_exec.js'
    script.onload = () => resolve()
    script.onerror = () => reject(new Error('Could not load the Go WASM runtime'))
    document.head.appendChild(script)
  })
}

async function instantiate(
  go: InstanceType<typeof Go>,
): Promise<WebAssembly.Instance> {
  const response = await fetch('/napkin.wasm')
  if (!response.ok) {
    throw new Error(`Could not load napkin.wasm (${response.status})`)
  }

  try {
    const result = await WebAssembly.instantiateStreaming(
      response.clone(),
      go.importObject,
    )
    return result.instance
  } catch {
    const result = await WebAssembly.instantiate(
      await response.arrayBuffer(),
      go.importObject,
    )
    return result.instance
  }
}

async function waitForAPI(): Promise<NapkinAPI> {
  for (let attempt = 0; attempt < 100; attempt += 1) {
    if (globalThis.napkin) return globalThis.napkin
    await new Promise((resolve) => setTimeout(resolve, 10))
  }
  throw new Error('Go WASM started but did not expose the Napkin API')
}

export function loadNapkin(): Promise<NapkinAPI> {
  loadPromise ??= (async () => {
    await loadRuntime()
    const go = new Go()
    const instance = await instantiate(go)
    void go.run(instance).catch((error: unknown) => {
      console.error('Napkin WASM runtime stopped', error)
    })
    return waitForAPI()
  })()

  return loadPromise
}

export function parseWasmResult(raw: string): WasmResult {
  return JSON.parse(raw) as WasmResult
}
