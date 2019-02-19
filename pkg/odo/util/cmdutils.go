package util

import (
	"fmt"
	"os"
	"strings"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/redhat-developer/odo/pkg/component"
	"github.com/redhat-developer/odo/pkg/log"
	"github.com/redhat-developer/odo/pkg/url"
)

// LogErrorAndExit prints the cause of the given error and exits the code with an
// exit code of 1.
// If the context is provided, then that is printed, if not, then the cause is
// detected using errors.Cause(err)
func LogErrorAndExit(err error, context string, a ...interface{}) {
	if err != nil {
		glog.V(4).Infof("Error:\n%v", err)
		if context == "" {
			log.Error(errors.Cause(err))
		} else {
			log.Errorf(fmt.Sprintf("%s\n", context), a...)
		}
		os.Exit(1)
	}
}

// CheckOutputFlag validates the -o flag
func CheckOutputFlag(outputFlag string) error {
	switch outputFlag {
	case "", "json":
		return nil
	default:
		return fmt.Errorf("Please input valid output format. available format: json")
	}

}

var CmdUsageTemplate = `Usage:{{if .Runnable}}
  {{if .HasAvailableFlags}}{{appendIfNotPresent .UseLine "[flags]"}}{{else}}{{.UseLine}}{{end}}{{end}}{{if .HasAvailableSubCommands}}
  {{ .CommandPath}} [command]{{end}}{{if gt .Aliases 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{ .Example }}{{end}}{{ if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if .IsAvailableCommand}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{ if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimRightSpace}}{{end}}{{ if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimRightSpace}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsHelpCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{ if .HasAvailableSubCommands }}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

// PrintComponentInfo prints Component Information like path, URL & storage
func PrintComponentInfo(currentComponentName string, componentDesc component.Description) {
	fmt.Printf("Component Name: %v\nType: %v\n", currentComponentName, componentDesc.ComponentImageType)
	// Source
	if componentDesc.Path != "" {
		fmt.Printf("Source: %v\n", componentDesc.Path)
	}

	// Env
	if componentDesc.Env != nil {
		fmt.Println("\nEnvironment variables:")
		for _, env := range componentDesc.Env {
			fmt.Printf(" - %v=%v\n", env.Name, env.Value)
		}
	}
	// Storage
	if len(componentDesc.Storage.Items) > 0 {
		fmt.Println("\nStorage:")
		for _, store := range componentDesc.Storage.Items {
			fmt.Printf(" - %v of size %v mounted to %v\n", store.Name, store.Spec.Size, store.Spec.Path)
		}
	}
	// URL
	if componentDesc.URLs.Items != nil {
		fmt.Println("\nURLs")
		for _, componentUrl := range componentDesc.URLs.Items {

			fmt.Printf(" - %v exposed via %v\n", url.GetURLString(componentUrl.Spec.Protocol, componentUrl.Spec.URL), componentUrl.Spec.Port)
		}

	}
	// Linked services
	if len(componentDesc.LinkedServices) > 0 {
		fmt.Print("Linked Services: ")
		fmt.Printf("%v\n", strings.Join(componentDesc.LinkedServices, ","))
	}
	// Linked components
	if len(componentDesc.LinkedComponents) > 0 {
		fmt.Println("Linked Components")
		for name, ports := range componentDesc.LinkedComponents {
			if len(ports) > 0 {
				fmt.Printf("Name: %v - Port(s): %v\n", name, strings.Join(ports, ","))
			}
		}
	}
}

// GetFullName generates a command's full name based on its parent's full name and its own name
func GetFullName(parentName, name string) string {
	return parentName + " " + name
}
