import { useState, type ReactNode } from 'react'
import { useServerEvents } from '../lib/events'
import { Sidebar } from './Sidebar'
import { Topbar } from './Topbar'
import styles from './AppShell.module.css'

const COLLAPSE_KEY = 'rivly:sidebar-collapsed'

type Props = {
  children: ReactNode
}

export function AppShell({ children }: Props) {
  useServerEvents()

  const [menuOpen, setMenuOpen] = useState(false)
  const [collapsed, setCollapsed] = useState(
    () => localStorage.getItem(COLLAPSE_KEY) === '1',
  )

  function toggleCollapsed() {
    setCollapsed((value) => {
      const next = !value
      localStorage.setItem(COLLAPSE_KEY, next ? '1' : '0')
      return next
    })
  }

  return (
    <div className={styles.shell}>
      <Sidebar
        open={menuOpen}
        collapsed={collapsed}
        onNavigate={() => setMenuOpen(false)}
        onToggleCollapse={toggleCollapsed}
      />
      {menuOpen && (
        <button
          type="button"
          className={styles.backdrop}
          aria-label="Close menu"
          onClick={() => setMenuOpen(false)}
        />
      )}
      <div className={styles.body}>
        <Topbar onMenuToggle={() => setMenuOpen((open) => !open)} />
        <main className={styles.main}>{children}</main>
      </div>
    </div>
  )
}
