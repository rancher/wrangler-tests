package integration

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/rancher/wrangler-tests/tests/integration/framework"
	"github.com/stretchr/testify/require"
)

func TestConfigMapHandlerIndependence(t *testing.T) {
	h := framework.NewHarness(t)

	name := fmt.Sprintf("handler-independence-test-%d", time.Now().UnixNano())

	var failingHandlerCalls atomic.Int32
	var healthyHandlerObserved atomic.Bool

	h.Factory.Core().V1().ConfigMap().OnChange(
		h.Ctx,
		"handler-independence-failing",
		func(key string, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm != nil && cm.Name == name {
				failingHandlerCalls.Add(1)
				return cm, fmt.Errorf("simulated persistent handler failure")
			}
			return cm, nil
		},
	)

	h.Factory.Core().V1().ConfigMap().OnChange(
		h.Ctx,
		"handler-independence-healthy",
		func(key string, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm != nil && cm.Name == name {
				healthyHandlerObserved.Store(true)
			}
			return cm, nil
		},
	)

	h.CreateConfigMap(name, map[string]string{"test": "value"})

	require.Eventually(t, func() bool {
		return healthyHandlerObserved.Load()
	}, 10*time.Second, 200*time.Millisecond,
		"healthy handler never observed the event -- a failing sibling handler may be blocking it")

	require.Eventually(t, func() bool {
		return failingHandlerCalls.Load() >= 1
	}, 10*time.Second, 200*time.Millisecond,
		"failing handler was never invoked -- test setup issue, not a real pass")

	h.EventuallyInCache(name)
}
