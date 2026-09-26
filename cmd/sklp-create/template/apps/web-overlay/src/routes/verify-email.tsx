import { createFileRoute, Link, useNavigate } from '@tanstack/react-router'
import { VerifyEmailForm } from '@lalternative/auth'
import { authClient } from '@/lib/auth-client'
import { AuthScreen } from '@/components/auth-screen'

export const Route = createFileRoute('/verify-email')({
  validateSearch: (search: Record<string, unknown>) => ({
    email: typeof search.email === 'string' ? search.email : '',
  }),
  component: VerifyEmail,
})

function VerifyEmail() {
  const navigate = useNavigate()
  const { email } = Route.useSearch()

  return (
    <AuthScreen subtitle={email ? `Un code a été envoyé à ${email}` : undefined}>
      <VerifyEmailForm
        authClient={authClient}
        email={email}
        linkComponent={Link}
        loginUrl="/sign-in"
        onSuccess={() => navigate({ to: '/' })}
      />
    </AuthScreen>
  )
}
