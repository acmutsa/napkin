import { useEffect, useState } from 'react'
import './App.css'
import { loadNapkin, parseWasmResult, type NapkinAPI } from './lib/wasm'

type RuntimeState =
  | { status: 'loading' }
  | { status: 'ready'; api: NapkinAPI; version: string }
  | { status: 'error'; message: string }

function App() {
  const [runtime, setRuntime] = useState<RuntimeState>({ status: 'loading' })
  const [input, setInput] = useState('Hello from React')
  const [output, setOutput] = useState('')

  useEffect(() => {
    let active = true

    loadNapkin()
      .then((api) => {
        if (active) setRuntime({ status: 'ready', api, version: api.version() })
      })
      .catch((error: unknown) => {
        if (active) {
          setRuntime({
            status: 'error',
            message: error instanceof Error ? error.message : String(error),
          })
        }
      })

    return () => {
      active = false
    }
  }, [])

  function runEcho() {
    if (runtime.status !== 'ready') return

    try {
      const result = parseWasmResult(runtime.api.echo(input))
      setOutput(result.ok ? (result.value ?? '') : (result.error ?? 'Unknown error'))
    } catch (error) {
      setOutput(error instanceof Error ? error.message : String(error))
    }
  }

  return (
    <main>
      <header>
        <p className="eyebrow">Napkin V2</p>
        <h1>Go, running in your browser.</h1>
        <p className="lede">
          A fresh React frontend connected to a Go WebAssembly backend.
        </p>
      </header>

      <section className="card" aria-labelledby="runtime-title">
        <div className="card-heading">
          <div>
            <h2 id="runtime-title">WASM runtime</h2>
            <p>The status below is reported by the Go module.</p>
          </div>
          <span className={`status ${runtime.status}`}>
            {runtime.status === 'loading' && 'Loading'}
            {runtime.status === 'ready' && 'Ready'}
            {runtime.status === 'error' && 'Failed'}
          </span>
        </div>

        {runtime.status === 'ready' && (
          <p className="detail">
            Version: <code>{runtime.version}</code>
          </p>
        )}
        {runtime.status === 'error' && (
          <p className="error" role="alert">
            {runtime.message}
          </p>
        )}
      </section>

      <section className="card" aria-labelledby="interop-title">
        <h2 id="interop-title">JS → Go → JS</h2>
        <p>Send a string through the typed WebAssembly interop boundary.</p>
        <div className="controls">
          <label htmlFor="echo-input">Message</label>
          <div className="input-row">
            <input
              id="echo-input"
              value={input}
              onChange={(event) => setInput(event.target.value)}
            />
            <button onClick={runEcho} disabled={runtime.status !== 'ready'}>
              Call Go
            </button>
          </div>
        </div>
        {output && <output>{output}</output>}
      </section>
    </main>
  )
}

export default App
