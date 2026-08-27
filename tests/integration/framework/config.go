package framework

import (
	"flag"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	Kubeconfig string
	Namespace  = "wrangler-tests"
)

func init() {
	flag.StringVar(
		&Kubeconfig,
		"kubeconfig",
		"",
		"Path to kubeconfig (optional, uses default kubeconfig if empty)",
	)
}

func LoadKubeConfig() (*rest.Config, error) {
	if Kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", Kubeconfig)
	}

	if kube := os.Getenv("KUBECONFIG"); kube != "" {
		return clientcmd.BuildConfigFromFlags("", kube)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	kubeconfig := filepath.Join(home, ".kube", "config")
	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}

func NewRawClient() (*kubernetes.Clientset, error) {
	cfg, err := LoadKubeConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(cfg)
}
