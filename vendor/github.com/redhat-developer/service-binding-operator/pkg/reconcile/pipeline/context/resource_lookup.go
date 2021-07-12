package context

import (
	"github.com/redhat-developer/service-binding-operator/pkg/client/kubernetes"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type resourceLookup struct {
	restMapper meta.RESTMapper
}

func ResourceLookup(restMapper meta.RESTMapper) K8STypeLookup {
	return &resourceLookup{
		restMapper: restMapper,
	}
}

func (i *resourceLookup) ResourceForReferable(obj kubernetes.Referable) (*schema.GroupVersionResource, error) {
	gvr, err := obj.GroupVersionResource()
	if err == nil {
		return gvr, nil
	}
	gvk, err := obj.GroupVersionKind()
	if err != nil {
		return nil, err
	}
	return i.ResourceForKind(*gvk)
}

func (i *resourceLookup) ResourceForKind(gvk schema.GroupVersionKind) (*schema.GroupVersionResource, error) {
	mapping, err := i.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}
	return &mapping.Resource, nil
}

func (i *resourceLookup) KindForResource(gvr schema.GroupVersionResource) (*schema.GroupVersionKind, error) {
	gvk, err := i.restMapper.KindFor(gvr)
	return &gvk, err
}
