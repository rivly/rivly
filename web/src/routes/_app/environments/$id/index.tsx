import { createFileRoute } from '@tanstack/react-router'
import { StatusBadge } from '../../../../components/StatusBadge'
import { useEnvironment, type SystemInfo } from '../../../../lib/environments'
import { formatBytes } from '../../../../lib/format'
import styles from './index.module.css'

export const Route = createFileRoute('/_app/environments/$id/')({
  component: OverviewPage,
})

function OverviewPage() {
  const { id } = Route.useParams()
  const { data: env } = useEnvironment(Number(id))

  if (!env?.system) {
    return null
  }

  return (
    <div className={styles.page}>
      <header className={styles.head}>
        <div className={styles.titleRow}>
          <h1 className={styles.title}>{env.name}</h1>
          <StatusBadge status={env.status} />
        </div>
        <span className={styles.url}>{env.url}</span>
      </header>

      <div className={styles.content}>
        <ContainersCard system={env.system} />

        <div className={styles.metrics}>
          <Metric label="Images" value={env.system.images} />
          <Metric label="CPUs" value={env.system.ncpu} />
          <Metric label="Memory" value={formatBytes(env.system.memTotal)} />
          {env.system.swarm && <Metric label="Nodes" value={env.system.nodes} />}
        </div>

        <div className={styles.card}>
          <h2 className={styles.cardTitle}>System</h2>
          <dl className={styles.details}>
            <Detail label="Docker version" value={env.system.serverVersion} />
            <Detail label="Mode" value={env.system.swarm ? 'Swarm' : 'Standalone'} />
            <Detail label="Engine" value={env.system.name} />
            <Detail label="Operating system" value={env.system.operatingSystem} />
            <Detail
              label="Architecture"
              value={`${env.system.osType} / ${env.system.architecture}`}
            />
            <Detail label="Kernel" value={env.system.kernelVersion} />
          </dl>
        </div>
      </div>
    </div>
  )
}

function ContainersCard({ system }: { system: SystemInfo }) {
  const total = system.containers
  const width = (n: number) => (total > 0 ? `${(n / total) * 100}%` : '0%')

  return (
    <div className={styles.card}>
      <div className={styles.containersHead}>
        <h2 className={styles.cardTitle}>Containers</h2>
        <span className={styles.total}>{total}</span>
      </div>

      <div className={styles.bar}>
        {total === 0 ? (
          <span className={styles.barEmpty} />
        ) : (
          <>
            {system.containersRunning > 0 && (
              <span
                className={styles.barRunning}
                style={{ width: width(system.containersRunning) }}
              />
            )}
            {system.containersPaused > 0 && (
              <span
                className={styles.barPaused}
                style={{ width: width(system.containersPaused) }}
              />
            )}
            {system.containersStopped > 0 && (
              <span
                className={styles.barStopped}
                style={{ width: width(system.containersStopped) }}
              />
            )}
          </>
        )}
      </div>

      <div className={styles.legend}>
        <span className={styles.legendItem}>
          <span className={`${styles.dot} ${styles.dotRunning}`} />
          {system.containersRunning} running
        </span>
        {system.containersPaused > 0 && (
          <span className={styles.legendItem}>
            <span className={`${styles.dot} ${styles.dotPaused}`} />
            {system.containersPaused} paused
          </span>
        )}
        <span className={styles.legendItem}>
          <span className={`${styles.dot} ${styles.dotStopped}`} />
          {system.containersStopped} stopped
        </span>
      </div>
    </div>
  )
}

function Metric({ label, value }: { label: string; value: number | string }) {
  return (
    <div className={styles.metric}>
      <span className={styles.metricLabel}>{label}</span>
      <span className={styles.metricValue}>{value}</span>
    </div>
  )
}

function Detail({ label, value }: { label: string; value: string }) {
  return (
    <div className={styles.detail}>
      <dt className={styles.detailLabel}>{label}</dt>
      <dd className={styles.detailValue}>{value}</dd>
    </div>
  )
}
