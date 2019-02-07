package config

import (
	"fmt"

	"github.com/redhat-developer/odo/pkg/config"
	"github.com/redhat-developer/odo/pkg/odo/util"
	"github.com/spf13/cobra"
	ktemplates "k8s.io/kubernetes/pkg/kubectl/cmd/templates"
)

// RecommendedCommandName is the recommended config command name
const RecommendedCommandName = "config"

var configLongDesc = ktemplates.LongDesc(`Modifies Odo specific configuration settings within the config file.

%[1]s
%[2]s
`)

// NewCmdConfiguration implements the utils config odo command
func NewCmdConfiguration(name, fullName string) *cobra.Command {
	configurationViewCmd := NewCmdView(viewCommandName, util.GetFullName(fullName, viewCommandName))
	configurationSetCmd := NewCmdSet(setCommandName, util.GetFullName(fullName, setCommandName))
	configurationDeleteCmd := NewCmdDelete(deleteCommandName, util.GetFullName(fullName, deleteCommandName))
	configurationCmd := &cobra.Command{
		Use:   name,
		Short: "Modifies configuration settings",
		Long:  fmt.Sprintf(configLongDesc, config.FormatSupportedParameters(), config.FormatLocallySupportedParameters()),
		Example: fmt.Sprintf("%s\n%s\n%s",
			configurationViewCmd.Example,
			configurationSetCmd.Example,
			configurationDeleteCmd.Example,
		),
		Aliases: []string{"configuration"},
	}

	configurationCmd.AddCommand(configurationViewCmd, configurationSetCmd)
	configurationCmd.AddCommand(configurationDeleteCmd)
	configurationCmd.SetUsageTemplate(util.CmdUsageTemplate)

	return configurationCmd
}
