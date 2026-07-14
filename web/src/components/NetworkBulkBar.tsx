import { useNetworkActions, type Network } from '../lib/networks'
import { bulkActionError, reportBulk } from '../lib/bulk'
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
        onSuccess: (data) =>
          reportBulk(data.results, {
            verb: 'Removed',
            noun: 'network',
            failHint: (failed) => `${failed} in use or predefined`,
            clear,
          }),
        onError: bulkActionError,
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
