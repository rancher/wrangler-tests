package integration

import (
	"fmt"
	"sync"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/rancher/wrangler-tests/tests/integration/framework"
	"github.com/stretchr/testify/require"
)

func TestConfigMapEventOrdering(t *testing.T) {
	h := framework.NewHarness(t)

	name := fmt.Sprintf("event-order-test-%d", time.Now().UnixNano())

	var mu sync.Mutex
	var sequence []string

	h.Factory.Core().V1().ConfigMap().OnChange(
		h.Ctx,
		"event-order-watch",
		func(key string, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm == nil {
				mu.Lock()
				sequence = append(sequence, "delete")
				mu.Unlock()
				return cm, nil
			}

			if cm.Name != name {
				return cm, nil
			}

			mu.Lock()
			switch cm.Data["state"] {
			case "created":
				sequence = append(sequence, "create")
			case "updated":
				sequence = append(sequence, "update")
			}
			mu.Unlock()

			return cm, nil
		},
	)

	h.CreateConfigMap(name, map[string]string{
		"state": "created",
	})

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(sequence) >= 1
	}, 10*time.Second, 100*time.Millisecond, "create event never observed")

	h.UpdateConfigMap(name, func(cm *corev1.ConfigMap) {
		cm.Data["state"] = "updated"
	})

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(sequence) >= 2
	}, 10*time.Second, 100*time.Millisecond, "update event never observed")

	h.DeleteConfigMap(name)

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(sequence) >= 3
	}, 10*time.Second, 100*time.Millisecond, "delete event never observed")

	mu.Lock()
	final := append([]string(nil), sequence...)
	mu.Unlock()

	require.Equal(t, []string{"create", "update", "delete"}, final,
		"event callbacks fired out of order")
}
