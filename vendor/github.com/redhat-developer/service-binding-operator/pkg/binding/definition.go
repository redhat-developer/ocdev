package binding

import (
	"encoding/base64"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type objectType string

type elementType string

const (
	// configMapObjectType indicates the path contains a name for a ConfigMap containing the binding
	// data.
	configMapObjectType objectType = "ConfigMap"
	// secretObjectType indicates the path contains a name for a Secret containing the binding data.
	secretObjectType objectType = "Secret"
	// stringObjectType indicates the path contains a value string.
	stringObjectType objectType = "string"
	// emptyObjectType is used as default value when the objectType key is present in the string
	// provided by the user but no value has been provided; can be used by the user to force the
	// system to use the default objectType.
	emptyObjectType objectType = ""

	// mapElementType indicates the value found at path is a map[string]interface{}.
	mapElementType elementType = "map"
	// sliceOfMapsElementType indicates the value found at path is a slice of maps.
	sliceOfMapsElementType elementType = "sliceOfMaps"
	// sliceOfStringsElementType indicates the value found at path is a slice of strings.
	sliceOfStringsElementType elementType = "sliceOfStrings"
	// stringElementType indicates the value found at path is a string.
	stringElementType elementType = "string"
)

//go:generate mockgen -destination=mocks/mocks.go -package=mocks . Definition,Value

type Definition interface {
	GetPath() []string
	Apply(u *unstructured.Unstructured) (Value, error)
}

type DefinitionBuilder interface {
	Build() (Definition, error)
}

type stringDefinition struct {
	outputName string
	path       []string
}

var _ Definition = (*stringDefinition)(nil)

func (d *stringDefinition) getOutputName() string {
	outputName := d.outputName
	if len(outputName) == 0 {
		outputName = d.path[len(d.path)-1]
	}
	return outputName
}

func (d *stringDefinition) GetPath() []string { return d.path[0 : len(d.path)-1] }

func (d *stringDefinition) Apply(u *unstructured.Unstructured) (Value, error) {
	val, ok, err := unstructured.NestedFieldCopy(u.Object, d.path...)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("not found")
	}

	m := map[string]interface{}{
		d.getOutputName(): fmt.Sprintf("%v", val),
	}

	return &value{v: m}, nil
}

type stringFromDataFieldDefinition struct {
	secretConfigMapReader *secretConfigMapReader
	objectType            objectType
	outputName            string
	path                  []string
	sourceKey             string
}

var _ Definition = (*stringFromDataFieldDefinition)(nil)

func (d *stringFromDataFieldDefinition) GetPath() []string { return d.path }

func (d *stringFromDataFieldDefinition) Apply(u *unstructured.Unstructured) (Value, error) {
	if d.secretConfigMapReader == nil {
		return nil, errors.New("kubeClient required for this functionality")
	}

	resourceName, ok, err := unstructured.NestedString(u.Object, d.path...)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("not found")
	}

	var otherObj *unstructured.Unstructured
	if d.objectType == secretObjectType {
		otherObj, err = d.secretConfigMapReader.secretReader(u.GetNamespace(), resourceName)
	} else if d.objectType == configMapObjectType {
		otherObj, err = d.secretConfigMapReader.configMapReader(u.GetNamespace(), resourceName)
	}

	if err != nil {
		return nil, err
	}

	val, ok, err := unstructured.NestedString(otherObj.Object, "data", d.sourceKey)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("not found")
	}
	if d.objectType == secretObjectType {
		n, err := base64.StdEncoding.DecodeString(val)
		if err != nil {
			return nil, err
		}
		val = string(n)
	}
	v := map[string]interface{}{
		"": val,
	}
	return &value{v: v}, nil
}

type mapFromDataFieldDefinition struct {
	secretConfigMapReader *secretConfigMapReader
	objectType            objectType
	outputName            string
	sourceValue           string
	path                  []string
}

var _ Definition = (*mapFromDataFieldDefinition)(nil)

func (d *mapFromDataFieldDefinition) GetPath() []string { return d.path }

func (d *mapFromDataFieldDefinition) Apply(u *unstructured.Unstructured) (Value, error) {
	if d.secretConfigMapReader == nil {
		return nil, errors.New("kubeClient required for this functionality")
	}

	resourceName, ok, err := unstructured.NestedString(u.Object, d.path...)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("not found")
	}

	var otherObj *unstructured.Unstructured
	if d.objectType == secretObjectType {
		otherObj, err = d.secretConfigMapReader.secretReader(u.GetNamespace(), resourceName)
	} else if d.objectType == configMapObjectType {
		otherObj, err = d.secretConfigMapReader.configMapReader(u.GetNamespace(), resourceName)
	}

	if err != nil {
		return nil, err
	}

	val, ok, err := unstructured.NestedStringMap(otherObj.Object, "data")
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("not found")
	}

	outputVal := make(map[string]string)

	for k, v := range val {
		if len(d.sourceValue) > 0 && k != d.sourceValue {
			continue
		}
		var n string
		if d.objectType == secretObjectType {
			b, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				return nil, err
			}
			n = string(b)
		} else {
			n = v
		}
		if len(d.sourceValue) > 0 && len(d.outputName) > 0 {
			outputVal[d.outputName] = string(n)
		} else {
			outputVal[k] = string(n)
		}
	}

	return &value{v: outputVal}, nil
}

type stringOfMapDefinition struct {
	outputName string
	path       []string
}

var _ Definition = (*stringOfMapDefinition)(nil)

func (d *stringOfMapDefinition) GetPath() []string { return d.path }

func (d *stringOfMapDefinition) Apply(u *unstructured.Unstructured) (Value, error) {
	val, ok, err := unstructured.NestedFieldNoCopy(u.Object, d.path...)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("not found")
	}

	outputName := d.outputName
	if len(outputName) == 0 {
		outputName = d.path[len(d.path)-1]
	}
	v := map[string]interface{}{
		outputName: val,
	}
	return &value{v: v}, nil

}

type sliceOfMapsFromPathDefinition struct {
	outputName  string
	path        []string
	sourceKey   string
	sourceValue string
}

var _ Definition = (*sliceOfMapsFromPathDefinition)(nil)

func (d *sliceOfMapsFromPathDefinition) GetPath() []string { return d.path[0 : len(d.path)-1] }

func (d *sliceOfMapsFromPathDefinition) Apply(u *unstructured.Unstructured) (Value, error) {
	val, ok, err := unstructured.NestedSlice(u.Object, d.path...)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("not found")
	}

	v := make(map[string]interface{})
	for _, e := range val {
		if mm, ok := e.(map[string]interface{}); ok {
			key := mm[d.sourceKey]
			ks := key.(string)
			value := mm[d.sourceValue]
			v[ks] = value
		}
	}

	return &value{v: map[string]interface{}{d.outputName: v}}, nil
}

type sliceOfStringsFromPathDefinition struct {
	outputName  string
	path        []string
	sourceValue string
}

var _ Definition = (*sliceOfStringsFromPathDefinition)(nil)

func (d *sliceOfStringsFromPathDefinition) GetPath() []string { return d.path[0 : len(d.path)-1] }

func (d *sliceOfStringsFromPathDefinition) Apply(u *unstructured.Unstructured) (Value, error) {
	val, ok, err := unstructured.NestedSlice(u.Object, d.path...)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("not found")
	}

	v := make([]interface{}, 0, len(val))
	for _, e := range val {
		if d.sourceValue != "" {
			if mm, ok := e.(map[string]interface{}); ok {
				sourceValue := mm[d.sourceValue].(string)
				v = append(v, sourceValue)
			}
		} else {
			if x, ok := e.(string); ok {
				v = append(v, x)
			}
		}

	}

	return &value{v: map[string]interface{}{d.outputName: v}}, nil
}
