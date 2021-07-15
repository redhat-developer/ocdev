package context

import (
	"github.com/redhat-developer/service-binding-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"reflect"
)

const defaultContainerPath = "spec.template.spec.containers"

type application struct {
	gvr               *schema.GroupVersionResource
	persistedResource *unstructured.Unstructured
	resource          *unstructured.Unstructured
	bindingPath       *v1alpha1.BindingPath
}

func (a *application) SecretPath() string {
	if a.bindingPath != nil {
		return a.bindingPath.SecretPath
	}
	return ""
}

func (a *application) Resource() *unstructured.Unstructured {
	if a.resource == nil {
		a.resource = a.persistedResource.DeepCopy()
	}
	return a.resource
}

func (a *application) ContainersPath() string {
	if a.bindingPath == nil || a.bindingPath.ContainersPath == "" {
		return defaultContainerPath
	}
	return a.bindingPath.ContainersPath
}

func (a *application) IsUpdated() bool {
	return !reflect.DeepEqual(a.persistedResource, a.resource)
}
