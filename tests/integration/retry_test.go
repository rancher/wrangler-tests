package integration

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/rancher/wrangler-tests/tests/integration/framework"
	"github.com/stretchr/testify/require"
)

func TestControllerRetriesOnHandlerError(t *testing.T) {
	h := framework.NewHarness(t)

	name := fmt.Sprintf("retry-test-%d", time.Now().UnixNano())

	var attempts atomic.Int32
	var succeeded atomic.Bool
	var mu sync.Mutex
	var attemptLog []int32

	h.Factory.Core().V1().ConfigMap().OnChange(
		h.Ctx,
		"retry-watch",
		func(key string, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm == nil || cm.Name != name {
				return cm, nil
			}

			n := attempts.Add(1)

			mu.Lock()
			attemptLog = append(attemptLog, n)
			mu.Unlock()

			if n == 1 {
				return nil, fmt.Errorf("simulated transient error on first attempt")
			}

			succeeded.Store(true)
			return cm, nil
		},
	)

	h.CreateConfigMap(name, map[string]string{"test": "value"})

	require.Eventually(t, func() bool {
		return succeeded.Load()
	}, 15*time.Second, 200*time.Millisecond,
		"handler never succeeded -- framework did not retry after the first error")

	mu.Lock()
	log := append([]int32(nil), attemptLog...)
	mu.Unlock()

	require.GreaterOrEqual(t, len(log), 2,
		"expected at least 2 attempts (1 failure + 1+ retries), got %d: %v", len(log), log)
	require.Equal(t, int32(1), log[0], "first recorded attempt should be attempt #1")

	h.EventuallyInCache(name)

	t.Logf("handler required %d attempt(s) before succeeding: %v", attempts.Load(), log)
}
