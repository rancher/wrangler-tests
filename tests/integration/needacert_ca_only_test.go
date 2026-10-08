package integration

import (
	"encoding/pem"
	"fmt"
	"testing"
	"time"

	adminregv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rancher/wrangler-tests/tests/integration/framework"
	"github.com/rancher/wrangler/v3/pkg/needacert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNeedACertCABundleModeCAOnly(t *testing.T) {
	requireWranglerVersion37(t)

	h := framework.NewHarness(t)

	name := fmt.Sprintf("needacert-test-%d", time.Now().UnixNano())
	secretName := name + "-tls"
	webhookName := fmt.Sprintf("%s.needacert-test.example.com", name)

	_, err := h.Client.CoreV1().Services(h.Namespace).Create(
		h.Ctx,
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: h.Namespace,
				Annotations: map[string]string{
					needacert.SecretAnnotation:       secretName,
					needacert.CABundleModeAnnotation: needacert.CABundleModeCAOnly,
				},
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{Port: 443, Protocol: corev1.ProtocolTCP},
				},
			},
		},
		metav1.CreateOptions{},
	)
	require.NoError(t, err, "failed to create annotated service")

	var secret *corev1.Secret
	secretReady := assert.Eventually(t, func() bool {
		var getErr error
		secret, getErr = h.Client.CoreV1().Secrets(h.Namespace).Get(
			h.Ctx, secretName, metav1.GetOptions{},
		)
		return getErr == nil && secret != nil && len(secret.Data[corev1.TLSCertKey]) > 0
	}, 15*time.Second, 200*time.Millisecond)
	require.True(t, secretReady, "timed out waiting for needacert to generate the TLS secret")

	failurePolicy := adminregv1.Ignore
	sideEffects := adminregv1.SideEffectClassNone
	scope := adminregv1.NamespacedScope
	webhookPath := "/validate"

	vwc := &adminregv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: webhookName},
		Webhooks: []adminregv1.ValidatingWebhook{
			{
				Name: webhookName,
				ClientConfig: adminregv1.WebhookClientConfig{
					Service: &adminregv1.ServiceReference{
						Namespace: h.Namespace,
						Name:      name,
						Path:      &webhookPath,
					},
				},
				Rules: []adminregv1.RuleWithOperations{
					{
						Operations: []adminregv1.OperationType{adminregv1.Create},
						Rule: adminregv1.Rule{
							APIGroups:   []string{"needacert-test.example.com"},
							APIVersions: []string{"v1"},
							Resources:   []string{"widgets"},
							Scope:       &scope,
						},
					},
				},
				FailurePolicy:           &failurePolicy,
				SideEffects:             &sideEffects,
				AdmissionReviewVersions: []string{"v1"},
			},
		},
	}

	_, err = h.Client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(
		h.Ctx, vwc, metav1.CreateOptions{},
	)
	require.NoError(t, err, "failed to create validating webhook configuration")

	t.Cleanup(func() {
		_ = h.Client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(
			h.Ctx, webhookName, metav1.DeleteOptions{},
		)
	})

	var caBundle []byte
	success := assert.Eventually(t, func() bool {
		current, getErr := h.Client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(
			h.Ctx, webhookName, metav1.GetOptions{},
		)
		if getErr != nil || current == nil || len(current.Webhooks) == 0 {
			return false
		}
		caBundle = current.Webhooks[0].ClientConfig.CABundle
		return len(caBundle) > 0
	}, 15*time.Second, 200*time.Millisecond)

	if !success {
		t.Log("--- DEBUG: CABundle never got populated. Dumping namespace state ---")

		events, eventErr := h.Client.CoreV1().Events(h.Namespace).List(h.Ctx, metav1.ListOptions{})
		if eventErr == nil {
			t.Log("Namespace Events:")
			for _, event := range events.Items {
				t.Logf("  [%s] Reason: %s - %s", event.Type, event.Reason, event.Message)
			}
		} else {
			t.Logf("Failed to list events: %v", eventErr)
		}

		svc, svcErr := h.Client.CoreV1().Services(h.Namespace).Get(h.Ctx, name, metav1.GetOptions{})
		if svcErr == nil {
			t.Logf("Service Annotations: %v", svc.Annotations)
		}

		require.FailNow(t, "timed out waiting for needacert to populate webhook CABundle")
	}

	certCount := countPEMCertificates(caBundle)
	assert.Equal(t, 1, certCount, "ca-only mode must result in exactly 1 certificate block in the webhook CABundle (omitting the leaf certificate)")
}

func countPEMCertificates(data []byte) int {
	count := 0
	rest := data
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type == "CERTIFICATE" {
			count++
		}
	}
	return count
}
