# stack

Skalpai-style application stack. Generated from this template, each new
app starts with the full skalpai dev workflow + observability already wired.

## Stack
- **Backend**: Go 1.25 + Echo v4 + Postgres + NATS JetStream — port 4100
- **Frontend**: React 19 + TanStack Router + Tailwind v4 — port 5273
- **Auth**: urbangate, the suite's identity provider, behind this app's own screens via
  `@lalternative/auth/urbangate` (Kratos holds the identity; the core verifies the token
  urbangate issues for the person with `go/websession`)
- **API client**: `lib/front` (`@app/front`) — typed React Query client generated from the core OpenAPI (swag → orval), consumed as source
- **Pipeline**: host-installed `sklp` CLI (NOT vendored), config in `.sklp/`

## Architecture
- DDD + CQRS per bounded context under `apps/core/<context>/` (see `example/`)
- `account/` is a plain service (not CQRS): the core's account surface —
  authentication is urbangate's, so the core verifies the person's token, maps
  their identity to the local `"user"` row everything is keyed on
  (`ResolvePerson`), and serves `GET /me` and the account erasure
  `DELETE /account` the web's settings page calls. Grow it into the DDD
  layout only if account state moves into the core.
- Event-sourced via `github.com/lalternative/packages/go/eda`: aggregate
  (`ddd.BaseAggregateRoot` + `ddd.Raise`) → events → projection (read model) →
  typed command/query routers (`cqrs.Execute`/`cqrs.Ask`), wired with `di`.
  Modelled on `go/eda/examples/banking`. `example/`'s event store is in-memory
  (swap `db.InMemoryStore` for a Postgres `db.Store` to persist).
- Layout: `domain/ (aggregate+events) → application/<usecase>/ (command|query) →
  projection/ → infrastructure/ (repo on event store) → api.go + dto.go + bootstrap.go`
- Integration events: durable JetStream consumers under
  `<context>/application/event-handlers/` via `eda/pkg/consumer`
- File-based SQL migrations in `apps/core/migrations/postgres/` (the `"user"`
  row; the example context is in-memory, no table)
- Observability wired at boot through `apps/core/observability` using `skalpai/sdk-go`

## Conventions
- All code, comments, commit messages in English
- Commits must include `Co-Authored-By: codesyl <codesyl@pm.me>`
- Do NOT include `Co-Authored-By: Claude` or any Anthropic co-author
- Use `github.com/google/uuid` for IDs
- `middleware.RequireAuth` names the signed-in person; handlers read them via `middleware.GetUser(c)`
- The TS API client is generated, never hand-written: annotate Echo handlers
  with swaggo `// @...`, then `sklp run generate` (swag → orval). The typed
  React Query client lands in `lib/front/src/api` (`@app/front`), consumed as
  source via subpaths (`@app/front/api/<tag>/<tag>`). `lib/front/src/api` and
  `apps/core/docs` are checked in.
- **Never hand-roll a JetStream consumer.** Consume integration events through
  `github.com/lalternative/packages/go/eda/pkg/consumer` (`consumer.Run` + an
  `EventHandler`). It provides `Term`/`MaxDeliver`/`BackOff`/DLQ/heartbeat/
  idempotency/reconnect by default — you write only `Handle`. New handlers go
  under `apps/core/<context>/application/event-handlers/`; copy
  `example/application/event-handlers/` as the template.
  Do not call `js.Subscribe`/`Consume` directly. See `docs/ARCHITECTURE.md`.

## Feature / PR / Publish flow

Use the `sklp flow` helpers — they wrap this flow and, by creating a
dedicated worktree per feature, let several features be worked on in
parallel without branch-switching in a shared checkout. Prefer them over
raw `git checkout -b`.

1. `sklp flow start <name>` — opens `feat/<name>` in a dedicated worktree
   (default; `--branch` checks out in place instead). One worktree per
   feature keeps parallel work isolated.
2. `sklp run ci` before pushing.
3. `sklp flow end --pr` — diffs vs `--onto`, runs CI on the impacted scope,
   and opens the PR targeting `main`. CI runs again on PR.
4. After merge, update local `main`.
5. `sklp run publish` from clean local `main`.
6. Publish only impacted services. Tags `YYYYMMDD-HHMM-<short-sha>`.
7. ArgoCD / Image Updater handles rollout.

Rules:
- Never publish from a feature branch.
- Never claim deployment success when only the image was pushed.
- Treat `.sklp/pipelines/ci.yaml` and `.sklp/pipelines/publish.yaml` as the source of truth.

## Development

```bash
cp .env.example .env
sklp dev stack             # boots core → web
sklp dev stack --validate  # parse-only sanity check
```

Ports: core 4100, web 5273.

Other entry points:
- `sklp run build` — build `bin/core` and the web bundle
- `sklp run ci` — lint + test impacted services
- `sklp run secops` — gitleaks, semgrep, govulncheck, pnpm audit, trivy
- `sklp run publish` — build, scan, push impacted Docker images (main only)

## CI / triggers — NO GitHub Actions

Do NOT add `.github/workflows/*.yml` that drive `sklp run`. The skalpai UI is
the CI control plane (remote dispatch). CI runs locally via `sklp flow end`
before opening a PR, and again on the PR.

## Container registry

`rg.fr-par.scw.cloud/skaplai/<service>:<date>-<short-sha>`. Immutable tags
only. No `latest`. Trivy gates HIGH/CRITICAL fixable findings.
