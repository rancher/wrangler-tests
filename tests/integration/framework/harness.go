package framework

import (
	"context"
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	admissionregcontrollers "github.com/rancher/wrangler/v3/pkg/generated/controllers/admissionregistration.k8s.io"
	apiextcontrollers "github.com/rancher/wrangler/v3/pkg/generated/controllers/apiextensions.k8s.io"
	corecontrollers "github.com/rancher/wrangler/v3/pkg/generated/controllers/core"
	v1controllers "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/v3/pkg/needacert"

	"github.com/stretchr/testify/require"
)

type Harness struct {
	T           *testing.T
	Ctx         context.Context
	Cancel      context.CancelFunc
	Client      *kubernetes.Clientset
	Factory     *corecontrollers.Factory
	Cache       v1controllers.ConfigMapCache
	SecretCache v1controllers.SecretCache
	Namespace   string
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

	admissionregFactory, err := admissionregcontrollers.NewFactoryFromConfig(cfg)
	require.NoError(t, err)

	apiextFactory, err := apiextcontrollers.NewFactoryFromConfig(cfg)
	require.NoError(t, err)

	h := &Harness{
		T:         t,
		Ctx:       ctx,
		Cancel:    cancel,
		Client:    clientset,
		Factory:   factory,
		Namespace: Namespace,
	}

	secretController := factory.Core().V1().Secret()
	serviceController := factory.Core().V1().Service()
	cmController := factory.Core().V1().ConfigMap()

	cmController.OnChange(
		ctx,
		"startup-debug",
		func(key string, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			return cm, nil
		},
	)

	needacert.Register(
		ctx,
		secretController,
		serviceController,
		admissionregFactory.Admissionregistration().V1().MutatingWebhookConfiguration(),
		admissionregFactory.Admissionregistration().V1().ValidatingWebhookConfiguration(),
		apiextFactory.Apiextensions().V1().CustomResourceDefinition(),
	)

	factory.Start(ctx, 2)
	admissionregFactory.Start(ctx, 2)
	apiextFactory.Start(ctx, 2)

	require.Eventually(t, func() bool {
		return serviceController.Informer().HasSynced() &&
			secretController.Informer().HasSynced() &&
			cmController.Informer().HasSynced()
	}, 10*time.Second, 100*time.Millisecond, "controllers never synced")

	h.Cache = cmController.Cache()
	h.SecretCache = secretController.Cache()

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

func (h *Harness) EventuallyNotInCache(name string) {
	h.T.Helper()
	require.Eventually(h.T, func() bool {
		_, err := h.Cache.Get(h.Namespace, name)
		return err != nil
	}, 20*time.Second, 200*time.Millisecond,
		fmt.Sprintf("configmap %s still present in cache", name),
	)
}

func (h *Harness) ConsistentlyNotInCache(name string, duration time.Duration) {
	h.T.Helper()
	deadline := time.Now().Add(duration)
	for time.Now().Before(deadline) {
		_, err := h.Cache.Get(h.Namespace, name)
		require.Error(h.T, err, "configmap %s reappeared in cache after being deleted", name)
		time.Sleep(200 * time.Millisecond)
	}
}

func (h *Harness) EventuallyInCacheWithData(name, key, value string) {
	h.T.Helper()
	require.Eventually(h.T, func() bool {
		cm, err := h.Cache.Get(h.Namespace, name)
		if err != nil {
			return false
		}
		return cm.Data[key] == value
	}, 20*time.Second, 200*time.Millisecond,
		fmt.Sprintf("configmap %s never showed %s=%s in cache", name, key, value),
	)
}

func (h *Harness) CreateSecret(name string, data map[string][]byte) *corev1.Secret {
	s, err := h.Client.CoreV1().Secrets(h.Namespace).Create(
		h.Ctx,
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: h.Namespace,
			},
			Data: data,
		},
		metav1.CreateOptions{},
	)
	require.NoError(h.T, err)
	return s
}

func (h *Harness) UpdateSecret(name string, mutate func(*corev1.Secret)) {
	client := h.Client.CoreV1().Secrets(h.Namespace)

	s, err := client.Get(h.Ctx, name, metav1.GetOptions{})
	require.NoError(h.T, err)

	s = s.DeepCopy()
	mutate(s)

	_, err = client.Update(h.Ctx, s, metav1.UpdateOptions{})
	require.NoError(h.T, err)
}

func (h *Harness) DeleteSecret(name string) {
	err := h.Client.CoreV1().Secrets(h.Namespace).Delete(
		h.Ctx,
		name,
		metav1.DeleteOptions{},
	)
	require.NoError(h.T, err)
}

func (h *Harness) CreateConfigMapWithLabels(name string, labels map[string]string, data map[string]string) *corev1.ConfigMap {
	cm, err := h.Client.CoreV1().ConfigMaps(h.Namespace).Create(
		h.Ctx,
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: h.Namespace,
				Labels:    labels,
			},
			Data: data,
		},
		metav1.CreateOptions{},
	)
	require.NoError(h.T, err)
	return cm
}
