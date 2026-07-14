import { useState, type ReactNode } from 'react'
import { AlertDialog } from '@base-ui/react/alert-dialog'
import { LuPlay, LuRotateCw, LuSquare, LuTrash2, LuX } from 'react-icons/lu'
import { useStackActions, type Stack, type StackAction } from '../lib/stacks'
import { toast } from '../lib/toast'
import { Button } from './Button'
import styles from './ContainerBulkBar.module.css'

type ActionSpec = {
  key: Exclude<StackAction, 'remove'>
  label: string
  icon: ReactNode
  eligible: (stack: Stack) => boolean
}

const ACTIONS: ActionSpec[] = [
  { key: 'start', label: 'Start', icon: <LuPlay />, eligible: (s) => s.running < s.total },
  { key: 'restart', label: 'Restart', icon: <LuRotateCw />, eligible: (s) => s.running > 0 },
  { key: 'stop', label: 'Stop', icon: <LuSquare />, eligible: (s) => s.running > 0 },
]

const PAST: Record<StackAction, string> = {
  start: 'Started',
  stop: 'Stopped',
  restart: 'Restarted',
  remove: 'Removed',
}

type Props = {
  envId: number
  selected: Stack[]
  clear: () => void
}

export function StackBulkBar({ envId, selected, clear }: Props) {
  const mutation = useStackActions(envId)
  const [confirmRemove, setConfirmRemove] = useState(false)

  const run = (action: StackAction, names: string[]) => {
    if (names.length === 0) {
      return
    }
    mutation.mutate(
      { action, ids: names },
      {
        onSuccess: (data) => {
          const ok = data.results.filter((result) => result.ok).length
          const failed = data.results.length - ok
          if (failed === 0) {
            toast.success(`${PAST[action]} ${ok} stack${ok > 1 ? 's' : ''}`)
          } else {
            toast.error(
              `${PAST[action]} ${ok}/${data.results.length} stacks`,
              `${failed} did not fully ${action}`,
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
        {ACTIONS.filter((action) => selected.some(action.eligible)).map((action) => (
          <Button
            key={action.key}
            variant="secondary"
            size="sm"
            icon={action.icon}
            disabled={mutation.isPending}
            loading={mutation.isPending && mutation.variables?.action === action.key}
            onClick={() => run(action.key, selected.filter(action.eligible).map((s) => s.name))}
          >
            {action.label}
          </Button>
        ))}

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
                Remove {selected.length} stack{selected.length > 1 ? 's' : ''}?
              </AlertDialog.Title>
              <AlertDialog.Description className={styles.dialogText}>
                This removes every container in the selected stacks. Volumes and networks are
                kept. This cannot be undone.
              </AlertDialog.Description>
              <div className={styles.dialogActions}>
                <AlertDialog.Close render={<Button variant="secondary" size="sm">Cancel</Button>} />
                <Button
                  variant="danger"
                  size="sm"
                  onClick={() => {
                    setConfirmRemove(false)
                    run('remove', selected.map((s) => s.name))
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
