import { Link, type LinkProps } from '@tanstack/react-router'
import type { IconType } from 'react-icons'
import { LuHouse, LuPanelLeftClose, LuPanelLeftOpen } from 'react-icons/lu'
import mark from '../assets/mark.png'
import styles from './Sidebar.module.css'

type NavItem = {
  to: LinkProps['to']
  label: string
  icon: IconType
  exact?: boolean
}

type NavSection = {
  label?: string
  items: NavItem[]
}

const SECTIONS: NavSection[] = [
  {
    items: [{ to: '/', label: 'Home', icon: LuHouse, exact: true }],
  },
]

type Props = {
  open: boolean
  collapsed: boolean
  onNavigate: () => void
  onToggleCollapse: () => void
}

export function Sidebar({ open, collapsed, onNavigate, onToggleCollapse }: Props) {
  return (
    <aside
      className={[
        styles.sidebar,
        open ? styles.open : '',
        collapsed ? styles.collapsed : '',
      ].join(' ')}
    >
      <div className={styles.header}>
        <img className={styles.mark} src={mark} alt="" width={22} height={22} />
        <span className={styles.wordmark}>Rivly</span>
      </div>
      <nav className={styles.nav}>
        {SECTIONS.map((section) => (
          <div key={section.items[0].to} className={styles.section}>
            {section.label && (
              <p className={styles.sectionLabel}>{section.label}</p>
            )}
            {section.items.map(({ to, label, icon: Icon, exact }) => (
              <Link
                key={to}
                to={to}
                activeOptions={{ exact }}
                className={styles.link}
                onClick={onNavigate}
                title={collapsed ? label : undefined}
              >
                <Icon className={styles.icon} />
                <span className={styles.label}>{label}</span>
              </Link>
            ))}
          </div>
        ))}
      </nav>
      <div className={styles.footer}>
        <button
          type="button"
          className={styles.toggle}
          onClick={onToggleCollapse}
          aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
        >
          {collapsed ? <LuPanelLeftOpen /> : <LuPanelLeftClose />}
        </button>
      </div>
    </aside>
  )
}
