import type { ComponentProps } from 'react'
import { createLink, type LinkComponent } from '@tanstack/react-router'
import styles from './NameLink.module.css'

function NameLinkBase({ children, ...props }: ComponentProps<'a'>) {
  return (
    <a {...props} className={styles.link}>
      {children}
    </a>
  )
}

const CreatedNameLink = createLink(NameLinkBase)

export const NameLink: LinkComponent<typeof NameLinkBase> = (props) => (
  <CreatedNameLink {...props} />
)
