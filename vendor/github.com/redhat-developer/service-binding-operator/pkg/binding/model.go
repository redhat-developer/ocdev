package binding

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

type model struct {
	path        []string
	elementType elementType
	objectType  objectType
	sourceKey   string
	sourceValue string
	bindAs      BindingType
}

func (m *model) isStringElementType() bool {
	return m.elementType == stringElementType
}

func (m *model) isStringObjectType() bool {
	return m.objectType == stringObjectType
}

func (m *model) isMapElementType() bool {
	return m.elementType == mapElementType
}

func (m *model) isSliceOfMapsElementType() bool {
	return m.elementType == sliceOfMapsElementType
}

func (m *model) isSliceOfStringsElementType() bool {
	return m.elementType == sliceOfStringsElementType
}

func (m *model) hasDataField() bool {
	return m.objectType == secretObjectType || m.objectType == configMapObjectType
}

func newModel(annotationValue string) (*model, error) {
	// re contains a regular expression to split the input string using '=' and ',' as separators
	re := regexp.MustCompile("[=,]")

	// split holds the tokens extracted from the input string
	split := re.Split(annotationValue, -1)

	// its length should be even, since from this point on is assumed a sequence of key and value
	// pairs as model source
	if len(split)%2 != 0 {
		m := fmt.Sprintf("invalid input, odd number of tokens: %q", split)
		return nil, errors.New(m)
	}

	// extract the tokens into a map, iterating a pair at a time and using the Nth element as key and
	// Nth+1 as value
	raw := make(map[modelKey]string)
	for i := 0; i < len(split); i += 2 {
		k := modelKey(split[i])
		// invalid object type can be created here e.g. "foobar"; this does not pose a problem since
		// the value will be used in a switch statement further on
		v := split[i+1]
		raw[k] = v
	}

	// assert PathModelKey is present
	path, found := raw[pathModelKey]
	if !found {
		return nil, fmt.Errorf("path not found: %q", annotationValue)
	}
	if !strings.HasPrefix(path, "{") || !strings.HasSuffix(path, "}") {
		return nil, fmt.Errorf("path has invalid syntax: %q", path)
	} else {
		// trim curly braces and initial dot
		path = strings.Trim(path, "{}.")
	}

	// ensure ObjectTypeModelKey has a default value
	var objType objectType
	if rawObjectType, found := raw[objectTypeModelKey]; !found {
		objType = stringObjectType
	} else {
		// in the case the key is present but the value isn't (for example, "objectType=,") the
		// default string object type should be set
		if objType = objectType(rawObjectType); objType == emptyObjectType {
			objType = stringObjectType
		}
	}

	// ensure sourceKey has a default value
	sourceKey, found := raw[sourceKeyModelKey]
	if !found {
		sourceKey = ""
	}

	sourceValue, found := raw[sourceValueModelKey]
	if !found {
		sourceValue, found = raw[sourceKeyModelKey]
		if !found {
			sourceValue = ""
		}
	}

	// hasData indicates the configured or inferred objectType is either a Secret or ConfigMap
	hasData := objType == secretObjectType || objType == configMapObjectType
	// hasSourceKey indicates a value for sourceKey has been informed

	var eltType elementType
	if rawEltType, found := raw[elementTypeModelKey]; found {
		// the input string contains an elementType configuration, use it
		eltType = elementType(rawEltType)
	} else if hasData {
		// the input doesn't contain an elementType configuration, does contain a sourceKey
		// configuration, and is either a Secret or ConfigMap
		eltType = mapElementType
	} else {
		// elementType configuration hasn't been informed and there's no extra hints, assume it is a
		// string element
		eltType = stringElementType
	}

	// ensure an error is returned if not all required information is available for sliceOfMaps
	// element type
	if eltType == sliceOfMapsElementType && (len(sourceValue) == 0 || len(sourceKey) == 0) {
		return nil, errors.New("sliceOfMaps elementType requires sourceKey and sourceValue to be present")
	}

	pathParts := strings.Split(path, ".")

	return &model{
		path:        pathParts,
		elementType: eltType,
		objectType:  objType,
		sourceValue: sourceValue,
		sourceKey:   sourceKey,
		bindAs:      TypeEnvVar,
	}, nil
}
