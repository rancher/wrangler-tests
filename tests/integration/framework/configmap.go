package framework

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	corecontrollers "github.com/rancher/wrangler/pkg/generated/controllers/core"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type EventType string

const (
	EventCreate EventType = "create"
	EventUpdate EventType = "update"
	EventDelete EventType = "delete"
)

type ConfigMapEventInput struct {
	T         *testing.T
	Name      string
	Event     EventType
	MatchFunc func(*corev1.ConfigMap) bool
	Mutate    func(*corev1.ConfigMap)
	Timeout   time.Duration
}

func WaitForConfigMapEvent(factory *corecontrollers.Factory, in ConfigMapEventInput) {
	in.T.Helper()

	var observed atomic.Bool

	ctx, cancel := context.WithTimeout(context.Background(), in.Timeout)
	defer cancel()

	cmClient := factory.Core().V1().ConfigMap()

	cmClient.OnChange(
		ctx,
		"deterministic-test-"+in.Name,
		func(_ string, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm != nil && in.MatchFunc(cm) {
				observed.Store(true)
			}
			return cm, nil
		},
	)

	cache := cmClient.Cache()

	require.Eventually(in.T, func() bool {
		_, err := cache.List("wrangler-tests", labels.Everything())
		return err == nil
	}, 10*time.Second, 200*time.Millisecond, "cache not ready")

	client := factory.Core().V1().ConfigMap()

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      in.Name,
			Namespace: "wrangler-tests",
		},
	}

	_, err := client.Create(cm)
	require.NoError(in.T, err)

	if in.Event == EventUpdate && in.Mutate != nil {
		require.Eventually(in.T, func() bool {
			obj, err := client.Get("wrangler-tests", in.Name, metav1.GetOptions{})
			if err != nil {
				return false
			}

			obj = obj.DeepCopy()
			in.Mutate(obj)

			_, err = client.Update(obj)
			return err == nil
		}, 5*time.Second, 200*time.Millisecond)
	}

	if in.Event == EventDelete {
		_ = client.Delete("wrangler-tests", in.Name, &metav1.DeleteOptions{})
	}

	require.Eventually(in.T, func() bool {
		return observed.Load()
	}, in.Timeout, 200*time.Millisecond, "expected ConfigMap event")
}
