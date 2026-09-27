package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const generatedRoot = `import { HeadContent, Scripts } from '@tanstack/react-router'

function RootDocument({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        {children}
        <Scripts />
      </body>
    </html>
  )
}
`

func TestNakodaAnalyticsIsMountedOnceInTheRootLayout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "__root.tsx")
	if err := os.WriteFile(path, []byte(generatedRoot), 0o644); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := wireNakodaAnalytics(path); err != nil {
			t.Fatal(err)
		}
	}
	b, _ := os.ReadFile(path)
	got := string(b)
	if strings.Count(got, "<NakodaAnalytics") != 1 || strings.Count(got, "from '@lalternative/auth'") != 1 {
		t.Fatalf("wired more or less than once:\n%s", got)
	}
	if strings.Index(got, "<NakodaAnalytics") > strings.Index(got, "<Scripts />") {
		t.Fatalf("NakodaAnalytics must come before <Scripts />:\n%s", got)
	}
}

func TestARootLayoutWithoutScriptsIsAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "__root.tsx")
	_ = os.WriteFile(path, []byte("export const Route = {}\n"), 0o644)
	if err := wireNakodaAnalytics(path); err == nil {
		t.Fatal("a changed CLI output must fail loudly, not skip the wiring")
	}
}
