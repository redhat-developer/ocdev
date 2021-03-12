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

	ktemplates "k8s.io/kubectl/pkg/util/templates"

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
	*ComponentOptions
}

// NewDescribeOptions returns new instance of ListOptions
func NewDescribeOptions() *DescribeOptions {
	return &DescribeOptions{"", &ComponentOptions{}}
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

	return nil
}

// Run has the logic to perform the required actions as part of command
func (do *DescribeOptions) Run() (err error) {
	if (len(do.componentName) <= 0 || len(do.Application) <= 0 || len(do.Project) <= 0) && !do.LocalConfigInfo.Exists() {
		return fmt.Errorf("Component %v does not exist", do.componentName)
	}

	cfd, err := component.NewComponentFullDescriptionFromClientAndLocalConfig(do.Context.Client, do.LocalConfigInfo, do.EnvSpecificInfo, do.componentName, do.Context.Application, do.Context.Project)
	if err != nil {
		return err
	}

	if log.IsJSON() {
		machineoutput.OutputSuccess(cfd)
	} else {
		err = cfd.Print(do.Context.Client)
		if err != nil {
			return err
		}
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
