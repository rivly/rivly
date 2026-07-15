import { Link, useNavigate } from '@tanstack/react-router'
import { LuCloudDownload, LuPencil } from 'react-icons/lu'
import { ApiError } from '../lib/api'
import { useContainers } from '../lib/containers'
import {
  pollLabel,
  useDeployStack,
  useStackDetail,
  useStacks,
  type DeployStackInput,
} from '../lib/stacks'
import { formatDateTime } from '../lib/format'
import { toast } from '../lib/toast'
import { BackLink } from './BackLink'
import { Button } from './Button'
import { ContainerMiniTable } from './ContainerMiniTable'
import { DetailHeader } from './DetailHeader'
import { LimitedBadge } from './LimitedBadge'
import { Loader } from './Loader'
import { StackActionButtons, StackDeleteButton } from './StackActionButtons'
import { StackStateBadge } from './StackStateBadge'
import styles from './resourceDetail.module.css'

export function StackDetail({ envId, name }: { envId: number; name: string }) {
  const navigate = useNavigate()
  const { data: stacks, isPending, isError } = useStacks(envId)
  const { data: containers } = useContainers(envId)
  const isManaged = stacks?.find((s) => s.name === name)?.type === 'rivly'
  const { data: detail } = useStackDetail(envId, name, Boolean(isManaged))
  const redeploy = useDeployStack(envId)

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
  const git = detail?.git

  const onRedeploy = () => {
    if (!detail) {
      return
    }
    const input: DeployStackInput =
      detail.source === 'git' && detail.git
        ? {
            name,
            source: 'git',
            content: '',
            env: detail.env,
            git: {
              url: detail.git.url,
              ref: detail.git.ref,
              path: detail.git.path,
              credentialId: detail.git.credentialId,
              autoUpdate: detail.git.autoUpdate,
              pollInterval: detail.git.pollInterval,
            },
          }
        : { name, source: 'content', content: detail.content, env: detail.env }

    redeploy.mutate(input, {
      onSuccess: () => toast.success(`Redeployed ${name}`),
      onError: (err) =>
        toast.error(
          'Redeploy failed',
          err instanceof ApiError ? err.message : 'Please try again',
        ),
    })
  }

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
                icon={<LuCloudDownload />}
                loading={redeploy.isPending}
                disabled={!detail}
                onClick={onRedeploy}
              >
                Redeploy
              </Button>
            )}
            <StackActionButtons envId={envId} items={[stack]} />
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
            <StackDeleteButton
              envId={envId}
              items={[stack]}
              onDone={() => navigate(backTo)}
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

      {git && (
        <section className={styles.section}>
          <span className={styles.sectionTitle}>Git</span>
          <div className={styles.labels}>
            <div className={styles.labelRow}>
              <span className={styles.labelKey}>Repository</span>
              <a
                className={styles.labelValue}
                href={git.url}
                target="_blank"
                rel="noreferrer"
              >
                {git.url}
              </a>
            </div>
            <div className={styles.labelRow}>
              <span className={styles.labelKey}>Reference</span>
              <span className={styles.labelValue}>{git.ref || 'default branch'}</span>
            </div>
            <div className={styles.labelRow}>
              <span className={styles.labelKey}>Compose path</span>
              <span className={styles.labelValue}>{git.path}</span>
            </div>
            {git.commit && (
              <div className={styles.labelRow}>
                <span className={styles.labelKey}>Deployed commit</span>
                <span className={styles.labelValue}>{git.commit.slice(0, 12)}</span>
              </div>
            )}
            <div className={styles.labelRow}>
              <span className={styles.labelKey}>Automatic updates</span>
              <span className={styles.labelValue}>
                {git.autoUpdate ? `every ${pollLabel(git.pollInterval)}` : 'off'}
              </span>
            </div>
            {git.autoUpdate && git.lastCheckedAt > 0 && (
              <div className={styles.labelRow}>
                <span className={styles.labelKey}>Last checked</span>
                <span className={styles.labelValue}>{formatDateTime(git.lastCheckedAt)}</span>
              </div>
            )}
            {git.lastError && (
              <div className={styles.labelRow}>
                <span className={styles.labelKey}>Last error</span>
                <span className={styles.gitError}>{git.lastError}</span>
              </div>
            )}
          </div>
        </section>
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
