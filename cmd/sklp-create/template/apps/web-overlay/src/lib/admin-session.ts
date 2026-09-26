import { createServerFn } from '@tanstack/react-start'
import { getRequestHeaders, setResponseHeader } from '@tanstack/react-start/server'
import { auth } from '@/lib/auth'
import { adminAccess } from '@/lib/admin-guard'
import type { AdminAccess } from '@/lib/admin-guard'

export const checkAdminAccess = createServerFn({ method: 'GET' }).handler(
  async (): Promise<AdminAccess> => {
    const guarded = await auth.requireAdmin(new Headers(getRequestHeaders()))
    if ('setCookies' in guarded && guarded.setCookies.length > 0)
      setResponseHeader('set-cookie', guarded.setCookies)
    return adminAccess(guarded)
  },
)
