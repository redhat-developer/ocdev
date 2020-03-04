package meta

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// TypeMeta creates v1.TypeMeta
func TypeMeta(kind, apiVersion string) v1.TypeMeta {
	return v1.TypeMeta{
		Kind:       kind,
		APIVersion: apiVersion,
	}
}

// ObjectMeta creates v1.ObjectMeta
func ObjectMeta(n types.NamespacedName) v1.ObjectMeta {
	return v1.ObjectMeta{
		Namespace: n.Namespace,
		Name:      n.Name,
	}
}

type objectMetaFunc func(om *v1.ObjectMeta)

// CreateObjectMeta creates v1.ObjectMeta from ns and name
func CreateObjectMeta(ns, name string, opts ...objectMetaFunc) v1.ObjectMeta {
	om := v1.ObjectMeta{
		Name:      name,
		Namespace: ns,
	}
	for _, o := range opts {
		o(&om)
	}
	return om
}

// NamespacedName creates types.NamespacedName
func NamespacedName(ns, name string) types.NamespacedName {
	return types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}
}
