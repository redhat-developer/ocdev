package cmd

import (
	"fmt"
	"os"

	"github.com/redhat-developer/odo/pkg/component"
	"github.com/spf13/cobra"
)

var describeCmd = &cobra.Command{
	Use:   "describe [component_name]",
	Short: "Describe the given component",
	Long:  `Describe the given component.`,
	Example: `  # Describe nodejs component,
  odo describe nodejs
	`,
	Args: cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		client := getOcClient()

		projectName := setNamespace(client)
		applicationName := getAppName(client)

		var componentName string
		if len(args) == 0 {
			componentName = getComponent(client, "", applicationName, projectName)
		} else {

			componentName = args[0]

			// Checks to see if the component actually exists
			exists, err := component.Exists(client, componentName, applicationName, projectName)
			checkError(err, "")
			if !exists {
				fmt.Printf("Component with the name %s does not exist in the current application\n", componentName)
				os.Exit(1)
			}
		}
		// currentComponent := getComponent(client, componentName, Application, Project)
		componentType, path, componentURL, appStore, err := component.GetComponentDesc(client, componentName, applicationName, projectName)
		checkError(err, "")
		printComponentInfo(componentName, componentType, path, componentURL, appStore)
	},
}

func init() {
	// Add a defined annotation in order to appear in the help menu
	describeCmd.Annotations = map[string]string{"command": "component"}
	describeCmd.SetUsageTemplate(cmdUsageTemplate)

	//Adding `--project` flag
	addProjectFlag(describeCmd)
	//Adding `--application` flag
	addApplicationFlag(describeCmd)

	rootCmd.AddCommand(describeCmd)
}
