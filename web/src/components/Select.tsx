import { Select as BaseSelect } from '@base-ui/react/select'
import { LuChevronDown } from 'react-icons/lu'
import type { ReactNode } from 'react'
import styles from './Select.module.css'

export type SelectItem = {
  label: string
  value: string
}

type Props = {
  items: SelectItem[]
  defaultValue?: string
  value?: string
  onValueChange?: (value: string | null) => void
  icon?: ReactNode
  'aria-label'?: string
}

export function Select({
  items,
  defaultValue,
  value,
  onValueChange,
  icon,
  'aria-label': ariaLabel,
}: Props) {
  return (
    <BaseSelect.Root
      items={items}
      defaultValue={defaultValue}
      value={value}
      onValueChange={onValueChange}
    >
      <BaseSelect.Trigger className={styles.trigger} aria-label={ariaLabel}>
        {icon && <span className={styles.leading}>{icon}</span>}
        <BaseSelect.Value className={styles.value} />
        <BaseSelect.Icon className={styles.caret}>
          <LuChevronDown />
        </BaseSelect.Icon>
      </BaseSelect.Trigger>
      <BaseSelect.Portal>
        <BaseSelect.Positioner sideOffset={6} className={styles.positioner}>
          <BaseSelect.Popup className={styles.popup}>
            <BaseSelect.List>
              {items.map((item) => (
                <BaseSelect.Item
                  key={item.value}
                  value={item.value}
                  className={styles.item}
                >
                  <BaseSelect.ItemText>{item.label}</BaseSelect.ItemText>
                </BaseSelect.Item>
              ))}
            </BaseSelect.List>
          </BaseSelect.Popup>
        </BaseSelect.Positioner>
      </BaseSelect.Portal>
    </BaseSelect.Root>
  )
}
