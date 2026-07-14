import { useEffect, useState } from 'react'
import { Link, useNavigate } from '@tanstack/react-router'
import CodeMirror from '@uiw/react-codemirror'
import { yaml } from '@codemirror/lang-yaml'
import { LuArrowLeft, LuGitBranch, LuPencil, LuPlus, LuUpload, LuX } from 'react-icons/lu'
import { ApiError } from '../lib/api'
import { fetchStackContent, useDeployStack, type EnvVar } from '../lib/stacks'
import { toast } from '../lib/toast'
import { Button } from './Button'
import { RequiredMark } from './RequiredMark'
import styles from './StackEditor.module.css'

type Props = {
  envId: number
  name?: string
}

function toRawEnv(vars: EnvVar[]): string {
  return vars
    .filter((v) => v.key.trim() !== '')
    .map((v) => `${v.key}=${v.value}`)
    .join('\n')
}

function parseRawEnv(raw: string): EnvVar[] {
  const out: EnvVar[] = []
  for (const line of raw.split('\n')) {
    const trimmed = line.trim()
    if (trimmed === '' || trimmed.startsWith('#')) {
      continue
    }
    const eq = trimmed.indexOf('=')
    const key = (eq === -1 ? trimmed : trimmed.slice(0, eq)).trim()
    if (key === '') {
      continue
    }
    out.push({ key, value: eq === -1 ? '' : trimmed.slice(eq + 1) })
  }
  return out
}

export function StackEditor({ envId, name: editName }: Props) {
  const editing = editName !== undefined
  const navigate = useNavigate()
  const [name, setName] = useState(editName ?? '')
  const [content, setContent] = useState('')
  const [env, setEnv] = useState<EnvVar[]>([])
  const [rawEnv, setRawEnv] = useState('')
  const [advanced, setAdvanced] = useState(false)
  const [source, setSource] = useState<'editor' | 'upload'>('editor')
  const [dragOver, setDragOver] = useState(false)
  const [loading, setLoading] = useState(editing)
  const [error, setError] = useState<string | null>(null)
  const deploy = useDeployStack(envId)

  const loadFile = async (file: File | undefined) => {
    if (!file) {
      return
    }
    const lower = file.name.toLowerCase()
    if (!lower.endsWith('.yml') && !lower.endsWith('.yaml')) {
      toast.error('Unsupported file', 'Choose a .yml or .yaml file')
      return
    }
    try {
      setContent(await file.text())
      setSource('editor')
      toast.success(`Loaded ${file.name}`)
    } catch {
      toast.error('Could not read the file', 'Please try again')
    }
  }

  const addVar = () => setEnv((prev) => [...prev, { key: '', value: '' }])
  const removeVar = (index: number) =>
    setEnv((prev) => prev.filter((_, i) => i !== index))
  const updateVar = (index: number, field: keyof EnvVar, value: string) =>
    setEnv((prev) => prev.map((v, i) => (i === index ? { ...v, [field]: value } : v)))

  const toggleAdvanced = () => {
    if (advanced) {
      setEnv(parseRawEnv(rawEnv))
    } else {
      setRawEnv(toRawEnv(env))
    }
    setAdvanced((value) => !value)
  }

  const backTo = {
    to: '/environments/$id/stacks' as const,
    params: { id: String(envId) },
  }

  useEffect(() => {
    if (!editing) {
      return
    }
    let active = true
    fetchStackContent(envId, editName)
      .then((stack) => {
        if (active) {
          setContent(stack.content)
          setEnv(stack.env ?? [])
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
  }, [editing, envId, editName])

  const submit = () => {
    setError(null)
    const finalEnv = advanced
      ? parseRawEnv(rawEnv)
      : env.filter((v) => v.key.trim() !== '')
    deploy.mutate(
      { name: name.trim(), content, env: finalEnv },
      {
        onSuccess: () => {
          toast.success(editing ? `Redeployed ${name}` : `Deployed ${name}`)
          navigate(backTo)
        },
        onError: (err) => {
          setError(err instanceof ApiError ? err.message : 'Deployment failed')
        },
      },
    )
  }

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <Link {...backTo} className={styles.back} aria-label="Back to stacks">
          <LuArrowLeft />
        </Link>
        <h1 className={styles.title}>{editing ? `Edit ${editName}` : 'Deploy a stack'}</h1>
      </header>

      {!editing && (
        <label className={styles.nameField}>
          <span className={styles.nameLabel}>
            Name
            <RequiredMark />
          </span>
          <input
            className={styles.nameInput}
            value={name}
            onChange={(event) => setName(event.target.value)}
            placeholder="my-app"
            autoComplete="off"
            spellCheck={false}
            required
          />
        </label>
      )}

      {!editing && (
        <div className={styles.sources} role="tablist" aria-label="Compose source">
          <button
            type="button"
            role="tab"
            aria-selected={source === 'editor'}
            className={`${styles.source} ${source === 'editor' ? styles.sourceActive : ''}`}
            onClick={() => setSource('editor')}
          >
            <LuPencil />
            Editor
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={source === 'upload'}
            className={`${styles.source} ${source === 'upload' ? styles.sourceActive : ''}`}
            onClick={() => setSource('upload')}
          >
            <LuUpload />
            Upload
          </button>
          <button type="button" className={styles.source} disabled>
            <LuGitBranch />
            Git Repository
            <span className={styles.soon}>Soon</span>
          </button>
        </div>
      )}

      {source === 'upload' ? (
        <label
          className={`${styles.dropzone} ${dragOver ? styles.dropzoneActive : ''}`}
          onDragOver={(event) => {
            event.preventDefault()
            setDragOver(true)
          }}
          onDragLeave={() => setDragOver(false)}
          onDrop={(event) => {
            event.preventDefault()
            setDragOver(false)
            loadFile(event.dataTransfer.files?.[0])
          }}
        >
          <input
            type="file"
            accept=".yml,.yaml"
            className={styles.fileInput}
            onChange={(event) => {
              const file = event.target.files?.[0]
              event.target.value = ''
              loadFile(file)
            }}
          />
          <LuUpload className={styles.dropIcon} />
          <span className={styles.dropText}>Drop a compose file here, or click to browse</span>
          <span className={styles.dropHint}>.yml or .yaml</span>
        </label>
      ) : (
        <div className={styles.editor}>
          <CodeMirror
            value={content}
            height="360px"
            theme="light"
            extensions={[yaml()]}
            editable={!loading}
            onChange={setContent}
          />
        </div>
      )}

      <section className={styles.envSection}>
        <div className={styles.envHead}>
          <span className={styles.envTitle}>Environment variables</span>
          <button type="button" className={styles.envToggle} onClick={toggleAdvanced}>
            {advanced ? 'Simple editor' : 'Advanced'}
          </button>
        </div>

        {advanced ? (
          <textarea
            className={styles.envTextarea}
            value={rawEnv}
            onChange={(e) => setRawEnv(e.target.value)}
            spellCheck={false}
          />
        ) : (
          <>
            {env.length > 0 && (
              <div className={styles.envRows}>
                {env.map((variable, index) => (
                  <div key={index} className={styles.envRow}>
                    <input
                      className={styles.envKey}
                      value={variable.key}
                      onChange={(e) => updateVar(index, 'key', e.target.value)}
                      placeholder="KEY"
                      autoComplete="off"
                      spellCheck={false}
                    />
                    <input
                      className={styles.envValue}
                      value={variable.value}
                      onChange={(e) => updateVar(index, 'value', e.target.value)}
                      placeholder="value"
                      autoComplete="off"
                      spellCheck={false}
                    />
                    <button
                      type="button"
                      className={styles.envRemove}
                      onClick={() => removeVar(index)}
                      aria-label="Remove variable"
                    >
                      <LuX />
                    </button>
                  </div>
                ))}
              </div>
            )}
            <Button variant="secondary" size="sm" icon={<LuPlus />} onClick={addVar}>
              Add variable
            </Button>
          </>
        )}
      </section>

      {error && <pre className={styles.error}>{error}</pre>}

      <footer className={styles.footer}>
        <Button variant="secondary" render={<Link {...backTo} />}>
          Cancel
        </Button>
        <Button
          onClick={submit}
          loading={deploy.isPending}
          disabled={loading || name.trim() === '' || content.trim() === ''}
        >
          {editing ? 'Redeploy' : 'Deploy'}
        </Button>
      </footer>
    </div>
  )
}
