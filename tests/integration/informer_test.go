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

func TestConfigMapInformerReceivesCreate(t *testing.T) {
	h := framework.NewHarness(t)

	name := fmt.Sprintf("integration-test-%d", time.Now().UnixNano())

	var observed atomic.Bool

	h.Factory.Core().V1().ConfigMap().OnChange(
		h.Ctx,
		"create-watch",
		func(key string, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm != nil && cm.Name == name {
				observed.Store(true)
			}
			return cm, nil
		},
	)

	h.CreateConfigMap(name, nil)

	require.Eventually(t, func() bool {
		return observed.Load()
	}, 10*time.Second, 200*time.Millisecond)
}

func TestConfigMapInformerReceivesUpdate(t *testing.T) {
	h := framework.NewHarness(t)

	name := fmt.Sprintf("integration-test-%d", time.Now().UnixNano())

	var observed atomic.Bool

	h.Factory.Core().V1().ConfigMap().OnChange(
		h.Ctx,
		"update-watch",
		func(key string, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm != nil &&
				cm.Name == name &&
				cm.Annotations["updated"] == "true" {
				observed.Store(true)
			}
			return cm, nil
		},
	)

	h.CreateConfigMap(name, nil)

	h.UpdateConfigMap(name, func(cm *corev1.ConfigMap) {
		if cm.Annotations == nil {
			cm.Annotations = map[string]string{}
		}
		cm.Annotations["updated"] = "true"
	})

	require.Eventually(t, func() bool {
		return observed.Load()
	}, 10*time.Second, 200*time.Millisecond)
}

func TestConfigMapInformerReceivesDelete(t *testing.T) {
	h := framework.NewHarness(t)

	name := fmt.Sprintf("integration-test-%d", time.Now().UnixNano())

	var deleted atomic.Bool

	h.Factory.Core().V1().ConfigMap().OnChange(
		h.Ctx,
		"delete-watch",
		func(key string, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm == nil {
				deleted.Store(true)
			}
			return cm, nil
		},
	)

	h.CreateConfigMap(name, nil)
	h.DeleteConfigMap(name)

	require.Eventually(t, func() bool {
		return deleted.Load()
	}, 10*time.Second, 200*time.Millisecond)
}