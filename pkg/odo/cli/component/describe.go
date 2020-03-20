package component

import (
	"fmt"

	"github.com/openshift/odo/pkg/log"
	"github.com/openshift/odo/pkg/machineoutput"
	"github.com/openshift/odo/pkg/odo/genericclioptions"

	"github.com/openshift/odo/pkg/component"
	appCmd "github.com/openshift/odo/pkg/odo/cli/application"
	projectCmd "github.com/openshift/odo/pkg/odo/cli/project"
	odoutil "github.com/openshift/odo/pkg/odo/util"
	"github.com/openshift/odo/pkg/odo/util/completion"

	ktemplates "k8s.io/kubernetes/pkg/kubectl/util/templates"

	"github.com/spf13/cobra"
)

// DescribeRecommendedCommandName is the recommended describe command name
const DescribeRecommendedCommandName = "describe"

var describeExample = ktemplates.Examples(`  # Describe nodejs component
%[1]s nodejs
`)

// DescribeOptions is a dummy container to attach complete, validate and run pattern
type DescribeOptions struct {
	componentContext string
	isPushed         bool
	*ComponentOptions
}

// NewDescribeOptions returns new instance of ListOptions
func NewDescribeOptions() *DescribeOptions {
	return &DescribeOptions{"", false, &ComponentOptions{}}
}

// Complete completes describe args
func (do *DescribeOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	err = do.ComponentOptions.Complete(name, cmd, args)
	if err != nil {
		return err
	}
	return nil
}

// Validate validates the describe parameters
func (do *DescribeOptions) Validate() (err error) {
	if do.Context.Project == "" || do.Application == "" {
		return odoutil.ThrowContextError()
	}

	existsInCluster, err := component.Exists(do.Context.Client, do.componentName, do.Context.Application)
	if err != nil {
		return err
	}
	if existsInCluster {
		do.isPushed = true
	}

	return nil
}

// Run has the logic to perform the required actions as part of command
func (do *DescribeOptions) Run() (err error) {
	var componentDesc component.Component
	if !do.isPushed {
		componentDesc, err = component.GetComponentFromConfig(do.LocalConfigInfo)
		if err != nil {
			return err
		}
	} else {
		componentDesc, err = component.GetComponent(do.Context.Client, do.componentName, do.Context.Application, do.Context.Project)
		if err != nil {
			return err
		}
	}

	if log.IsJSON() {
		componentDesc.Spec.Ports = do.LocalConfigInfo.GetPorts()
		machineoutput.OutputSuccess(componentDesc)
	} else {

		odoutil.PrintComponentInfo(do.Context.Client, do.componentName, componentDesc, do.Context.Application, do.Context.Project)
	}

	return
}

// NewCmdDescribe implements the describe odo command
func NewCmdDescribe(name, fullName string) *cobra.Command {
	do := NewDescribeOptions()

	var describeCmd = &cobra.Command{
		Use:         fmt.Sprintf("%s [component_name]", name),
		Short:       "Describe component",
		Long:        `Describe component.`,
		Example:     fmt.Sprintf(describeExample, fullName),
		Args:        cobra.RangeArgs(0, 1),
		Annotations: map[string]string{"machineoutput": "json", "command": "component"},
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(do, cmd, args)
		},
	}

	describeCmd.SetUsageTemplate(odoutil.CmdUsageTemplate)
	completion.RegisterCommandHandler(describeCmd, completion.ComponentNameCompletionHandler)
	// Adding --context flag
	genericclioptions.AddContextFlag(describeCmd, &do.componentContext)

	//Adding `--project` flag
	projectCmd.AddProjectFlag(describeCmd)
	//Adding `--application` flag
	appCmd.AddApplicationFlag(describeCmd)

	return describeCmd
}
