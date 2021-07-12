package pipeline

import (
	"encoding/base64"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ Bindings = &SecretBackedBindings{}

// bindings whose life-cycle is bound to k8s secret
type SecretBackedBindings struct {
	// service associated to the bindings
	Service Service

	// secret containing the bindings
	// each binding correspond to a (key, value) pair
	Secret *unstructured.Unstructured
	items  BindingItems
}

func (s *SecretBackedBindings) Items() (BindingItems, error) {
	if s.items != nil {
		return s.items, nil
	}
	data, found, err := unstructured.NestedStringMap(s.Secret.Object, "data")
	if err != nil {
		return nil, err
	}
	if found {
		for k, v := range data {
			val, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				return nil, err
			}
			s.items = append(s.items, &BindingItem{
				Name:   k,
				Value:  string(val),
				Source: s.Service,
			})
		}
	} else {
		s.items = make([]*BindingItem, 0)
	}
	return s.items, nil
}

func (s *SecretBackedBindings) Source() *corev1.ObjectReference {
	ref := &corev1.ObjectReference{
		Kind:       s.Secret.GetKind(),
		APIVersion: s.Secret.GetAPIVersion(),
		Name:       s.Secret.GetName(),
		Namespace:  s.Secret.GetNamespace(),
	}
	if s.items == nil {
		return ref
	}
	val, found, err := unstructured.NestedStringMap(s.Secret.Object, "data")
	if err != nil || !found {
		return nil
	}
	for _, item := range s.items {
		if _, ok := val[item.Name]; !ok {
			return nil
		}
	}
	return ref
}
