package binding

import (
	"context"
	"fmt"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"strings"

	"github.com/mitchellh/copystructure"
	"github.com/redhat-developer/service-binding-operator/pkg/nested"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// bindingType encodes the medium the binding should deliver the configuration value.
type BindingType string

const (
	// TypeVolumeMount indicates the binding should happen through a volume mount.
	TypeVolumeMount BindingType = "volumemount"
	// TypeEnvVar indicates the binding should happen through environment variables.
	TypeEnvVar BindingType = "env"
)

// result contains data that has been collected by an annotation handler.
type result struct {
	// Data contains the annotation data collected by an annotation handler inside a deep structure
	// with its root being the value specified in the Path field.
	Data map[string]interface{}
	// Type indicates where the Object field should be injected in the application; can be either
	// "env" or "volumemount".
	Type BindingType
	// Path is the nested location the collected data can be found in the Data field.
	Path string
	// RawData contains the annotation data collected by an annotation handler
	// inside a deep structure with its root being composed by the path where
	// the external resource name was extracted and the path within the external
	// resource.
	RawData map[string]interface{}
}

type errHandlerNotFound string

func (e errHandlerNotFound) Error() string {
	return fmt.Sprintf("could not find handler for annotation value %q", string(e))
}

func IsErrHandlerNotFound(err error) bool {
	_, ok := err.(errHandlerNotFound)
	return ok
}

type SpecHandler struct {
	kubeClient      dynamic.Interface
	obj             unstructured.Unstructured
	annotationKey   string
	annotationValue string
}

func configMapsReader(client dynamic.Interface) UnstructuredResourceReader {
	return func(namespace string, name string) (*unstructured.Unstructured, error) {
		return client.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}).Namespace(namespace).Get(context.TODO(), name, v1.GetOptions{})
	}
}

func secretsReader(client dynamic.Interface) UnstructuredResourceReader {
	return func(namespace string, name string) (*unstructured.Unstructured, error) {
		return client.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}).Namespace(namespace).Get(context.TODO(), name, v1.GetOptions{})
	}
}

func (s *SpecHandler) Handle() (result, error) {
	builder := NewDefinitionBuilder(s.annotationKey, s.annotationValue, configMapsReader(s.kubeClient), secretsReader(s.kubeClient))

	d, err := builder.Build()
	if err != nil {
		return result{}, err
	}

	val, err := d.Apply(&s.obj)
	if err != nil {
		return result{}, err
	}

	v := val.Get()

	path := strings.Join(d.GetPath(), ".")

	out := make(map[string]interface{})

	switch t := v.(type) {
	case map[string]string:
		for k, v := range t {
			out[k] = v
		}
	case map[string]interface{}:
		for k, v := range t {
			out[k] = v
		}
	}

	cpy, err := copystructure.Copy(out)
	if err != nil {
		return result{}, err
	}

	rawData := nested.ComposeValue(cpy, nested.NewPath(path))

	return result{
		Data:    out,
		RawData: rawData,
	}, nil
}

func NewSpecHandler(
	kubeClient dynamic.Interface,
	annotationKey string,
	annotationValue string,
	obj unstructured.Unstructured,
) (*SpecHandler, error) {
	return &SpecHandler{
		kubeClient:      kubeClient,
		obj:             obj,
		annotationKey:   annotationKey,
		annotationValue: annotationValue,
	}, nil
}
