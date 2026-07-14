import type { ComponentProps } from 'react'
import { createLink, type LinkComponent } from '@tanstack/react-router'
import { LuArrowLeft } from 'react-icons/lu'
import styles from './BackLink.module.css'

function BackLinkBase({ children, ...props }: ComponentProps<'a'>) {
  return (
    <a {...props} className={styles.link}>
      <LuArrowLeft /> {children}
    </a>
  )
}

const CreatedBackLink = createLink(BackLinkBase)

export const BackLink: LinkComponent<typeof BackLinkBase> = (props) => (
  <CreatedBackLink {...props} />
)
