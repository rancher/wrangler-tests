package framework

import (
	"context"
	"time"

	"k8s.io/client-go/kubernetes"

	corecontrollers "github.com/rancher/wrangler/pkg/generated/controllers/core"
)

type WranglerContext struct {
	Context   context.Context
	Cancel    context.CancelFunc
	Clientset *kubernetes.Clientset
	Factory   *corecontrollers.Factory
}

func StartWrangler(setup func(*corecontrollers.Factory)) (*WranglerContext, error) {
	cfg, err := LoadKubeConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	factory, err := corecontrollers.NewFactoryFromConfig(cfg)
	if err != nil {
		cancel()
		return nil, err
	}

	factory.Start(ctx, 2)

	setup(factory)

	time.Sleep(2 * time.Second)

	return &WranglerContext{
		Context:   ctx,
		Cancel:    cancel,
		Clientset: clientset,
		Factory:   factory,
	}, nil
}
