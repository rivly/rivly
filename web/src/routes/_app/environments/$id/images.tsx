import { createFileRoute } from '@tanstack/react-router'
import { useMemo, useState } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { AlertDialog } from '@base-ui/react/alert-dialog'
import { LuDownload, LuTrash2 } from 'react-icons/lu'
import { Button } from '../../../../components/Button'
import { DataTable } from '../../../../components/DataTable'
import { ImageBulkBar } from '../../../../components/ImageBulkBar'
import { Loader } from '../../../../components/Loader'
import { PullImageDialog } from '../../../../components/PullImageDialog'
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
      <header className={styles.head}>
        <h1 className={styles.title}>Images</h1>
        <div className={styles.headActions}>
          <PruneButton envId={Number(id)} />
          <Button size="sm" icon={<LuDownload />} onClick={() => setPullOpen(true)}>
            Pull image
          </Button>
        </div>
      </header>

      {isPending && <Loader />}
      {isError && <p className={styles.message}>Could not load images.</p>}
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

      <PullImageDialog envId={Number(id)} open={pullOpen} onClose={() => setPullOpen(false)} />
    </div>
  )
}

function PruneButton({ envId }: { envId: number }) {
  const mutation = useImagePrune(envId)
  const [open, setOpen] = useState(false)

  const prune = () => {
    setOpen(false)
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
    <AlertDialog.Root open={open} onOpenChange={setOpen}>
      <AlertDialog.Trigger
        render={
          <Button variant="danger" size="sm" icon={<LuTrash2 />} loading={mutation.isPending}>
            Prune
          </Button>
        }
      />
      <AlertDialog.Portal>
        <AlertDialog.Backdrop className={styles.backdrop} />
        <AlertDialog.Popup className={styles.dialog}>
          <AlertDialog.Title className={styles.dialogTitle}>Prune unused images?</AlertDialog.Title>
          <AlertDialog.Description className={styles.dialogText}>
            This removes every image not used by a container. This cannot be undone.
          </AlertDialog.Description>
          <div className={styles.dialogActions}>
            <AlertDialog.Close render={<Button variant="secondary" size="sm">Cancel</Button>} />
            <Button variant="danger" size="sm" onClick={prune}>
              Prune
            </Button>
          </div>
        </AlertDialog.Popup>
      </AlertDialog.Portal>
    </AlertDialog.Root>
  )
}

function ImageCell({ image }: { image: Image }) {
  return (
    <span className={styles.imageCell}>
      {image.tags.length > 0 ? (
        <span className={styles.name}>{image.tags.join(', ')}</span>
      ) : (
        <span className={styles.untagged}>{'<none>'}</span>
      )}
      {!image.inUse && <span className={styles.unusedBadge}>Unused</span>}
    </span>
  )
}
