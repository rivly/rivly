import { useEffect, useState } from 'react'
import { Dialog } from '@base-ui/react/dialog'
import CodeMirror from '@uiw/react-codemirror'
import { yaml } from '@codemirror/lang-yaml'
import { LuX } from 'react-icons/lu'
import { ApiError } from '../lib/api'
import { fetchStackContent, useDeployStack } from '../lib/stacks'
import { toast } from '../lib/toast'
import { Button } from './Button'
import styles from './StackEditor.module.css'

export type EditorState = { mode: 'new' } | { mode: 'edit'; name: string }

type Props = {
  envId: number
  state: EditorState | null
  onClose: () => void
}

export function StackEditor({ envId, state, onClose }: Props) {
  return (
    <Dialog.Root
      open={state !== null}
      onOpenChange={(open) => {
        if (!open) {
          onClose()
        }
      }}
    >
      <Dialog.Portal>
        <Dialog.Backdrop className={styles.backdrop} />
        <Dialog.Popup className={styles.popup}>
          {state && <EditorBody key={editorKey(state)} envId={envId} state={state} onClose={onClose} />}
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  )
}

function editorKey(state: EditorState) {
  return state.mode === 'edit' ? `edit:${state.name}` : 'new'
}

function EditorBody({ envId, state, onClose }: { envId: number; state: EditorState; onClose: () => void }) {
  const editing = state.mode === 'edit'
  const [name, setName] = useState(editing ? state.name : '')
  const [content, setContent] = useState('')
  const [loading, setLoading] = useState(editing)
  const [error, setError] = useState<string | null>(null)
  const deploy = useDeployStack(envId)

  useEffect(() => {
    if (!editing) {
      return
    }
    let active = true
    fetchStackContent(envId, state.name)
      .then((stack) => {
        if (active) {
          setContent(stack.content)
        }
      })
      .catch(() => {
        if (active) {
          setError('Could not load this stack.')
        }
      })
      .finally(() => {
        if (active) {
          setLoading(false)
        }
      })
    return () => {
      active = false
    }
  }, [editing, envId, state])

  const submit = () => {
    setError(null)
    deploy.mutate(
      { name: name.trim(), content },
      {
        onSuccess: () => {
          toast.success(editing ? `Redeployed ${name}` : `Deployed ${name}`)
          onClose()
        },
        onError: (err) => {
          setError(err instanceof ApiError ? err.message : 'Deployment failed')
        },
      },
    )
  }

  return (
    <>
      <header className={styles.header}>
        <Dialog.Title className={styles.title}>
          {editing ? `Edit ${state.name}` : 'Deploy a stack'}
        </Dialog.Title>
        <Dialog.Close
          render={<Button variant="secondary" size="sm" iconOnly icon={<LuX />} aria-label="Close" />}
        />
      </header>

      <div className={styles.body}>
        {!editing && (
          <label className={styles.nameField}>
            <span className={styles.nameLabel}>Name</span>
            <input
              className={styles.nameInput}
              value={name}
              onChange={(event) => setName(event.target.value)}
              placeholder="my-app"
              autoComplete="off"
              spellCheck={false}
            />
          </label>
        )}

        <div className={styles.editor}>
          <CodeMirror
            value={content}
            height="100%"
            theme="light"
            placeholder={'services:\n  app:\n    image: nginx\n    ports:\n      - "8080:80"'}
            extensions={[yaml()]}
            editable={!loading}
            onChange={setContent}
          />
        </div>

        {error && <pre className={styles.error}>{error}</pre>}
      </div>

      <footer className={styles.footer}>
        <Dialog.Close render={<Button variant="secondary">Cancel</Button>} />
        <Button
          onClick={submit}
          loading={deploy.isPending}
          disabled={loading || name.trim() === '' || content.trim() === ''}
        >
          {editing ? 'Redeploy' : 'Deploy'}
        </Button>
      </footer>
    </>
  )
}
