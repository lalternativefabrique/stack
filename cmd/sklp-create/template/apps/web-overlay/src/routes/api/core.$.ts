import { createFileRoute } from '@tanstack/react-router'
import { coreProxy } from '@/lib/core-proxy'

export const Route = createFileRoute('/api/core/$')({
  server: {
    handlers: {
      GET: ({ request }: { request: Request }) => coreProxy(request),
      POST: ({ request }: { request: Request }) => coreProxy(request),
      PUT: ({ request }: { request: Request }) => coreProxy(request),
      PATCH: ({ request }: { request: Request }) => coreProxy(request),
      DELETE: ({ request }: { request: Request }) => coreProxy(request),
    },
  },
})
