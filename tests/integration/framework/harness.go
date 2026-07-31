package framework

import (
	"context"
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	corecontrollers "github.com/rancher/wrangler/pkg/generated/controllers/core"
	v1controllers "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"

	"github.com/stretchr/testify/require"
)

type Harness struct {
	T         *testing.T
	Ctx       context.Context
	Cancel    context.CancelFunc
	Client    *kubernetes.Clientset
	Factory   *corecontrollers.Factory
	Cache     v1controllers.ConfigMapCache
	Namespace string
}

func NewHarness(t *testing.T) *Harness {
	t.Helper()

	cfg, err := LoadKubeConfig()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	clientset, err := kubernetes.NewForConfig(cfg)
	require.NoError(t, err)

	factory, err := corecontrollers.NewFactoryFromConfig(cfg)
	require.NoError(t, err)

	h := &Harness{
		T:         t,
		Ctx:       ctx,
		Cancel:    cancel,
		Client:    clientset,
		Factory:   factory,
		Namespace: Namespace,
	}

	cmController := factory.Core().V1().ConfigMap()

	cmController.OnChange(
		ctx,
		"startup-debug",
		func(key string, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			return cm, nil
		},
	)

	factory.Start(ctx, 2)

	require.Eventually(t, func() bool {
		return cmController.Informer().HasSynced()
	}, 10*time.Second, 100*time.Millisecond, "ConfigMap informer never synced")

	h.Cache = cmController.Cache()

	return h
}

func (h *Harness) CreateConfigMap(name string, data map[string]string) *corev1.ConfigMap {
	cm, err := h.Client.CoreV1().ConfigMaps(h.Namespace).Create(
		h.Ctx,
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: h.Namespace,
			},
			Data: data,
		},
		metav1.CreateOptions{},
	)
	require.NoError(h.T, err)
	return cm
}

func (h *Harness) UpdateConfigMap(name string, mutate func(*corev1.ConfigMap)) {
	client := h.Client.CoreV1().ConfigMaps(h.Namespace)

	cm, err := client.Get(h.Ctx, name, metav1.GetOptions{})
	require.NoError(h.T, err)

	cm = cm.DeepCopy()
	mutate(cm)

	_, err = client.Update(h.Ctx, cm, metav1.UpdateOptions{})
	require.NoError(h.T, err)
}

func (h *Harness) DeleteConfigMap(name string) {
	err := h.Client.CoreV1().ConfigMaps(h.Namespace).Delete(
		h.Ctx,
		name,
		metav1.DeleteOptions{},
	)
	require.NoError(h.T, err)
}

func (h *Harness) EventuallyInCache(name string) {
	require.Eventually(h.T, func() bool {
		_, err := h.Cache.Get(h.Namespace, name)
		return err == nil
	}, 20*time.Second, 200*time.Millisecond,
		fmt.Sprintf("configmap %s never appeared in cache", name),
	)
}