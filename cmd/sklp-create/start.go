package main

import (
	"bufio"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

//go:embed all:template
var templateFS embed.FS

const templateRoot = "template"

// webOverlayRoot is the embedded overlay laid over the freshly-scaffolded web
// app by overlayWeb. It lives under template/ so `go:embed all:template` ships
// it, but extractTemplate must NOT drop it into the generated project as-is —
// it belongs inside apps/web, not at apps/web-overlay.
const webOverlayRoot = templateRoot + "/apps/web-overlay"

var nameRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

const (
	digstackPrivatePattern = "github.com/digstack/*"
	digstackHTTPSPrefix    = "https://github.com/digstack/"
	digstackSSHPrefix      = "git@github.com:digstack/"
)

func newStartCmd() *cobra.Command {
	var assumeYes bool
	cmd := &cobra.Command{
		Use:   "start <name>",
		Short: "Scaffold a new skalpai-style project in ./<name>",
		Long: `Create a new project from the embedded stack template.

The template ships the Go backend: Postgres + Echo REST (DDD), NATS
JetStream event consumers (eda lib), and a generated TS SDK (lib/front).
Drop the parts you don't need (e.g. the NATS wiring) after scaffolding.

The web app (apps/web) is NOT embedded: it is scaffolded on the fly with
the official TanStack CLI (` + "`pnpm dlx @tanstack/cli create`" + `), so every
new project starts from an up-to-date TanStack Start app instead of a
frozen, drifting template. Requires pnpm on PATH.

On first run, prompts to set GOPRIVATE and a git insteadOf rule so the
private skalpai sdk-go module can be fetched over SSH. Pass --yes to
apply both without confirmation.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStart(args[0], assumeYes)
		},
	}
	cmd.Flags().BoolVarP(&assumeYes, "yes", "y", false, "apply machine config changes (GOPRIVATE, git insteadOf) without confirmation")
	return cmd
}

func runStart(name string, assumeYes bool) error {
	if !nameRe.MatchString(name) {
		return fmt.Errorf("name must be kebab-case [a-z][a-z0-9-]*, got %q", name)
	}
	if _, err := os.Stat(name); err == nil {
		return fmt.Errorf("./%s already exists", name)
	}
	if _, err := exec.LookPath("pnpm"); err != nil {
		return fmt.Errorf("pnpm not found on PATH — required to scaffold the web app: %w", err)
	}

	if err := ensureMachineConfig(assumeYes); err != nil {
		return err
	}

	if err := extractTemplate(templateFS, templateRoot, name); err != nil {
		return fmt.Errorf("extract template: %w", err)
	}
	// Generate the web app fresh with the official TanStack CLI, then rewire it
	// into the stack (name, port). Done BEFORE rename so the @app/ placeholders
	// we write are rewritten alongside the rest of the template.
	if err := scaffoldWeb(name); err != nil {
		return fmt.Errorf("scaffold web app: %w", err)
	}
	// Lay the admin/auth overlay over the fresh TanStack app: the shared admin
	// UI (@lalternative/admin) plus the thin routes, auth wiring and server
	// endpoints that mount it. Done after scaffoldWeb (needs apps/web to exist)
	// and before rename (so any placeholders are rewritten with the rest).
	if err := overlayWeb(name); err != nil {
		return fmt.Errorf("overlay web app: %w", err)
	}
	if err := rename(name); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	if err := copyEnvExample(name); err != nil {
		return fmt.Errorf("seed .env: %w", err)
	}
	if err := gitInit(name); err != nil {
		fmt.Fprintf(os.Stderr, "warn: git init failed (%v) — continuing\n", err)
	}

	fmt.Printf("✓ created ./%s\n\n", name)
	fmt.Printf("next:\n")
	fmt.Printf("  cd %s\n", name)
	fmt.Printf("  sklp dev stack\n")
	return nil
}

// scaffoldWeb generates apps/web on the fly with the official TanStack CLI
// (React + Nitro SSR + TanStack Query), then rewires it into the stack:
// package name @app/web and dev/preview on the stack port (5273). The stack's
// dev/CI/build/publish all reference apps/web + @app/web, so those two must
// match; everything else is left as the CLI produced it (kept up to date).
func scaffoldWeb(name string) error {
	// Absolute target so the CLI writes into the generated project regardless
	// of the process cwd.
	abs, err := filepath.Abs(filepath.Join(name, "apps", "web"))
	if err != nil {
		return err
	}
	args := []string{
		"dlx", "@tanstack/cli", "create",
		"--framework", "React",
		"--target-dir", abs,
		"--package-manager", "pnpm",
		"--deployment", "nitro",
		"--add-ons", "tanstack-query",
		"--toolchain", "eslint",
		"--no-git", "--no-install", "--no-examples",
		"--non-interactive",
	}
	cmd := exec.Command("pnpm", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pnpm dlx @tanstack/cli create: %w", err)
	}
	if err := rewireWebPackageJSON(filepath.Join(abs, "package.json")); err != nil {
		return err
	}
	// The lockfile is written later by overlayWeb, after patchWebDeps adds the
	// admin/auth/pg dependencies — otherwise it would omit them and the stack
	// CI's --frozen-lockfile install would fail.
	return relaxWebTSConfig(filepath.Join(abs, "tsconfig.json"))
}

// webOverlayDeps are the dependencies the admin/auth overlay needs, added to
// apps/web/package.json by patchWebDeps. @lalternative/{auth,admin} are the
// shared packages; better-auth is their peer; pg backs the setup route's direct
// SQL. Kept in one place so the versions are easy to bump.
var webOverlayDeps = map[string]string{
	"@lalternative/auth":  "^0.1.2",
	"@lalternative/admin": "^0.1.0",
	"better-auth":         "^1.6.11",
	"pg":                  "^8.18.0",
	"@types/pg":           "^8.16.0",
}

// overlayWeb lays the admin/auth overlay over the freshly-scaffolded web app:
// it copies template/apps/web-overlay/** into apps/web/**, adds the overlay's
// dependencies to package.json, then (re)writes the lockfile so it accounts for
// them. Read straight from the embedded FS — the overlay is deliberately not
// extracted into the project by extractTemplate.
func overlayWeb(name string) error {
	webDir := filepath.Join(name, "apps", "web")
	err := fs.WalkDir(templateFS, webOverlayRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(p, webOverlayRoot)
		rel = strings.TrimPrefix(rel, "/")
		if rel == "" {
			return nil
		}
		target := filepath.Join(webDir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := templateFS.ReadFile(p)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
	if err != nil {
		return fmt.Errorf("copy overlay: %w", err)
	}
	if err := patchWebDeps(filepath.Join(webDir, "package.json")); err != nil {
		return err
	}
	return writeWebLockfile(name)
}

// patchWebDeps adds webOverlayDeps to the web package.json's "dependencies".
// The TanStack CLI always emits a "dependencies" block, so we insert right
// after its opening brace — a textual edit (like rewireWebPackageJSON) rather
// than a full JSON round-trip, to keep the CLI's formatting untouched.
func patchWebDeps(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	s := string(b)
	// Deterministic order so the generated file is stable across runs.
	keys := make([]string, 0, len(webOverlayDeps))
	for k := range webOverlayDeps {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var lines strings.Builder
	for _, k := range keys {
		if strings.Contains(s, `"`+k+`"`) {
			continue // already present — don't duplicate
		}
		lines.WriteString(fmt.Sprintf("\n    %q: %q,", k, webOverlayDeps[k]))
	}
	if lines.Len() == 0 {
		return nil
	}
	marker := `"dependencies": {`
	i := strings.Index(s, marker)
	if i < 0 {
		return fmt.Errorf("no \"dependencies\" block in %s (TanStack CLI output changed?)", path)
	}
	at := i + len(marker)
	s = s[:at] + lines.String() + s[at:]
	return os.WriteFile(path, []byte(s), 0o644)
}

// writeWebLockfile generates the workspace pnpm-lock.yaml. The CLI runs with
// --no-install (no lockfile), but the stack CI installs with --frozen-lockfile
// and fails without one. `--lockfile-only` resolves the graph and writes the
// lockfile without downloading/linking packages, so it stays fast.
func writeWebLockfile(name string) error {
	cmd := exec.Command("pnpm", "install", "--lockfile-only", "--ignore-scripts")
	cmd.Dir = name
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pnpm install --lockfile-only: %w", err)
	}
	return nil
}

// relaxWebTSConfig turns off noUnusedLocals/noUnusedParameters in the generated
// tsconfig. The TanStack CLI ships a router.tsx with unused imports, so a fresh
// scaffold fails `tsc --noEmit` out of the box — and the stack CI runs exactly
// that. eslint still flags real unused code; this only stops the generator's
// own dead imports from turning the very first CI run red.
func relaxWebTSConfig(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	s := string(b)
	s = strings.ReplaceAll(s, `"noUnusedLocals": true`, `"noUnusedLocals": false`)
	s = strings.ReplaceAll(s, `"noUnusedParameters": true`, `"noUnusedParameters": false`)
	return os.WriteFile(path, []byte(s), 0o644)
}

// rewireWebPackageJSON aligns the generated web package with the stack: the
// name must be @app/web (the stack filters `pnpm --filter @app/web ...`), and
// the dev/preview servers must bind the stack port (5273, proxied by sklp dev).
func rewireWebPackageJSON(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	s := string(b)
	repl := []struct{ from, to string }{
		{`"name": "web"`, `"name": "@app/web"`},
		// --host binds 0.0.0.0: sklp dev runs this inside a container, and a
		// server left on the loopback answers nothing through the published
		// port, so the readiness probe times out on a server that did start.
		{`"dev": "vite dev --port 3000"`, `"dev": "vite dev --port 5273 --host 0.0.0.0"`},
		{`"preview": "vite preview"`, `"preview": "vite preview --port 5273 --host 0.0.0.0"`},
		// The stack CI runs `pnpm --filter @app/web typecheck`; the TanStack CLI
		// ships `lint`/`test` but no typecheck script. Generate the route tree
		// first (it is not committed, so `tsc` alone fails with "Cannot find
		// module './routeTree.gen'"). Use `vite build`, NOT `tsr generate`: the
		// admin overlay's server routes (`server: { handlers }`) only typecheck
		// when routeTree.gen.ts carries the `@tanstack/react-start` module
		// augmentation, which the Start Vite plugin emits and the bare `tsr`
		// CLI does not.
		{`"lint": "eslint",`, "\"lint\": \"eslint\",\n    \"typecheck\": \"vite build && tsc --noEmit\","},
	}
	for _, r := range repl {
		s = strings.Replace(s, r.from, r.to, 1)
	}
	if !strings.Contains(s, `"name": "@app/web"`) {
		return fmt.Errorf("could not set @app/web name in %s (TanStack CLI output changed?)", path)
	}
	// The CLI writes a `pnpm.onlyBuiltDependencies` block in the web package;
	// pnpm ignores it outside the workspace root (it lives in the root
	// package.json instead), so drop it to silence the install warning.
	s = stripWebPnpmBlock(s)
	return os.WriteFile(path, []byte(s), 0o644)
}

// webPnpmBlockRe matches the trailing `,"pnpm": { ... }` object the TanStack
// CLI appends to the web package.json (with its leading comma) right before the
// closing brace of the document. Go's RE2 has no lookahead, so the final `}` is
// captured and restored. Non-greedy inner match stops at the block's own `}`.
var webPnpmBlockRe = regexp.MustCompile(`(?s),\s*"pnpm"\s*:\s*\{.*?\}\s*(\n?\})\s*$`)

func stripWebPnpmBlock(s string) string {
	return webPnpmBlockRe.ReplaceAllString(s, "$1")
}

// ensureMachineConfig checks the two prerequisites for fetching the
// private skalpai sdk-go module and offers to set them. Without these,
// `go run` from the generated project fails with a HTTPS prompt.
func ensureMachineConfig(assumeYes bool) error {
	if err := ensureGoPrivate(assumeYes); err != nil {
		return err
	}
	return ensureGitInsteadOf(assumeYes)
}

func ensureGoPrivate(assumeYes bool) error {
	out, err := exec.Command("go", "env", "GOPRIVATE").Output()
	if err != nil {
		return fmt.Errorf("read GOPRIVATE: %w", err)
	}
	current := strings.TrimSpace(string(out))
	if strings.Contains(current, "github.com/digstack/") || strings.Contains(current, digstackPrivatePattern) {
		return nil
	}

	next := digstackPrivatePattern
	if current != "" {
		next = current + "," + digstackPrivatePattern
	}
	prompt := fmt.Sprintf("GOPRIVATE does not include %q.\n  current:  %q\n  proposed: %q\nApply with `go env -w GOPRIVATE=...`?", digstackPrivatePattern, current, next)
	if !assumeYes && !confirm(prompt) {
		fmt.Fprintln(os.Stderr, "skipping — `sklp dev` will fail until GOPRIVATE is set.")
		return nil
	}
	if err := exec.Command("go", "env", "-w", "GOPRIVATE="+next).Run(); err != nil {
		return fmt.Errorf("set GOPRIVATE: %w", err)
	}
	fmt.Printf("✓ GOPRIVATE=%s\n", next)
	return nil
}

func ensureGitInsteadOf(assumeYes bool) error {
	key := "url." + digstackSSHPrefix + ".insteadOf"
	out, _ := exec.Command("git", "config", "--global", "--get", key).Output()
	if strings.TrimSpace(string(out)) == digstackHTTPSPrefix {
		return nil
	}
	prompt := fmt.Sprintf("git is not configured to fetch %s over SSH.\nApply `git config --global %s %q`?", digstackHTTPSPrefix, key, digstackHTTPSPrefix)
	if !assumeYes && !confirm(prompt) {
		fmt.Fprintln(os.Stderr, "skipping — `sklp dev` will fail until the SSH rewrite is set.")
		return nil
	}
	if err := exec.Command("git", "config", "--global", key, digstackHTTPSPrefix).Run(); err != nil {
		return fmt.Errorf("set git insteadOf: %w", err)
	}
	fmt.Printf("✓ git rewrite %s → %s\n", digstackHTTPSPrefix, digstackSSHPrefix)
	return nil
}

func confirm(prompt string) bool {
	fmt.Fprintf(os.Stderr, "\n%s [y/N] ", prompt)
	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	if err != nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	}
	return false
}

func extractTemplate(fsys embed.FS, root, dst string) error {
	return fs.WalkDir(fsys, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// The web overlay is copied into apps/web by overlayWeb, not extracted
		// into the project as its own tree.
		if p == webOverlayRoot || strings.HasPrefix(p, webOverlayRoot+"/") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		rel := strings.TrimPrefix(p, root)
		rel = strings.TrimPrefix(rel, "/")
		if rel == "" {
			return os.MkdirAll(dst, 0o755)
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := fsys.ReadFile(p)
		if err != nil {
			return err
		}
		// go:embed refuses to walk into subtrees that contain a go.mod, so
		// template/.../go.mod is shipped as go.mod.tmpl and restored here.
		target = strings.TrimSuffix(target, ".tmpl")
		mode := fs.FileMode(0o644)
		if strings.HasSuffix(rel, ".sh") {
			mode = 0o755
		}
		return os.WriteFile(target, b, mode)
	})
}

// rename rewrites the template placeholders to the chosen project name,
// in-process from the embedded copy.
func rename(name string) error {
	return filepath.WalkDir(name, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		// The generated web app carries no @app/ placeholders except the ones we
		// wrote (package name); its node_modules is absent (--no-install), so the
		// whole tree is safe to walk.
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		s := string(b)
		s = strings.ReplaceAll(s, "@app/", "@"+name+"/")
		s = strings.ReplaceAll(s, `"app/`, `"`+name+`/`)
		s = strings.ReplaceAll(s, "module app/", "module "+name+"/")
		// Registry root in .sklp/space.yaml has no trailing service segment
		// (rg.fr-par.scw.cloud/skaplai); rewrite it before the trailing-slash
		// form used by publish image paths (skaplai/core).
		s = strings.ReplaceAll(s, ".scw.cloud/skaplai", ".scw.cloud/"+name)
		s = strings.ReplaceAll(s, "skaplai/", name+"/")
		if s == string(b) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(p, []byte(s), info.Mode())
	})
}

func copyEnvExample(dst string) error {
	src := filepath.Join(dst, ".env.example")
	b, err := os.ReadFile(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.WriteFile(filepath.Join(dst, ".env"), b, 0o600)
}

func gitInit(dir string) error {
	if _, err := exec.LookPath("git"); err != nil {
		return err
	}
	cmd := exec.Command("git", "init", "--quiet")
	cmd.Dir = dir
	return cmd.Run()
}
