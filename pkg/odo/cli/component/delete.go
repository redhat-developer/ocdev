package component

import (
	"fmt"
	"github.com/redhat-developer/odo/pkg/odo/genericclioptions"
	"strings"

	"github.com/pkg/errors"
	appCmd "github.com/redhat-developer/odo/pkg/odo/cli/application"
	projectCmd "github.com/redhat-developer/odo/pkg/odo/cli/project"
	"github.com/redhat-developer/odo/pkg/odo/util/completion"
	ktemplates "k8s.io/kubernetes/pkg/kubectl/cmd/templates"

	"github.com/redhat-developer/odo/pkg/log"
	odoutil "github.com/redhat-developer/odo/pkg/odo/util"

	"github.com/golang/glog"
	"github.com/redhat-developer/odo/pkg/component"
	"github.com/spf13/cobra"
)

// RecommendedDeleteCommandName is the recommended delete command name
const RecommendedDeleteCommandName = "delete"

var deleteExample = ktemplates.Examples(`  # Delete component named 'frontend'. 
%[1]s frontend
  `)

// DeleteOptions is a container to attach complete, validate and run pattern
type DeleteOptions struct {
	componentForceDeleteFlag bool
	*ComponentOptions
}

// NewDeleteOptions returns new instance of DeleteOptions
func NewDeleteOptions() *DeleteOptions {
	return &DeleteOptions{false, &ComponentOptions{}}
}

// Complete completes log args
func (do *DeleteOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	err = do.ComponentOptions.Complete(name, cmd, args)
	return
}

// Validate validates the list parameters
func (do *DeleteOptions) Validate() (err error) {
	isExists, err := component.Exists(do.Context.Client, do.componentName, do.Context.Application)
	if err != nil {
		return err
	}
	if !isExists {
		return fmt.Errorf("failed to delete component %s as it doesn't exist", do.componentName)
	}
	return
}

// Run has the logic to perform the required actions as part of command
func (do *DeleteOptions) Run() (err error) {
	glog.V(4).Infof("component delete called")
	glog.V(4).Infof("args: %#v", do)

	var confirmDeletion string
	if do.componentForceDeleteFlag {
		confirmDeletion = "y"
	} else {
		log.Askf("Are you sure you want to delete %v from %v? [y/N]: ", do.componentName, do.Context.Application)
		fmt.Scanln(&confirmDeletion)
	}

	if strings.ToLower(confirmDeletion) == "y" {
		err := component.Delete(do.Context.Client, do.componentName, do.Context.Application)
		if err != nil {
			return err
		}
		log.Successf("Component %s from application %s has been deleted", do.componentName, do.Context.Application)

		currentComponent, err := component.GetCurrent(do.Context.Application, do.Context.Project)
		if err != nil {
			return errors.Wrapf(err, "Unable to get current component")
		}

		if currentComponent == "" {
			log.Info("No default component has been set")
		} else {
			log.Infof("Default component set to: %s", currentComponent)
		}

	} else {
		log.Infof("Aborting deletion of component: %v", do.componentName)
	}

	return
}

// NewCmdDelete implements the delete odo command
func NewCmdDelete(name, fullName string) *cobra.Command {

	do := NewDeleteOptions()

	var componentDeleteCmd = &cobra.Command{
		Use:     fmt.Sprintf("%s <component_name>", name),
		Short:   "Delete an existing component",
		Long:    "Delete an existing component.",
		Example: fmt.Sprintf(deleteExample, fullName),
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(do, cmd, args)
		},
	}

	componentDeleteCmd.Flags().BoolVarP(&do.componentForceDeleteFlag, "force", "f", false, "Delete component without prompting")

	// Add a defined annotation in order to appear in the help menu
	componentDeleteCmd.Annotations = map[string]string{"command": "component"}
	componentDeleteCmd.SetUsageTemplate(odoutil.CmdUsageTemplate)
	completion.RegisterCommandHandler(componentDeleteCmd, completion.ComponentNameCompletionHandler)

	//Adding `--project` flag
	projectCmd.AddProjectFlag(componentDeleteCmd)
	//Adding `--application` flag
	appCmd.AddApplicationFlag(componentDeleteCmd)

	return componentDeleteCmd
}
