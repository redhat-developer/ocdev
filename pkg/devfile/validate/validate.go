package validate

import (
	"fmt"

	adaptersCommon "github.com/openshift/odo/pkg/devfile/adapters/common"
	"github.com/openshift/odo/pkg/devfile/parser/data"
	"github.com/openshift/odo/pkg/devfile/parser/data/common"
	"github.com/openshift/odo/pkg/util"
	"k8s.io/klog"

	v200 "github.com/openshift/odo/pkg/devfile/parser/data/2.0.0"
)

// ValidateDevfileData validates whether sections of devfile are odo compatible
func ValidateDevfileData(data interface{}) error {
	var components []common.DevfileComponent
	var commands map[string]common.DevfileCommand
	var events common.DevfileEvents

	switch d := data.(type) {
	case *v200.Devfile200:
		components = d.GetComponents()
		commands = d.GetCommands()
		events = d.GetEvents()

		// Validate all the devfile components before validating commands
		if err := validateComponents(components); err != nil {
			return err
		}

		// Validate all the devfile commands before validating events
		if err := validateCommands(commands, components); err != nil {
			return err
		}

		// Validate all the events
		if err := validateEvents(events, commands, components); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown devfile type %T", d)
	}

	// Successful
	klog.V(2).Info("Successfully validated devfile sections")
	return nil

}

// ValidateContainerName validates whether the container name is valid for K8
func ValidateContainerName(devfileData data.DevfileData) error {
	containerComponents := adaptersCommon.GetDevfileContainerComponents(devfileData)
	for _, comp := range containerComponents {
		err := util.ValidateK8sResourceName("container name", comp.Name)
		if err != nil {
			return err
		}
	}
	return nil
}
