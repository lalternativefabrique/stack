import { createUrbangateAuth } from '@lalternative/auth/urbangate'
import { accountDeletion } from '@/lib/account-deletion'
import { CORE_URL } from '@/lib/core-url'

/**
 * Nobody has an account here: a person signs up and signs in on this app's
 * screens, Kratos holds the identity, and the core trusts the token urbangate
 * issues for them (urbangate ADR 0009). Built on first use, so a public page
 * needs no secret.
 */
function build() {
  const required = (name: string) => {
    const value = process.env[name]
    if (!value) throw new Error(`${name} environment variable is required`)
    return value
  }
  const issuerUrl = process.env.URBANGATE_ISSUER_URL ?? 'https://id.urbangate.dev'
  return createUrbangateAuth({
    product: '__APP_NAME__',
    productName: '__APP_NAME__',
    kratosUrl: process.env.KRATOS_PUBLIC_URL ?? issuerUrl,
    urbangate: {
      issuerUrl,
      provisioner: {
        clientId: process.env.URBANGATE_PROVISIONER_CLIENT_ID ?? '__APP_NAME__-provisioner',
        clientSecret: required('URBANGATE_PROVISIONER_CLIENT_SECRET'),
      },
      admin: {
        clientId: process.env.URBANGATE_CLIENT_ID ?? '__APP_NAME__-admin',
        clientSecret: required('URBANGATE_CLIENT_SECRET'),
      },
    },
    coreUrl: CORE_URL,
    accountDeletion: ({ accessToken }) => accountDeletion(accessToken),
    sso: { appUrl: process.env.APP_URL ?? 'http://localhost:5273' },
    nakoda: process.env.NAKODA_KEY ? { key: process.env.NAKODA_KEY } : undefined,
  })
}

export type Auth = ReturnType<typeof build>

let built: Auth | undefined

export function getAuth(): Auth {
  built ??= build()
  return built
}

export const auth: Auth = new Proxy({} as Auth, {
  get: (_, prop: string | symbol) => Reflect.get(getAuth(), prop) as unknown,
})
