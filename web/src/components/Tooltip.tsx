import { Tooltip as BaseTooltip } from '@base-ui/react/tooltip'
import type { ReactElement, ReactNode } from 'react'
import styles from './Tooltip.module.css'

type Props = {
  content: ReactNode
  children: ReactElement
}

export function Tooltip({ content, children }: Props) {
  return (
    <BaseTooltip.Root>
      <BaseTooltip.Trigger render={children} />
      <BaseTooltip.Portal>
        <BaseTooltip.Positioner className={styles.positioner} sideOffset={6}>
          <BaseTooltip.Popup className={styles.popup}>{content}</BaseTooltip.Popup>
        </BaseTooltip.Positioner>
      </BaseTooltip.Portal>
    </BaseTooltip.Root>
  )
}
