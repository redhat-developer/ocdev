package binding

import (
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"strings"

	"github.com/pkg/errors"
)

type UnstructuredResourceReader func(namespace string, name string) (*unstructured.Unstructured, error)

type secretConfigMapReader struct {
	configMapReader UnstructuredResourceReader
	secretReader    UnstructuredResourceReader
}

type annotationBackedDefinitionBuilder struct {
	*secretConfigMapReader
	name  string
	value string
}

var _ DefinitionBuilder = (*annotationBackedDefinitionBuilder)(nil)

type modelKey string

const (
	pathModelKey        modelKey = "path"
	objectTypeModelKey  modelKey = "objectType"
	sourceKeyModelKey   modelKey = "sourceKey"
	sourceValueModelKey modelKey = "sourceValue"
	elementTypeModelKey modelKey = "elementType"
	AnnotationPrefix             = "service.binding"
)

func NewDefinitionBuilder(annotationName string, annotationValue string, configMapReader UnstructuredResourceReader, secretReader UnstructuredResourceReader) *annotationBackedDefinitionBuilder {
	return &annotationBackedDefinitionBuilder{
		name:  annotationName,
		value: annotationValue,
		secretConfigMapReader: &secretConfigMapReader{
			configMapReader: configMapReader,
			secretReader:    secretReader,
		},
	}
}

func (m *annotationBackedDefinitionBuilder) outputName() (string, error) {
	// bail out in the case the annotation name doesn't start with "service.binding"
	if m.name != AnnotationPrefix && !strings.HasPrefix(m.name, AnnotationPrefix+"/") {
		return "", fmt.Errorf("can't process annotation with name %q", m.name)
	}

	if p := strings.SplitN(m.name, "/", 2); len(p) > 1 && len(p[1]) > 0 {
		return p[1], nil
	}

	return "", nil
}

func (m *annotationBackedDefinitionBuilder) Build() (Definition, error) {

	outputName, err := m.outputName()
	if err != nil {
		return nil, err
	}

	mod, err := newModel(m.value)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create binding model for annotation key %s and value %s", m.name, m.value)
	}

	if len(outputName) == 0 {
		outputName = mod.path[len(mod.path)-1]
	}

	switch {
	case mod.isStringElementType() && mod.isStringObjectType():
		return &stringDefinition{
			outputName: outputName,
			path:       mod.path,
		}, nil

	case mod.isStringElementType() && mod.hasDataField():
		return &stringFromDataFieldDefinition{
			secretConfigMapReader: m.secretConfigMapReader,
			objectType:            mod.objectType,
			outputName:            outputName,
			path:                  mod.path,
			sourceKey:             mod.sourceKey,
		}, nil

	case mod.isMapElementType() && mod.hasDataField():
		return &mapFromDataFieldDefinition{
			secretConfigMapReader: m.secretConfigMapReader,
			objectType:            mod.objectType,
			outputName:            outputName,
			path:                  mod.path,
			sourceValue:           mod.sourceValue,
		}, nil

	case mod.isMapElementType() && mod.isStringObjectType():
		return &stringOfMapDefinition{
			outputName: outputName,
			path:       mod.path,
		}, nil

	case mod.isSliceOfMapsElementType():
		return &sliceOfMapsFromPathDefinition{
			outputName:  outputName,
			path:        mod.path,
			sourceKey:   mod.sourceKey,
			sourceValue: mod.sourceValue,
		}, nil

	case mod.isSliceOfStringsElementType():
		return &sliceOfStringsFromPathDefinition{
			outputName:  outputName,
			path:        mod.path,
			sourceValue: mod.sourceValue,
		}, nil
	}

	panic(fmt.Sprintf("Annotation %s=%s not implemented!", m.name, m.value))
}
