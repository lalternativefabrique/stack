import { createFileRoute, Link, redirect } from '@tanstack/react-router'
import { AccountSettings } from '@lalternative/auth'
import { authClient } from '@/lib/auth-client'
import { getServerSession } from '@/lib/get-server-session'

export const Route = createFileRoute('/settings')({
  beforeLoad: async () => {
    const session = await getServerSession()
    if (!session) throw redirect({ to: '/sign-in' })
    return { session }
  },
  component: Settings,
})

function Settings() {
  const { session } = Route.useRouteContext()
  return (
    <div className="mx-auto max-w-2xl px-6 py-10">
      <Link to="/" className="text-sm text-muted-foreground underline underline-offset-4">
        ← Retour
      </Link>
      <div className="mt-6">
        <AccountSettings
          client={authClient}
          user={{ name: session.user.name, email: session.user.email }}
          setPasswordHref="/forgot-password"
          deleteAccount={{ onDeleted: () => window.location.assign('/sign-in') }}
        />
      </div>
    </div>
  )
}
