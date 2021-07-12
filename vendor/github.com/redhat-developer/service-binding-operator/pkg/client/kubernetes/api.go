package kubernetes

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ConfigMapReader interface {
	ReadConfigMap(namespace string, name string) (*unstructured.Unstructured, error)
}

type SecretReader interface {
	ReadSecret(namespace string, name string) (*unstructured.Unstructured, error)
}

type Referable interface {
	GroupVersionResource() (*schema.GroupVersionResource, error)
	GroupVersionKind() (*schema.GroupVersionKind, error)
}
