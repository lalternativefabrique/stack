import { createFileRoute, Link, useNavigate } from '@tanstack/react-router'
import { LoginForm } from '@lalternative/auth'
import { authClient } from '@/lib/auth-client'
import { AuthScreen } from '@/components/auth-screen'

export const Route = createFileRoute('/sign-in')({
  component: SignIn,
  validateSearch: (search: Record<string, unknown>): { next?: string } =>
    typeof search.next === 'string' ? { next: search.next } : {},
})

function SignIn() {
  const navigate = useNavigate()
  const { next } = Route.useSearch()

  return (
    <AuthScreen
      footer={
        <Link to="/sign-in/code" className="underline underline-offset-4">
          Recevoir un code par e-mail
        </Link>
      }
    >
      <LoginForm
        authClient={authClient}
        linkComponent={Link}
        registerUrl="/register"
        forgotPasswordUrl="/forgot-password"
        coreTokenUrl={null}
        onSuccess={() => navigate({ to: next ?? '/' })}
        onEmailNotVerified={(email) => navigate({ to: '/verify-email', search: { email } })}
      />
    </AuthScreen>
  )
}
