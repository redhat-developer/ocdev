package component

import (
	"fmt"
	"os"

	appCmd "github.com/openshift/odo/pkg/odo/cli/application"
	projectCmd "github.com/openshift/odo/pkg/odo/cli/project"
	"github.com/openshift/odo/pkg/odo/util/completion"
	"github.com/pkg/errors"
	ktemplates "k8s.io/kubernetes/pkg/kubectl/cmd/templates"

	"github.com/golang/glog"
	"github.com/openshift/odo/pkg/odo/genericclioptions"

	"github.com/openshift/odo/pkg/component"
	odoutil "github.com/openshift/odo/pkg/odo/util"
	"github.com/openshift/odo/pkg/util"
	"github.com/spf13/cobra"
)

// WatchRecommendedCommandName is the recommended watch command name
const WatchRecommendedCommandName = "watch"

var watchLongDesc = ktemplates.LongDesc(`Watch for changes, update component on change.`)
var watchExample = ktemplates.Examples(`  # Watch for changes in directory for current component
%[1]s

# Watch for changes in directory for component called frontend 
%[1]s frontend
  `)

// WatchOptions contains attributes of the watch command
type WatchOptions struct {
	ignores []string
	delay   int

	options genericclioptions.ComponentOptions

	*genericclioptions.Context
}

// NewWatchOptions returns new instance of WatchOptions
func NewWatchOptions() *WatchOptions {
	return &WatchOptions{}
}

// Complete completes watch args
func (wo *WatchOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	wo.options.Client = genericclioptions.Client(cmd)

	// Retrieve configuration configuration as well as SourcePath
	conf, err := genericclioptions.RetrieveLocalConfigInfo(wo.options.ComponentContext)
	if err != nil {
		return errors.Wrap(err, "unable to retrieve config information")
	}

	// Set the necessary values within WatchOptions
	wo.options.LocalConfig = conf.LocalConfig
	wo.options.SourcePath = conf.SourcePath
	wo.options.SourceType = conf.LocalConfig.GetSourceType()

	// Apply ignore information
	err = genericclioptions.ApplyIgnore(&wo.ignores, wo.options.SourcePath)
	if err != nil {
		return errors.Wrap(err, "unable to apply ignore information")
	}

	// Set the correct context
	wo.Context = genericclioptions.NewContextCreatingAppIfNeeded(cmd)
	return
}

// Validate validates the watch parameters
func (wo *WatchOptions) Validate() (err error) {

	// Validate component path existence and accessibility permissions for odo
	if _, err := os.Stat(wo.options.SourcePath); err != nil {
		return errors.Wrapf(err, "Cannot watch %s", wo.options.SourcePath)
	}

	// Validate source of component is either local source or binary path until git watch is supported
	if wo.options.SourceType != "binary" && wo.options.SourceType != "local" {
		return fmt.Errorf("Watch is supported by binary and local components only and source type of component %s is %s",
			wo.options.LocalConfig.GetName(),
			wo.options.SourceType)
	}

	// Delay interval cannot be -ve
	if wo.delay < 0 {
		return fmt.Errorf("Delay cannot be lesser than 0 and delay=0 means changes will be pushed as soon as they are detected which can cause performance issues")
	}
	// Print a debug message warning user if delay is set to 0
	if wo.delay == 0 {
		glog.V(4).Infof("delay=0 means changes will be pushed as soon as they are detected which can cause performance issues")
	}

	return
}

// Run has the logic to perform the required actions as part of command
func (wo *WatchOptions) Run() (err error) {
	err = component.WatchAndPush(
		wo.Context.Client,
		os.Stdout,
		component.WatchParameters{
			ComponentName:   wo.options.LocalConfig.GetName(),
			ApplicationName: wo.Context.Application,
			Path:            wo.options.SourcePath,
			FileIgnores:     util.GetAbsGlobExps(wo.options.SourcePath, wo.ignores),
			PushDiffDelay:   wo.delay,
			StartChan:       nil,
			ExtChan:         make(chan bool),
			WatchHandler:    component.PushLocal,
		},
	)
	if err != nil {
		return errors.Wrapf(err, "Error while trying to watch %s", wo.options.SourcePath)
	}
	return
}

// NewCmdWatch implements the watch odo command
func NewCmdWatch(name, fullName string) *cobra.Command {
	wo := NewWatchOptions()

	var watchCmd = &cobra.Command{
		Use:     fmt.Sprintf("%s [component name]", name),
		Short:   "Watch for changes, update component on change",
		Long:    watchLongDesc,
		Example: fmt.Sprintf(watchExample, fullName),
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(wo, cmd, args)
		},
	}

	watchCmd.Flags().StringSliceVar(&wo.ignores, "ignore", []string{}, "Files or folders to be ignored via glob expressions.")
	watchCmd.Flags().IntVar(&wo.delay, "delay", 1, "Time in seconds between a detection of code change and push.delay=0 means changes will be pushed as soon as they are detected which can cause performance issues")
	watchCmd.Flags().StringVarP(&wo.options.ComponentContext, "context", "c", "", "Use given context directory as a source for component settings")

	// Add a defined annotation in order to appear in the help menu
	watchCmd.Annotations = map[string]string{"command": "component"}
	watchCmd.SetUsageTemplate(odoutil.CmdUsageTemplate)

	//Adding `--application` flag
	appCmd.AddApplicationFlag(watchCmd)

	//Adding `--project` flag
	projectCmd.AddProjectFlag(watchCmd)

	completion.RegisterCommandHandler(watchCmd, completion.ComponentNameCompletionHandler)

	return watchCmd
}
