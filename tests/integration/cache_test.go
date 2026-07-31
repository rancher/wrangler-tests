package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/rancher/wrangler-tests/tests/integration/framework"
)

func TestConfigMapCacheGet(t *testing.T) {
	h := framework.NewHarness(t)

	name := fmt.Sprintf("cache-test-%d", time.Now().UnixNano())

	h.CreateConfigMap(name, map[string]string{
		"test": "value",
	})

	h.EventuallyInCache(name)
}
