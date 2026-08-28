package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rancher/wrangler-tests/tests/integration/framework"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConfigMapInformerInitialList(t *testing.T) {
	ctx := context.Background()

	client, err := framework.NewRawClient()
	require.NoError(t, err)

	const preExistingCount = 3
	names := make([]string, preExistingCount)

	for i := 0; i < preExistingCount; i++ {
		name := fmt.Sprintf("initial-list-test-%d-%d", time.Now().UnixNano(), i)
		names[i] = name

		_, err := client.CoreV1().ConfigMaps(framework.Namespace).Create(
			ctx,
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: framework.Namespace,
				},
				Data: map[string]string{
					"preexisting": "true",
				},
			},
			metav1.CreateOptions{},
		)
		require.NoError(t, err)
	}

	t.Cleanup(func() {
		for _, name := range names {
			_ = client.CoreV1().ConfigMaps(framework.Namespace).Delete(ctx, name, metav1.DeleteOptions{})
		}
	})

	h := framework.NewHarness(t)

	for _, name := range names {
		cm, err := h.Cache.Get(h.Namespace, name)
		require.NoError(t, err, "pre-existing configmap %s missing from cache after initial sync", name)
		require.Equal(t, "true", cm.Data["preexisting"], "cached configmap %s has unexpected data", name)
	}
}
