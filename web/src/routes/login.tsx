import { createFileRoute, Link, redirect, useNavigate } from '@tanstack/react-router'
import { useState, type FormEvent } from 'react'
import { Form } from '@base-ui/react/form'
import { AuthLayout } from '../components/AuthLayout'
import { Field } from '../components/Field'
import { PasswordField } from '../components/PasswordField'
import { Checkbox } from '../components/Checkbox'
import { Button } from '../components/Button'
import { FormError } from '../components/FormError'
import { ApiError } from '../lib/api'
import { loadAuthState, useLogin } from '../lib/auth'
import styles from './login.module.css'

export const Route = createFileRoute('/login')({
  head: () => ({ meta: [{ title: 'Sign in · Rivly' }] }),
  beforeLoad: async ({ context }) => {
    const { needsSetup, me } = await loadAuthState(context.queryClient)
    if (me) {
      throw redirect({ to: '/' })
    }
    if (needsSetup) {
      throw redirect({ to: '/setup' })
    }
  },
  component: LoginPage,
})

function LoginPage() {
  const navigate = useNavigate()
  const login = useLogin()
  const [error, setError] = useState('')

  function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    if (!event.currentTarget.checkValidity()) {
      return
    }
    const form = new FormData(event.currentTarget)
    login.mutate(
      {
        email: String(form.get('email')),
        password: String(form.get('password')),
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
    <AuthLayout title="Sign in to Rivly">
      <Form onSubmit={onSubmit} noValidate>
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
          autoComplete="current-password"
          required
          action={
            <Link to="/forgot-password" className={styles.forgot}>
              Forgot password?
            </Link>
          }
        />
        <div className={styles.remember}>
          <Checkbox name="remember" label="Remember me" />
        </div>
        <FormError message={error} />
        <Button type="submit" fullWidth loading={login.isPending}>
          Sign in
        </Button>
      </Form>
    </AuthLayout>
  )
}
