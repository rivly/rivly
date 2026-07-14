import { createFileRoute } from '@tanstack/react-router'
import { useCallback, useMemo, useState } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { LuPencil, LuPlus, LuTrash2 } from 'react-icons/lu'
import { Button } from '../../components/Button'
import { ConfirmDialog } from '../../components/ConfirmDialog'
import { DataTable } from '../../components/DataTable'
import { GitCredentialDialog } from '../../components/GitCredentialDialog'
import { PageHeader } from '../../components/PageHeader'
import { QueryState } from '../../components/QueryState'
import {
  useDeleteGitCredential,
  useGitCredentials,
  type GitCredential,
} from '../../lib/gitCredentials'
import { formatDateTime } from '../../lib/format'
import { toast } from '../../lib/toast'
import styles from './git-credentials.module.css'

export const Route = createFileRoute('/_app/git-credentials')({
  head: () => ({ meta: [{ title: 'Git Credentials · Rivly' }] }),
  component: GitCredentialsPage,
})

function GitCredentialsPage() {
  const { data: credentials, isPending, isError } = useGitCredentials()
  const deletion = useDeleteGitCredential()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<GitCredential | null>(null)

  const openAdd = () => {
    setEditing(null)
    setDialogOpen(true)
  }
  const openEdit = useCallback((credential: GitCredential) => {
    setEditing(credential)
    setDialogOpen(true)
  }, [])

  const remove = useCallback(
    (credential: GitCredential) => {
      deletion.mutate(credential.id, {
        onSuccess: () => toast.success(`Removed ${credential.name}`),
        onError: () => toast.error('Could not remove', 'Please try again'),
      })
    },
    [deletion],
  )

  const columns = useMemo<ColumnDef<GitCredential>[]>(
    () => [
      {
        accessorKey: 'name',
        header: 'Name',
        cell: (cell) => cell.row.original.name,
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
                <Button
                  variant="danger"
                  size="sm"
                  iconOnly
                  icon={<LuTrash2 />}
                  aria-label="Remove credential"
                />
              }
              title={`Remove ${cell.row.original.name}?`}
              description="Stacks that rely on this credential will no longer be able to reach their repository. This cannot be undone."
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
        title="Git Credentials"
        subtitle="Credentials to clone private repositories when deploying or updating stacks from Git."
        action={
          <Button size="sm" icon={<LuPlus />} onClick={openAdd}>
            Add credential
          </Button>
        }
      />

      <QueryState
        pending={isPending}
        error={isError}
        errorMessage="Could not load git credentials."
      >
        {credentials && (
          <DataTable
            data={credentials}
            columns={columns}
            searchPlaceholder="Search credentials…"
            emptyMessage="No credentials yet. Add one to deploy stacks from a private repository."
            initialPageSize={25}
          />
        )}
      </QueryState>

      <GitCredentialDialog
        open={dialogOpen}
        editing={editing}
        onClose={() => setDialogOpen(false)}
      />
    </div>
  )
}
