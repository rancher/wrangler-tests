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

func TestSecretInformerReceivesCreate(t *testing.T) {
	h := framework.NewHarness(t)

	name := fmt.Sprintf("secret-integration-test-%d", time.Now().UnixNano())

	var observed atomic.Bool

	h.Factory.Core().V1().Secret().OnChange(
		h.Ctx,
		"secret-create-watch",
		func(key string, s *corev1.Secret) (*corev1.Secret, error) {
			if s != nil && s.Name == name {
				observed.Store(true)
			}
			return s, nil
		},
	)

	h.CreateSecret(name, nil)

	require.Eventually(t, func() bool {
		return observed.Load()
	}, 10*time.Second, 200*time.Millisecond)
}

func TestSecretInformerReceivesUpdate(t *testing.T) {
	h := framework.NewHarness(t)

	name := fmt.Sprintf("secret-integration-test-%d", time.Now().UnixNano())

	var observed atomic.Bool

	h.Factory.Core().V1().Secret().OnChange(
		h.Ctx,
		"secret-update-watch",
		func(key string, s *corev1.Secret) (*corev1.Secret, error) {
			if s != nil &&
				s.Name == name &&
				s.Annotations["updated"] == "true" {
				observed.Store(true)
			}
			return s, nil
		},
	)

	h.CreateSecret(name, nil)

	h.UpdateSecret(name, func(s *corev1.Secret) {
		if s.Annotations == nil {
			s.Annotations = map[string]string{}
		}
		s.Annotations["updated"] = "true"
	})

	require.Eventually(t, func() bool {
		return observed.Load()
	}, 10*time.Second, 200*time.Millisecond)
}

func TestSecretInformerReceivesDelete(t *testing.T) {
	h := framework.NewHarness(t)

	name := fmt.Sprintf("secret-integration-test-%d", time.Now().UnixNano())

	var deleted atomic.Bool

	h.Factory.Core().V1().Secret().OnChange(
		h.Ctx,
		"secret-delete-watch",
		func(key string, s *corev1.Secret) (*corev1.Secret, error) {
			if s == nil {
				deleted.Store(true)
			}
			return s, nil
		},
	)

	h.CreateSecret(name, nil)
	h.DeleteSecret(name)

	require.Eventually(t, func() bool {
		return deleted.Load()
	}, 10*time.Second, 200*time.Millisecond)
}

func TestSecretCacheGet(t *testing.T) {
	h := framework.NewHarness(t)

	name := fmt.Sprintf("secret-cache-test-%d", time.Now().UnixNano())

	h.CreateSecret(name, map[string][]byte{
		"test": []byte("value"),
	})

	require.Eventually(t, func() bool {
		_, err := h.SecretCache.Get(h.Namespace, name)
		return err == nil
	}, 20*time.Second, 200*time.Millisecond,
		fmt.Sprintf("secret %s never appeared in cache", name),
	)
}
