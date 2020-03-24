package labels

import (
	applabels "github.com/openshift/odo/pkg/application/labels"
)

// ComponentLabel is a label key used to identify the component name
const ComponentLabel = "app.kubernetes.io/instance"

// ComponentTypeLabel is Kubernetes label that identifies the type of a component being used
const ComponentTypeLabel = "app.kubernetes.io/name"

// ComponentTypeVersion is a Kubernetes label that identifies the component version
const ComponentTypeVersion = "app.openshift.io/runtime-version"

// GetLabels return labels that should be applied to every object for given component in active application
// additional labels are used only for creating object
// if you are creating something use additional=true
// if you need labels to filter component then use additional=false
func GetLabels(componentName string, applicationName string, additional bool) map[string]string {
	labels := applabels.GetLabels(applicationName, additional)
	labels[ComponentLabel] = componentName
	return labels
}
