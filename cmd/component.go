package cmd

import (
	"fmt"
	"github.com/redhat-developer/odo/pkg/component"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// componentCmd represents the component command
var componentCmd = &cobra.Command{
	Use:   "component",
	Short: "Components of application.",
	// 'odo component' is the same as 'odo component get'
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 && args[0] != "get" && args[0] != "set" {
			componentSetCmd.Run(cmd, args)
		} else {
			componentGetCmd.Run(cmd, args)
		}
	},
}

var componentGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get currently active component",
	Long:  "Get currently active component.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debugf("component get called")
		client := getOcClient()
		component, err := component.GetCurrent(client)
		checkError(err, "unable to get current component")
		if componentShortFlag {
			fmt.Print(component)
		} else {
			if component == "" {
				fmt.Printf("No component is set as current\n")
				return
			}
			fmt.Printf("The current component is: %v\n", component)
		}
	},
}

var componentSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set active component.",
	Long:  "Set component as active.",
	Example: `  # Set component named 'frontend' as active
  odo set component frontend
  `,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := getOcClient()
		err := component.SetCurrent(client, args[0])
		checkError(err, "")
		fmt.Printf("Switched to component: %v\n", args[0])
	},
}

func init() {

	componentGetCmd.Flags().BoolVarP(&componentShortFlag, "short", "q", false, "If true, display only the component name")

	// add flags from 'get' to component command
	componentCmd.Flags().AddFlagSet(applicationGetCmd.Flags())

	componentCmd.AddCommand(componentGetCmd)
	componentCmd.AddCommand(componentSetCmd)

	rootCmd.AddCommand(componentCmd)
}
