package integration

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rancher/wrangler-tests/tests/integration/framework"
	corecontrollers "github.com/rancher/wrangler/v3/pkg/generated/controllers/core"
	"github.com/stretchr/testify/require"
)

func TestConfigMapWatcherStopsOnContextCancel(t *testing.T) {
	cfg, err := framework.LoadKubeConfig()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	client, err := framework.NewRawClient()
	require.NoError(t, err)

	factory, err := corecontrollers.NewFactoryFromConfig(cfg)
	require.NoError(t, err)

	cmController := factory.Core().V1().ConfigMap()

	var eventCount atomic.Int32

	cmController.OnChange(
		ctx,
		"cleanup-watch",
		func(key string, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			eventCount.Add(1)
			return cm, nil
		},
	)

	factory.Start(ctx, 2)

	require.Eventually(t, func() bool {
		return cmController.Informer().HasSynced()
	}, 10*time.Second, 100*time.Millisecond, "informer never synced")

	name := fmt.Sprintf("cleanup-test-%d", time.Now().UnixNano())

	_, err = client.CoreV1().ConfigMaps(framework.Namespace).Create(
		ctx,
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: framework.Namespace,
			},
		},
		metav1.CreateOptions{},
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = client.CoreV1().ConfigMaps(framework.Namespace).Delete(
			context.Background(), name, metav1.DeleteOptions{},
		)
	})

	require.Eventually(t, func() bool {
		return eventCount.Load() >= 1
	}, 10*time.Second, 200*time.Millisecond, "watcher never observed the initial create")

	countAtCancel := eventCount.Load()
	cancel()

	time.Sleep(1 * time.Second)

	for i := 0; i < 3; i++ {
		cm, getErr := client.CoreV1().ConfigMaps(framework.Namespace).Get(
			context.Background(), name, metav1.GetOptions{},
		)
		require.NoError(t, getErr)

		cm = cm.DeepCopy()
		if cm.Annotations == nil {
			cm.Annotations = map[string]string{}
		}
		cm.Annotations["post-cancel-rev"] = fmt.Sprintf("%d", i)

		_, err = client.CoreV1().ConfigMaps(framework.Namespace).Update(
			context.Background(), cm, metav1.UpdateOptions{},
		)
		require.NoError(t, err)
	}

	time.Sleep(3 * time.Second)

	require.Equal(t, countAtCancel, eventCount.Load(),
		"watcher delivered events after context cancellation -- shutdown was not clean")
}

func TestHarnessResourceCleanup(t *testing.T) {
	h := framework.NewHarness(t)

	name := fmt.Sprintf("cleanup-verify-test-%d", time.Now().UnixNano())

	h.CreateConfigMap(name, map[string]string{"test": "value"})
	h.EventuallyInCache(name)

	h.DeleteConfigMap(name)

	require.Eventually(t, func() bool {
		_, err := h.Client.CoreV1().ConfigMaps(h.Namespace).Get(h.Ctx, name, metav1.GetOptions{})
		return err != nil
	}, 10*time.Second, 200*time.Millisecond,
		"configmap still exists on the API server after cleanup delete")

	h.EventuallyNotInCache(name)
}
