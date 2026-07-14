import type { ReactNode } from 'react'
import { LuPlay, LuRotateCw, LuSquare, LuTrash2 } from 'react-icons/lu'
import { useStackActions, type Stack, type StackAction } from '../lib/stacks'
import { bulkActionError, reportBulk } from '../lib/bulk'
import { Button } from './Button'
import { ConfirmDialog } from './ConfirmDialog'

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
  items: Stack[]
  onDone?: (action: StackAction) => void
}

export function StackActionButtons({ envId, items, onDone }: Props) {
  const mutation = useStackActions(envId)

  const run = (action: StackAction, names: string[]) => {
    if (names.length === 0) {
      return
    }
    mutation.mutate(
      { action, ids: names },
      {
        onSuccess: (data) =>
          reportBulk(data.results, {
            verb: PAST[action],
            noun: 'stack',
            failHint: (failed) => `${failed} did not fully ${action}`,
            clear: () => onDone?.(action),
          }),
        onError: bulkActionError,
      },
    )
  }

  const loading = (action: StackAction) =>
    mutation.isPending && mutation.variables?.action === action

  return (
    <>
      {ACTIONS.filter((action) => items.some(action.eligible)).map((action) => (
        <Button
          key={action.key}
          variant="secondary"
          size="sm"
          icon={action.icon}
          loading={loading(action.key)}
          disabled={mutation.isPending}
          onClick={() => run(action.key, items.filter(action.eligible).map((stack) => stack.name))}
        >
          {action.label}
        </Button>
      ))}
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
        title={`Remove ${items.length} stack${items.length > 1 ? 's' : ''}?`}
        description="This removes every container in the selected stacks. Volumes and networks are kept. This cannot be undone."
        onConfirm={() => run('remove', items.map((stack) => stack.name))}
      />
    </>
  )
}
