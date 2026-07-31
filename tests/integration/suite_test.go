package integration

import (
	"os"
	"testing"
)

// TestMain is reserved for suite-wide integration test setup/teardown.
func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}