package context

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"sort"

	"github.com/redhat-developer/service-binding-operator/api/v1alpha1"
	"github.com/redhat-developer/service-binding-operator/pkg/client/kubernetes"
	"github.com/redhat-developer/service-binding-operator/pkg/converter"
	"github.com/redhat-developer/service-binding-operator/pkg/reconcile/pipeline"
	"github.com/redhat-developer/service-binding-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var _ pipeline.Context = &impl{}

type impl struct {
	serviceBinding *v1alpha1.ServiceBinding

	client dynamic.Interface

	typeLookup K8STypeLookup

	services []*service

	applications []*application

	bindingItems pipeline.BindingItems

	bindings []pipeline.Bindings

	retry bool
	stop  bool
	err   error

	conditions map[string]*metav1.Condition
}

func (i *impl) UnbindRequested() bool {
	return !i.serviceBinding.DeletionTimestamp.IsZero()
}

type provider struct {
	client     dynamic.Interface
	typeLookup K8STypeLookup
}

func (p *provider) Get(sb *v1alpha1.ServiceBinding) (pipeline.Context, error) {
	return &impl{
		conditions:     make(map[string]*metav1.Condition),
		client:         p.client,
		typeLookup:     p.typeLookup,
		serviceBinding: sb,
	}, nil
}

var Provider = func(client dynamic.Interface, typeLookup K8STypeLookup) pipeline.ContextProvider {
	return &provider{
		client:     client,
		typeLookup: typeLookup,
	}
}

func (i *impl) BindingName() string {
	return i.serviceBinding.Name
}

func (i *impl) MountPath() string {
	return i.serviceBinding.Spec.MountPath
}

func (i *impl) FlowStatus() pipeline.FlowStatus {
	return pipeline.FlowStatus{
		Retry: i.retry,
		Stop:  i.stop,
		Err:   i.err,
	}
}

func (i *impl) Mappings() map[string]string {
	result := make(map[string]string)
	for _, m := range i.serviceBinding.Spec.Mappings {
		result[m.Name] = m.Value
	}
	return result
}

func (i *impl) Services() ([]pipeline.Service, error) {
	if i.services == nil {
		serviceRefs := i.serviceBinding.Spec.Services
		for idx := 0; idx < len(serviceRefs); idx++ {
			serviceRef := serviceRefs[idx]
			gvr, err := i.typeLookup.ResourceForReferable(&serviceRef)
			if err != nil {
				return nil, err
			}
			if serviceRef.Namespace == nil {
				serviceRef.Namespace = &i.serviceBinding.Namespace
			}
			u, err := i.client.Resource(*gvr).Namespace(*serviceRef.Namespace).Get(context.Background(), serviceRef.Name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			i.services = append(i.services, &service{client: i.client, resource: u, groupVersionResource: gvr, serviceRef: &serviceRef, lookForOwnedResources: i.serviceBinding.Spec.DetectBindingResources})
		}
	}
	services := make([]pipeline.Service, len(i.services))
	for idx := 0; idx < len(i.services); idx++ {
		services[idx] = i.services[idx]
	}
	return services, nil
}

func (i *impl) Applications() ([]pipeline.Application, error) {
	if i.applications == nil {
		if ref := i.serviceBinding.Spec.Application; ref != nil {
			gvr, err := i.typeLookup.ResourceForReferable(ref)
			if err != nil {
				return nil, err
			}
			if i.serviceBinding.Spec.Application.Name != "" {
				u, err := i.client.Resource(*gvr).Namespace(i.serviceBinding.Namespace).Get(context.Background(), ref.Name, metav1.GetOptions{})
				if err != nil {
					return nil, err
				}
				i.applications = append(i.applications, &application{gvr: gvr, persistedResource: u, bindingPath: i.serviceBinding.Spec.Application.BindingPath})
			}
			if i.serviceBinding.Spec.Application.LabelSelector != nil && i.serviceBinding.Spec.Application.LabelSelector.MatchLabels != nil {
				matchLabels := i.serviceBinding.Spec.Application.LabelSelector.MatchLabels
				opts := metav1.ListOptions{
					LabelSelector: labels.Set(matchLabels).String(),
				}

				emptyResult := make([]pipeline.Application, 0)
				objList, err := i.client.Resource(*gvr).Namespace(i.serviceBinding.Namespace).List(context.Background(), opts)
				if err != nil {
					return emptyResult, err
				}

				if len(objList.Items) == 0 {
					return emptyResult, nil
				}

				for index := range objList.Items {
					i.applications = append(i.applications, &application{gvr: gvr, persistedResource: &(objList.Items[index]), bindingPath: i.serviceBinding.Spec.Application.BindingPath})
				}
			}
		} else {
			i.applications = make([]*application, 0)
		}
	}
	result := make([]pipeline.Application, len(i.applications))
	for l, a := range i.applications {
		result[l] = a
	}
	return result, nil
}

func (i *impl) AddBindingItem(item *pipeline.BindingItem) {
	i.bindingItems = append(i.bindingItems, item)
}

func (i *impl) BindingItems() pipeline.BindingItems {
	var allItems pipeline.BindingItems
	for _, b := range i.bindings {
		items, err := b.Items()
		if err != nil {
			continue
		}
		allItems = append(allItems, items...)
	}
	if len(i.bindingItems) > 0 {
		allItems = append(allItems, i.bindingItems...)
	}
	return allItems
}

func (i *impl) BindingSecretName() string {
	name, _ := i.bindingSecretName()
	return name
}

func (i *impl) bindingSecretName() (string, bool) {
	if i.UnbindRequested() {
		return i.serviceBinding.Status.Secret, true
	}
	if i.bindingItems == nil && len(i.bindings) == 1 {
		ref := i.bindings[0].Source()
		if ref != nil && ref.Namespace == i.serviceBinding.GetNamespace() {
			return ref.Name, true
		}
	}
	data := i.bindingItemMap()
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	hash := sha1.New()
	for _, k := range keys {
		_, _ = hash.Write([]byte(k))
		_, _ = hash.Write([]byte(data[k]))
	}
	return i.serviceBinding.Name + "-" + string(hex.EncodeToString(hash.Sum(nil))[:8]), false
}

func (i *impl) bindingItemMap() map[string]string {
	data := make(map[string]string)
	for _, b := range i.bindings {
		items, err := b.Items()
		if err != nil {
			continue
		}
		util.MergeMaps(data, items.AsMap())
	}
	if len(i.bindingItems) > 0 {
		util.MergeMaps(data, i.bindingItems.AsMap())
	}
	return data
}

func (i *impl) NamingTemplate() string {
	return i.serviceBinding.Spec.NamingTemplate()
}

func (i *impl) BindAsFiles() bool {
	return i.serviceBinding.Spec.BindAsFiles
}

func (i *impl) RetryProcessing(reason error) {
	i.retry = true
	i.stop = true
	i.err = reason
}

func (i *impl) Error(err error) {
	i.err = err
}

func (i *impl) StopProcessing() {
	i.stop = true
}

func (i *impl) persistBinding() error {
	if i.serviceBinding.UID == "" {
		return nil
	}
	for _, c := range i.conditions {
		meta.SetStatusCondition(&i.serviceBinding.Status.Conditions, *c)
	}
	u, err := converter.ToUnstructured(i.serviceBinding)
	if err != nil {
		return err
	}
	client := i.client.Resource(v1alpha1.GroupVersionResource).Namespace(i.serviceBinding.Namespace)
	_, err = client.UpdateStatus(context.Background(), u, metav1.UpdateOptions{})
	return err
}

func (i *impl) persistSecret() (string, error) {
	name, secretExist := i.bindingSecretName()
	if secretExist {
		return name, nil
	}
	data := i.bindingItemMap()
	if len(data) == 0 {
		return "", nil
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: i.serviceBinding.Namespace,
			Name:      name,
		},
		StringData: data,
	}
	if i.serviceBinding.UID != "" {
		secret.OwnerReferences = []metav1.OwnerReference{i.serviceBinding.AsOwnerReference()}
	}
	u, err := converter.ToUnstructuredAsGVK(secret, corev1.SchemeGroupVersion.WithKind("Secret"))
	if err != nil {
		return name, err
	}

	secretClient := i.client.Resource(corev1.SchemeGroupVersion.WithResource("secrets")).Namespace(i.serviceBinding.Namespace)

	_, err = secretClient.Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = secretClient.Create(context.Background(), u, metav1.CreateOptions{})
			return name, err
		}
		return name, err
	}
	_, err = secretClient.Update(context.Background(), u, metav1.UpdateOptions{})
	return name, err
}

func (i *impl) Close() error {
	if i.err != nil {
		i.SetCondition(v1alpha1.Conditions().NotBindingReady().Reason("ProcessingError").Msg(i.err.Error()).Build())
		return i.persistBinding()
	}
	secretName, err := i.persistSecret()
	if err != nil {
		i.SetCondition(v1alpha1.Conditions().NotBindingReady().Reason("ErrorPersistingSecret").Msg(err.Error()).Build())
		_ = i.persistBinding()
		return err
	}
	if secretName != "" {
		i.serviceBinding.Status.Secret = secretName
	}
	for _, app := range i.applications {
		if app.IsUpdated() {
			_, err = i.client.Resource(*app.gvr).Namespace(i.serviceBinding.Namespace).Update(context.Background(), app.Resource(), metav1.UpdateOptions{})
			if err != nil {
				i.SetCondition(v1alpha1.Conditions().NotBindingReady().Reason("ApplicationUpdateError").Msg(err.Error()).Build())
				_ = i.persistBinding()
				return err
			}
		}
	}
	i.SetCondition(v1alpha1.Conditions().BindingReady().Reason("ApplicationsBound").Build())
	return i.persistBinding()
}

func (i *impl) SetCondition(condition *metav1.Condition) {
	i.conditions[condition.Type] = condition
}

func (i *impl) ReadConfigMap(namespace string, name string) (*unstructured.Unstructured, error) {
	return i.client.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}).Namespace(namespace).Get(context.Background(), name, metav1.GetOptions{})
}

func (i *impl) ReadSecret(namespace string, name string) (*unstructured.Unstructured, error) {
	return i.client.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}).Namespace(namespace).Get(context.Background(), name, metav1.GetOptions{})
}

func (i *impl) AddBindings(bindings pipeline.Bindings) {
	i.bindings = append(i.bindings, bindings)
}

//go:generate mockgen -destination=mocks/mocks.go -package=mocks . K8STypeLookup

type K8STypeLookup interface {
	ResourceForReferable(obj kubernetes.Referable) (*schema.GroupVersionResource, error)
	ResourceForKind(gvk schema.GroupVersionKind) (*schema.GroupVersionResource, error)
	KindForResource(gvr schema.GroupVersionResource) (*schema.GroupVersionKind, error)
}
