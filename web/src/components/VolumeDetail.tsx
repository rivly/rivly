import { useNavigate } from '@tanstack/react-router'
import { LuTrash2 } from 'react-icons/lu'
import { useContainers } from '../lib/containers'
import { useVolumeActions, useVolumeDetail } from '../lib/volumes'
import { bulkActionError } from '../lib/bulk'
import { formatDateTime } from '../lib/format'
import { toast } from '../lib/toast'
import { BackLink } from './BackLink'
import { Button } from './Button'
import { ConfirmDialog } from './ConfirmDialog'
import { ContainerMiniTable } from './ContainerMiniTable'
import { DetailHeader } from './DetailHeader'
import { Loader } from './Loader'
import styles from './resourceDetail.module.css'

export function VolumeDetail({ envId, name }: { envId: number; name: string }) {
  const navigate = useNavigate()
  const { data, isPending, isError } = useVolumeDetail(envId, name)
  const { data: containers } = useContainers(envId)
  const mutation = useVolumeActions(envId)

  const backTo = {
    to: '/environments/$id/volumes' as const,
    params: { id: String(envId) },
  }

  if (isPending) {
    return <Loader />
  }
  if (isError || !data) {
    return (
      <div>
        <BackLink {...backTo}>Volumes</BackLink>
        <p className={styles.message}>Could not load this volume.</p>
      </div>
    )
  }

  const usingIds = new Set(data.containers.map((c) => c.id))
  const usingContainers = (containers ?? []).filter((c) => usingIds.has(c.id))
  const labels = data.labels ?? {}

  const remove = () => {
    mutation.mutate(
      { action: 'remove', ids: [name] },
      {
        onSuccess: (res) => {
          if (res.results[0]?.ok) {
            toast.success(`Removed ${name}`)
            navigate(backTo)
          } else {
            toast.error('Could not remove', `${name} is in use`)
          }
        },
        onError: bulkActionError,
      },
    )
  }

  return (
    <div className={styles.page}>
      <div>
        <BackLink {...backTo}>Volumes</BackLink>
      </div>

      <DetailHeader
        title={name}
        actions={
          <ConfirmDialog
            trigger={
              <Button variant="danger" size="sm" icon={<LuTrash2 />} loading={mutation.isPending}>
                Remove
              </Button>
            }
            title={`Remove ${name}?`}
            description="This permanently removes the volume and its data. This cannot be undone."
            onConfirm={remove}
          />
        }
      />

      <div className={styles.meta}>
        <span>Driver {data.driver}</span>
        <span className={styles.dot}>·</span>
        <span>Scope {data.scope}</span>
        {data.created > 0 && (
          <>
            <span className={styles.dot}>·</span>
            <span>Created {formatDateTime(data.created)}</span>
          </>
        )}
        <span className={styles.dot}>·</span>
        <code className={styles.mono}>{data.mountpoint}</code>
      </div>

      {Object.keys(labels).length > 0 && (
        <section className={styles.section}>
          <span className={styles.sectionTitle}>Labels</span>
          <div className={styles.labels}>
            {Object.entries(labels).map(([key, value]) => (
              <div key={key} className={styles.labelRow}>
                <span className={styles.labelKey}>{key}</span>
                <span className={styles.labelValue}>{value}</span>
              </div>
            ))}
          </div>
        </section>
      )}

      <section className={styles.section}>
        <span className={styles.sectionTitle}>Containers</span>
        <ContainerMiniTable
          envId={envId}
          containers={usingContainers}
          emptyMessage="No containers use this volume."
        />
      </section>
    </div>
  )
}
