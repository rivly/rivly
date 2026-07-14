import { useState, type ReactNode } from 'react'
import { AlertDialog } from '@base-ui/react/alert-dialog'
import { LuPause, LuPlay, LuRotateCw, LuSquare, LuTrash2, LuX, LuZap } from 'react-icons/lu'
import {
  useContainerActions,
  type Container,
  type ContainerAction,
} from '../lib/containers'
import { toast } from '../lib/toast'
import { Button } from './Button'
import styles from './ContainerBulkBar.module.css'

const RUNNING = new Set(['running', 'restarting'])
const STOPPED = new Set(['exited', 'created', 'dead'])

type ActionSpec = {
  key: ContainerAction
  label: string
  icon: ReactNode
  eligible: (state: string) => boolean
}

const ACTIONS: ActionSpec[] = [
  { key: 'start', label: 'Start', icon: <LuPlay />, eligible: (s) => STOPPED.has(s) },
  { key: 'unpause', label: 'Resume', icon: <LuPlay />, eligible: (s) => s === 'paused' },
  { key: 'restart', label: 'Restart', icon: <LuRotateCw />, eligible: (s) => RUNNING.has(s) || s === 'paused' },
  { key: 'pause', label: 'Pause', icon: <LuPause />, eligible: (s) => s === 'running' },
  { key: 'stop', label: 'Stop', icon: <LuSquare />, eligible: (s) => RUNNING.has(s) },
  { key: 'kill', label: 'Kill', icon: <LuZap />, eligible: (s) => RUNNING.has(s) },
]

const PAST: Record<ContainerAction, string> = {
  start: 'Started',
  stop: 'Stopped',
  restart: 'Restarted',
  pause: 'Paused',
  unpause: 'Resumed',
  kill: 'Killed',
  remove: 'Removed',
}

type Props = {
  envId: number
  selected: Container[]
  clear: () => void
}

export function ContainerBulkBar({ envId, selected, clear }: Props) {
  const mutation = useContainerActions(envId)
  const [confirmRemove, setConfirmRemove] = useState(false)

  const run = (action: ContainerAction, ids: string[]) => {
    if (ids.length === 0) {
      return
    }
    mutation.mutate(
      { action, ids },
      {
        onSuccess: (data) => {
          const ok = data.results.filter((result) => result.ok).length
          const failed = data.results.length - ok
          if (failed === 0) {
            toast.success(`${PAST[action]} ${ok} container${ok > 1 ? 's' : ''}`)
          } else {
            toast.error(
              `${PAST[action]} ${ok}/${data.results.length} containers`,
              `${failed} could not be ${PAST[action].toLowerCase()}`,
            )
          }
          clear()
        },
        onError: () => toast.error('Action failed', 'Could not reach the environment'),
      },
    )
  }

  return (
    <div className={styles.bar}>
      <button
        type="button"
        className={styles.clear}
        onClick={clear}
        aria-label="Clear selection"
      >
        <LuX />
      </button>
      <span className={styles.count}>{selected.length} selected</span>

      <div className={styles.actions}>
        {ACTIONS.filter((action) => selected.some((c) => action.eligible(c.state))).map(
          (action) => (
            <Button
              key={action.key}
              variant="secondary"
              size="sm"
              icon={action.icon}
              disabled={mutation.isPending}
              loading={mutation.isPending && mutation.variables?.action === action.key}
              onClick={() =>
                run(
                  action.key,
                  selected.filter((c) => action.eligible(c.state)).map((c) => c.id),
                )
              }
            >
              {action.label}
            </Button>
          ),
        )}

        <AlertDialog.Root open={confirmRemove} onOpenChange={setConfirmRemove}>
          <AlertDialog.Trigger
            render={
              <Button variant="danger" size="sm" icon={<LuTrash2 />} disabled={mutation.isPending}>
                Remove
              </Button>
            }
          />
          <AlertDialog.Portal>
            <AlertDialog.Backdrop className={styles.backdrop} />
            <AlertDialog.Popup className={styles.dialog}>
              <AlertDialog.Title className={styles.dialogTitle}>
                Remove {selected.length} container{selected.length > 1 ? 's' : ''}?
              </AlertDialog.Title>
              <AlertDialog.Description className={styles.dialogText}>
                This permanently removes the selected containers, running ones included. This
                cannot be undone.
              </AlertDialog.Description>
              <div className={styles.dialogActions}>
                <AlertDialog.Close render={<Button variant="secondary" size="sm">Cancel</Button>} />
                <Button
                  variant="danger"
                  size="sm"
                  onClick={() => {
                    setConfirmRemove(false)
                    run('remove', selected.map((c) => c.id))
                  }}
                >
                  Remove
                </Button>
              </div>
            </AlertDialog.Popup>
          </AlertDialog.Portal>
        </AlertDialog.Root>
      </div>
    </div>
  )
}
