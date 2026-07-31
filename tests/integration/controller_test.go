package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rancher/wrangler-tests/tests/integration/framework"
	"github.com/stretchr/testify/require"
)

func TestConfigMapTriggersSecretCreation(t *testing.T) {
	h := framework.NewHarness(t)

	name := fmt.Sprintf("controller-test-%d", time.Now().UnixNano())

	h.Factory.Core().V1().ConfigMap().OnChange(
		context.Background(),
		"cm-to-secret",
		func(key string, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {

			if cm != nil && cm.Namespace == h.Namespace && cm.Name == name {
				t.Logf("configmap handler saw: %s", cm.Name)

				t.Logf("creating secret: %s-secret", name)

				_, err := h.Client.CoreV1().Secrets(h.Namespace).Create(
					h.Ctx,
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      name + "-secret",
							Namespace: h.Namespace,
						},
					},
					metav1.CreateOptions{},
				)

				if err != nil {
					t.Logf("secret create error: %v", err)
				}
			}

			return cm, nil
		},
	)

	h.Factory.Core().V1().Secret().OnChange(
		context.Background(),
		"secret-watch",
		func(key string, s *corev1.Secret) (*corev1.Secret, error) {

			if s != nil && s.Namespace == h.Namespace && s.Name == name+"-secret" {
				t.Logf("secret watcher saw: %s", s.Name)
			}

			return s, nil
		},
	)

	t.Log("creating manual secret")

	_, err := h.Client.CoreV1().Secrets(h.Namespace).Create(
		h.Ctx,
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name + "-manual",
				Namespace: h.Namespace,
			},
		},
		metav1.CreateOptions{},
	)
	require.NoError(t, err)

	t.Log("creating configmap")

	h.CreateConfigMap(name, nil)

	require.Eventually(t, func() bool {
		_, err := h.Client.CoreV1().Secrets(h.Namespace).Get(
			h.Ctx,
			name+"-secret",
			metav1.GetOptions{},
		)
		return err == nil
	}, 10*time.Second, 200*time.Millisecond)
}