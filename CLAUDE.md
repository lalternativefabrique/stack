# stack

Scaffolder for Skalpai-style application stacks. This repo is a **single Go
module**: the `sklp-create` CLI and the embedded template it ships. There is no
backend, `.sklp/`, frontend, or SDK at the root — all of that lives *inside*
the template and only materialises in a generated project.

## Layout

```
cmd/sklp-create/
  main.go        root + version
  start.go       `sklp-create start <name>` (NOT copied into projects)
  template/      the checked-in project template (backend + .sklp + Dockerfile)
go.mod           module github.com/lalternative/stack
```

Edit the generated stack by editing `cmd/sklp-create/template/`. `start.go`
rewrites module paths (`app/` → `<name>/`) and npm scopes (`@app/` → `@<name>/`)
on extraction, then scaffolds `apps/web` on the fly with the official TanStack
CLI (nothing frontend is embedded, so it never drifts). `pnpm` must be on PATH.

## Conventions
- All code, comments, commit messages in English
- **Minimal comments (MANDATORY)**: the code is the only real source of truth — it is what runs, and it is always current. A comment is an unverified claim: nothing compiles it, nothing tests it, and it goes stale the first time the code next to it changes. A stale comment is worse than none, because it actively misleads the next reader into trusting behaviour that no longer exists. So write the strict minimum, especially in frontend (TS/React) and Go API code. No comments restating the code, no section banners, no step narration (`// 1. validate`), no JSDoc repeating TS types. Only tooling directives (`//go:generate`, swag annotations), linter-required exported Go doc comments, and short notes for genuinely non-obvious external constraints (protocol quirk, upstream bug, race) — things the code truly cannot express. If a comment feels needed, rename or extract a function instead: that explanation lives in the code and cannot drift. Remove noise comments in code you touch. Applies to subagents and skills too.
- Commits must include `Co-Authored-By: codesyl <codesyl@pm.me>`
- Do NOT include `Co-Authored-By: Claude` or any Anthropic co-author

## The template ships (into generated projects, not run here)
- **Backend**: Go 1.25 + Echo v4 + Postgres (pgx), DDD per bounded context
- **`.sklp/`**: dev.yaml + pipelines (ci/build/publish/secops) driven by the
  host-installed `sklp` CLI
- **Dockerfile**: distroless, runs as `USER nonroot`
- Dev deps (postgres, nats) are `image:` services so `sklp dev`'s embedded runc
  backend pulls them into spacenet — never `run: docker run` (no docker in the
  space).

Changes to the shipped stack go under `cmd/sklp-create/template/`; verify by
scaffolding a throwaway project and running `sklp dev stack` in it.

## Feature / PR flow

Use the `sklp flow` helpers (one worktree per feature):

1. `sklp flow start <name>` — opens `feat/<name>` (`--branch` for in-place).
2. `sklp flow end --pr` — diff vs `main`, run CI on impacted scope, open the PR.
3. After merge, sync local `main`.

Rules:
- "Merge a PR" = merge on the GitHub remote via `gh`, then `git pull`. Never a
  local merge; never push `main` directly without explicit approval.
- Distribution is `go install github.com/lalternative/stack/cmd/sklp-create` —
  there is no Docker image for this repo.

## CI / triggers — NO GitHub Actions

Do NOT add `.github/workflows/*.yml` that drive `sklp run`. The skalpai UI is
the CI control plane (remote dispatch). Local pre-push gates use git hooks.
