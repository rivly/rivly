import { useEffect, useLayoutEffect, useRef, useState } from 'react'
import { Dialog } from '@base-ui/react/dialog'
import { LuArrowDown, LuTrash2, LuWrapText, LuX } from 'react-icons/lu'
import type { ContainerRef } from '../lib/containers'
import { useContainerLogs, type LogStatus } from '../lib/logs'
import { Button } from './Button'
import { Loader } from './Loader'
import { Tooltip } from './Tooltip'
import styles from './LogsViewer.module.css'

type Props = {
  envId: number
  container: ContainerRef | null
  onClose: () => void
}

export function LogsViewer({ envId, container, onClose }: Props) {
  const [active, setActive] = useState<ContainerRef | null>(container)

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
          {active && <LogsPanel key={active.id} envId={envId} container={active} />}
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  )
}

function streamBadge(status: LogStatus, paused: boolean, styles: Record<string, string>) {
  if (status === 'connecting') {
    return { label: 'Connecting', tone: styles.connecting }
  }
  if (status === 'error') {
    return { label: 'Reconnecting', tone: styles.connecting }
  }
  if (status === 'ended') {
    return { label: 'Ended', tone: styles.ended }
  }
  if (paused) {
    return { label: 'Paused', tone: styles.paused }
  }
  return { label: 'Live', tone: styles.streaming }
}

function LogsPanel({ envId, container }: { envId: number; container: ContainerRef }) {
  const { lines, status } = useContainerLogs(envId, container.id)
  const [wrap, setWrap] = useState(true)
  const [clearedAt, setClearedAt] = useState(-1)

  const scrollRef = useRef<HTMLDivElement>(null)
  const stick = useRef(true)
  const [showJump, setShowJump] = useState(false)

  const visible = lines.filter((line) => line.id > clearedAt)

  useLayoutEffect(() => {
    if (stick.current && scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [visible.length, wrap])

  const handleScroll = () => {
    const el = scrollRef.current
    if (!el) {
      return
    }
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 60
    stick.current = atBottom
    setShowJump(!atBottom)
  }

  const jumpToLatest = () => {
    const el = scrollRef.current
    if (el) {
      el.scrollTop = el.scrollHeight
      stick.current = true
      setShowJump(false)
    }
  }

  const clear = () => {
    if (lines.length > 0) {
      setClearedAt(lines[lines.length - 1].id)
    }
  }

  return (
    <>
      <header className={styles.header}>
        <div className={styles.heading}>
          <Dialog.Title className={styles.title}>{container.name}</Dialog.Title>
          {(() => {
            const badge = streamBadge(status, showJump, styles)
            return (
              <span className={`${styles.status} ${badge.tone}`}>
                <span className={styles.statusDot} />
                {badge.label}
              </span>
            )
          })()}
        </div>
        <div className={styles.controls}>
          <Tooltip content={wrap ? 'Disable wrapping' : 'Wrap lines'}>
            <Button
              variant="secondary"
              size="sm"
              iconOnly
              icon={<LuWrapText />}
              aria-label="Toggle line wrapping"
              aria-pressed={wrap}
              className={wrap ? styles.active : undefined}
              onClick={() => setWrap((value) => !value)}
            />
          </Tooltip>
          <Tooltip content="Clear">
            <Button
              variant="secondary"
              size="sm"
              iconOnly
              icon={<LuTrash2 />}
              aria-label="Clear logs"
              onClick={clear}
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
        <div
          ref={scrollRef}
          onScroll={handleScroll}
          className={`${styles.stream} ${wrap ? styles.wrap : ''}`}
        >
          {visible.length === 0 ? (
            <EmptyState status={status} />
          ) : (
            visible.map((line) => (
              <div
                key={line.id}
                className={`${styles.line} ${line.stream === 'stderr' ? styles.stderr : ''}`}
              >
                {line.message === '' ? ' ' : line.message}
              </div>
            ))
          )}
        </div>
        {showJump && (
          <button type="button" className={styles.jump} onClick={jumpToLatest}>
            <LuArrowDown />
            Jump to latest
          </button>
        )}
      </div>
    </>
  )
}

function EmptyState({ status }: { status: LogStatus }) {
  if (status === 'connecting') {
    return (
      <div className={styles.empty}>
        <Loader />
      </div>
    )
  }
  return (
    <div className={styles.empty}>
      {status === 'ended' ? 'This container produced no logs.' : 'No logs yet.'}
    </div>
  )
}
