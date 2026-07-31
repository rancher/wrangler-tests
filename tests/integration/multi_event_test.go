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

func TestConfigMapReceivesMultipleUpdates(t *testing.T) {
	h := framework.NewHarness(t)

	name := fmt.Sprintf("multi-update-%d", time.Now().UnixNano())

	var count atomic.Int32

	h.Factory.Core().V1().ConfigMap().OnChange(
		h.Ctx,
		"multi-update",
		func(key string, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm != nil && cm.Name == name {
				count.Add(1)
			}
			return cm, nil
		},
	)

	h.CreateConfigMap(name, nil)

	for i := 0; i < 3; i++ {
		h.UpdateConfigMap(name, func(cm *corev1.ConfigMap) {
			if cm.Annotations == nil {
				cm.Annotations = map[string]string{}
			}
			cm.Annotations["rev"] = fmt.Sprintf("%d", i)
		})
	}

	require.Eventually(t, func() bool {
		return count.Load() >= 4
	}, 10*time.Second, 200*time.Millisecond)
}
