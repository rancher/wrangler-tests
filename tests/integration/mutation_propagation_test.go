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

func TestConfigMapLabelMutationPropagation(t *testing.T) {
	h := framework.NewHarness(t)

	name := fmt.Sprintf("label-mutation-test-%d", time.Now().UnixNano())

	var observed atomic.Bool

	h.Factory.Core().V1().ConfigMap().OnChange(
		h.Ctx,
		"label-mutation-watch",
		func(key string, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm != nil && cm.Name == name && cm.Labels["team"] == "platform" {
				observed.Store(true)
			}
			return cm, nil
		},
	)

	h.CreateConfigMap(name, nil)

	h.UpdateConfigMap(name, func(cm *corev1.ConfigMap) {
		if cm.Labels == nil {
			cm.Labels = map[string]string{}
		}
		cm.Labels["team"] = "platform"
	})

	require.Eventually(t, func() bool {
		return observed.Load()
	}, 10*time.Second, 200*time.Millisecond, "handler never observed label mutation")

	require.Eventually(t, func() bool {
		cm, err := h.Cache.Get(h.Namespace, name)
		return err == nil && cm.Labels["team"] == "platform"
	}, 10*time.Second, 200*time.Millisecond, "cache never reflected label mutation")
}

func TestConfigMapAnnotationMutationPropagation(t *testing.T) {
	h := framework.NewHarness(t)

	name := fmt.Sprintf("annotation-mutation-test-%d", time.Now().UnixNano())

	var observed atomic.Bool

	h.Factory.Core().V1().ConfigMap().OnChange(
		h.Ctx,
		"annotation-mutation-watch",
		func(key string, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm != nil && cm.Name == name && cm.Annotations["owner"] == "wrangler-tests" {
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
		cm.Annotations["owner"] = "wrangler-tests"
	})

	require.Eventually(t, func() bool {
		return observed.Load()
	}, 10*time.Second, 200*time.Millisecond, "handler never observed annotation mutation")

	require.Eventually(t, func() bool {
		cm, err := h.Cache.Get(h.Namespace, name)
		return err == nil && cm.Annotations["owner"] == "wrangler-tests"
	}, 10*time.Second, 200*time.Millisecond, "cache never reflected annotation mutation")
}

func TestConfigMapDataMutationPropagation(t *testing.T) {
	h := framework.NewHarness(t)

	name := fmt.Sprintf("data-mutation-test-%d", time.Now().UnixNano())

	var observed atomic.Bool

	h.Factory.Core().V1().ConfigMap().OnChange(
		h.Ctx,
		"data-mutation-watch",
		func(key string, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm != nil && cm.Name == name && cm.Data["config.yaml"] == "replicas: 3" {
				observed.Store(true)
			}
			return cm, nil
		},
	)

	h.CreateConfigMap(name, map[string]string{
		"config.yaml": "replicas: 1",
	})

	h.UpdateConfigMap(name, func(cm *corev1.ConfigMap) {
		if cm.Data == nil {
			cm.Data = map[string]string{}
		}
		cm.Data["config.yaml"] = "replicas: 3"
	})

	require.Eventually(t, func() bool {
		return observed.Load()
	}, 10*time.Second, 200*time.Millisecond, "handler never observed data mutation")

	require.Eventually(t, func() bool {
		cm, err := h.Cache.Get(h.Namespace, name)
		return err == nil && cm.Data["config.yaml"] == "replicas: 3"
	}, 10*time.Second, 200*time.Millisecond, "cache never reflected data mutation")
}
