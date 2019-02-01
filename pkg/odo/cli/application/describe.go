package application

import (
	"fmt"
	"github.com/redhat-developer/odo/pkg/component"
	"github.com/redhat-developer/odo/pkg/log"
	"github.com/redhat-developer/odo/pkg/odo/cli/project"
	"github.com/redhat-developer/odo/pkg/odo/genericclioptions"
	"github.com/redhat-developer/odo/pkg/odo/util"
	"github.com/redhat-developer/odo/pkg/odo/util/completion"
	"github.com/redhat-developer/odo/pkg/service"
	"github.com/spf13/cobra"
	ktemplates "k8s.io/kubernetes/pkg/kubectl/cmd/templates"
)

const describeRecommendedCommandName = "describe"

var (
	describeExample = ktemplates.Examples(`  # Describe 'webapp' application,
  %[1]s webapp`)
)

// DescribeOptions encapsulates the options for the odo command
type DescribeOptions struct {
	appName string
	*genericclioptions.Context
}

// NewDescribeOptions creates a new DescribeOptions instance
func NewDescribeOptions() *DescribeOptions {
	return &DescribeOptions{}
}

// Complete completes DescribeOptions after they've been created
func (o *DescribeOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	o.Context = genericclioptions.NewContext(cmd)
	o.appName = o.Application
	if len(args) == 1 {
		o.appName = args[0]
	}
	return
}

// Validate validates the DescribeOptions based on completed values
func (o *DescribeOptions) Validate() (err error) {
	if o.appName == "" {
		return fmt.Errorf("There's no active application in project: %v", o.Project)
	}

	return validateApp(o.Client, o.appName, o.Project)
}

// Run contains the logic for the odo command
func (o *DescribeOptions) Run() (err error) {
	// List of Component
	componentList, err := component.List(o.Client, o.appName)
	if err != nil {
		return err
	}

	//we ignore service errors here because it's entirely possible that the service catalog has not been installed
	serviceList, _ := service.ListWithDetailedStatus(o.Client, o.appName)

	if len(componentList) == 0 && len(serviceList) == 0 {
		log.Errorf("Application %s has no components or services deployed.", o.appName)
	} else {
		fmt.Printf("Application Name: %s has %v component(s) and %v service(s):\n--------------------------------------\n",
			o.appName, len(componentList), len(serviceList))
		if len(componentList) > 0 {
			for _, currentComponent := range componentList {
				componentDesc, err := component.GetComponentDesc(o.Client, currentComponent.Name, o.appName, o.Project)
				util.LogErrorAndExit(err, "")
				util.PrintComponentInfo(currentComponent.Name, componentDesc)
				fmt.Println("--------------------------------------")
			}
		}
		if len(serviceList) > 0 {
			for _, currentService := range serviceList {
				fmt.Printf("Service Name: %s\n", currentService.Name)
				fmt.Printf("Type: %s\n", currentService.Type)
				fmt.Printf("Status: %s\n", currentService.Status)
				fmt.Println("--------------------------------------")
			}
		}
	}

	return
}

// NewCmdDescribe implements the odo command.
func NewCmdDescribe(name, fullName string) *cobra.Command {
	o := NewDescribeOptions()
	command := &cobra.Command{
		Use:     fmt.Sprintf("%s [application_name]", name),
		Short:   "Describe the given application",
		Long:    "Describe the given application",
		Example: fmt.Sprintf(describeExample, fullName),
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			util.LogErrorAndExit(o.Complete(name, cmd, args), "")
			util.LogErrorAndExit(o.Validate(), "")
			util.LogErrorAndExit(o.Run(), "")
		},
	}

	completion.RegisterCommandHandler(command, completion.AppCompletionHandler)
	project.AddProjectFlag(command)
	return command
}
