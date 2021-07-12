package service

import (
	"fmt"
	"path/filepath"

	"github.com/openshift/odo/pkg/odo/cli/component"
	"github.com/openshift/odo/pkg/util"
)

// validDevfileDirectory returns an error if the "odo service" command is executed from a directory not containing devfile.yaml
func validDevfileDirectory(componentContext string) error {
	if componentContext == "" {
		componentContext = component.LocalDirectoryDefaultLocation
	}
	devfilePath := filepath.Join(componentContext, component.DevfilePath)
	if !util.CheckPathExists(devfilePath) {
		return fmt.Errorf("service can be created/deleted from a valid component directory only\n"+
			"refer %q for more information", "odo servce create -h")
	}
	return nil
}
