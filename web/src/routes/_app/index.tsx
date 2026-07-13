import { createFileRoute, Link } from '@tanstack/react-router'
import {
  LuBox,
  LuChevronRight,
  LuCpu,
  LuLayers,
  LuMemoryStick,
  LuServer,
} from 'react-icons/lu'
import { Loader } from '../../components/Loader'
import { StatusBadge } from '../../components/StatusBadge'
import type { EnvironmentDetail } from '../../lib/environments'
import { useEnvironments } from '../../lib/environments'
import { formatBytes } from '../../lib/format'
import { toast } from '../../lib/toast'
import styles from './index.module.css'

export const Route = createFileRoute('/_app/')({
  component: HomePage,
})

function HomePage() {
  const { data: environments, isPending, isError } = useEnvironments()

  return (
    <div className={styles.page}>
      <header className={styles.head}>
        <h1 className={styles.title}>Environments</h1>
        <p className={styles.subtitle}>
          Select an environment to manage its containers, images, and more.
        </p>
      </header>

      {isPending && <Loader />}
      {isError && (
        <p className={styles.state}>Could not load environments. Try again.</p>
      )}
      {environments && environments.length === 0 && (
        <p className={styles.state}>No environments yet.</p>
      )}

      {environments && environments.length > 0 && (
        <ul className={styles.list}>
          {environments.map((env) => (
            <li key={env.id}>
              <EnvironmentRow env={env} />
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

function EnvironmentRow({ env }: { env: EnvironmentDetail }) {
  const body = (
    <>
      <LuServer className={styles.rowIcon} />
      <span className={styles.info}>
        <span className={styles.topLine}>
          <span className={styles.name}>{env.name}</span>
          {env.system && (
            <span className={styles.type}>
              {env.system.swarm ? 'Swarm' : 'Standalone'}{' '}
              {env.system.serverVersion}
            </span>
          )}
          <span className={styles.url}>{env.url}</span>
        </span>
        {env.system && (
          <span className={styles.stats}>
            <span className={styles.stat}>
              <LuBox className={styles.statIcon} />
              {env.system.containers} containers
            </span>
            <span className={styles.stat}>
              <LuLayers className={styles.statIcon} />
              {env.system.images} images
            </span>
            <span className={styles.stat}>
              <LuCpu className={styles.statIcon} />
              {env.system.ncpu} CPU
            </span>
            <span className={styles.stat}>
              <LuMemoryStick className={styles.statIcon} />
              {formatBytes(env.system.memTotal)}
            </span>
            {env.system.swarm && (
              <span className={styles.stat}>
                <LuServer className={styles.statIcon} />
                {env.system.nodes} nodes
              </span>
            )}
          </span>
        )}
      </span>
      <StatusBadge status={env.status} />
      <LuChevronRight className={styles.chevron} />
    </>
  )

  if (env.status !== 'up') {
    return (
      <button
        type="button"
        className={styles.row}
        onClick={() =>
          toast.error(
            'Environment unreachable',
            `${env.name} is not responding.`,
          )
        }
      >
        {body}
      </button>
    )
  }

  return (
    <Link
      to="/environments/$id"
      params={{ id: String(env.id) }}
      className={styles.row}
    >
      {body}
    </Link>
  )
}
