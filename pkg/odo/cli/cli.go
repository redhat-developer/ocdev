package cli

import (
	"flag"
	"fmt"

	"github.com/redhat-developer/odo/pkg/odo/cli/application"
	"github.com/redhat-developer/odo/pkg/odo/cli/catalog"
	"github.com/redhat-developer/odo/pkg/odo/cli/component"
	"github.com/redhat-developer/odo/pkg/odo/cli/login"
	"github.com/redhat-developer/odo/pkg/odo/cli/logout"
	"github.com/redhat-developer/odo/pkg/odo/cli/project"
	"github.com/redhat-developer/odo/pkg/odo/cli/service"
	"github.com/redhat-developer/odo/pkg/odo/cli/storage"
	"github.com/redhat-developer/odo/pkg/odo/cli/url"
	"github.com/redhat-developer/odo/pkg/odo/cli/utils"
	"github.com/redhat-developer/odo/pkg/odo/cli/version"
	"github.com/redhat-developer/odo/pkg/odo/genericclioptions"
	"github.com/redhat-developer/odo/pkg/odo/util"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	ktemplates "k8s.io/kubernetes/pkg/kubectl/cmd/templates"
)

// OdoRecommendedName is the recommended odo command name
const OdoRecommendedName = "odo"

var (
	odoLong = ktemplates.LongDesc(`
Odo (OpenShift Do) is a CLI tool for running OpenShift applications in a fast and automated matter. Odo reduces the complexity of deployment by adding iterative development without the worry of deploying your source code.

Find more information at https://github.com/redhat-developer/odo`)
	odoExample = ktemplates.Examples(`  # Creating and deploying a Node.js project
  git clone https://github.com/openshift/nodejs-ex && cd nodejs-ex
  %[1]s create nodejs
  %[1]s push

  # Accessing your Node.js component
  %[1]s url create`)
	rootUsageTemplate = `Usage:{{if .Runnable}}
  {{if .HasAvailableFlags}}{{appendIfNotPresent .UseLine "[flags]"}}{{else}}{{.UseLine}}{{end}}{{end}}{{if .HasAvailableSubCommands}}
  {{ .CommandPath}} [command]{{end}}{{if gt .Aliases 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{ .Example }}{{end}}{{ if .HasAvailableSubCommands}}

Component Commands:{{range .Commands}}{{if eq .Annotations.command "component"}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{ if .HasAvailableLocalFlags}}

Other Commands:{{range .Commands}}{{if eq .Annotations.command "other"}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{ if .HasAvailableLocalFlags}}

Utility Commands:{{range .Commands}}{{if or (eq .Annotations.command "utility") (eq .Name "help") }}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{ if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimRightSpace}}{{end}}{{ if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimRightSpace}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsHelpCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{ if .HasAvailableSubCommands }}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`
)

// NewCmdOdo creates a new root command for odo
func NewCmdOdo(name, fullName string) *cobra.Command {
	// rootCmd represents the base command when called without any subcommands
	rootCmd := &cobra.Command{
		Use:     name,
		Short:   "Odo (OpenShift Do)",
		Long:    odoLong,
		Example: fmt.Sprintf(odoExample, fullName),
	}
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.odo.yaml)")

	rootCmd.PersistentFlags().Bool(genericclioptions.SkipConnectionCheckFlagName, false, "Skip cluster check")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.CommandLine.Set("logtostderr", "true")

	// Override the verbosity flag description
	verbosity := pflag.Lookup("v")
	verbosity.Usage += ". Level varies from 0 to 9 (default 0)."

	rootCmd.SetUsageTemplate(rootUsageTemplate)

	rootCmd.AddCommand(
		application.NewCmdApplication(application.RecommendedCommandName, util.GetFullName(fullName, application.RecommendedCommandName)),
		catalog.NewCmdCatalog(catalog.RecommendedCommandName, util.GetFullName(fullName, catalog.RecommendedCommandName)),
		component.NewCmdComponent(component.RecommendedCommandName, util.GetFullName(fullName, component.RecommendedCommandName)),
		component.NewCmdCreate(component.CreateRecommendedCommandName, util.GetFullName(fullName, component.CreateRecommendedCommandName)),
		component.NewCmdDelete(component.DeleteRecommendedCommandName, util.GetFullName(fullName, component.DeleteRecommendedCommandName)),
		component.NewCmdDescribe(component.DescribeRecommendedCommandName, util.GetFullName(fullName, component.DescribeRecommendedCommandName)),
		component.NewCmdLink(component.LinkRecommendedCommandName, util.GetFullName(fullName, component.LinkRecommendedCommandName)),
		component.NewCmdUnlink(component.UnlinkRecommendedCommandName, util.GetFullName(fullName, component.UnlinkRecommendedCommandName)),
		component.NewCmdList(component.ListRecommendedCommandName, util.GetFullName(fullName, component.ListRecommendedCommandName)),
		component.NewCmdLog(component.LogRecommendedCommandName, util.GetFullName(fullName, component.LogRecommendedCommandName)),
		component.NewCmdPush(component.PushRecommendedCommandName, util.GetFullName(fullName, component.PushRecommendedCommandName)),
		component.NewCmdUpdate(component.UpdateRecommendedCommandName, util.GetFullName(fullName, component.UpdateRecommendedCommandName)),
		component.NewCmdWatch(component.WatchRecommendedCommandName, util.GetFullName(fullName, component.WatchRecommendedCommandName)),
		login.NewCmdLogin(login.RecommendedCommandName, util.GetFullName(fullName, login.RecommendedCommandName)),
		logout.NewCmdLogout(logout.RecommendedCommandName, util.GetFullName(fullName, logout.RecommendedCommandName)),
		project.NewCmdProject(project.RecommendedCommandName, util.GetFullName(fullName, project.RecommendedCommandName)),
		service.NewCmdService(service.RecommendedCommandName, util.GetFullName(fullName, service.RecommendedCommandName)),
		storage.NewCmdStorage(storage.RecommendedCommandName, util.GetFullName(fullName, storage.RecommendedCommandName)),
		url.NewCmdURL(url.RecommendedCommandName, util.GetFullName(fullName, url.RecommendedCommandName)),
		utils.NewCmdUtils(utils.RecommendedCommandName, util.GetFullName(fullName, utils.RecommendedCommandName)),
		version.NewCmdVersion(version.RecommendedCommandName, util.GetFullName(fullName, version.RecommendedCommandName)),
	)

	return rootCmd
}
