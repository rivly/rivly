import { useEffect, useRef, useState } from 'react'
import { Dialog } from '@base-ui/react/dialog'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { LuEraser, LuX } from 'react-icons/lu'
import '@xterm/xterm/css/xterm.css'
import type { Container } from '../lib/containers'
import { Button } from './Button'
import { Tooltip } from './Tooltip'
import styles from './TerminalViewer.module.css'

type Props = {
  envId: number
  container: Container | null
  onClose: () => void
}

type TermStatus = 'connecting' | 'connected' | 'closed' | 'error'

const STATUS_LABEL: Record<TermStatus, string> = {
  connecting: 'Connecting',
  connected: 'Connected',
  closed: 'Session ended',
  error: 'Disconnected',
}

const THEME = {
  background: '#0d1117',
  foreground: '#e6edf3',
  cursor: '#e6edf3',
  cursorAccent: '#0d1117',
  selectionBackground: 'rgba(47, 123, 255, 0.35)',
  black: '#484f58',
  red: '#ff7b72',
  green: '#3fb950',
  yellow: '#d29922',
  blue: '#58a6ff',
  magenta: '#bc8cff',
  cyan: '#39c5cf',
  white: '#b1bac4',
  brightBlack: '#6e7681',
  brightRed: '#ffa198',
  brightGreen: '#56d364',
  brightYellow: '#e3b341',
  brightBlue: '#79c0ff',
  brightMagenta: '#d2a8ff',
  brightCyan: '#56d4dd',
  brightWhite: '#f0f6fc',
}

export function TerminalViewer({ envId, container, onClose }: Props) {
  const [active, setActive] = useState<Container | null>(container)

  useEffect(() => {
    if (container) {
      setActive(container)
    }
  }, [container])

  return (
    <Dialog.Root
      open={container !== null}
      onOpenChange={(open) => {
        if (!open) {
          onClose()
        }
      }}
      onOpenChangeComplete={(open) => {
        if (!open) {
          setActive(null)
        }
      }}
    >
      <Dialog.Portal>
        <Dialog.Backdrop className={styles.backdrop} />
        <Dialog.Popup className={styles.popup}>
          {active && <TerminalPanel key={active.id} envId={envId} container={active} />}
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  )
}

function TerminalPanel({ envId, container }: { envId: number; container: Container }) {
  const mountRef = useRef<HTMLDivElement>(null)
  const termRef = useRef<Terminal | null>(null)
  const [status, setStatus] = useState<TermStatus>('connecting')

  useEffect(() => {
    const mount = mountRef.current
    if (!mount) {
      return
    }

    const term = new Terminal({
      cursorBlink: true,
      fontFamily: '"Geist Mono Variable", ui-monospace, "SF Mono", Menlo, monospace',
      fontSize: 13,
      theme: THEME,
    })
    termRef.current = term
    const fit = new FitAddon()
    term.loadAddon(fit)
    term.open(mount)
    requestAnimationFrame(() => {
      try {
        fit.fit()
      } catch {
        /* container not measured yet */
      }
    })

    const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
    const ws = new WebSocket(
      `${protocol}://${window.location.host}/api/v1/environments/${envId}/containers/${container.id}/exec`,
    )
    ws.binaryType = 'arraybuffer'
    const encoder = new TextEncoder()

    const sendResize = () => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }))
      }
    }

    ws.onopen = () => {
      setStatus('connected')
      sendResize()
      term.focus()
    }
    ws.onmessage = (event) => {
      if (typeof event.data === 'string') {
        try {
          const message = JSON.parse(event.data)
          if (message.type === 'error') {
            term.write(`\r\n\x1b[31m${message.message}\x1b[0m\r\n`)
          }
        } catch {
          /* ignore malformed control frame */
        }
        return
      }
      term.write(new Uint8Array(event.data))
    }
    ws.onclose = () => {
      setStatus((current) => (current === 'error' ? current : 'closed'))
      term.write('\r\n\x1b[90m[session ended]\x1b[0m\r\n')
    }
    ws.onerror = () => setStatus('error')

    const dataListener = term.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(encoder.encode(data))
      }
    })
    const resizeListener = term.onResize(sendResize)

    const observer = new ResizeObserver(() => {
      try {
        fit.fit()
      } catch {
        /* ignore */
      }
    })
    observer.observe(mount)

    return () => {
      observer.disconnect()
      dataListener.dispose()
      resizeListener.dispose()
      ws.close()
      term.dispose()
      termRef.current = null
    }
  }, [envId, container.id])

  return (
    <>
      <header className={styles.header}>
        <div className={styles.heading}>
          <Dialog.Title className={styles.title}>{container.name}</Dialog.Title>
          <span className={`${styles.status} ${styles[status]}`}>
            <span className={styles.statusDot} />
            {STATUS_LABEL[status]}
          </span>
        </div>
        <div className={styles.controls}>
          <Tooltip content="Clear">
            <Button
              variant="secondary"
              size="sm"
              iconOnly
              icon={<LuEraser />}
              aria-label="Clear terminal"
              onClick={() => termRef.current?.clear()}
            />
          </Tooltip>
          <Dialog.Close
            render={
              <Button variant="secondary" size="sm" iconOnly icon={<LuX />} aria-label="Close" />
            }
          />
        </div>
      </header>

      <div className={styles.body}>
        <div ref={mountRef} className={styles.term} />
      </div>
    </>
  )
}
