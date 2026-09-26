import { createFileRoute, Link, useNavigate } from '@tanstack/react-router'
import { EmailCodeSignInForm } from '@lalternative/auth'
import { authClient } from '@/lib/auth-client'
import { AuthScreen } from '@/components/auth-screen'

export const Route = createFileRoute('/sign-in_/code')({
  validateSearch: (search: Record<string, unknown>): { next?: string } =>
    typeof search.next === 'string' ? { next: search.next } : {},
  component: CodeSignIn,
})

function CodeSignIn() {
  const navigate = useNavigate()
  const { next } = Route.useSearch()

  return (
    <AuthScreen subtitle="Connexion par code">
      <EmailCodeSignInForm
        authClient={authClient}
        linkComponent={Link}
        passwordSignInUrl="/sign-in"
        coreTokenUrl={null}
        onSuccess={() => void navigate({ to: next ?? '/' })}
      />
    </AuthScreen>
  )
}
