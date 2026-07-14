import type { ReactNode } from 'react'
import { LuPause, LuPlay, LuRotateCw, LuSquare, LuTrash2, LuZap } from 'react-icons/lu'
import { useContainerActions, type ContainerAction } from '../lib/containers'
import { bulkActionError, reportBulk } from '../lib/bulk'
import { Button } from './Button'
import { ConfirmDialog } from './ConfirmDialog'

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

type Item = { id: string; state: string }

type Props = {
  envId: number
  items: Item[]
  onDone?: (action: ContainerAction) => void
}

export function ContainerActionButtons({ envId, items, onDone }: Props) {
  const mutation = useContainerActions(envId)

  const run = (action: ContainerAction, ids: string[]) => {
    if (ids.length === 0) {
      return
    }
    mutation.mutate(
      { action, ids },
      {
        onSuccess: (data) =>
          reportBulk(data.results, {
            verb: PAST[action],
            noun: 'container',
            failHint: (failed) => `${failed} could not be ${PAST[action].toLowerCase()}`,
            clear: () => onDone?.(action),
          }),
        onError: bulkActionError,
      },
    )
  }

  const loading = (action: ContainerAction) =>
    mutation.isPending && mutation.variables?.action === action

  return (
    <>
      {ACTIONS.filter((action) => items.some((item) => action.eligible(item.state))).map(
        (action) => (
          <Button
            key={action.key}
            variant="secondary"
            size="sm"
            icon={action.icon}
            loading={loading(action.key)}
            disabled={mutation.isPending}
            onClick={() =>
              run(action.key, items.filter((item) => action.eligible(item.state)).map((item) => item.id))
            }
          >
            {action.label}
          </Button>
        ),
      )}
      <ConfirmDialog
        trigger={
          <Button
            variant="danger"
            size="sm"
            icon={<LuTrash2 />}
            loading={loading('remove')}
            disabled={mutation.isPending}
          >
            Remove
          </Button>
        }
        title={`Remove ${items.length} container${items.length > 1 ? 's' : ''}?`}
        description="This permanently removes the selected containers, running ones included. This cannot be undone."
        onConfirm={() => run('remove', items.map((item) => item.id))}
      />
    </>
  )
}
