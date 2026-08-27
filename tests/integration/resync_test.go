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

func TestConfigMapMissedEventRecovery(t *testing.T) {
	ctx := context.Background()

	client, err := framework.NewRawClient()
	require.NoError(t, err)

	name := fmt.Sprintf("resync-test-%d", time.Now().UnixNano())

	_, err = client.CoreV1().ConfigMaps(framework.Namespace).Create(
		ctx,
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: framework.Namespace,
			},
			Data: map[string]string{
				"state": "created",
			},
		},
		metav1.CreateOptions{},
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = client.CoreV1().ConfigMaps(framework.Namespace).Delete(ctx, name, metav1.DeleteOptions{})
	})

	existing, err := client.CoreV1().ConfigMaps(framework.Namespace).Get(ctx, name, metav1.GetOptions{})
	require.NoError(t, err)

	existing = existing.DeepCopy()
	existing.Data["state"] = "updated-before-watch"

	_, err = client.CoreV1().ConfigMaps(framework.Namespace).Update(ctx, existing, metav1.UpdateOptions{})
	require.NoError(t, err)

	h := framework.NewHarness(t)

	cm, err := h.Cache.Get(h.Namespace, name)
	require.NoError(t, err, "configmap %s missing from cache after sync", name)
	require.Equal(t, "updated-before-watch", cm.Data["state"],
		"cache reflects stale pre-update state -- the update made before watcher establishment was lost")
}
