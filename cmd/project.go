package cmd

import (
	"fmt"
	"github.com/redhat-developer/ocdev/pkg/project"
	"github.com/spf13/cobra"
	"os"
)

var (
	projectShortFlag bool
)

var projectCmd = &cobra.Command{
	Use:   "project [options]",
	Short: "Perform project operations",
	Run:   projectGetCmd.Run,
}

var projectSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set the current active project",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		client := getOcClient()
		current, err := project.GetCurrent(client)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		err = project.SetCurrent(client, projectName)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if projectShortFlag {
			fmt.Println(projectName)
		} else {
			if current == projectName {
				fmt.Printf("Already on project : %v\n", projectName)
			} else {
				fmt.Printf("Now using project : %v\n", projectName)
			}
		}
	},
}

var projectGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get the active project",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		client := getOcClient()
		project, err := project.GetCurrent(client)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if projectShortFlag {
			fmt.Println(project)
		} else {
			fmt.Printf("The current project is: %v\n", project)
		}
	},
}

var projectCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new project",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		client := getOcClient()
		err := project.CreateProject(client, projectName)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Printf("New project created and now using project : %v\n", projectName)
	},
}

func init() {
	projectGetCmd.Flags().BoolVarP(&projectShortFlag, "short", "q", false, "If true, display only the application name")
	projectSetCmd.Flags().BoolVarP(&projectShortFlag, "short", "q", false, "If true, display only the application name")
	projectCmd.Flags().AddFlagSet(projectGetCmd.Flags())
	projectCmd.AddCommand(projectGetCmd)
	projectCmd.AddCommand(projectSetCmd)
	projectCmd.AddCommand(projectCreateCmd)
	rootCmd.AddCommand(projectCmd)
}
