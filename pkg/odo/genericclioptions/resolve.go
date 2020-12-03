package genericclioptions

import (
	"fmt"

	"github.com/openshift/odo/pkg/envinfo"
	"github.com/openshift/odo/pkg/odo/util"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResolveAppFlag resolves the app from the flag
func ResolveAppFlag(command *cobra.Command) string {
	appFlag := FlagValueIfSet(command, ApplicationFlagName)
	if len(appFlag) > 0 {
		return appFlag
	}
	return DefaultAppName
}

// resolveProject resolves project
func (o *internalCxt) resolveProject(localConfiguration envinfo.LocalConfigProvider) {
	var namespace string
	command := o.command
	projectFlag := FlagValueIfSet(command, ProjectFlagName)
	if len(projectFlag) > 0 {
		// if project flag was set, check that the specified project exists and use it
		project, err := o.Client.GetProject(projectFlag)
		if err != nil || project == nil {
			util.LogErrorAndExit(err, "")
		}
		namespace = projectFlag
	} else {
		namespace = localConfiguration.GetNamespace()
		if namespace == "" {
			namespace = o.Client.Namespace
			if len(namespace) <= 0 {
				errFormat := "Could not get current project. Please create or set a project\n\t%s project create|set <project_name>"
				checkProjectCreateOrDeleteOnlyOnInvalidNamespace(command, errFormat)
			}
		}

		// check that the specified project exists
		_, err := o.Client.GetProject(namespace)
		if err != nil {
			e1 := fmt.Sprintf("You don't have permission to create or set project '%s' or the project doesn't exist. Please create or set a different project\n\t", namespace)
			errFormat := fmt.Sprint(e1, "%s project create|set <project_name>")
			checkProjectCreateOrDeleteOnlyOnInvalidNamespace(command, errFormat)
		}
	}
	o.Client.GetKubeClient().Namespace = namespace
	o.Client.Namespace = namespace
	o.Project = namespace
	if o.KClient != nil {
		o.KClient.Namespace = namespace
	}
}

// resolveNamespace resolves namespace for devfile component
func (o *internalCxt) resolveNamespace(configProvider envinfo.LocalConfigProvider) {
	var namespace string
	command := o.command
	projectFlag := FlagValueIfSet(command, ProjectFlagName)
	if len(projectFlag) > 0 {
		// if namespace flag was set, check that the specified namespace exists and use it
		_, err := o.KClient.KubeClient.CoreV1().Namespaces().Get(projectFlag, metav1.GetOptions{})
		// do not error out when its odo delete -a, so that we let users delete the local config on missing namespace
		if command.HasParent() && command.Parent().Name() != "project" && !(command.Name() == "delete" && command.Flags().Changed("all")) {
			util.LogErrorAndExit(err, "")
		}
		namespace = projectFlag
	} else {
		namespace = configProvider.GetNamespace()
		if namespace == "" {
			namespace = o.KClient.Namespace
			if len(namespace) <= 0 {
				errFormat := "Could not get current namespace. Please create or set a namespace\n"
				checkProjectCreateOrDeleteOnlyOnInvalidNamespace(command, errFormat)
			}
		}

		// check that the specified namespace exists
		_, err := o.KClient.KubeClient.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
		if err != nil {
			errFormat := fmt.Sprintf("You don't have permission to create or set namespace '%s' or the namespace doesn't exist. Please create or set a different namespace\n\t", namespace)
			// errFormat := fmt.Sprint(e1, "%s project create|set <project_name>")
			checkProjectCreateOrDeleteOnlyOnInvalidNamespaceNoFmt(command, errFormat)
		}
	}
	o.Client.Namespace = namespace
	o.Client.GetKubeClient().Namespace = namespace
	o.KClient.Namespace = namespace
	o.Project = namespace
}

// resolveApp resolves the app
func (o *internalCxt) resolveApp(createAppIfNeeded bool, localConfiguration envinfo.LocalConfigProvider) {
	var app string
	command := o.command
	appFlag := FlagValueIfSet(command, ApplicationFlagName)
	if len(appFlag) > 0 {
		app = appFlag
	} else {
		app = localConfiguration.GetApplication()
		if app == "" {
			if createAppIfNeeded {
				app = DefaultAppName
			}
		}
	}
	o.Application = app
}

// resolveComponent resolves component
func (o *internalCxt) resolveAndSetComponent(command *cobra.Command, localConfiguration envinfo.LocalConfigProvider) string {
	var cmp string
	cmpFlag := FlagValueIfSet(command, ComponentFlagName)
	if len(cmpFlag) == 0 {
		// retrieve the current component if it exists if we didn't set the component flag
		cmp = localConfiguration.GetName()
	} else {
		// if flag is set, check that the specified component exists
		o.checkComponentExistsOrFail(cmpFlag)
		cmp = cmpFlag
	}
	o.cmp = cmp
	return cmp
}
