import { createFileRoute } from '@tanstack/react-router'
import { StatusBadge } from '../../../../components/StatusBadge'
import { useEnvironment } from '../../../../lib/environments'
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
