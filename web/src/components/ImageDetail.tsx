import { useNavigate } from '@tanstack/react-router'
import { LuTrash2 } from 'react-icons/lu'
import { useContainers } from '../lib/containers'
import { useImageActions, useImageDetail } from '../lib/images'
import { bulkActionError } from '../lib/bulk'
import { formatBytes, formatDateTime } from '../lib/format'
import { toast } from '../lib/toast'
import { BackLink } from './BackLink'
import { Button } from './Button'
import { ConfirmDialog } from './ConfirmDialog'
import { ContainerMiniTable } from './ContainerMiniTable'
import { DetailHeader } from './DetailHeader'
import { Loader } from './Loader'
import { UnusedBadge } from './UnusedBadge'
import styles from './resourceDetail.module.css'

export function ImageDetail({ envId, imageId }: { envId: number; imageId: string }) {
  const navigate = useNavigate()
  const { data, isPending, isError } = useImageDetail(envId, imageId)
  const { data: containers } = useContainers(envId)
  const mutation = useImageActions(envId)

  const backTo = {
    to: '/environments/$id/images' as const,
    params: { id: String(envId) },
  }

  if (isPending) {
    return <Loader />
  }
  if (isError || !data) {
    return (
      <div>
        <BackLink {...backTo}>Images</BackLink>
        <p className={styles.message}>Could not load this image.</p>
      </div>
    )
  }

  const title = data.tags[0] ?? (data.id ? data.id.slice(0, 12) : 'Image')
  const usingIds = new Set(data.containers.map((c) => c.id))
  const usingContainers = (containers ?? []).filter((c) => usingIds.has(c.id))
  const labels = data.labels ?? {}

  const config: [string, string][] = []
  if (data.tags.length > 0) config.push(['Tags', data.tags.join(', ')])
  if (data.entrypoint && data.entrypoint.length > 0) config.push(['Entrypoint', data.entrypoint.join(' ')])
  if (data.command && data.command.length > 0) config.push(['Command', data.command.join(' ')])
  if (data.workingDir) config.push(['Working dir', data.workingDir])
  if (data.exposedPorts && data.exposedPorts.length > 0)
    config.push(['Exposed ports', data.exposedPorts.join(', ')])

  const remove = () => {
    mutation.mutate(
      { action: 'remove', ids: [imageId] },
      {
        onSuccess: (res) => {
          if (res.results[0]?.ok) {
            toast.success(`Removed ${title}`)
            navigate(backTo)
          } else {
            toast.error('Could not remove', `${title} is in use`)
          }
        },
        onError: bulkActionError,
      },
    )
  }

  return (
    <div className={styles.page}>
      <div>
        <BackLink {...backTo}>Images</BackLink>
      </div>

      <DetailHeader
        title={title}
        badges={data.containers.length === 0 ? <UnusedBadge /> : undefined}
        actions={
          <ConfirmDialog
            trigger={
              <Button variant="danger" size="sm" icon={<LuTrash2 />} loading={mutation.isPending}>
                Remove
              </Button>
            }
            title={`Remove ${title}?`}
            description="This permanently removes the image. This cannot be undone."
            onConfirm={remove}
          />
        }
      />

      <div className={styles.meta}>
        <code className={styles.mono}>{data.id.slice(0, 12)}</code>
        {data.architecture && (
          <>
            <span className={styles.dot}>·</span>
            <span>
              {data.os}/{data.architecture}
            </span>
          </>
        )}
        <span className={styles.dot}>·</span>
        <span>{formatBytes(data.size)}</span>
        {data.created > 0 && (
          <>
            <span className={styles.dot}>·</span>
            <span>Created {formatDateTime(data.created)}</span>
          </>
        )}
      </div>

      {config.length > 0 && (
        <section className={styles.section}>
          <span className={styles.sectionTitle}>Configuration</span>
          <div className={styles.labels}>
            {config.map(([key, value]) => (
              <div key={key} className={styles.labelRow}>
                <span className={styles.labelKey}>{key}</span>
                <span className={styles.labelValue}>{value}</span>
              </div>
            ))}
          </div>
        </section>
      )}

      {data.env && data.env.length > 0 && (
        <section className={styles.section}>
          <span className={styles.sectionTitle}>Environment</span>
          <div className={styles.codeList}>
            {data.env.map((line) => (
              <code key={line} className={styles.codeLine}>
                {line}
              </code>
            ))}
          </div>
        </section>
      )}

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

      {data.digests.length > 0 && (
        <section className={styles.section}>
          <span className={styles.sectionTitle}>Digests</span>
          <div className={styles.codeList}>
            {data.digests.map((digest) => (
              <code key={digest} className={styles.codeLine}>
                {digest}
              </code>
            ))}
          </div>
        </section>
      )}

      <section className={styles.section}>
        <span className={styles.sectionTitle}>Containers</span>
        <ContainerMiniTable
          envId={envId}
          containers={usingContainers}
          emptyMessage="No containers use this image."
        />
      </section>
    </div>
  )
}
