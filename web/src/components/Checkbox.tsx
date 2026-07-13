import { Checkbox as BaseCheckbox } from '@base-ui/react/checkbox'
import { LuCheck } from 'react-icons/lu'
import type { ComponentProps } from 'react'
import styles from './Checkbox.module.css'

type Props = Omit<ComponentProps<typeof BaseCheckbox.Root>, 'className'> & {
  label: string
}

export function Checkbox({ label, ...props }: Props) {
  return (
    <label className={styles.wrap}>
      <BaseCheckbox.Root className={styles.root} {...props}>
        <BaseCheckbox.Indicator className={styles.indicator}>
          <LuCheck />
        </BaseCheckbox.Indicator>
      </BaseCheckbox.Root>
      <span className={styles.text}>{label}</span>
    </label>
  )
}
