package application

import (
	"fmt"
	"path/filepath"

	applabels "github.com/openshift/odo/pkg/application/labels"

	odoutil "github.com/openshift/odo/pkg/util"

	"github.com/openshift/odo/pkg/application"
	"github.com/openshift/odo/pkg/component"
	"github.com/openshift/odo/pkg/log"
	"github.com/openshift/odo/pkg/machineoutput"
	"github.com/openshift/odo/pkg/odo/cli/project"
	"github.com/openshift/odo/pkg/odo/genericclioptions"
	"github.com/openshift/odo/pkg/odo/util"
	"github.com/openshift/odo/pkg/odo/util/completion"
	"github.com/spf13/cobra"
	ktemplates "k8s.io/kubectl/pkg/util/templates"
)

const describeRecommendedCommandName = "describe"

var (
	describeExample = ktemplates.Examples(`  # Describe 'webapp' application
  %[1]s webapp`)
)

// DescribeOptions encapsulates the options for the odo command
type DescribeOptions struct {
	appName      string
	outputFormat string
	*genericclioptions.Context
}

// NewDescribeOptions creates a new DescribeOptions instance
func NewDescribeOptions() *DescribeOptions {
	return &DescribeOptions{}
}

// Complete completes DescribeOptions after they've been created
func (o *DescribeOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	if odoutil.CheckPathExists(filepath.Join(".odo", "config.yaml")) {
		o.Context, err = genericclioptions.NewContext(cmd)
	} else {
		o.Context, err = genericclioptions.NewDevfileContext(cmd)
	}
	if err != nil {
		return err
	}
	o.appName = o.Application
	if len(args) == 1 {
		o.appName = args[0]
	}
	return
}

// Validate validates the DescribeOptions based on completed values
func (o *DescribeOptions) Validate() (err error) {
	if o.Context.Project == "" || o.appName == "" {
		return util.ThrowContextError()
	}
	err = util.CheckOutputFlag(o.outputFormat)
	if err != nil {
		return err
	}
	if o.appName == "" {
		return fmt.Errorf("There's no active application in project: %v", o.Project)
	}

	exist, err := application.Exists(o.appName, o.Client, o.KClient)
	if !exist {
		return fmt.Errorf("%s app does not exists", o.appName)
	}
	return err
}

// Run contains the logic for the odo command
func (o *DescribeOptions) Run(cmd *cobra.Command) (err error) {
	if log.IsJSON() {
		appDef := application.GetMachineReadableFormat(o.Client, o.appName, o.Project)
		machineoutput.OutputSuccess(appDef)
	} else {
		var selector string
		if o.appName != "" {
			selector = applabels.GetSelector(o.appName)
		}
		componentList, err := component.List(o.Client, selector, nil)
		if err != nil {
			return err
		}

		if len(componentList.Items) == 0 {
			fmt.Printf("Application %s has no components or services deployed.", o.appName)
		} else {
			fmt.Printf("Application Name: %s has %v component(s):\n--------------------------------------\n",
				o.appName, len(componentList.Items))
			if len(componentList.Items) > 0 {
				for _, currentComponent := range componentList.Items {
					err := util.PrintComponentInfo(o.Client, currentComponent.Name, currentComponent, o.appName, o.Project)
					if err != nil {
						return err
					}
					fmt.Println("--------------------------------------")
				}
			}
		}
	}

	return
}

// NewCmdDescribe implements the odo command.
func NewCmdDescribe(name, fullName string) *cobra.Command {
	o := NewDescribeOptions()
	command := &cobra.Command{
		Use:         fmt.Sprintf("%s [application_name]", name),
		Short:       "Describe the given application",
		Long:        "Describe the given application",
		Example:     fmt.Sprintf(describeExample, fullName),
		Args:        cobra.MaximumNArgs(1),
		Annotations: map[string]string{"machineoutput": "json"},
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(o, cmd, args)
		},
	}

	completion.RegisterCommandHandler(command, completion.AppCompletionHandler)

	project.AddProjectFlag(command)
	return command
}
