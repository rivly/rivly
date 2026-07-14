import { createFileRoute } from '@tanstack/react-router'
import { useCallback, useMemo, useState } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { LuPencil, LuPlus, LuTrash2 } from 'react-icons/lu'
import { Button } from '../../components/Button'
import { ConfirmDialog } from '../../components/ConfirmDialog'
import { DataTable } from '../../components/DataTable'
import { PageHeader } from '../../components/PageHeader'
import { QueryState } from '../../components/QueryState'
import { RegistryDialog } from '../../components/RegistryDialog'
import { useDeleteRegistry, useRegistries, type Registry } from '../../lib/registries'
import { formatDateTime } from '../../lib/format'
import { toast } from '../../lib/toast'
import styles from './registries.module.css'

export const Route = createFileRoute('/_app/registries')({
  head: () => ({ meta: [{ title: 'Registries · Rivly' }] }),
  component: RegistriesPage,
})

function RegistriesPage() {
  const { data: registries, isPending, isError } = useRegistries()
  const deletion = useDeleteRegistry()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<Registry | null>(null)

  const openAdd = () => {
    setEditing(null)
    setDialogOpen(true)
  }
  const openEdit = useCallback((registry: Registry) => {
    setEditing(registry)
    setDialogOpen(true)
  }, [])

  const remove = useCallback(
    (registry: Registry) => {
      deletion.mutate(registry.id, {
        onSuccess: () => toast.success(`Removed ${registry.server}`),
        onError: () => toast.error('Could not remove', 'Please try again'),
      })
    },
    [deletion],
  )

  const columns = useMemo<ColumnDef<Registry>[]>(
    () => [
      {
        accessorKey: 'name',
        header: 'Name',
        cell: (cell) =>
          cell.row.original.name || <span className={styles.muted}>-</span>,
      },
      {
        accessorKey: 'server',
        header: 'Registry URL',
        cell: (cell) => cell.row.original.server,
      },
      {
        accessorKey: 'username',
        header: 'Username',
        cell: (cell) => cell.row.original.username,
      },
      {
        accessorKey: 'createdAt',
        header: 'Added',
        cell: (cell) => (
          <span className={styles.muted}>
            {formatDateTime(cell.row.original.createdAt)}
            {cell.row.original.createdBy && ` by ${cell.row.original.createdBy}`}
          </span>
        ),
      },
      {
        id: 'actions',
        header: 'Actions',
        enableSorting: false,
        enableHiding: false,
        cell: (cell) => (
          <div className={styles.actions}>
            <Button
              variant="secondary"
              size="sm"
              icon={<LuPencil />}
              onClick={() => openEdit(cell.row.original)}
            >
              Edit
            </Button>
            <ConfirmDialog
              trigger={
                <Button variant="danger" size="sm" iconOnly icon={<LuTrash2 />} aria-label="Remove registry" />
              }
              title={`Remove ${cell.row.original.server}?`}
              description="Rivly will no longer authenticate to this registry. This cannot be undone."
              onConfirm={() => remove(cell.row.original)}
            />
          </div>
        ),
      },
    ],
    [openEdit, remove],
  )

  return (
    <div>
      <PageHeader
        title="Registries"
        subtitle="Credentials to pull private images. Rivly matches an image's registry automatically when pulling or running a container."
        action={
          <Button size="sm" icon={<LuPlus />} onClick={openAdd}>
            Add registry
          </Button>
        }
      />

      <QueryState pending={isPending} error={isError} errorMessage="Could not load registries.">
        {registries && (
          <DataTable
            data={registries}
            columns={columns}
            searchPlaceholder="Search registries…"
            emptyMessage="No registries yet. Add one to pull private images."
            initialPageSize={25}
          />
        )}
      </QueryState>

      <RegistryDialog open={dialogOpen} editing={editing} onClose={() => setDialogOpen(false)} />
    </div>
  )
}
