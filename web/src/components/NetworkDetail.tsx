import { Fragment } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { LuTrash2 } from 'react-icons/lu'
import { useContainers } from '../lib/containers'
import { useNetworkActions, useNetworkDetail } from '../lib/networks'
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

export function NetworkDetail({ envId, networkId }: { envId: number; networkId: string }) {
  const navigate = useNavigate()
  const { data, isPending, isError } = useNetworkDetail(envId, networkId)
  const { data: containers } = useContainers(envId)
  const mutation = useNetworkActions(envId)

  const backTo = {
    to: '/environments/$id/networks' as const,
    params: { id: String(envId) },
  }

  if (isPending) {
    return <Loader />
  }
  if (isError || !data) {
    return (
      <div>
        <BackLink {...backTo}>Networks</BackLink>
        <p className={styles.message}>Could not load this network.</p>
      </div>
    )
  }

  const attachedIds = new Set(data.containers.map((c) => c.id))
  const attachedContainers = (containers ?? []).filter((c) => attachedIds.has(c.id))
  const labels = data.labels ?? {}

  const remove = () => {
    mutation.mutate(
      { action: 'remove', ids: [networkId] },
      {
        onSuccess: (res) => {
          if (res.results[0]?.ok) {
            toast.success(`Removed ${data.name}`)
            navigate(backTo)
          } else {
            toast.error('Could not remove', `${data.name} is in use`)
          }
        },
        onError: bulkActionError,
      },
    )
  }

  return (
    <div className={styles.page}>
      <div>
        <BackLink {...backTo}>Networks</BackLink>
      </div>

      <DetailHeader
        title={data.name}
        actions={
          <ConfirmDialog
            trigger={
              <Button variant="danger" size="sm" icon={<LuTrash2 />} loading={mutation.isPending}>
                Remove
              </Button>
            }
            title={`Remove ${data.name}?`}
            description="This permanently removes the network. Containers must be detached first. This cannot be undone."
            onConfirm={remove}
          />
        }
      />

      <div className={styles.meta}>
        <span>Driver {data.driver}</span>
        <span className={styles.dot}>·</span>
        <span>Scope {data.scope}</span>
        {data.subnets.map((subnet) => (
          <Fragment key={subnet.subnet + subnet.gateway}>
            <span className={styles.dot}>·</span>
            <span>
              {subnet.subnet}
              {subnet.gateway && ` → ${subnet.gateway}`}
            </span>
          </Fragment>
        ))}
        {data.internal && (
          <>
            <span className={styles.dot}>·</span>
            <span>Internal</span>
          </>
        )}
        {data.attachable && (
          <>
            <span className={styles.dot}>·</span>
            <span>Attachable</span>
          </>
        )}
        {data.created > 0 && (
          <>
            <span className={styles.dot}>·</span>
            <span>Created {formatDateTime(data.created)}</span>
          </>
        )}
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
          containers={attachedContainers}
          emptyMessage="No containers attached to this network."
        />
      </section>
    </div>
  )
}
