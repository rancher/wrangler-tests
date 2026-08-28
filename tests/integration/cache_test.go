package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/rancher/wrangler-tests/tests/integration/framework"
	corev1 "k8s.io/api/core/v1"
)

func TestConfigMapCacheGet(t *testing.T) {
	h := framework.NewHarness(t)

	name := fmt.Sprintf("cache-test-%d", time.Now().UnixNano())

	h.CreateConfigMap(name, map[string]string{
		"test": "value",
	})

	h.EventuallyInCache(name)
}

func TestConfigMapCacheRapidChurn(t *testing.T) {
	h := framework.NewHarness(t)

	const iterations = 5

	for i := 0; i < iterations; i++ {
		name := fmt.Sprintf("cache-churn-test-%d-%d", time.Now().UnixNano(), i)

		h.CreateConfigMap(name, map[string]string{
			"state": "created",
		})

		h.UpdateConfigMap(name, func(cm *corev1.ConfigMap) {
			cm.Data["state"] = "updated"
		})

		h.EventuallyInCacheWithData(name, "state", "updated")

		h.DeleteConfigMap(name)

		h.EventuallyNotInCache(name)

		h.ConsistentlyNotInCache(name, 3*time.Second)
	}
}
