package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/rancher/wrangler-tests/tests/integration/framework"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func TestConfigMapCacheListBySelector(t *testing.T) {
	h := framework.NewHarness(t)

	runID := fmt.Sprintf("%d", time.Now().UnixNano())

	targetLabels := map[string]string{
		"group": "target-" + runID,
	}

	targetNames := []string{
		fmt.Sprintf("selector-target-a-%s", runID),
		fmt.Sprintf("selector-target-b-%s", runID),
	}

	nonMatchName := fmt.Sprintf("selector-nonmatch-%s", runID)

	for _, name := range targetNames {
		h.CreateConfigMapWithLabels(name, targetLabels, nil)
	}
	h.CreateConfigMapWithLabels(nonMatchName, map[string]string{
		"group": "other-" + runID,
	}, nil)

	t.Cleanup(func() {
		for _, name := range append(targetNames, nonMatchName) {
			_ = h.Client.CoreV1().ConfigMaps(h.Namespace).Delete(h.Ctx, name, metav1.DeleteOptions{})
		}
	})

	selector := labels.SelectorFromSet(targetLabels)

	var results []string
	require.Eventually(t, func() bool {
		objs, err := h.Cache.List(h.Namespace, selector)
		if err != nil {
			return false
		}

		results = nil
		for _, obj := range objs {
			results = append(results, obj.Name)
		}

		return len(results) >= len(targetNames)
	}, 10*time.Second, 200*time.Millisecond, "cache list never returned expected matching objects")

	require.ElementsMatch(t, targetNames, results,
		"cache list by selector returned wrong set: got %v, want %v", results, targetNames)
}
