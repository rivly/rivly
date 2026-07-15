import { useEffect, useState } from 'react'
import { Link, useNavigate } from '@tanstack/react-router'
import CodeMirror from '@uiw/react-codemirror'
import { yaml } from '@codemirror/lang-yaml'
import { LuArrowLeft, LuGitBranch, LuPencil, LuPlus, LuUpload, LuX } from 'react-icons/lu'
import { ApiError } from '../lib/api'
import { useGitCredentials } from '../lib/gitCredentials'
import {
  fetchStackContent,
  useDeployStack,
  type DeployStackInput,
  type EnvVar,
  type StackSource,
} from '../lib/stacks'
import { toast } from '../lib/toast'
import { Button } from './Button'
import { Checkbox } from './Checkbox'
import { RequiredMark } from './RequiredMark'
import { Select } from './Select'
import styles from './StackEditor.module.css'

type Props = {
  envId: number
  name?: string
}

type Tab = 'editor' | 'upload' | 'git'

type GitFields = {
  url: string
  ref: string
  path: string
  credentialId: string
  autoUpdate: boolean
  pollInterval: string
}

const NO_CREDENTIAL = '0'
const DEFAULT_POLL = '30'

const POLL_ITEMS = [
  { label: '15 seconds', value: '15' },
  { label: '30 seconds', value: '30' },
  { label: '45 seconds', value: '45' },
  { label: '1 minute', value: '60' },
  { label: '2 minutes', value: '120' },
  { label: '5 minutes', value: '300' },
  { label: '10 minutes', value: '600' },
  { label: '30 minutes', value: '1800' },
]

function pollValue(seconds: number): string {
  const match = POLL_ITEMS.find((item) => item.value === String(seconds))
  return match ? match.value : DEFAULT_POLL
}

function snapshot(content: string, env: EnvVar[], git: GitFields | null): string {
  return JSON.stringify({ content, env, git })
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
  const [tab, setTab] = useState<Tab>('editor')
  const [stackSource, setStackSource] = useState<StackSource>('content')
  const [gitUrl, setGitUrl] = useState('')
  const [gitRef, setGitRef] = useState('')
  const [gitPath, setGitPath] = useState('docker-compose.yml')
  const [gitCredentialId, setGitCredentialId] = useState(NO_CREDENTIAL)
  const [autoUpdate, setAutoUpdate] = useState(false)
  const [pollInterval, setPollInterval] = useState(DEFAULT_POLL)
  const [initial, setInitial] = useState('')
  const [dragOver, setDragOver] = useState(false)
  const [loading, setLoading] = useState(editing)
  const [error, setError] = useState<string | null>(null)
  const deploy = useDeployStack(envId)
  const { data: credentials } = useGitCredentials()

  const fromGit = editing ? stackSource === 'git' : tab === 'git'

  const credentialItems = [
    { label: 'None, public repository', value: NO_CREDENTIAL },
    ...(credentials ?? []).map((c) => ({ label: c.name, value: String(c.id) })),
  ]

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
      setTab('editor')
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
        if (!active) {
          return
        }
        const loadedEnv = (stack.env ?? []).filter((v) => v.key.trim() !== '')
        setContent(stack.content)
        setEnv(loadedEnv)
        setStackSource(stack.source)
        let loadedGit: GitFields | null = null
        if (stack.git) {
          loadedGit = {
            url: stack.git.url,
            ref: stack.git.ref,
            path: stack.git.path,
            credentialId: String(stack.git.credentialId),
            autoUpdate: stack.git.autoUpdate,
            pollInterval: pollValue(stack.git.pollInterval),
          }
          setGitUrl(loadedGit.url)
          setGitRef(loadedGit.ref)
          setGitPath(loadedGit.path)
          setGitCredentialId(loadedGit.credentialId)
          setAutoUpdate(loadedGit.autoUpdate)
          setPollInterval(loadedGit.pollInterval)
        }
        setInitial(snapshot(stack.content, loadedEnv, loadedGit))
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

  const canDeploy = fromGit
    ? name.trim() !== '' && gitUrl.trim() !== '' && gitPath.trim() !== ''
    : name.trim() !== '' && content.trim() !== ''

  const finalEnv = advanced
    ? parseRawEnv(rawEnv)
    : env.filter((v) => v.key.trim() !== '')

  const currentGit: GitFields | null = fromGit
    ? {
        url: gitUrl.trim(),
        ref: gitRef.trim(),
        path: gitPath.trim(),
        credentialId: gitCredentialId,
        autoUpdate,
        pollInterval,
      }
    : null

  const changed = !editing || snapshot(content, finalEnv, currentGit) !== initial

  const submit = () => {
    setError(null)

    const input: DeployStackInput = fromGit
      ? {
          name: name.trim(),
          source: 'git',
          content: '',
          env: finalEnv,
          git: {
            url: gitUrl.trim(),
            ref: gitRef.trim(),
            path: gitPath.trim(),
            credentialId: Number(gitCredentialId),
            autoUpdate,
            pollInterval: Number(pollInterval),
          },
        }
      : { name: name.trim(), source: 'content', content, env: finalEnv }

    deploy.mutate(input, {
      onSuccess: () => {
        toast.success(editing ? `Redeployed ${name}` : `Deployed ${name}`)
        navigate(backTo)
      },
      onError: (err) => {
        setError(err instanceof ApiError ? err.message : 'Deployment failed')
      },
    })
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
            aria-selected={tab === 'editor'}
            className={`${styles.source} ${tab === 'editor' ? styles.sourceActive : ''}`}
            onClick={() => setTab('editor')}
          >
            <LuPencil />
            Editor
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={tab === 'upload'}
            className={`${styles.source} ${tab === 'upload' ? styles.sourceActive : ''}`}
            onClick={() => setTab('upload')}
          >
            <LuUpload />
            Upload
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={tab === 'git'}
            className={`${styles.source} ${tab === 'git' ? styles.sourceActive : ''}`}
            onClick={() => setTab('git')}
          >
            <LuGitBranch />
            Git Repository
          </button>
        </div>
      )}

      {fromGit && (
        <div className={styles.gitForm}>
          <label className={styles.gitField}>
            <span className={styles.gitLabel}>
              Repository URL
              <RequiredMark />
            </span>
            <input
              className={styles.gitInput}
              value={gitUrl}
              onChange={(e) => setGitUrl(e.target.value)}
              autoComplete="off"
              spellCheck={false}
            />
            <span className={styles.gitHint}>An http or https URL. SSH is not supported yet.</span>
          </label>

          <div className={styles.gitRow}>
            <label className={styles.gitField}>
              <span className={styles.gitLabel}>Branch or tag</span>
              <input
                className={styles.gitInput}
                value={gitRef}
                onChange={(e) => setGitRef(e.target.value)}
                autoComplete="off"
                spellCheck={false}
              />
              <span className={styles.gitHint}>Leave blank for the default branch.</span>
            </label>

            <label className={styles.gitField}>
              <span className={styles.gitLabel}>
                Compose path
                <RequiredMark />
              </span>
              <input
                className={styles.gitInput}
                value={gitPath}
                onChange={(e) => setGitPath(e.target.value)}
                autoComplete="off"
                spellCheck={false}
              />
              <span className={styles.gitHint}>Path from the repository root.</span>
            </label>
          </div>

          <div className={`${styles.gitField} ${styles.gitFieldHalf}`}>
            <span className={styles.gitLabel}>Credential</span>
            <Select
              items={credentialItems}
              size="md"
              value={gitCredentialId}
              onValueChange={(value) => setGitCredentialId(value ?? NO_CREDENTIAL)}
              aria-label="Git credential"
            />
            <span className={styles.gitHint}>
              Only needed for a private repository. Manage them in{' '}
              <Link to="/git-credentials" className={styles.gitHintLink}>
                Git Credentials
              </Link>
              .
            </span>
          </div>

          <div className={styles.autoUpdate}>
            <Checkbox
              label="Redeploy automatically on a new commit"
              checked={autoUpdate}
              onCheckedChange={(checked) => setAutoUpdate(checked === true)}
            />
            {autoUpdate && (
              <div className={`${styles.gitField} ${styles.gitFieldHalf}`}>
                <span className={styles.gitLabel}>Check every</span>
                <Select
                  items={POLL_ITEMS}
                  size="md"
                  value={pollInterval}
                  onValueChange={(value) => setPollInterval(value ?? DEFAULT_POLL)}
                  aria-label="Check interval"
                />
              </div>
            )}
          </div>
        </div>
      )}

      {tab === 'upload' && !editing ? (
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
        !fromGit && (
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
        )
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
          disabled={loading || !canDeploy || !changed}
        >
          {editing ? 'Save and redeploy' : 'Deploy'}
        </Button>
      </footer>
    </div>
  )
}
