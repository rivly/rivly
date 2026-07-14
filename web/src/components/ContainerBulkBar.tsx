import { type Container } from '../lib/containers'
import { BulkBar } from './BulkBar'
import { ContainerActionButtons } from './ContainerActionButtons'

type Props = {
  envId: number
  selected: Container[]
  clear: () => void
}

export function ContainerBulkBar({ envId, selected, clear }: Props) {
  return (
    <BulkBar count={selected.length} clear={clear}>
      <ContainerActionButtons envId={envId} items={selected} onDone={() => clear()} />
    </BulkBar>
  )
}
