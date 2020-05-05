package component

import (
	"fmt"
	"github.com/openshift/odo/pkg/envinfo"
	projectCmd "github.com/openshift/odo/pkg/odo/cli/project"
	"github.com/openshift/odo/pkg/odo/genericclioptions"
	odoutil "github.com/openshift/odo/pkg/odo/util"
	"github.com/openshift/odo/pkg/odo/util/completion"
	"github.com/openshift/odo/pkg/odo/util/experimental"
	"github.com/openshift/odo/pkg/odo/util/pushtarget"
	"github.com/openshift/odo/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	ktemplates "k8s.io/kubectl/pkg/util/templates"
	"path/filepath"
)

// ExecRecommendedCommandName is the recommended exec command name
const ExecRecommendedCommandName = "exec"

var execExample = ktemplates.Examples(`  # Executes a command inside the component
%[1]s
`)

// ExecOptions contains exec options
type ExecOptions struct {
	componentContext string
	componentOptions *ComponentOptions
	devfilePath      string
	namespace        string

	command []string
}

// NewExecOptions returns new instance of ExecOptions
func NewExecOptions() *ExecOptions {
	return &ExecOptions{
		componentOptions: &ComponentOptions{},
	}
}

// Complete completes exec args
func (eo *ExecOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	if cmd.ArgsLenAtDash() <= -1 {
		return fmt.Errorf("no command was given for the exec command")
	}

	eo.command = args[cmd.ArgsLenAtDash():]

	if len(eo.command) <= 0 {
		return fmt.Errorf("no command was given for the exec command")
	}

	if len(eo.command) != len(args) {
		return fmt.Errorf("no parameter is expected for the command")
	}

	eo.devfilePath = filepath.Join(eo.componentContext, eo.devfilePath)
	// if experimental mode is enabled and devfile is present
	if experimental.IsExperimentalModeEnabled() && util.CheckPathExists(eo.devfilePath) {
		envInfo, err := envinfo.NewEnvSpecificInfo(eo.componentContext)
		if err != nil {
			return errors.Wrap(err, "unable to retrieve configuration information")
		}
		eo.componentOptions.Context = genericclioptions.NewDevfileContext(cmd)
		eo.componentOptions.EnvSpecificInfo = envInfo

		if !pushtarget.IsPushTargetDocker() {
			// The namespace was retrieved from the --project flag (or from the kube client if not set) and stored in kclient when initalizing the context
			eo.namespace = eo.componentOptions.KClient.Namespace
		}
		return nil
	}
	return
}

// Validate validates the exec parameters
func (eo *ExecOptions) Validate() (err error) {
	return
}

// Run has the logic to perform the required actions as part of command
func (eo *ExecOptions) Run() (err error) {
	return eo.DevfileComponentExec(eo.command)
}

// NewCmdExec implements the exec odo command
func NewCmdExec(name, fullName string) *cobra.Command {
	o := NewExecOptions()

	var execCmd = &cobra.Command{
		Use:         name,
		Short:       "Executes a command inside the component",
		Long:        `Executes a command inside the component`,
		Example:     fmt.Sprintf(execExample, fullName),
		Annotations: map[string]string{"command": "component"},
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(o, cmd, args)
		},
	}

	execCmd.Flags().StringVar(&o.devfilePath, "devfile", "./devfile.yaml", "Path to a devfile.yaml")

	execCmd.SetUsageTemplate(odoutil.CmdUsageTemplate)
	completion.RegisterCommandHandler(execCmd, completion.ComponentNameCompletionHandler)
	genericclioptions.AddContextFlag(execCmd, &o.componentContext)

	//Adding `--project` flag
	projectCmd.AddProjectFlag(execCmd)

	return execCmd
}
