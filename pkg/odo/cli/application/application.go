package application

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/redhat-developer/odo/pkg/occlient"
	"github.com/redhat-developer/odo/pkg/odo/genericclioptions"
	"github.com/redhat-developer/odo/pkg/odo/util"
	"os"
	"strings"

	"github.com/redhat-developer/odo/pkg/odo/util/completion"

	"text/tabwriter"

	"github.com/redhat-developer/odo/pkg/application"
	"github.com/redhat-developer/odo/pkg/component"
	"github.com/spf13/cobra"
)

var (
	applicationShortFlag       bool
	applicationForceDeleteFlag bool
)

// applicationCmd represents the app command
var applicationCmd = &cobra.Command{
	Use:   "app",
	Short: "Perform application operations",
	Long:  `Performs application operations related to your OpenShift project.`,
	Example: fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		applicationCreateCmd.Example,
		applicationGetCmd.Example,
		applicationDeleteCmd.Example,
		applicationDescribeCmd.Example,
		applicationListCmd.Example,
		applicationSetCmd.Example),
	Aliases: []string{"application"},
	// 'odo app' is the same as 'odo app get'
	// 'odo app <application_name>' is the same as 'odo app set <application_name>'
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 && args[0] != "get" && args[0] != "set" {
			applicationSetCmd.Run(cmd, args)
		} else {
			applicationGetCmd.Run(cmd, args)
		}
	},
}

var applicationCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an application",
	Long: `Create an application.
If no app name is passed, a default app name will be auto-generated.
	`,
	Example: `  # Create an application
  odo app create myapp
  odo app create
	`,
	Args: cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		context := genericclioptions.NewContext(cmd)
		client := context.Client
		projectName := context.Project

		var appName string
		if len(args) == 1 {
			// The only arg passed is the app name
			appName = args[0]
		} else {
			// Desired app name is not passed so, generate a new app name
			// Fetch existing list of apps
			apps, err := application.List(client)
			util.CheckError(err, "")

			// Generate a random name that's not already in use for the existing apps
			appName, err = application.GetDefaultAppName(apps)
			util.CheckError(err, "")
		}
		// validate application name
		err := util.ValidateName(appName)
		util.CheckError(err, "")
		fmt.Printf("Creating application: %v in project: %v\n", appName, projectName)
		err = application.Create(client, appName)
		util.CheckError(err, "")
		err = application.SetCurrent(client, appName)

		// TODO: updating the app name should be done via SetCurrent and passing the Context
		// not strictly needed here but Context should stay in sync
		context.Application = appName

		util.CheckError(err, "")
		fmt.Printf("Switched to application: %v in project: %v\n", appName, projectName)
	},
}

var applicationGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get the active application",
	Long:  "Get the active application",
	Example: `  # Get the currently active application
  odo app get
	`,
	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		context := genericclioptions.NewContext(cmd)
		projectName := context.Project
		app := context.Application
		if applicationShortFlag {
			fmt.Print(app)
			return
		}
		if app == "" {
			fmt.Printf("There's no active application.\nYou can create one by running 'odo application create <name>'.\n")
			return
		}
		fmt.Printf("The current application is: %v in project: %v\n", app, projectName)
	},
}

var applicationDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete the given application",
	Long:  "Delete the given application",
	Example: `  # Delete the application
  odo app delete myapp
	`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		context := genericclioptions.NewContext(cmd)
		client := context.Client
		projectName := context.Project
		appName := context.Application
		if len(args) == 1 {
			// If app name passed, consider it for deletion
			appName = args[0]
		}

		var confirmDeletion string

		// Print App Information which will be deleted
		err := printDeleteAppInfo(client, appName)
		util.CheckError(err, "")
		exists, err := application.Exists(client, appName)
		util.CheckError(err, "")
		if !exists {
			fmt.Printf("Application %v in project %v does not exist\n", appName, projectName)
			os.Exit(1)
		}

		if applicationForceDeleteFlag {
			confirmDeletion = "y"
		} else {
			fmt.Printf("Are you sure you want to delete the application: %v from project: %v? [y/N] ", appName, projectName)
			fmt.Scanln(&confirmDeletion)
		}

		if strings.ToLower(confirmDeletion) == "y" {
			err := application.Delete(client, appName)
			util.CheckError(err, "")
			fmt.Printf("Deleted application: %s from project: %v\n", appName, projectName)
		} else {
			fmt.Printf("Aborting deletion of application: %v\n", appName)
		}
	},
}

var applicationListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all applications in the current project",
	Long:  "List all applications in the current project.",
	Example: `  # List all applications in the current project
  odo app list

  # List all applications in the specified project
  odo app list --project myproject
	`,
	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		context := genericclioptions.NewContext(cmd)
		client := context.Client
		projectName := context.Project

		apps, err := application.ListInProject(client)
		util.CheckError(err, "unable to get list of applications")
		if len(apps) > 0 {
			fmt.Printf("The project '%v' has the following applications:\n", projectName)
			tabWriter := tabwriter.NewWriter(os.Stdout, 5, 2, 3, ' ', tabwriter.TabIndent)
			fmt.Fprintln(tabWriter, "ACTIVE", "\t", "NAME")
			for _, app := range apps {
				activeMark := " "
				if app.Active {
					activeMark = "*"
				}
				fmt.Fprintln(tabWriter, activeMark, "\t", app.Name)
			}
			tabWriter.Flush()
		} else {
			fmt.Printf("There are no applications deployed in the project '%v'.\n", projectName)
		}
	},
}

var applicationSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set application as active",
	Long:  "Set application as active",
	Example: `  # Set an application as active
  odo app set myapp
	`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("Please provide application name")
		}
		if len(args) > 1 {
			return fmt.Errorf("Only one argument (application name) is allowed")
		}
		return nil
	}, Run: func(cmd *cobra.Command, args []string) {
		context := genericclioptions.NewContext(cmd)
		client := context.Client
		projectName := context.Project

		// error if application does not exist
		appName := args[0]
		exists, err := application.Exists(client, appName)
		util.CheckError(err, "unable to check if application exists")
		if !exists {
			fmt.Printf("Application %v does not exist\n", appName)
			os.Exit(1)
		}

		err = application.SetCurrent(client, appName)
		util.CheckError(err, "")
		fmt.Printf("Switched to application: %v in project: %v\n", args[0], projectName)

		// TODO: updating the app name should be done via SetCurrent and passing the Context
		// not strictly needed here but Context should stay in sync
		context.Application = appName
	},
}

var applicationDescribeCmd = &cobra.Command{
	Use:   "describe [application_name]",
	Short: "Describe the given application",
	Long:  "Describe the given application",
	Args:  cobra.MaximumNArgs(1),
	Example: `  # Describe webapp application,
  odo app describe webapp
	`,
	Run: func(cmd *cobra.Command, args []string) {
		context := genericclioptions.NewContext(cmd)
		client := context.Client
		projectName := context.Project

		appName := context.Application
		if len(args) == 0 {
			if appName == "" {
				fmt.Printf("There's no active application in project: %v\n", projectName)
				os.Exit(1)
			}
		} else {
			appName = args[0]
			//Check whether application exist or not
			exists, err := application.Exists(client, appName)
			util.CheckError(err, "")
			if !exists {
				fmt.Printf("Application with the name %s does not exist in %s \n", appName, projectName)
				os.Exit(1)
			}
		}

		// List of Component
		componentList, err := component.List(client, appName)
		util.CheckError(err, "")
		if len(componentList) == 0 {
			fmt.Printf("Application %s has no components deployed.\n", appName)
			os.Exit(1)
		}
		fmt.Printf("Application %s has:\n", appName)

		for _, currentComponent := range componentList {
			componentType, path, componentURL, appStore, err := component.GetComponentDesc(client, currentComponent.Name, appName)
			util.CheckError(err, "")
			util.PrintComponentInfo(currentComponent.Name, componentType, path, componentURL, appStore)
		}
	},
}

func NewCmdApplication() *cobra.Command {
	applicationDeleteCmd.Flags().BoolVarP(&applicationForceDeleteFlag, "force", "f", false, "Delete application without prompting")

	applicationGetCmd.Flags().BoolVarP(&applicationShortFlag, "short", "q", false, "If true, display only the application name")
	// add flags from 'get' to application command
	applicationCmd.Flags().AddFlagSet(applicationGetCmd.Flags())

	applicationCmd.AddCommand(applicationListCmd)
	applicationCmd.AddCommand(applicationDeleteCmd)
	applicationCmd.AddCommand(applicationGetCmd)
	applicationCmd.AddCommand(applicationCreateCmd)
	applicationCmd.AddCommand(applicationSetCmd)
	applicationCmd.AddCommand(applicationDescribeCmd)

	//Adding `--project` flag
	addProjectFlag(applicationListCmd)
	addProjectFlag(applicationCreateCmd)
	addProjectFlag(applicationDeleteCmd)
	addProjectFlag(applicationDescribeCmd)
	addProjectFlag(applicationSetCmd)
	addProjectFlag(applicationGetCmd)

	// Add a defined annotation in order to appear in the help menu
	applicationCmd.Annotations = map[string]string{"command": "other"}
	applicationCmd.SetUsageTemplate(util.CmdUsageTemplate)

	completion.RegisterCommandHandler(applicationDescribeCmd, completion.AppCompletionHandler)
	completion.RegisterCommandHandler(applicationDeleteCmd, completion.AppCompletionHandler)
	completion.RegisterCommandHandler(applicationSetCmd, completion.AppCompletionHandler)

	return applicationCmd
}

func addProjectFlag(cmd *cobra.Command) {
	genericclioptions.AddProjectFlag(cmd)
	completion.RegisterCommandFlagHandler(cmd, "project", completion.ProjectNameCompletionHandler)
}

// printDeleteAppInfo will print things which will be deleted
func printDeleteAppInfo(client *occlient.Client, appName string) error {
	componentList, err := component.List(client, appName)
	if err != nil {
		return errors.Wrap(err, "failed to get Component list")
	}

	for _, currentComponent := range componentList {
		_, _, componentURL, appStore, err := component.GetComponentDesc(client, currentComponent.Name, appName)
		if err != nil {
			return errors.Wrap(err, "unable to get component description")
		}
		fmt.Println("Component", currentComponent.Name, "will be deleted.")

		if len(componentURL) != 0 {
			fmt.Println("  Externally exposed URL will be removed")
		}

		for _, store := range appStore {
			fmt.Println("  Storage", store.Name, "of size", store.Size, "will be removed")
		}

	}
	return nil
}
