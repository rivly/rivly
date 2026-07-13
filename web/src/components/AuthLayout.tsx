import type { ReactNode } from 'react'
import { LuGlobe } from 'react-icons/lu'
import { Select } from './Select'
import logo from '../assets/logo.png'
import styles from './AuthLayout.module.css'

const LANGUAGES = [{ label: 'English', value: 'en' }]

type Props = {
  title: string
  children: ReactNode
}

export function AuthLayout({ title, children }: Props) {
  return (
    <div className={styles.wrap}>
      <div className={styles.stack}>
        <img
          className={styles.logo}
          src={logo}
          alt="Rivly"
          width={116}
          height={36}
        />
        <main className={styles.card}>
          <h1 className={styles.title}>{title}</h1>
          {children}
        </main>
        <footer className={styles.footer}>
          <Select
            items={LANGUAGES}
            defaultValue="en"
            icon={<LuGlobe />}
            aria-label="Language"
          />
        </footer>
      </div>
    </div>
  )
}
