package application

import (
	"fmt"
	"github.com/redhat-developer/odo/pkg/application"
	"github.com/redhat-developer/odo/pkg/log"
	"github.com/redhat-developer/odo/pkg/odo/cli/project"
	"github.com/redhat-developer/odo/pkg/odo/cli/ui"
	"github.com/redhat-developer/odo/pkg/odo/genericclioptions"
	"github.com/redhat-developer/odo/pkg/odo/util/completion"
	"github.com/spf13/cobra"
	ktemplates "k8s.io/kubernetes/pkg/kubectl/cmd/templates"
)

const deleteRecommendedCommandName = "delete"

var (
	deleteExample = ktemplates.Examples(`  # Delete the application
  %[1]s myapp`)
)

// DeleteOptions encapsulates the options for the odo command
type DeleteOptions struct {
	appName string
	force   bool
	*genericclioptions.Context
}

// NewDeleteOptions creates a new DeleteOptions instance
func NewDeleteOptions() *DeleteOptions {
	return &DeleteOptions{}
}

// Complete completes DeleteOptions after they've been created
func (o *DeleteOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	o.Context = genericclioptions.NewContext(cmd)
	o.appName = o.Application
	if len(args) == 1 {
		// If app name passed, consider it for deletion
		o.appName = args[0]
	}
	return
}

// Validate validates the DeleteOptions based on completed values
func (o *DeleteOptions) Validate() (err error) {
	return ensureAppExists(o.Client, o.appName, o.Project)
}

// Run contains the logic for the odo command
func (o *DeleteOptions) Run() (err error) {
	// Print App Information which will be deleted
	err = printDeleteAppInfo(o.Client, o.appName, o.Project)
	if err != nil {
		return err
	}

	if o.force || ui.Proceed(fmt.Sprintf("Are you sure you want to delete the application: %v from project: %v", o.appName, o.Project)) {
		err = application.Delete(o.Client, o.appName)
		if err != nil {
			return err
		}
		log.Infof("Deleted application: %s from project: %v", o.appName, o.Project)
	} else {
		log.Infof("Aborting deletion of application: %v", o.appName)
	}
	return
}

// NewCmdDelete implements the odo command.
func NewCmdDelete(name, fullName string) *cobra.Command {
	o := NewDeleteOptions()
	command := &cobra.Command{
		Use:     name,
		Short:   "Delete the given application",
		Long:    "Delete the given application",
		Example: fmt.Sprintf(deleteExample, fullName),
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(o, cmd, args)
		},
	}

	command.Flags().BoolVarP(&o.force, "force", "f", false, "Delete application without prompting")

	project.AddProjectFlag(command)
	completion.RegisterCommandHandler(command, completion.AppCompletionHandler)
	return command
}
