package context

import (
	"context"
	e "errors"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/redhat-developer/service-binding-operator/api/v1alpha1"
	"github.com/redhat-developer/service-binding-operator/pkg/binding"
	"github.com/redhat-developer/service-binding-operator/pkg/reconcile/pipeline"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"reflect"
)

var _ pipeline.Service = &service{}

var crdGVRs = []schema.GroupVersionResource{
	{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	},
	{
		Group:    "apiextensions.k8s.io",
		Version:  "v1beta1",
		Resource: "customresourcedefinitions",
	},
}

var bindableResourceGVRs = []schema.GroupVersionResource{
	{Group: "", Version: "v1", Resource: "configmaps"},
	{Group: "", Version: "v1", Resource: "secrets"},
	{Group: "", Version: "v1", Resource: "services"},
	{Group: "route.openshift.io", Version: "v1", Resource: "routes"},
}

type service struct {
	client                dynamic.Interface
	serviceRef            *v1alpha1.Service
	resource              *unstructured.Unstructured
	groupVersionResource  *schema.GroupVersionResource
	crd                   *customResourceDefinition
	crdLookup             bool
	lookForOwnedResources bool
	bindingDefinitions    []binding.Definition
}

func (s *service) OwnedResources() ([]*unstructured.Unstructured, error) {
	uid := s.Resource().GetUID()
	var result []*unstructured.Unstructured
	if !s.lookForOwnedResources {
		return result, nil
	}
	for _, gvr := range bindableResourceGVRs {
		list, err := s.client.Resource(gvr).Namespace(*s.serviceRef.Namespace).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			return nil, err
		}
		for i := range list.Items {
			item := list.Items[i]
			for _, ownerRef := range item.GetOwnerReferences() {
				if reflect.DeepEqual(ownerRef.UID, uid) {
					result = append(result, &item)
				}
			}
		}
	}
	return result, nil
}

func (s *service) Id() *string {
	return s.serviceRef.Id
}

func (s *service) Resource() *unstructured.Unstructured {
	return s.resource
}

func (s *service) CustomResourceDefinition() (pipeline.CRD, error) {
	if s.crd == nil {
		if s.crdLookup {
			return nil, nil
		}
		var err error
		var u *unstructured.Unstructured
		for _, crd := range crdGVRs {
			u, err = s.client.Resource(crd).Get(context.Background(), s.groupVersionResource.GroupResource().String(), metav1.GetOptions{})
			if err == nil {
				s.crd = &customResourceDefinition{resource: u, client: s.client, ns: *s.serviceRef.Namespace, serviceGVR: s.groupVersionResource}
				return s.crd, nil
			}
		}
		if errors.IsNotFound(err) {
			s.crdLookup = true
			return nil, nil
		}
		return nil, err
	}
	return s.crd, nil
}

func (s *service) AddBindingDef(def binding.Definition) {
	s.bindingDefinitions = append(s.bindingDefinitions, def)
}

func (s *service) BindingDefs() []binding.Definition {
	return s.bindingDefinitions
}

type customResourceDefinition struct {
	resource   *unstructured.Unstructured
	serviceGVR *schema.GroupVersionResource
	client     dynamic.Interface
	ns         string
}

func (c *customResourceDefinition) Resource() *unstructured.Unstructured {
	return c.resource
}

func (c *customResourceDefinition) kind() string {
	val, found, _ := unstructured.NestedString(c.resource.Object, "spec", "names", "kind")
	if found {
		return val
	}
	return ""
}

func (c *customResourceDefinition) Descriptor() (*olmv1alpha1.CRDDescription, error) {
	csvs, err := c.client.Resource(olmv1alpha1.SchemeGroupVersion.WithResource("clusterserviceversions")).Namespace(c.ns).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	if len(csvs.Items) == 0 {
		return nil, nil
	}
	for _, csv := range csvs.Items {
		ownedPath := []string{"spec", "customresourcedefinitions", "owned"}

		ownedCRDs, exists, err := unstructured.NestedSlice(csv.Object, ownedPath...)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}

		for _, crd := range ownedCRDs {
			crdDesciption := &olmv1alpha1.CRDDescription{}
			data, ok := crd.(map[string]interface{})
			if !ok {
				return nil, e.New("cannot cast to map")
			}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(data, crdDesciption)
			if err != nil {
				return nil, err
			}
			if crdDesciption.Name == c.Resource().GetName() && crdDesciption.Kind == c.kind() && crdDesciption.Version == c.serviceGVR.Version {
				return crdDesciption, nil
			}
		}
	}
	return nil, nil
}
