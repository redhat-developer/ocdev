package component

import (
	"fmt"
	"github.com/openshift/odo/pkg/component"
	"github.com/openshift/odo/pkg/config"
	"github.com/openshift/odo/pkg/log"
	"github.com/openshift/odo/pkg/odo/genericclioptions"
	"github.com/openshift/odo/pkg/odo/util/completion"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	odoutil "github.com/openshift/odo/pkg/odo/util"

	ktemplates "k8s.io/kubernetes/pkg/kubectl/cmd/templates"
)

var pushCmdExample = ktemplates.Examples(`  # Push source code to the current component
%[1]s

# Push data to the current component from the original source.
%[1]s

# Push source code in ~/mycode to component called my-component
%[1]s my-component --context ~/mycode
  `)

// PushRecommendedCommandName is the recommended push command name
const PushRecommendedCommandName = "push"

// PushOptions encapsulates options that push command uses
type PushOptions struct {
	*CommonPushOptions
}

// NewPushOptions returns new instance of PushOptions
// with "default" values for certain values, for example, show is "false"
func NewPushOptions() *PushOptions {
	return &PushOptions{
		CommonPushOptions: NewCommonPushOptions(),
	}
}

// Complete completes push args
func (po *PushOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	conf, err := config.NewLocalConfigInfo(po.componentContext)
	if err != nil {
		return errors.Wrap(err, "unable to retrieve configuration information")
	}

	// Set the necessary values within WatchOptions
	po.localConfigInfo = conf
	err = po.SetSourceInfo()
	if err != nil {
		return errors.Wrap(err, "unable to set source information")
	}
	// Apply ignore information
	err = genericclioptions.ApplyIgnore(&po.ignores, po.sourcePath)
	if err != nil {
		return errors.Wrap(err, "unable to apply ignore information")
	}

	// Set the correct context
	po.Context = genericclioptions.NewContextCreatingAppIfNeeded(cmd)
	prjName := po.localConfigInfo.GetProject()
	po.ResolveSrcAndConfigFlags()
	err = po.ResolveProject(prjName)
	if err != nil {
		return err
	}
	return
}

// Validate validates the push parameters
func (po *PushOptions) Validate() (err error) {

	log.Info("Validation")

	// First off, we check to see if the component exists. This is ran each time we do `odo push`
	s := log.Spinner("Checking component")
	defer s.End(false)

	po.isCmpExists, err = component.Exists(po.Context.Client, po.localConfigInfo.GetName(), po.localConfigInfo.GetApplication())
	if err != nil {
		return errors.Wrapf(err, "failed to check if component of name %s exists in application %s", po.localConfigInfo.GetName(), po.localConfigInfo.GetApplication())
	}

	if err = component.ValidateComponentCreateRequest(po.Context.Client, po.localConfigInfo.GetComponentSettings(), po.isCmpExists, false); err != nil {
		s.End(false)
		log.Info("Run 'odo catalog list components' for a list of supported component types")
		return fmt.Errorf("Invalid component type %s, %v", *po.localConfigInfo.GetComponentSettings().Type, errors.Cause(err))
	}

	if !po.isCmpExists && po.pushSource && !po.pushConfig {
		return fmt.Errorf("Component %s does not exist and hence cannot push only source. Please use `odo push` without any flags or with both `--source` and `--config` flags", po.localConfigInfo.GetName())
	}

	s.End(true)
	return nil
}

// Run has the logic to perform the required actions as part of command
func (po *PushOptions) Run() (err error) {
	return po.Push()
}

// NewCmdPush implements the push odo command
func NewCmdPush(name, fullName string) *cobra.Command {
	po := NewPushOptions()

	var pushCmd = &cobra.Command{
		Use:     fmt.Sprintf("%s [component name]", name),
		Short:   "Push source code to a component",
		Long:    `Push source code to a component.`,
		Example: fmt.Sprintf(pushCmdExample, fullName),
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(po, cmd, args)
		},
	}
	genericclioptions.AddContextFlag(pushCmd, &po.componentContext)
	pushCmd.Flags().BoolVar(&po.show, "show-log", false, "If enabled, logs will be shown when built")
	pushCmd.Flags().StringSliceVar(&po.ignores, "ignore", []string{}, "Files or folders to be ignored via glob expressions.")
	pushCmd.Flags().BoolVar(&po.pushConfig, "config", false, "Use config flag to only apply config on to cluster")
	pushCmd.Flags().BoolVar(&po.pushSource, "source", false, "Use source flag to only push latest source on to cluster")
	pushCmd.Flags().BoolVarP(&po.forceBuild, "force-build", "f", false, "Use force-build flag to force building the component")

	// Add a defined annotation in order to appear in the help menu
	pushCmd.Annotations = map[string]string{"command": "component"}
	pushCmd.SetUsageTemplate(odoutil.CmdUsageTemplate)
	completion.RegisterCommandHandler(pushCmd, completion.ComponentNameCompletionHandler)
	completion.RegisterCommandFlagHandler(pushCmd, "context", completion.FileCompletionHandler)

	return pushCmd
}
