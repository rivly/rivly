import { Link, useParams } from '@tanstack/react-router'
import {
  LuBox,
  LuBoxes,
  LuDatabase,
  LuHouse,
  LuLayers,
  LuLayoutDashboard,
  LuNetwork,
  LuPanelLeftClose,
  LuPanelLeftOpen,
} from 'react-icons/lu'
import mark from '../assets/mark.png'
import { useEnvironments } from '../lib/environments'
import styles from './Sidebar.module.css'

type Props = {
  open: boolean
  collapsed: boolean
  onNavigate: () => void
  onToggleCollapse: () => void
}

export function Sidebar({ open, collapsed, onNavigate, onToggleCollapse }: Props) {
  const params = useParams({ strict: false })
  const { data: environments } = useEnvironments()
  const currentEnv =
    typeof params.id === 'string'
      ? environments?.find((env) => String(env.id) === params.id)
      : undefined

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
        {currentEnv && (
          <div className={styles.section}>
            <p className={styles.sectionLabel}>
              <span className={styles.envName}>{currentEnv.name}</span>
            </p>
            <Link
              to="/environments/$id"
              params={{ id: String(currentEnv.id) }}
              activeOptions={{ exact: true }}
              className={styles.link}
              onClick={onNavigate}
              title={collapsed ? 'Overview' : undefined}
            >
              <LuLayoutDashboard className={styles.icon} />
              <span className={styles.label}>Overview</span>
            </Link>
            <Link
              to="/environments/$id/stacks"
              params={{ id: String(currentEnv.id) }}
              className={styles.link}
              onClick={onNavigate}
              title={collapsed ? 'Stacks' : undefined}
            >
              <LuBoxes className={styles.icon} />
              <span className={styles.label}>Stacks</span>
            </Link>
            <Link
              to="/environments/$id/containers"
              params={{ id: String(currentEnv.id) }}
              className={styles.link}
              onClick={onNavigate}
              title={collapsed ? 'Containers' : undefined}
            >
              <LuBox className={styles.icon} />
              <span className={styles.label}>Containers</span>
            </Link>
            <Link
              to="/environments/$id/images"
              params={{ id: String(currentEnv.id) }}
              className={styles.link}
              onClick={onNavigate}
              title={collapsed ? 'Images' : undefined}
            >
              <LuLayers className={styles.icon} />
              <span className={styles.label}>Images</span>
            </Link>
            <Link
              to="/environments/$id/volumes"
              params={{ id: String(currentEnv.id) }}
              className={styles.link}
              onClick={onNavigate}
              title={collapsed ? 'Volumes' : undefined}
            >
              <LuDatabase className={styles.icon} />
              <span className={styles.label}>Volumes</span>
            </Link>
            <Link
              to="/environments/$id/networks"
              params={{ id: String(currentEnv.id) }}
              className={styles.link}
              onClick={onNavigate}
              title={collapsed ? 'Networks' : undefined}
            >
              <LuNetwork className={styles.icon} />
              <span className={styles.label}>Networks</span>
            </Link>
          </div>
        )}

        <div className={styles.section}>
          <p className={styles.sectionLabel}>
            <span className={styles.envName}>Rivly</span>
          </p>
          <Link
            to="/"
            activeOptions={{ exact: true }}
            className={styles.link}
            onClick={onNavigate}
            title={collapsed ? 'Home' : undefined}
          >
            <LuHouse className={styles.icon} />
            <span className={styles.label}>Home</span>
          </Link>
        </div>
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
