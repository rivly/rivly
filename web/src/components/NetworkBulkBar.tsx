import { useNetworkActions, type Network } from '../lib/networks'
import { toast } from '../lib/toast'
import { RemoveBulkBar } from './RemoveBulkBar'

type Props = {
  envId: number
  selected: Network[]
  clear: () => void
}

export function NetworkBulkBar({ envId, selected, clear }: Props) {
  const mutation = useNetworkActions(envId)

  const remove = () => {
    mutation.mutate(
      { action: 'remove', ids: selected.map((network) => network.id) },
      {
        onSuccess: (data) => {
          const ok = data.results.filter((result) => result.ok).length
          const failed = data.results.length - ok
          if (failed === 0) {
            toast.success(`Removed ${ok} network${ok > 1 ? 's' : ''}`)
          } else {
            toast.error(
              `Removed ${ok}/${data.results.length} networks`,
              `${failed} in use or predefined`,
            )
          }
          clear()
        },
        onError: () => toast.error('Action failed', 'Could not reach the environment'),
      },
    )
  }

  return (
    <RemoveBulkBar
      selectedCount={selected.length}
      inUseCount={selected.filter((network) => network.inUse).length}
      noun="network"
      pending={mutation.isPending}
      onRemove={remove}
      clear={clear}
    />
  )
}
