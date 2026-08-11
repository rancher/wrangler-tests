package framework

import (
	"context"
	"time"

	corecontrollers "github.com/rancher/wrangler/v3/pkg/generated/controllers/core"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Env struct {
	Client    kubernetes.Interface
	Wrangler  *corecontrollers.Factory
	Namespace string
}

func NewEnv(kubeConfigPath string, namespace string) (*Env, error) {
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	factory, err := corecontrollers.NewFactoryFromConfig(cfg)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	factory.Start(ctx, 2)

	time.Sleep(2 * time.Second)

	return &Env{
		Client:    clientset,
		Wrangler:  factory,
		Namespace: namespace,
	}, nil
}
