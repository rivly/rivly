import { createFileRoute, Link, useNavigate } from '@tanstack/react-router'
import { useEffect } from 'react'
import { LuArrowLeft } from 'react-icons/lu'
import { Loader } from '../../../components/Loader'
import { StatusBadge } from '../../../components/StatusBadge'
import { useEnvironment } from '../../../lib/environments'
import { formatBytes } from '../../../lib/format'
import { toast } from '../../../lib/toast'
import styles from './$id.module.css'

export const Route = createFileRoute('/_app/environments/$id')({
  component: EnvironmentDetailPage,
})

function EnvironmentDetailPage() {
  const { id } = Route.useParams()
  const navigate = useNavigate()
  const { data: env, isPending, isError } = useEnvironment(Number(id))

  useEffect(() => {
    if (env && env.status !== 'up') {
      toast.error('Environment unreachable', `${env.name} is not responding.`)
      navigate({ to: '/' })
    }
  }, [env, navigate])

  return (
    <div className={styles.page}>
      <Link to="/" className={styles.back}>
        <LuArrowLeft />
        Environments
      </Link>

      {isPending && <Loader />}
      {isError && (
        <p className={styles.state}>Could not load this environment.</p>
      )}

      {env?.status === 'up' && env.system && (
        <>
          <header className={styles.head}>
            <div className={styles.titleRow}>
              <h1 className={styles.title}>{env.name}</h1>
              <StatusBadge status={env.status} />
            </div>
            <span className={styles.url}>{env.url}</span>
          </header>

          <div className={styles.panel}>
            <div className={styles.stats}>
              <div className={styles.stat}>
                <span className={styles.statLabel}>Containers</span>
                <span className={styles.statValue}>{env.system.containers}</span>
                <span className={styles.statHint}>
                  {env.system.containersRunning} running,{' '}
                  {env.system.containersStopped} stopped
                </span>
              </div>
              <div className={styles.stat}>
                <span className={styles.statLabel}>Images</span>
                <span className={styles.statValue}>{env.system.images}</span>
              </div>
              <div className={styles.stat}>
                <span className={styles.statLabel}>CPUs</span>
                <span className={styles.statValue}>{env.system.ncpu}</span>
              </div>
              <div className={styles.stat}>
                <span className={styles.statLabel}>Memory</span>
                <span className={styles.statValue}>
                  {formatBytes(env.system.memTotal)}
                </span>
              </div>
            </div>

            <dl className={styles.details}>
              <Detail label="Docker version" value={env.system.serverVersion} />
              <Detail label="Engine" value={env.system.name} />
              <Detail label="Operating system" value={env.system.operatingSystem} />
              <Detail
                label="Architecture"
                value={`${env.system.osType} / ${env.system.architecture}`}
              />
              <Detail label="Kernel" value={env.system.kernelVersion} />
            </dl>
          </div>
        </>
      )}
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
