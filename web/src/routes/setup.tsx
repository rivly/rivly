import { createFileRoute, redirect, useNavigate } from '@tanstack/react-router'
import { useState, type FormEvent } from 'react'
import { Form } from '@base-ui/react/form'
import { AuthLayout } from '../components/AuthLayout'
import { Field } from '../components/Field'
import { PasswordField } from '../components/PasswordField'
import { Button } from '../components/Button'
import { FormError } from '../components/FormError'
import { ApiError } from '../lib/api'
import { loadAuthState, useSetup } from '../lib/auth'

export const Route = createFileRoute('/setup')({
  beforeLoad: async ({ context }) => {
    const { needsSetup, me } = await loadAuthState(context.queryClient)
    if (me) {
      throw redirect({ to: '/' })
    }
    if (!needsSetup) {
      throw redirect({ to: '/login' })
    }
  },
  component: SetupPage,
})

function SetupPage() {
  const navigate = useNavigate()
  const setup = useSetup()
  const [error, setError] = useState('')

  function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    if (!event.currentTarget.checkValidity()) {
      return
    }
    const form = new FormData(event.currentTarget)
    setup.mutate(
      {
        email: String(form.get('email')),
        password: String(form.get('password')),
        displayName: String(form.get('displayName')),
      },
      {
        onSuccess: () => navigate({ to: '/' }),
        onError: (err) =>
          setError(
            err instanceof ApiError ? err.message : 'Something went wrong',
          ),
      },
    )
  }

  return (
    <AuthLayout title="Set up Rivly">
      <Form onSubmit={onSubmit} noValidate>
        <Field
          label="Name"
          name="displayName"
          autoComplete="name"
          placeholder="Admin"
        />
        <Field
          label="Email"
          name="email"
          type="email"
          autoComplete="email"
          placeholder="you@company.com"
          required
        />
        <PasswordField
          label="Password"
          name="password"
          autoComplete="new-password"
          minLength={8}
          required
        />
        <FormError message={error} />
        <Button type="submit" fullWidth loading={setup.isPending}>
          Create account
        </Button>
      </Form>
    </AuthLayout>
  )
}
