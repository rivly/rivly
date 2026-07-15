import { type Stack } from '../lib/stacks'
import { BulkBar } from './BulkBar'
import { StackActionButtons, StackDeleteButton } from './StackActionButtons'

type Props = {
  envId: number
  selected: Stack[]
  clear: () => void
}

export function StackBulkBar({ envId, selected, clear }: Props) {
  return (
    <BulkBar count={selected.length} clear={clear}>
      <StackActionButtons envId={envId} items={selected} onDone={() => clear()} />
      <StackDeleteButton envId={envId} items={selected} onDone={() => clear()} />
    </BulkBar>
  )
}
