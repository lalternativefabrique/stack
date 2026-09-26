import { auth } from '@/lib/auth'

type Proxy = ReturnType<typeof auth.coreProxy>

let built: Proxy | undefined

export function coreProxy(request: Request): ReturnType<Proxy> {
  built ??= auth.coreProxy({ stripPrefix: '/api/core' })
  return built(request)
}
