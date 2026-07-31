package framework

import (
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetConfig is the single source of truth for kubeconfig loading.
func GetConfig() (*rest.Config, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()

	if Kubeconfig != "" {
		rules.ExplicitPath = Kubeconfig
	}

	overrides := &clientcmd.ConfigOverrides{}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		rules,
		overrides,
	).ClientConfig()
}
