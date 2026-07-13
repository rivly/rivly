import type { ComponentProps, ReactNode } from 'react'
import { Button as BaseButton } from '@base-ui/react/button'
import styles from './Button.module.css'

type Props = Omit<ComponentProps<typeof BaseButton>, 'className'> & {
  variant?: 'primary' | 'secondary'
  size?: 'md' | 'sm'
  icon?: ReactNode
  iconPosition?: 'start' | 'end'
  iconOnly?: boolean
  loading?: boolean
  fullWidth?: boolean
  className?: string
}

export function Button({
  variant = 'primary',
  size = 'md',
  icon,
  iconPosition = 'start',
  iconOnly = false,
  loading = false,
  fullWidth = false,
  disabled,
  className,
  children,
  ...props
}: Props) {
  const classes = [
    styles.btn,
    styles[variant],
    styles[size],
    iconOnly && styles.iconOnly,
    fullWidth && styles.fullWidth,
    className,
  ]
    .filter(Boolean)
    .join(' ')

  const showStartIcon = icon && iconPosition === 'start' && !loading
  const showEndIcon = icon && iconPosition === 'end' && !loading

  return (
    <BaseButton
      className={classes}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      {...props}
    >
      {loading && <span className={styles.spinner} aria-hidden />}
      {showStartIcon && (
        <span className={styles.icon} aria-hidden>
          {icon}
        </span>
      )}
      {children}
      {showEndIcon && (
        <span className={styles.icon} aria-hidden>
          {icon}
        </span>
      )}
    </BaseButton>
  )
}
