package cmd

import (
	"fmt"
	"net/url"
	"os"

	"github.com/fatih/color"
	"github.com/redhat-developer/odo/pkg/application"
	"github.com/redhat-developer/odo/pkg/component"
	"github.com/redhat-developer/odo/pkg/project"

	"path/filepath"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push [component name]",
	Short: "Push source code to a component",
	Long:  `Push source code to a component.`,
	Example: `  # Push source code to the current component
  odo push

  # Push data to the current component from the original source.
  odo push

  # Push source code in ~/mycode to component called my-component
  odo push my-component --local ~/mycode
	`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		stdout := color.Output
		client := getOcClient()
		applicationName, err := application.GetCurrent(client)
		checkError(err, "")
		projectName := project.GetCurrent(client)

		var componentName string
		if len(args) == 0 {
			var err error
			glog.V(4).Info("No component name passed, assuming current component")
			componentName, err = component.GetCurrent(client, applicationName, projectName)
			checkError(err, "unable to get current component")
			if componentName == "" {
				fmt.Println("No component is set as active.")
				fmt.Println("Use 'odo component set <component name> to set and existing component as active or call this command with component name as and argument.")
				os.Exit(1)
			}
		} else {
			componentName = args[0]
			exists, err := component.Exists(client, componentName, applicationName, projectName)
			checkError(err, "")
			if !exists {
				fmt.Printf("Component with name %s does not exist in the current application\n", componentName)
				os.Exit(1)
			}
		}
		fmt.Printf("Pushing changes to component: %v\n", componentName)

		sourceType, sourcePath, err := component.GetComponentSource(client, componentName, applicationName, projectName)
		checkError(err, "unable to get component source")
		switch sourceType {
		case "local", "binary":
			// use value of '--dir' as source if it was used
			if len(componentLocal) != 0 {
				if sourceType == "binary" {
					fmt.Printf("Unable to push local directory:%s to component %s that uses binary %s.\n", componentLocal, componentName, sourcePath)
					os.Exit(1)
				}
				sourcePath = componentLocal
			}
			u, err := url.Parse(sourcePath)
			checkError(err, fmt.Sprintf("unable to parse source %s from component %s", sourcePath, componentName))

			if u.Scheme != "" && u.Scheme != "file" {
				fmt.Printf("Component %s has invalid source path %s", componentName, u.Scheme)
				os.Exit(1)
			}
			_, err = os.Stat(u.Path)
			if err != nil {
				checkError(err, "")
			}

			if sourceType == "local" {
				glog.V(4).Infof("Copying directory %s to pod", u.Path)
				err = component.PushLocal(client, componentName, applicationName, u.Path, os.Stdout, []string{})
			} else {
				dir := filepath.Dir(u.Path)
				glog.V(4).Infof("Copying file %s to pod", u.Path)
				err = component.PushLocal(client, componentName, applicationName, dir, os.Stdout, []string{u.Path})
			}
			checkError(err, fmt.Sprintf("failed to push component: %v", componentName))

		case "git":
			// currently we don't support changing build type
			// it doesn't make sense to use --dir with git build
			if len(componentLocal) != 0 {
				fmt.Printf("Unable to push local directory:%s to component %s that uses Git repository:%s.\n", componentLocal, componentName, sourcePath)
				os.Exit(1)
			}
			err := component.Build(client, componentName, applicationName, true, true, stdout)
			checkError(err, fmt.Sprintf("failed to push component: %v", componentName))
		}

		fmt.Printf("changes successfully pushed to component: %v\n", componentName)
	},
}

func init() {
	pushCmd.Flags().StringVar(&componentLocal, "local", "", "Use given local directory as a source for component. (It must be a local component)")

	// Add a defined annotation in order to appear in the help menu
	pushCmd.Annotations = map[string]string{"command": "component"}
	pushCmd.SetUsageTemplate(cmdUsageTemplate)

	rootCmd.AddCommand(pushCmd)
}
