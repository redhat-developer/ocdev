package converter

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ToUnstructured converts a runtime object into Unstructured, and can return errors related to it.
func ToUnstructured(obj interface{}) (*unstructured.Unstructured, error) {
	data, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: data}, nil
}

// ToUnstructuredAsGVK converts a runtime object into Unstructured, and set as given GVK. It can
// return errors related to conversion.
func ToUnstructuredAsGVK(
	obj interface{},
	gvk schema.GroupVersionKind,
) (*unstructured.Unstructured, error) {
	u, err := ToUnstructured(obj)
	if err != nil {
		return nil, err
	}
	u.SetGroupVersionKind(gvk)
	return u, nil
}
