package integration

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rancher/wrangler-tests/tests/integration/framework"
	"github.com/stretchr/testify/require"
)

func TestControllerIdempotentOnRepeatEvents(t *testing.T) {
	h := framework.NewHarness(t)

	name := fmt.Sprintf("idempotency-test-%d", time.Now().UnixNano())
	secretName := name + "-secret"

	var handlerRuns atomic.Int32
	var createAttempts atomic.Int32
	var alreadyExistsCount atomic.Int32
	var unexpectedErrCount atomic.Int32

	h.Factory.Core().V1().ConfigMap().OnChange(
		h.Ctx,
		"idempotency-watch",
		func(key string, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm == nil || cm.Name != name {
				return cm, nil
			}
			handlerRuns.Add(1)

			createAttempts.Add(1)
			_, err := h.Client.CoreV1().Secrets(h.Namespace).Create(
				h.Ctx,
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: h.Namespace,
					},
				},
				metav1.CreateOptions{},
			)

			if apierrors.IsAlreadyExists(err) {
				alreadyExistsCount.Add(1)
				return cm, nil
			}
			if err != nil {
				unexpectedErrCount.Add(1)
				return cm, err
			}
			return cm, nil
		},
	)

	h.CreateConfigMap(name, map[string]string{"rev": "0"})

	for i := 1; i <= 4; i++ {
		h.UpdateConfigMap(name, func(cm *corev1.ConfigMap) {
			if cm.Data == nil {
				cm.Data = map[string]string{}
			}
			cm.Data["rev"] = fmt.Sprintf("%d", i)
		})
	}

	require.Eventually(t, func() bool {
		return handlerRuns.Load() >= 5 // 1 create + 4 updates
	}, 10*time.Second, 200*time.Millisecond, "handler did not observe all repeat events")

	require.Zero(t, unexpectedErrCount.Load(),
		"handler hit an unexpected (non-AlreadyExists) error while creating the secret")

	secrets, err := h.Client.CoreV1().Secrets(h.Namespace).List(h.Ctx, metav1.ListOptions{
		FieldSelector: "metadata.name=" + secretName,
	})
	require.NoError(t, err)
	require.Len(t, secrets.Items, 1,
		"expected exactly one dependent secret, found %d -- controller created duplicates on repeat events",
		len(secrets.Items))

	t.Logf("handler ran %d times, attempted create %d times (%d already-existed), resulting in %d secret(s)",
		handlerRuns.Load(), createAttempts.Load(), alreadyExistsCount.Load(), len(secrets.Items))

	t.Cleanup(func() {
		_ = h.Client.CoreV1().Secrets(h.Namespace).Delete(h.Ctx, secretName, metav1.DeleteOptions{})
	})
}
