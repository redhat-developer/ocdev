package project

import (
	"github.com/golang/glog"

	"github.com/pkg/errors"
	"github.com/redhat-developer/odo/pkg/occlient"
)

// ApplicationInfo holds information about one project
type ProjectInfo struct {
	// Name of the project
	Name string
	// is this project active?
	Active bool
}

func GetCurrent(client *occlient.Client) string {
	project := client.GetCurrentProjectName()
	return project
}

func SetCurrent(client *occlient.Client, project string) error {
	err := client.SetCurrentProject(project)
	if err != nil {
		return errors.Wrap(err, "unable to set current project to"+project)
	}
	return nil
}

func Create(client *occlient.Client, projectName string) error {
	err := client.CreateNewProject(projectName)
	if err != nil {
		return errors.Wrap(err, "unable to create new project")
	}
	return nil
}

// Delete deletes the project with name projectName and sets the project with name currentProject as current project and returns errors if any
func Delete(client *occlient.Client, projectName string) error {
	currentProject := GetCurrent(client)

	projects, err := List(client)
	if err != nil {
		return errors.Wrapf(err, "unable to fetch list of projects")
	}

	//Iterate the project list and see the expected change post deletion
	for ind, prj := range projects {
		if prj.Name == projectName {
			projects = append(projects[:ind], projects[ind+1:]...)
		}
	}

	// If there will be any projects post the current deletion,
	// Randomly choose a project from remainder of project list to set as current
	if len(projects) > 0 {
		currentProject = projects[0].Name
	} else {
		// Set the current project to NOPROJECT
		currentProject = "NOPROJECT"
	}

	// If current project is not same as the project to be deleted, set it as current
	if currentProject != projectName {
		// Set the project to be deleted as current inorder to be able to delete it
		err = SetCurrent(client, projectName)
		if err != nil {
			return errors.Wrapf(err, "Unable to delete project %s", projectName)
		}
	}

	// Delete the requested project
	err = client.DeleteProject(projectName)
	if err != nil {
		return errors.Wrap(err, "unable to delete project")
	}

	// If current project is not NOPROJECT, set currentProject as current project
	if currentProject != "NOPROJECT" {
		glog.V(4).Infof("Setting the current project to %s\n", currentProject)
		err = SetCurrent(client, currentProject)
		if err != nil {
			return errors.Wrapf(err, "unable to set %s as the current project\n", currentProject)
		}
	} else {
		// Nothing to do if there's no project left -- Default oc client way
		glog.V(4).Info("No projects available to mark as current\n")
	}
	return nil
}

func List(client *occlient.Client) ([]ProjectInfo, error) {
	currentProject := client.GetCurrentProjectName()
	allProjects, err := client.GetProjectNames()
	if err != nil {
		return nil, errors.Wrap(err, "cannot get all the projects")
	}
	projects := []ProjectInfo{}
	for _, project := range allProjects {
		isActive := false
		if project == currentProject {
			isActive = true
		}
		projects = append(projects, ProjectInfo{
			Name:   project,
			Active: isActive,
		})
	}
	return projects, nil
}

// Checks whether a project with the given name exists or not
// projectName is the project name to perform check for
// The first returned parameter is a bool indicating if a project with the given name already exists or not
// The second returned parameter is the error that might occurs while execution
func Exists(client *occlient.Client, projectName string) (bool, error) {
	projects, err := List(client)
	if err != nil {
		return false, errors.Wrap(err, "unable to get the project list")
	}
	for _, project := range projects {
		if project.Name == projectName {
			return true, nil
		}
	}
	return false, nil
}
