import { createFileRoute, Outlet, useNavigate } from '@tanstack/react-router'
import { useEffect } from 'react'
import { Loader } from '../../../components/Loader'
import { useEnvironment } from '../../../lib/environments'
import { toast } from '../../../lib/toast'

export const Route = createFileRoute('/_app/environments/$id')({
  component: EnvironmentLayout,
})

function EnvironmentLayout() {
  const { id } = Route.useParams()
  const navigate = useNavigate()
  const { data: env, isPending } = useEnvironment(Number(id))

  useEffect(() => {
    if (env && env.status !== 'up') {
      toast.error('Environment unreachable', `${env.name} is not responding.`)
      navigate({ to: '/' })
    }
  }, [env, navigate])

  if (isPending) {
    return <Loader />
  }
  if (!env || env.status !== 'up') {
    return null
  }
  return <Outlet />
}
