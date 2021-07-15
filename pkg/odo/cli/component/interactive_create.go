package component

import (
	"os"

	devfilev1 "github.com/devfile/api/v2/pkg/apis/workspaces/v1alpha2"
	"github.com/openshift/odo/pkg/catalog"
	"github.com/openshift/odo/pkg/config"
	catalogutil "github.com/openshift/odo/pkg/odo/cli/catalog/util"
	"github.com/openshift/odo/pkg/odo/cli/component/ui"
	commonui "github.com/openshift/odo/pkg/odo/cli/ui"
	"github.com/openshift/odo/pkg/util"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

func (co *CreateOptions) SetComponentSettingsInteractively(catalogList catalog.ComponentTypeList) error {

	componentTypeCandidates := catalogutil.FilterHiddenComponents(catalogList.Items)
	selectedComponentType := ui.SelectComponentType(componentTypeCandidates)
	selectedImageTag := ui.SelectImageTag(componentTypeCandidates, selectedComponentType)
	componentType := selectedComponentType + ":" + selectedImageTag
	co.componentSettings.Type = &componentType

	// Ask for the type of source if not provided
	selectedSourceType := ui.SelectSourceType([]config.SrcType{config.LOCAL, config.GIT, config.BINARY})
	co.componentSettings.SourceType = &selectedSourceType
	selectedSourcePath := LocalDirectoryDefaultLocation

	// Get the current directory
	currentDirectory, err := os.Getwd()
	if err != nil {
		return err
	}

	if selectedSourceType == config.BINARY {

		// We ask for the source of the component context
		co.componentContext = ui.EnterInputTypePath("context", currentDirectory, ".")
		klog.V(4).Infof("Context: %s", co.componentContext)

		// If it's a binary, we have to ask where the actual binary in relation
		// to the context
		selectedSourcePath = ui.EnterInputTypePath("binary", ".")

		// Get the correct source location
		sourceLocation, err := getSourceLocation(selectedSourcePath, co.componentContext)
		if err != nil {
			return errors.Wrapf(err, "unable to get source location")
		}
		co.componentSettings.SourceLocation = &sourceLocation

	} else if selectedSourceType == config.GIT {

		// For git, we ask for the Git URL and set that as the source location
		cmpSrcLOC, selectedGitRef := ui.EnterGitInfo()
		co.componentSettings.SourceLocation = &cmpSrcLOC
		co.componentSettings.Ref = &selectedGitRef

	} else if selectedSourceType == config.LOCAL {

		// We ask for the source of the component, in this case the "path"!
		co.componentContext = ui.EnterInputTypePath("path", currentDirectory, ".")

		// Get the correct source location
		if co.componentContext == "" {
			co.componentContext = LocalDirectoryDefaultLocation
		}
		co.componentSettings.SourceLocation = &co.componentContext

	}

	defaultComponentName, err := createDefaultComponentName(selectedComponentType, selectedSourceType, selectedSourcePath)
	if err != nil {
		return err
	}
	componentName := ui.EnterComponentName(defaultComponentName, co.Context)

	appName := ui.EnterOpenshiftName(co.Context.Application, "Which application do you want the component to be associated with", co.Context)
	co.componentSettings.Application = &appName

	projectName := ui.EnterOpenshiftName(co.Context.Project, "Which project go you want the component to be created in", co.Context)
	co.componentSettings.Project = &projectName

	co.componentSettings.Name = &componentName

	var ports []string

	if commonui.Proceed("Do you wish to set advanced options") {
		// if the user doesn't opt for advanced options, ports field would remain unpopulated
		// we then set it at the end of this function
		ports = ui.EnterPorts()

		co.componentEnvVars = ui.EnterEnvVars()

		if commonui.Proceed("Do you wish to set resource limits") {
			memMax := ui.EnterMemory("maximum", "512Mi")
			memMin := ui.EnterMemory("minimum", memMax)
			cpuMax := ui.EnterCPU("maximum", "1")
			cpuMin := ui.EnterCPU("minimum", cpuMax)

			memoryQuantity, err := util.FetchResourceQuantity(corev1.ResourceMemory, memMin, memMax, "")
			if err != nil {
				return err
			}
			if memoryQuantity != nil {
				co.componentSettings.MinMemory = &memMin
				co.componentSettings.MaxMemory = &memMax
			}
			cpuQuantity, err := util.FetchResourceQuantity(corev1.ResourceCPU, cpuMin, cpuMax, "")
			if err != nil {
				return err
			}
			if cpuQuantity != nil {
				co.componentSettings.MinCPU = &cpuMin
				co.componentSettings.MaxCPU = &cpuMax
			}
		}
	}

	// if user didn't opt for advanced options, "ports" value remains empty which panics the "odo push"
	// so we set the ports here
	if len(ports) == 0 {
		ports, err = co.Client.GetPortsFromBuilderImage(*co.componentSettings.Type)
		if err != nil {
			return err
		}
	}

	co.componentSettings.Ports = &ports
	co.componentSettings.Project = &(co.Context.Project)
	envs, err := config.NewEnvVarListFromSlice(co.componentEnvVars)
	if err != nil {
		return err
	}
	co.componentSettings.Envs = envs
	co.ignores = []string{}
	// Above code is for INTERACTIVE mode
	return nil
}

// getStarterProjectInteractiveMode gets starter project value by asking user in interactive mode.
func getStarterProjectInteractiveMode(projects []devfilev1.StarterProject) *devfilev1.StarterProject {
	projectName := ui.SelectStarterProject(projects)

	// if user do not wish to download starter project or there are no projects in devfile, project name would be empty
	if projectName == "" {
		return nil
	}

	var project devfilev1.StarterProject

	for _, value := range projects {
		if value.Name == projectName {
			project = value
			break
		}
	}

	return &project
}
