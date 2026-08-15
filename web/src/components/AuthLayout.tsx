import type { ReactNode } from 'react'
import { LuGlobe } from 'react-icons/lu'
import { Select } from './Select'
import logo from '../assets/logo-white.png'
import styles from './AuthLayout.module.css'

const LANGUAGES = [{ label: 'English', value: 'en' }]

type Props = {
  title: string
  children: ReactNode
}

export function AuthLayout({ title, children }: Props) {
  return (
    <div className={styles.wrap}>
      <div className={styles.glow} aria-hidden="true" />

      <div className={styles.corner}>
        <Select
          items={LANGUAGES}
          defaultValue="en"
          icon={<LuGlobe />}
          aria-label="Language"
        />
      </div>

      <div className={styles.stack}>
        <img
          className={styles.logo}
          src={logo}
          alt="Rivly"
          width={103}
          height={32}
        />
        <main className={styles.card}>
          <div className={styles.inner}>
            <h1 className={styles.title}>{title}</h1>
            {children}
          </div>
        </main>
      </div>
    </div>
  )
}
