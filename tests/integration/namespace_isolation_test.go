package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/rancher/wrangler-tests/tests/integration/framework"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConfigMapNamespaceIsolation(t *testing.T) {
	h := framework.NewHarness(t)

	name := fmt.Sprintf("ns-isolation-test-%d", time.Now().UnixNano())
	secondNamespace := fmt.Sprintf("wrangler-tests-second-%d", time.Now().UnixNano())

	_, err := h.Client.CoreV1().Namespaces().Create(
		h.Ctx,
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: secondNamespace},
		},
		metav1.CreateOptions{},
	)
	require.NoError(t, err, "failed to create second namespace")

	t.Cleanup(func() {
		_ = h.Client.CoreV1().ConfigMaps(secondNamespace).Delete(h.Ctx, name, metav1.DeleteOptions{})
		_ = h.Client.CoreV1().Namespaces().Delete(h.Ctx, secondNamespace, metav1.DeleteOptions{})
	})

	type observedEvent struct {
		namespace string
		data      string
	}
	observed := make(chan observedEvent, 10)

	h.Factory.Core().V1().ConfigMap().OnChange(
		h.Ctx,
		"ns-isolation-watch",
		func(key string, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm != nil && cm.Name == name {
				observed <- observedEvent{namespace: cm.Namespace, data: cm.Data["source"]}
			}
			return cm, nil
		},
	)

	h.CreateConfigMap(name, map[string]string{
		"source": h.Namespace,
	})

	require.Eventually(t, func() bool {
		_, err := h.Client.CoreV1().ConfigMaps(secondNamespace).Create(
			h.Ctx,
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: secondNamespace,
				},
				Data: map[string]string{
					"source": secondNamespace,
				},
			},
			metav1.CreateOptions{},
		)
		return err == nil
	}, 5*time.Second, 200*time.Millisecond, "failed to create configmap in second namespace")

	require.Eventually(t, func() bool {
		cm, err := h.Cache.Get(h.Namespace, name)
		return err == nil && cm.Data["source"] == h.Namespace
	}, 10*time.Second, 200*time.Millisecond, "default-namespace configmap missing or wrong data in cache")

	require.Eventually(t, func() bool {
		cm, err := h.Cache.Get(secondNamespace, name)
		return err == nil && cm.Data["source"] == secondNamespace
	}, 10*time.Second, 200*time.Millisecond, "second-namespace configmap missing or wrong data in cache")

	seen := map[string]bool{}
	timeout := time.After(10 * time.Second)
	for len(seen) < 2 {
		select {
		case e := <-observed:
			require.Equal(t, e.namespace, e.data,
				"handler observed configmap %s/%s carrying data from a different namespace", e.namespace, name)
			seen[e.namespace] = true
		case <-timeout:
			t.Fatalf("timed out waiting for handler to observe both namespaced configmaps, saw: %v", seen)
		}
	}
}
