import { LuTrash2 } from 'react-icons/lu'
import { BulkBar } from './BulkBar'
import { Button } from './Button'
import { ConfirmDialog } from './ConfirmDialog'

type Props = {
  selectedCount: number
  inUseCount: number
  noun: string
  pending: boolean
  onRemove: () => void
  clear: () => void
}

export function RemoveBulkBar({
  selectedCount,
  inUseCount,
  noun,
  pending,
  onRemove,
  clear,
}: Props) {
  const plural = selectedCount > 1 ? 's' : ''

  return (
    <BulkBar count={selectedCount} clear={clear}>
      <ConfirmDialog
        trigger={
          <Button variant="danger" size="sm" icon={<LuTrash2 />} loading={pending}>
            Remove
          </Button>
        }
        title={`Remove ${selectedCount} ${noun}${plural}?`}
        description={
          inUseCount > 0
            ? `${inUseCount} of them ${inUseCount > 1 ? 'are' : 'is'} in use and will be skipped. This cannot be undone.`
            : `This permanently removes the selected ${noun}s. This cannot be undone.`
        }
        onConfirm={onRemove}
      />
    </BulkBar>
  )
}
