package component

import (
	"fmt"
	"github.com/redhat-developer/odo/pkg/odo/genericclioptions"
	"os"

	appCmd "github.com/redhat-developer/odo/pkg/odo/cli/application"
	projectCmd "github.com/redhat-developer/odo/pkg/odo/cli/project"
	"github.com/redhat-developer/odo/pkg/odo/util/completion"
	ktemplates "k8s.io/kubernetes/pkg/kubectl/cmd/templates"

	odoutil "github.com/redhat-developer/odo/pkg/odo/util"

	"github.com/redhat-developer/odo/pkg/component"
	"github.com/spf13/cobra"
)

// RecommendedLogCommandName is the recommended watch command name
const RecommendedLogCommandName = "log"

var logExample = ktemplates.Examples(`  # Get the logs for the nodejs component
%[1]s nodejs
`)

// LogOptions contains log options
type LogOptions struct {
	logFollow bool
	*ComponentOptions
}

// NewLogOptions returns new instance of LogOptions
func NewLogOptions() *LogOptions {
	return &LogOptions{false, &ComponentOptions{}}
}

// Complete completes log args
func (lo *LogOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	err = lo.ComponentOptions.Complete(name, cmd, args)
	return
}

// Validate validates the log parameters
func (lo *LogOptions) Validate() (err error) {
	return
}

// Run has the logic to perform the required actions as part of command
func (lo *LogOptions) Run() (err error) {
	stdout := os.Stdout

	// Retrieve the log
	err = component.GetLogs(lo.Context.Client, lo.componentName, lo.Context.Application, lo.logFollow, stdout)
	return
}

// NewCmdLog implements the log odo command
func NewCmdLog(name, fullName string) *cobra.Command {
	o := NewLogOptions()

	var logCmd = &cobra.Command{
		Use:     fmt.Sprintf("%s [component_name]", name),
		Short:   "Retrieve the log for the given component.",
		Long:    `Retrieve the log for the given component.`,
		Example: fmt.Sprintf(logExample, fullName),
		Args:    cobra.RangeArgs(0, 1),
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(o, cmd, args)
		},
	}

	logCmd.Flags().BoolVarP(&o.logFollow, "follow", "f", false, "Follow logs")

	// Add a defined annotation in order to appear in the help menu
	logCmd.Annotations = map[string]string{"command": "component"}
	logCmd.SetUsageTemplate(odoutil.CmdUsageTemplate)
	completion.RegisterCommandHandler(logCmd, completion.ComponentNameCompletionHandler)

	//Adding `--project` flag
	projectCmd.AddProjectFlag(logCmd)
	//Adding `--application` flag
	appCmd.AddApplicationFlag(logCmd)

	return logCmd
}
