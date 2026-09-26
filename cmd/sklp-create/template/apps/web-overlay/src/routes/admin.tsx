import { createFileRoute, ErrorComponent, Link, Outlet, redirect } from '@tanstack/react-router'
import type { ErrorComponentProps } from '@tanstack/react-router'
import { AdminLayout } from '@lalternative/admin'
import '@lalternative/admin/styles.css'
import { checkAdminAccess } from '@/lib/admin-session'

class IdentityProviderUnavailable extends Error {}

/**
 * `/admin/login` escapes this layout by the trailing underscore on its
 * `admin_` segment, so it stays reachable without an admin session.
 */
export const Route = createFileRoute('/admin')({
  beforeLoad: async () => {
    const access = await checkAdminAccess()
    if (access === 'sign_in') throw redirect({ to: '/admin/login', replace: true })
    if (access === 'unavailable') throw new IdentityProviderUnavailable()
  },
  errorComponent: AdminUnavailable,
  component: AdminShell,
})

function AdminUnavailable({ error }: ErrorComponentProps) {
  if (!(error instanceof IdentityProviderUnavailable)) return <ErrorComponent error={error} />
  return (
    <p role="alert" className="p-8 text-sm text-muted-foreground">
      Le service d'identité ne répond pas pour le moment. Réessayez dans un instant.
    </p>
  )
}

function AdminShell() {
  const linkClass = 'text-muted-foreground hover:text-foreground'
  const activeClass = 'text-foreground'
  return (
    <AdminLayout
      nav={
        <Link
          to="/admin"
          activeOptions={{ exact: true }}
          className={linkClass}
          activeProps={{ className: activeClass }}
        >
          Tableau de bord
        </Link>
      }
      backToApp={
        <Link to="/" className="text-muted-foreground hover:text-foreground">
          Retour à l'app
        </Link>
      }
    >
      <Outlet />
    </AdminLayout>
  )
}
