import { createFileRoute } from '@tanstack/react-router'
import { useMemo, useState } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { LuDownload, LuTrash2 } from 'react-icons/lu'
import { Button } from '../../../../components/Button'
import { ConfirmDialog } from '../../../../components/ConfirmDialog'
import { DataTable } from '../../../../components/DataTable'
import { ImageBulkBar } from '../../../../components/ImageBulkBar'
import { NameCell } from '../../../../components/NameCell'
import { PageHeader } from '../../../../components/PageHeader'
import { PullImageDialog } from '../../../../components/PullImageDialog'
import { QueryState } from '../../../../components/QueryState'
import { useImages, useImagePrune, type Image } from '../../../../lib/images'
import { formatBytes, timeAgo } from '../../../../lib/format'
import { toast } from '../../../../lib/toast'
import styles from './images.module.css'

export const Route = createFileRoute('/_app/environments/$id/images')({
  head: () => ({ meta: [{ title: 'Images · Rivly' }] }),
  component: ImagesPage,
})

function ImagesPage() {
  const { id } = Route.useParams()
  const { data: images, isPending, isError } = useImages(Number(id))
  const [pullOpen, setPullOpen] = useState(false)

  const columns = useMemo<ColumnDef<Image>[]>(
    () => [
      {
        id: 'image',
        header: 'Image',
        accessorFn: (row) => row.tags.join(' '),
        cell: (cell) => <ImageCell image={cell.row.original} />,
      },
      {
        accessorKey: 'id',
        header: 'Image ID',
        cell: (cell) => (
          <code className={styles.id}>{cell.row.original.id.slice(0, 12)}</code>
        ),
      },
      {
        accessorKey: 'size',
        header: 'Size',
        cell: (cell) => (
          <span className={styles.muted}>{formatBytes(cell.row.original.size)}</span>
        ),
      },
      {
        accessorKey: 'created',
        header: 'Created',
        cell: (cell) => (
          <span className={styles.muted}>{timeAgo(cell.row.original.created)}</span>
        ),
      },
    ],
    [],
  )

  return (
    <div>
      <PageHeader
        title="Images"
        action={
          <>
            <PruneButton envId={Number(id)} />
            <Button size="sm" icon={<LuDownload />} onClick={() => setPullOpen(true)}>
              Pull image
            </Button>
          </>
        }
      />

      <QueryState pending={isPending} error={isError} errorMessage="Could not load images.">
        {images && (
          <DataTable
            data={images}
            columns={columns}
            searchPlaceholder="Search images…"
            emptyMessage="No images on this host."
            initialPageSize={25}
            enableSelection
            getRowId={(image) => image.id}
            renderBulkActions={(selected, clear) => (
              <ImageBulkBar envId={Number(id)} selected={selected} clear={clear} />
            )}
          />
        )}
      </QueryState>

      <PullImageDialog envId={Number(id)} open={pullOpen} onClose={() => setPullOpen(false)} />
    </div>
  )
}

function PruneButton({ envId }: { envId: number }) {
  const mutation = useImagePrune(envId)

  const prune = () => {
    mutation.mutate(true, {
      onSuccess: (data) => {
        if (data.imagesDeleted === 0 && data.spaceReclaimed === 0) {
          toast.info('Nothing to prune', 'No unused images found.')
        } else {
          toast.success(
            data.imagesDeleted > 0
              ? `Removed ${data.imagesDeleted} image${data.imagesDeleted > 1 ? 's' : ''}`
              : 'Pruned unused images',
            `Reclaimed ${formatBytes(data.spaceReclaimed)}`,
          )
        }
      },
      onError: () => toast.error('Prune failed', 'Could not reach the environment'),
    })
  }

  return (
    <ConfirmDialog
      trigger={
        <Button variant="danger" size="sm" icon={<LuTrash2 />} loading={mutation.isPending}>
          Prune
        </Button>
      }
      title="Prune unused images?"
      description="This removes every image not used by a container. This cannot be undone."
      confirmLabel="Prune"
      onConfirm={prune}
    />
  )
}

function ImageCell({ image }: { image: Image }) {
  return (
    <NameCell inUse={image.inUse}>
      {image.tags.length > 0 ? (
        image.tags.join(', ')
      ) : (
        <span className={styles.untagged}>{'<none>'}</span>
      )}
    </NameCell>
  )
}
