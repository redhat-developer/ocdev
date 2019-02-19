package project

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/redhat-developer/odo/pkg/application"
	"github.com/redhat-developer/odo/pkg/component"
	"github.com/redhat-developer/odo/pkg/log"
	"github.com/redhat-developer/odo/pkg/occlient"
	"github.com/redhat-developer/odo/pkg/odo/genericclioptions"
	odoutil "github.com/redhat-developer/odo/pkg/odo/util"
	"github.com/redhat-developer/odo/pkg/odo/util/completion"

	"github.com/spf13/cobra"
)

// RecommendedCommandName is the recommended project command name
const RecommendedCommandName = "project"

// NewCmdProject implements the project odo command
func NewCmdProject(name, fullName string) *cobra.Command {

	projectCreateCmd := NewCmdProjectCreate(createRecommendedCommandName, odoutil.GetFullName(fullName, createRecommendedCommandName))
	projectSetCmd := NewCmdProjectSet(setRecommendedCommandName, odoutil.GetFullName(fullName, setRecommendedCommandName))
	projectListCmd := NewCmdProjectList(listRecommendedCommandName, odoutil.GetFullName(fullName, listRecommendedCommandName))
	projectDeleteCmd := NewCmdProjectDelete(deleteRecommendedCommandName, odoutil.GetFullName(fullName, deleteRecommendedCommandName))
	projectGetCmd := NewCmdProjectGet(getRecommendedCommandName, odoutil.GetFullName(fullName, getRecommendedCommandName))

	projectCmd := &cobra.Command{
		Use:   name + " [options]",
		Short: "Perform project operations",
		Long:  "Perform project operations",
		Example: fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s\n\n%s",
			projectSetCmd.Example,
			projectCreateCmd.Example,
			projectListCmd.Example,
			projectDeleteCmd.Example,
			projectGetCmd.Example),
		// 'odo project' is the same as 'odo project get'
		// 'odo project <project_name>' is the same as 'odo project set <project_name>'
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 && args[0] != getRecommendedCommandName && args[0] != setRecommendedCommandName {
				projectSetCmd.Run(cmd, args)
			} else {
				projectGetCmd.Run(cmd, args)
			}
		},
	}

	projectCmd.Flags().AddFlagSet(projectGetCmd.Flags())
	projectCmd.AddCommand(projectGetCmd)
	projectCmd.AddCommand(projectSetCmd)
	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectDeleteCmd)
	projectCmd.AddCommand(projectListCmd)

	// Add a defined annotation in order to appear in the help menu
	projectCmd.Annotations = map[string]string{"command": "other"}
	projectCmd.SetUsageTemplate(odoutil.CmdUsageTemplate)

	completion.RegisterCommandHandler(projectSetCmd, completion.ProjectNameCompletionHandler)
	completion.RegisterCommandHandler(projectDeleteCmd, completion.ProjectNameCompletionHandler)

	return projectCmd
}

// AddProjectFlag adds a `project` flag to the given cobra command
// Also adds a completion handler to the flag
func AddProjectFlag(cmd *cobra.Command) {
	cmd.Flags().String(genericclioptions.ProjectFlagName, "", "Project, defaults to active project")
	completion.RegisterCommandFlagHandler(cmd, "project", completion.ProjectNameCompletionHandler)
}

// printDeleteProjectInfo prints objects affected by project deletion
func printDeleteProjectInfo(client *occlient.Client, projectName string) error {
	applicationList, err := application.ListInProject(client, projectName)
	if err != nil {
		return errors.Wrap(err, "failed to get application list")
	}
	// List the applications
	if len(applicationList) != 0 {
		log.Info("This project contains the following applications, which will be deleted")
		for _, app := range applicationList {
			log.Info(" Application ", app.Name)

			// List the components
			componentList, err := component.List(client, app.Name)
			if err != nil {
				return errors.Wrap(err, "failed to get Component list")
			}
			if len(componentList) != 0 {
				log.Info("  This application has following components that will be deleted")

				for _, currentComponent := range componentList {
					componentDesc, err := component.GetComponentDesc(client, currentComponent.ComponentName, app.Name, app.Project)
					if err != nil {
						return errors.Wrap(err, "unable to get component description")
					}
					log.Info("  component named ", currentComponent.ComponentName)

					if len(componentDesc.URLs.Items) != 0 {
						log.Info("    This component has following urls that will be deleted with component")
						for _, url := range componentDesc.URLs.Items {
							log.Info("     URL named ", url.GetName(), " with value ", url.Spec.URL)
						}
					}

					if len(componentDesc.LinkedServices) != 0 {
						log.Info("    This component has following services linked to it, which will get unlinked")
						for _, linkedService := range componentDesc.LinkedServices {
							log.Info("     Service named ", linkedService)
						}
					}

					if len(componentDesc.Storage) != 0 {
						log.Info("    This component has following storages which will be deleted with the component")
						for _, store := range componentDesc.Storage {
							log.Info("     Storage named ", store.Name, " of size ", store.Size)
						}
					}
				}
			}
		}
	}
	return nil
}
