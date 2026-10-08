package integration

import (
	"os/exec"
	"strings"
	"testing"
)

func requireWranglerVersion37(t *testing.T) {
	t.Helper()

	output, err := exec.Command(
		"go", "list", "-m", "-f={{.Version}}",
		"github.com/rancher/wrangler/v3",
	).Output()
	if err != nil {
		t.Fatalf("failed to determine Wrangler version: %v", err)
	}

	version := strings.TrimSpace(string(output))
	if version == "" {
		t.Fatal("Wrangler version is empty")
	}

	// The feature was introduced in Wrangler v3.7.0.
	// Wrangler module versions here are standard v3.x.y releases.
	parts := strings.Split(strings.TrimPrefix(version, "v"), ".")
	if len(parts) < 3 {
		t.Fatalf("unexpected Wrangler version format %q", version)
	}

	major := 0
	minor := 0
	for _, part := range []struct {
		value string
		dest  *int
	}{
		{parts[0], &major},
		{parts[1], &minor},
	} {
		for _, c := range part.value {
			if c < '0' || c > '9' {
				t.Fatalf("unexpected Wrangler version format %q", version)
			}
			*part.dest = *part.dest*10 + int(c-'0')
		}
	}

	if major < 3 || (major == 3 && minor < 7) {
		t.Skipf(
			"skipping needacert CA bundle mode test: Wrangler %s does not support this feature (requires v3.7.0+)",
			version,
		)
	}
}
