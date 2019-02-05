package logout

import (
	"fmt"
	"github.com/redhat-developer/odo/pkg/odo/genericclioptions"
	odoutil "github.com/redhat-developer/odo/pkg/odo/util"
	"github.com/spf13/cobra"
	"k8s.io/kubernetes/pkg/kubectl/cmd/templates"
	"os"
)

// RecommendedCommandName is the recommended command name
const RecommendedCommandName = "logout"

var example = templates.Examples(`  # Logout
  %[1]s
`)

// LogoutOptions encapsulates the options for the odo logout command
type LogoutOptions struct {
	*genericclioptions.Context
}

// NewLogoutOptions creates a new LogoutOptions instance
func NewLogoutOptions() *LogoutOptions {
	return &LogoutOptions{}
}

// Complete completes LogoutOptions after they've been created
func (o *LogoutOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	o.Context = genericclioptions.NewContext(cmd)
	return
}

// Validate validates the LogoutOptions based on completed values
func (o *LogoutOptions) Validate() (err error) {
	return
}

// Run contains the logic for the odo logout command
func (o *LogoutOptions) Run() (err error) {
	return o.Client.RunLogout(os.Stdout)
}

// NewCmdLogout implements the logout odo command
func NewCmdLogout(name, fullName string) *cobra.Command {
	o := NewLogoutOptions()
	logoutCmd := &cobra.Command{
		Use:     name,
		Short:   "Log out of the current OpenShift session",
		Long:    "Log out of the current OpenShift session",
		Example: fmt.Sprintf(example, fullName),
		Args:    cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			odoutil.LogErrorAndExit(o.Complete(name, cmd, args), "")
			odoutil.LogErrorAndExit(o.Validate(), "")
			odoutil.LogErrorAndExit(o.Run(), "")
		},
	}

	// Add a defined annotation in order to appear in the help menu
	logoutCmd.Annotations = map[string]string{"command": "utility"}
	logoutCmd.SetUsageTemplate(odoutil.CmdUsageTemplate)

	return logoutCmd
}
