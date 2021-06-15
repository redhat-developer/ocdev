package project

import (
	applabels "github.com/openshift/odo/pkg/application/labels"
	"github.com/openshift/odo/pkg/util"
)

// GetSelector returns a selector to filter resource under the current project created by odo
func GetSelector() string {
	labels := map[string]string{
		applabels.ManagedBy: "odo",
	}

	return util.ConvertLabelsToSelector(labels)
}

func GetNonOdoSelector() string {
	labels := map[string]string{
		applabels.ManagedBy: "!odo",
	}

	return util.ConvertLabelsToSelector(labels)
}
