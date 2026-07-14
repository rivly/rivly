import { useMemo } from 'react'
import { Link, useNavigate } from '@tanstack/react-router'
import type { ColumnDef } from '@tanstack/react-table'
import { LuPencil } from 'react-icons/lu'
import { useContainers, type Container } from '../lib/containers'
import { useStacks } from '../lib/stacks'
import { formatDateTime, timeAgo } from '../lib/format'
import { BackLink } from './BackLink'
import { Button } from './Button'
import { ContainerStateBadge } from './ContainerStateBadge'
import { DataTable } from './DataTable'
import { DetailHeader } from './DetailHeader'
import { LimitedBadge } from './LimitedBadge'
import { Loader } from './Loader'
import { NameLink } from './NameLink'
import { PublishedPorts } from './PublishedPorts'
import { StackActionButtons } from './StackActionButtons'
import { StackStateBadge } from './StackStateBadge'
import styles from './StackDetail.module.css'

export function StackDetail({ envId, name }: { envId: number; name: string }) {
  const navigate = useNavigate()
  const { data: stacks, isPending, isError } = useStacks(envId)
  const { data: containers } = useContainers(envId)

  const backTo = {
    to: '/environments/$id/stacks' as const,
    params: { id: String(envId) },
  }

  const columns = useMemo<ColumnDef<Container>[]>(
    () => [
      {
        accessorKey: 'name',
        header: 'Name',
        cell: (cell) => (
          <NameLink
            to="/environments/$id/containers/$containerId"
            params={{ id: String(envId), containerId: cell.row.original.id }}
          >
            {cell.row.original.name}
          </NameLink>
        ),
      },
      {
        accessorKey: 'state',
        header: 'State',
        cell: (cell) => <ContainerStateBadge state={cell.row.original.state} />,
      },
      {
        accessorKey: 'image',
        header: 'Image',
        cell: (cell) => <code className={styles.image}>{cell.row.original.image}</code>,
      },
      {
        id: 'ports',
        header: 'Published ports',
        enableSorting: false,
        cell: (cell) => <PublishedPorts ports={cell.row.original.ports} />,
      },
      {
        accessorKey: 'ip',
        header: 'IP address',
        cell: (cell) => <span className={styles.muted}>{cell.row.original.ip || '-'}</span>,
      },
      {
        accessorKey: 'created',
        header: 'Created',
        cell: (cell) => <span className={styles.muted}>{timeAgo(cell.row.original.created)}</span>,
      },
    ],
    [envId],
  )

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

      <DataTable
        data={stackContainers}
        columns={columns}
        searchPlaceholder="Search containers…"
        emptyMessage="No containers in this stack."
        initialPageSize={25}
      />
    </div>
  )
}

