# Architecture

This stack mirrors the skalpai layout so a new project lands on the same
mental model from day one.

## Layered backend (DDD)

```
apps/core/<context>/          (see apps/core/example/ — CQRS via go/eda)
├── api.go              ── HTTP handlers (Echo). Thin: bind → cqrs.Execute/Ask → DTO
├── dto.go              ── transport types
├── bootstrap.go        ── DI registry: store, repo, projection, command/query
│                          routers; exposes NewService + RegisterRoutes
├── domain/             ── event-sourced aggregate (ddd.BaseAggregateRoot) + events
├── projection/         ── read model, updated from the event stream (Apply)
├── application/<usecase>/
│                       ── one folder per command/query (Command|Query + Handler).
│                          No HTTP.
├── application/event-handlers/  ── durable JetStream consumers (consumer.EventHandler)
└── infrastructure/     ── event-store-backed repository (go/eda db.Store)
```

Rationale: `apps/core/example/` is a runnable CQRS example built on
`github.com/lalternative/packages/go/eda` (aggregate → event → projection →
query, DI-wired), modelled on that lib's `examples/banking`. Its event store is
**in-memory** (an example — swap `db.InMemoryStore` for a Postgres `db.Store` to
persist). Copy the folder to a real context name, or delete it.

## Data access seam

Repositories receive a `*pgxpool.Pool` (see `apps/core/pkg/db`). Postgres via
pgx out of the box. Migrations are file-based SQL under `migrations/postgres/`,
applied in lexical order at boot (must be idempotent — they run every boot).

## Event-driven seam (durable consumers)

Integration events are consumed through `github.com/lalternative/packages/go/eda/pkg/consumer`,
**never** by hand-rolling a JetStream subscription. The brick owns every
redelivery concern once, so handlers can't get them wrong:

| Concern | Provided by |
| --- | --- |
| Permanent error → `Term()` | `consumer.Permanent(err)` / `ErrPermanent` |
| Bounded `MaxDeliver` | `EventHandler.MaxDeliver()` |
| Staged `BackOff` | `Config.BackOff` (default 30/60/120s) |
| Dead-letter stream (DLQ) | `MAX_DELIVERIES` advisory stream |
| Heartbeat anti-`AckWait` | `InProgress()` ticker (concurrent handlers) |
| Idempotency | optional `Config.Idempotency` (by `event_id`) |
| Reconnect / retry loop | `consumer.Run` |

A handler lives under `apps/core/<context>/application/event-handlers/`,
implements `consumer.EventHandler` (`Name/Subject/DurableName/MaxDeliver/Handle`)
and writes business logic **only** in `Handle`. `main.go` starts one
`consumer.Run` goroutine per handler. `apps/core/example/application/event-handlers/`
ships a working example — copy it, change the subject and the body of `Handle`.

> Anti-pattern this prevents: a service that re-implements a minimal JetStream
> consumer (Nak everywhere, no `Term`, no `MaxDeliver`/`BackOff`/DLQ) and
> silently loses or infinitely retries messages. If you find yourself calling
> `js.Subscribe`/`Consume` directly, stop — use the brick.

## Auth seam

Nobody has an account here. urbangate (Ory Kratos + Hydra) is the suite's
identity provider, behind this app's own screens (urbangate ADR 0009):

- `apps/web` renders sign-up, sign-in, the e-mail code, recovery and the
  account settings with `@lalternative/auth`, and its server drives the Kratos
  flows through `createUrbangateAuth` (`lib/auth.ts`). Session guards run
  server-side (`lib/get-server-session.ts`), never in the browser alone.
- The browser reaches the core only through the web's `/api/core/*` proxy
  (`auth.coreProxy`), which attaches the person's token; the token never
  reaches page scripts.
- `apps/core/middleware` verifies that token with `go/websession`
  (`OIDC_ISSUER_URL`, `OIDC_AUDIENCE`) and `account.ResolvePerson` maps the
  identity to the local `"user"` row every table is keyed on.
- `/admin` is the product's console: the suite's team signs in through
  urbangate's SSO (`sso` on the auth, `<product>:admin` role), people and
  roles are managed in urbangate's console, not here.
- Account deletion is the settings page's `DeleteAccountSteps` calling
  `DELETE /api/v1/account` on the core, idempotent so a failed step is
  retried whole.

Each product needs two Hydra clients declared in urbangate
(`ory/hydra/clients{,-dev}/<product>-admin.json` and `-provisioner.json`)
and the team identity granted `<product>:admin`.

## Observability seam

`apps/core/observability` wraps `github.com/lalternative/packages/skalpai/sdk-go`
and configures OTEL traces, metrics, and logs exporters at boot. Disabled
(no-op) when `SKALPAI_ENDPOINT` or `SKALPAI_API_KEY` are empty, so
`go test ./...` and offline dev don't fight the SDK. The web app should
ship `@lalternative/skalpai-sdk-browser` for RUM (left for the consuming project to
add — it's not in the stack to avoid a forced dep).

## sklp seam

The stack **uses** the host `sklp` CLI. `.sklp/space.yaml` is the runner
recipe (toolchain + registry); `.sklp/stack/dev.yaml` is the declarative
dev topology; `.sklp/pipelines/*.yaml` are the CI/build/publish pipelines.
New behavior goes into helpers/templates/codegen — not into new top-level
YAML fields (skalpai convention).

## Renaming

`sklp start <name>` renames the template in-process while scaffolding:
- replace `app/` Go module paths with `<name>/`
- replace `@app/` npm scopes with `@<name>/`
- replace `rg.fr-par.scw.cloud/skaplai/` images with your registry/team

## Container registry policy

`rg.fr-par.scw.cloud/skaplai/<service>:YYYYMMDD-HHMM-<short-sha>`. Set by
`tag.strategy: date-sha` in `.sklp/pipelines/publish.yaml`. Never `latest`,
never overwritten, Trivy gates HIGH/CRITICAL fixable findings.
