import { createServerFn } from '@tanstack/react-start'
import { getRequestHeaders, setResponseHeader } from '@tanstack/react-start/server'
import { auth } from '@/lib/auth'
import { sessionOf } from '@/lib/session-of'
import type { ServerSession } from '@/lib/session-of'

export type { ServerSession }

export const getServerSession = createServerFn({ method: 'GET' }).handler(
  async (): Promise<ServerSession | null> => {
    const guarded = await auth.requireSession(new Headers(getRequestHeaders()))
    if ('setCookies' in guarded && guarded.setCookies.length > 0)
      setResponseHeader('set-cookie', guarded.setCookies)
    return sessionOf(guarded)
  },
)
