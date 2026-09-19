import { createFileRoute } from '@tanstack/react-router'
import { auth } from '@/lib/auth'
import { pool } from '@/lib/db'

/**
 * First-admin bootstrap, in two steps.
 *
 * GET reports whether an admin already exists. POST without a code signs the
 * account up (which is what makes Better Auth send the verification code); POST
 * with a code verifies it and grants the admin role. The endpoint closes itself
 * once an admin is present — an open route that grants the admin role is how a
 * back-office is taken over.
 *
 * It is reachable with no session by design, so it proves only that the
 * operator can read the mailbox they claim. Gate the route itself (setup token,
 * allowed-emails list) if the deployment needs more than that.
 */

type OtpApi = {
  sendVerificationOTP: (args: {
    body: { email: string; type: 'email-verification' }
  }) => Promise<unknown>
  verifyEmailOTP: (args: { body: { email: string; otp: string } }) => Promise<unknown>
}

/**
 * The email-OTP endpoints exist on the instance but not on its inferred type:
 * createPlatformAuth always registers the emailOTP plugin, yet what it exports
 * does not carry the plugin's endpoints. Narrowed rather than `any` so an
 * upstream rename still breaks the build.
 */
function otpApi(): OtpApi {
  return auth.api as unknown as OtpApi
}

async function adminExists(): Promise<boolean> {
  const result = await pool.query(
    `SELECT COUNT(*) as count FROM "user" WHERE role = 'admin'`,
  )
  return parseInt(result.rows[0].count, 10) > 0
}

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

export const Route = createFileRoute('/api/admin/setup')({
  server: {
    handlers: {
      GET: async () => json({ hasAdmin: await adminExists() }),

      POST: async ({ request }: { request: Request }) => {
        if (await adminExists()) {
          return json({ error: 'Setup already completed' }, 403)
        }

        const body = (await request.json()) as {
          email?: string
          password?: string
          name?: string
          code?: string
        }
        if (!body.email || !body.password || !body.name) {
          return json({ error: 'Email, password and name are required' }, 400)
        }
        const email = body.email.trim().toLowerCase()

        if (!body.code?.trim()) {
          try {
            await auth.api.signUpEmail({
              body: { name: body.name.trim(), email, password: body.password },
            })
          } catch (err) {
            const message = err instanceof Error ? err.message : String(err)
            // Re-running the first step after a rejected code is normal: the
            // account is already there and what is wanted is another code.
            if (!/exists|existe|déjà/i.test(message)) {
              return json({ error: message }, 400)
            }
            try {
              await otpApi().sendVerificationOTP({
                body: { email, type: 'email-verification' },
              })
            } catch {
              return json({ error: message }, 400)
            }
          }
          return json({ success: true, codeSent: true })
        }

        try {
          await otpApi().verifyEmailOTP({ body: { email, otp: body.code.trim() } })
        } catch (err) {
          // The account stays unverified and roleless: the code expires on its
          // own and asking for another is the first step run again, which beats
          // deleting an account over a mistyped digit.
          const message = err instanceof Error ? err.message : 'Code rejected'
          return json({ error: `Code rejected: ${message}` }, 400)
        }

        // The advisory lock is what stops two racing callers from both passing
        // the existence check on an admin-less table and minting two admins.
        const client = await pool.connect()
        try {
          await client.query('SELECT pg_advisory_lock(hashtext($1))', ['admin-setup'])
          const existing = await client.query(
            `SELECT 1 FROM "user" WHERE role = 'admin'`,
          )
          if (existing.rowCount && existing.rowCount > 0) {
            return json({ error: 'Setup already completed' }, 403)
          }
          await client.query(`UPDATE "user" SET role = 'admin' WHERE email = $1`, [
            email,
          ])
        } finally {
          await client.query('SELECT pg_advisory_unlock(hashtext($1))', ['admin-setup'])
          client.release()
        }

        return json({ success: true, email })
      },
    },
  },
})
