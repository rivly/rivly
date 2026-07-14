import { useEffect, useRef, useState } from 'react'
import { Dialog } from '@base-ui/react/dialog'
import { useQueryClient } from '@tanstack/react-query'
import { LuX } from 'react-icons/lu'
import { toast } from '../lib/toast'
import { Button } from './Button'
import styles from './PullImageDialog.module.css'

type Props = {
  envId: number
  open: boolean
  onClose: () => void
}

type Phase = 'idle' | 'pulling' | 'done' | 'error'
type Layer = { key: string; text: string }

export function PullImageDialog({ envId, open, onClose }: Props) {
  return (
    <Dialog.Root
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          onClose()
        }
      }}
    >
      <Dialog.Portal>
        <Dialog.Backdrop className={styles.backdrop} />
        <Dialog.Popup className={styles.popup}>
          {open && <PullBody envId={envId} />}
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  )
}

function PullBody({ envId }: { envId: number }) {
  const queryClient = useQueryClient()
  const [ref, setRef] = useState('')
  const [phase, setPhase] = useState<Phase>('idle')
  const [layers, setLayers] = useState<Layer[]>([])
  const [error, setError] = useState<string | null>(null)
  const sourceRef = useRef<EventSource | null>(null)
  const errorRef = useRef<string | null>(null)
  const scrollRef = useRef<HTMLDivElement>(null)

  useEffect(() => () => sourceRef.current?.close(), [])

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [layers])

  const start = () => {
    const image = ref.trim()
    if (image === '' || phase === 'pulling') {
      return
    }
    setPhase('pulling')
    setLayers([])
    setError(null)
    errorRef.current = null

    const source = new EventSource(
      `/api/v1/environments/${envId}/images/pull?ref=${encodeURIComponent(image)}`,
    )
    sourceRef.current = source

    source.onmessage = (event) => {
      let p: { status?: string; id?: string; current?: number; total?: number; error?: string }
      try {
        p = JSON.parse(event.data)
      } catch {
        return
      }
      if (p.error) {
        errorRef.current = p.error
        setError(p.error)
        return
      }
      if (!p.status) {
        return
      }
      const bar = p.total ? ` (${Math.round(((p.current ?? 0) / p.total) * 100)}%)` : ''
      const text = `${p.id ? `${p.id}  ` : ''}${p.status}${bar}`
      setLayers((prev) => {
        if (p.id) {
          const idx = prev.findIndex((l) => l.key === p.id)
          if (idx >= 0) {
            const next = [...prev]
            next[idx] = { key: p.id, text }
            return next
          }
          return [...prev, { key: p.id, text }]
        }
        return [...prev, { key: `_${prev.length}`, text }]
      })
    }

    source.addEventListener('end', () => {
      source.close()
      if (errorRef.current) {
        setPhase('error')
        return
      }
      setPhase('done')
      queryClient.invalidateQueries({ queryKey: ['images', envId] })
      toast.success(`Pulled ${image}`)
    })

    source.onerror = () => {
      source.close()
      setPhase('error')
      if (!errorRef.current) {
        setError('Could not pull this image. Check the name and try again.')
      }
    }
  }

  return (
    <>
      <header className={styles.header}>
        <Dialog.Title className={styles.title}>Pull an image</Dialog.Title>
        <Dialog.Close
          render={<Button variant="secondary" size="sm" iconOnly icon={<LuX />} aria-label="Close" />}
        />
      </header>

      <div className={styles.body}>
        <form
          className={styles.form}
          onSubmit={(event) => {
            event.preventDefault()
            start()
          }}
        >
          <input
            className={styles.input}
            value={ref}
            onChange={(event) => setRef(event.target.value)}
            placeholder="nginx:latest"
            autoComplete="off"
            spellCheck={false}
            disabled={phase === 'pulling'}
          />
          <Button type="submit" loading={phase === 'pulling'} disabled={ref.trim() === ''}>
            Pull
          </Button>
        </form>

        {phase !== 'idle' && (
          <div ref={scrollRef} className={styles.progress}>
            {layers.map((l) => (
              <div key={l.key} className={styles.line}>
                {l.text}
              </div>
            ))}
            {error && <div className={styles.errorLine}>{error}</div>}
            {phase === 'done' && <div className={styles.doneLine}>Done.</div>}
          </div>
        )}
      </div>
    </>
  )
}
