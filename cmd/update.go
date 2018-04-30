package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/redhat-developer/odo/pkg/application"
	"github.com/redhat-developer/odo/pkg/component"
	"github.com/redhat-developer/odo/pkg/project"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:  "update",
	Args: cobra.MaximumNArgs(1),
	Example: `  # Change the source of a currently active component to local (use the current directory as a source)
  odo update --local

  # Change the source of the frontend component to local with source in ./frontend directory
  odo update frontend --local ./frontend

  # Change the source of a currently active component to git 
  odo update --git https://github.com/openshift/nodejs-ex.git

  # Change the source of the component named node-ex to git
  odo update node-ex --git https://github.com/openshift/nodejs-ex.git

  # Change the source of the component named wildfly to a binary named sample.war in ./downloads directory
  odo update wildfly --binary ./downloads/sample.war
	`,
	Short: "Change the source of a component",
	Run: func(cmd *cobra.Command, args []string) {
		client := getOcClient()
		applicationName, err := application.GetCurrent(client)
		checkError(err, "")
		projectName := project.GetCurrent(client)

		checkFlag := 0

		if len(componentBinary) != 0 {
			checkFlag++
		}
		if len(componentGit) != 0 {
			checkFlag++
		}
		if len(componentLocal) != 0 {
			checkFlag++
		}

		if checkFlag != 1 {
			fmt.Println("The source can be either --binary or --local or --git")
			os.Exit(1)
		}

		var (
			componentName string
		)
		if len(args) == 0 {
			componentName, err = component.GetCurrent(client, applicationName, projectName)
			checkError(err, "unable to get current component")
			if len(componentName) == 0 {
				appList, err := application.List(client)
				checkError(err, "")
				if len(appList) == 0 {
					fmt.Println("Cannot update as no application exists in the current project")
					os.Exit(1)
				}
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

		if len(applicationName) == 0 {
			fmt.Println("Cannot update as no application is set as active")
			os.Exit(1)
		}

		if len(componentGit) != 0 {
			err := component.Update(client, componentName, applicationName, "git", componentGit)
			checkError(err, "")
			fmt.Printf("The component %s was updated successfully\n", componentName)
		} else if len(componentLocal) != 0 {
			// we want to use and save absolute path for component
			dir, err := filepath.Abs(componentLocal)
			checkError(err, "")
			err = component.Update(client, componentName, applicationName, "local", dir)
			checkError(err, "")
			fmt.Printf("The component %s was updated successfully\n", componentName)
		} else if len(componentBinary) != 0 {
			path, err := filepath.Abs(componentBinary)
			checkError(err, "")
			err = component.Update(client, componentName, applicationName, "binary", path)
			checkError(err, "")
			fmt.Printf("The component %s was updated successfully\n", componentName)
		}
	},
}

func init() {
	updateCmd.Flags().StringVar(&componentBinary, "binary", "", "binary artifact")
	updateCmd.Flags().StringVar(&componentGit, "git", "", "git source")
	updateCmd.Flags().StringVar(&componentLocal, "local", "", "Use local directory as a source for component.")

	rootCmd.AddCommand(updateCmd)
}
