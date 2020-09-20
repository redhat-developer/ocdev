package component

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/openshift/odo/pkg/application"
	"github.com/openshift/odo/pkg/devfile"
	"github.com/openshift/odo/pkg/machineoutput"
	"github.com/openshift/odo/pkg/project"
	"github.com/openshift/odo/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/klog"

	applabels "github.com/openshift/odo/pkg/application/labels"

	"github.com/openshift/odo/pkg/component"
	"github.com/openshift/odo/pkg/log"
	appCmd "github.com/openshift/odo/pkg/odo/cli/application"
	projectCmd "github.com/openshift/odo/pkg/odo/cli/project"
	"github.com/openshift/odo/pkg/odo/genericclioptions"
	odoutil "github.com/openshift/odo/pkg/odo/util"
	"github.com/openshift/odo/pkg/odo/util/completion"

	ktemplates "k8s.io/kubectl/pkg/util/templates"
)

// ListRecommendedCommandName is the recommended watch command name
const ListRecommendedCommandName = "list"

var listExample = ktemplates.Examples(`  # List all components in the application
%[1]s
  `)

// ListOptions is a dummy container to attach complete, validate and run pattern
type ListOptions struct {
	pathFlag             string
	allAppsFlag          bool
	componentContext     string
	componentType        string
	hasDCSupport         bool
	hasDevfileComponents bool
	hasS2IComponents     bool
	devfilePath          string
	*genericclioptions.Context
}

// NewListOptions returns new instance of ListOptions
func NewListOptions() *ListOptions {
	return &ListOptions{}
}

// Complete completes log args
func (lo *ListOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {

	lo.devfilePath = filepath.Join(lo.componentContext, DevfilePath)

	if util.CheckPathExists(lo.devfilePath) {

		lo.Context = genericclioptions.NewDevfileContext(cmd)
		lo.Client = genericclioptions.Client(cmd)
		lo.hasDCSupport, err = lo.Client.IsDeploymentConfigSupported()
		if err != nil {
			return err
		}
		devfile, err := devfile.ParseAndValidate(lo.devfilePath)
		if err != nil {
			return err
		}
		lo.componentType = devfile.Data.GetMetadata().Name

	} else {
		// here we use the config.yaml derived context if its present, else we use information from user's kubeconfig
		// as odo list should work in a non-component directory too

		if util.CheckKubeConfigExist() {
			klog.V(4).Infof("New Context")
			lo.Context = genericclioptions.NewContext(cmd, false, true)
			// we intentionally leave this error out
			lo.hasDCSupport, _ = lo.Client.IsDeploymentConfigSupported()

		} else {
			klog.V(4).Infof("New Config Context")
			lo.Context = genericclioptions.NewConfigContext(cmd)
			// for disconnected situation we just assume we have DC support
			lo.hasDCSupport = true
		}
	}

	return

}

// Validate validates the list parameters
func (lo *ListOptions) Validate() (err error) {

	if len(lo.Application) != 0 && lo.allAppsFlag {
		klog.V(4).Infof("either --app and --all-apps both provided or provided --all-apps in a folder has app, use --all-apps anyway")
	}

	if util.CheckPathExists(lo.devfilePath) {
		if lo.Context.Application == "" && lo.Context.KClient.Namespace == "" {
			return odoutil.ThrowContextError()
		}
		return nil
	}
	var project, app string

	if !util.CheckKubeConfigExist() {
		project = lo.LocalConfigInfo.GetProject()
		app = lo.LocalConfigInfo.GetApplication()

	} else {
		project = lo.Context.Project
		app = lo.Application
	}
	if !lo.allAppsFlag && lo.pathFlag == "" && (project == "" || app == "") {
		return odoutil.ThrowContextError()
	}
	return nil

}

// Run has the logic to perform the required actions as part of command
func (lo *ListOptions) Run() (err error) {

	// --path workflow

	if len(lo.pathFlag) != 0 {

		devfileComps, err := component.ListDevfileComponentsInPath(lo.KClient, filepath.SplitList(lo.pathFlag))
		if err != nil {
			return err
		}
		s2iComps, err := component.ListIfPathGiven(lo.Context.Client, filepath.SplitList(lo.pathFlag))
		if err != nil {
			return err
		}
		combinedComponents := component.GetMachineReadableFormatForCombinedCompList(s2iComps, devfileComps)

		if log.IsJSON() {
			machineoutput.OutputSuccess(combinedComponents)
		} else {

			w := tabwriter.NewWriter(os.Stdout, 5, 2, 3, ' ', tabwriter.TabIndent)

			if len(devfileComps) != 0 {
				lo.hasDevfileComponents = true
				fmt.Fprintln(w, "Devfile Components: ")
				fmt.Fprintln(w, "APP", "\t", "NAME", "\t", "PROJECT", "\t", "STATE", "\t", "CONTEXT")
				for _, comp := range devfileComps {
					fmt.Fprintln(w, comp.Spec.App, "\t", comp.Name, "\t", comp.Namespace, "\t", comp.Status.State, "\t", comp.Status.Context)
				}
			}
			if lo.hasDevfileComponents {
				fmt.Fprintln(w)
			}

			if len(s2iComps) != 0 {
				lo.hasS2IComponents = true
				fmt.Fprintln(w, "S2I Components: ")
				fmt.Fprintln(w, "APP", "\t", "NAME", "\t", "PROJECT", "\t", "TYPE", "\t", "SOURCETYPE", "\t", "STATE", "\t", "CONTEXT")
				for _, comp := range s2iComps {
					fmt.Fprintln(w, comp.Spec.App, "\t", comp.Name, "\t", comp.Namespace, "\t", comp.Spec.Type, "\t", comp.Spec.SourceType, "\t", comp.Status.State, "\t", comp.Status.Context)

				}
			}

			// if we dont have any then
			if !lo.hasDevfileComponents && !lo.hasS2IComponents {
				fmt.Fprintln(w, "No components found")
			}

			w.Flush()
		}
		return nil
	}

	// non --path workflow below
	// read the code like
	// -> experimental
	//	or	|-> --all-apps
	//		|-> the current app
	// -> non-experimental
	//	or	|-> --all-apps
	//		|-> the current app

	// experimental workflow

	var deploymentList *appsv1.DeploymentList

	var selector string
	// TODO: wrap this into a component list for docker support
	if lo.allAppsFlag {
		selector = project.GetSelector()
	} else {
		selector = applabels.GetSelector(lo.Application)
	}

	var devfileComponents []component.DevfileComponent
	currentComponentState := component.StateTypeNotPushed

	if lo.KClient != nil {
		deploymentList, err = lo.KClient.ListDeployments(selector)
		if err != nil {
			return err
		}
		devfileComponents = append(devfileComponents, component.DevfileComponentsFromDeployments(deploymentList)...)
		for _, comp := range devfileComponents {
			if lo.EnvSpecificInfo != nil {
				// if we can find a component from the listing from server then the local state is pushed
				if lo.EnvSpecificInfo.EnvInfo.MatchComponent(comp.Spec.Name, comp.Spec.App, comp.Namespace) {
					currentComponentState = component.StateTypePushed
				}
			}
		}
	}

	// 1st condition - only if we are using the same application or all-apps are provided should we show the current component
	// 2nd condition - if the currentComponentState is unpushed that means it didn't show up in the list above
	if lo.EnvSpecificInfo != nil {
		envinfo := lo.EnvSpecificInfo.EnvInfo
		if (envinfo.GetApplication() == lo.Application || lo.allAppsFlag) && currentComponentState == component.StateTypeNotPushed {
			comp := component.NewDevfileComponent(envinfo.GetName())
			comp.Status.State = component.StateTypeNotPushed
			comp.Namespace = envinfo.GetNamespace()
			comp.Spec.App = envinfo.GetApplication()
			comp.Spec.Type = lo.componentType
			comp.Spec.Name = envinfo.GetName()
			devfileComponents = append(devfileComponents, comp)
		}
	}

	// non-experimental workflow

	var components []component.Component
	// we now check if DC is supported
	if lo.hasDCSupport {

		if lo.allAppsFlag {
			// retrieve list of application
			apps, err := application.List(lo.Client)
			if err != nil {
				return err
			}

			if len(apps) == 0 && lo.LocalConfigInfo.Exists() {
				comps, err := component.List(lo.Client, lo.LocalConfigInfo.GetApplication(), lo.LocalConfigInfo)
				if err != nil {
					return err
				}
				components = append(components, comps.Items...)
			}

			// interating over list of application and get list of all components
			for _, app := range apps {
				comps, err := component.List(lo.Client, app, lo.LocalConfigInfo)
				if err != nil {
					return err
				}
				components = append(components, comps.Items...)
			}
		} else {

			componentList, err := component.List(lo.Client, lo.Application, lo.LocalConfigInfo)
			// compat
			components = componentList.Items
			if err != nil {
				return errors.Wrapf(err, "failed to fetch component list")
			}
		}
	}

	w := tabwriter.NewWriter(os.Stdout, 5, 2, 3, ' ', tabwriter.TabIndent)

	if !log.IsJSON() {

		if len(devfileComponents) != 0 {
			lo.hasDevfileComponents = true
			fmt.Fprintln(w, "Devfile Components: ")
			fmt.Fprintln(w, "APP", "\t", "NAME", "\t", "PROJECT", "\t", "TYPE", "\t", "STATE")
			for _, comp := range devfileComponents {
				fmt.Fprintln(w, comp.Spec.App, "\t", comp.Spec.Name, "\t", comp.Namespace, "\t", comp.Spec.Type, "\t", comp.Status.State)
			}
			w.Flush()

		}
		if lo.hasDevfileComponents {
			fmt.Fprintln(w)
		}

		if len(components) != 0 {
			if lo.hasDevfileComponents {
				fmt.Println()
			}
			lo.hasS2IComponents = true
			w := tabwriter.NewWriter(os.Stdout, 5, 2, 3, ' ', tabwriter.TabIndent)
			fmt.Fprintln(w, "Openshift Components: ")
			fmt.Fprintln(w, "APP", "\t", "NAME", "\t", "PROJECT", "\t", "TYPE", "\t", "SOURCETYPE", "\t", "STATE")
			for _, comp := range components {
				fmt.Fprintln(w, comp.Spec.App, "\t", comp.Name, "\t", comp.Namespace, "\t", comp.Spec.Type, "\t", comp.Spec.SourceType, "\t", comp.Status.State)
			}
			w.Flush()
		}

		if !lo.hasDevfileComponents && !lo.hasS2IComponents {
			log.Error("There are no components deployed.")
			return
		}
	} else {
		combinedComponents := component.GetMachineReadableFormatForCombinedCompList(components, devfileComponents)
		machineoutput.OutputSuccess(combinedComponents)
	}

	return
}

// NewCmdList implements the list odo command
func NewCmdList(name, fullName string) *cobra.Command {
	o := NewListOptions()

	var componentListCmd = &cobra.Command{
		Use:         name,
		Short:       "List all components in the current application",
		Long:        "List all components in the current application.",
		Example:     fmt.Sprintf(listExample, fullName),
		Args:        cobra.NoArgs,
		Annotations: map[string]string{"machineoutput": "json", "command": "component"},
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(o, cmd, args)
		},
	}
	genericclioptions.AddContextFlag(componentListCmd, &o.componentContext)
	componentListCmd.Flags().StringVar(&o.pathFlag, "path", "", "path of the directory to scan for odo component directories")
	componentListCmd.Flags().BoolVar(&o.allAppsFlag, "all-apps", false, "list all components from all applications for the current set project")
	componentListCmd.SetUsageTemplate(odoutil.CmdUsageTemplate)

	//Adding `--project` flag
	projectCmd.AddProjectFlag(componentListCmd)
	//Adding `--application` flag
	appCmd.AddApplicationFlag(componentListCmd)

	completion.RegisterCommandFlagHandler(componentListCmd, "path", completion.FileCompletionHandler)

	return componentListCmd
}
