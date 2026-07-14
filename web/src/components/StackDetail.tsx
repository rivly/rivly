import { Link, useNavigate } from '@tanstack/react-router'
import { LuPencil } from 'react-icons/lu'
import { useContainers } from '../lib/containers'
import { useStacks } from '../lib/stacks'
import { formatDateTime } from '../lib/format'
import { BackLink } from './BackLink'
import { Button } from './Button'
import { ContainerMiniTable } from './ContainerMiniTable'
import { DetailHeader } from './DetailHeader'
import { LimitedBadge } from './LimitedBadge'
import { Loader } from './Loader'
import { StackActionButtons } from './StackActionButtons'
import { StackStateBadge } from './StackStateBadge'
import styles from './resourceDetail.module.css'

export function StackDetail({ envId, name }: { envId: number; name: string }) {
  const navigate = useNavigate()
  const { data: stacks, isPending, isError } = useStacks(envId)
  const { data: containers } = useContainers(envId)

  const backTo = {
    to: '/environments/$id/stacks' as const,
    params: { id: String(envId) },
  }

  if (isPending) {
    return <Loader />
  }

  const stack = stacks?.find((s) => s.name === name)
  if (isError || !stack) {
    return (
      <div>
        <BackLink {...backTo}>Stacks</BackLink>
        <p className={styles.message}>Could not find this stack.</p>
      </div>
    )
  }

  const managed = stack.type === 'rivly'
  const stackContainers = (containers ?? []).filter((c) => c.stack === name)

  return (
    <div className={styles.page}>
      <div>
        <BackLink {...backTo}>Stacks</BackLink>
      </div>

      <DetailHeader
        title={name}
        badges={
          <>
            {!managed && <LimitedBadge />}
            <StackStateBadge state={stack.state} />
          </>
        }
        actions={
          <>
            {managed && (
              <Button
                variant="secondary"
                size="sm"
                icon={<LuPencil />}
                render={
                  <Link
                    to="/environments/$id/stacks/$name/edit"
                    params={{ id: String(envId), name }}
                  />
                }
              >
                Edit stack
              </Button>
            )}
            <StackActionButtons
              envId={envId}
              items={[stack]}
              onDone={(action) => {
                if (action === 'remove') {
                  navigate(backTo)
                }
              }}
            />
          </>
        }
      />

      {(stack.createdAt > 0 || stack.updatedAt > 0) && (
        <div className={styles.meta}>
          {stack.createdAt > 0 && (
            <span>
              Created {formatDateTime(stack.createdAt)}
              {stack.createdBy && ` by ${stack.createdBy}`}
            </span>
          )}
          {stack.createdAt > 0 && stack.updatedAt > 0 && <span className={styles.dot}>·</span>}
          {stack.updatedAt > 0 && (
            <span>
              Updated {formatDateTime(stack.updatedAt)}
              {stack.updatedBy && ` by ${stack.updatedBy}`}
            </span>
          )}
        </div>
      )}

      <section className={styles.section}>
        <span className={styles.sectionTitle}>Containers</span>
        <ContainerMiniTable
          envId={envId}
          containers={stackContainers}
          emptyMessage="No containers in this stack."
        />
      </section>
    </div>
  )
}
