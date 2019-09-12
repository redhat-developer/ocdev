package component

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/golang/glog"
	"github.com/openshift/odo/pkg/application"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/openshift/odo/pkg/component"
	"github.com/openshift/odo/pkg/log"
	appCmd "github.com/openshift/odo/pkg/odo/cli/application"
	projectCmd "github.com/openshift/odo/pkg/odo/cli/project"
	"github.com/openshift/odo/pkg/odo/genericclioptions"
	odoutil "github.com/openshift/odo/pkg/odo/util"
	"github.com/openshift/odo/pkg/odo/util/completion"

	ktemplates "k8s.io/kubernetes/pkg/kubectl/cmd/templates"
)

// ListRecommendedCommandName is the recommended watch command name
const ListRecommendedCommandName = "list"

var listExample = ktemplates.Examples(`  # List all components in the application
%[1]s
  `)

// ListOptions is a dummy container to attach complete, validate and run pattern
type ListOptions struct {
	pathFlag         string
	allFlag          bool
	componentContext string
	*genericclioptions.Context
}

// NewListOptions returns new instance of ListOptions
func NewListOptions() *ListOptions {
	return &ListOptions{}
}

// Complete completes log args
func (lo *ListOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	lo.Context = genericclioptions.NewContext(cmd)
	return
}

// Validate validates the list parameters
func (lo *ListOptions) Validate() (err error) {
	if !lo.allFlag && lo.pathFlag == "" && (lo.Context.Project == "" || lo.Application == "") {
		return odoutil.ThrowContextError()
	}

	return nil
}

// Run has the logic to perform the required actions as part of command
func (lo *ListOptions) Run() (err error) {

	if len(lo.pathFlag) != 0 {
		components, err := component.ListIfPathGiven(lo.Context.Client, filepath.SplitList(lo.pathFlag))
		if err != nil {
			return err
		}
		if log.IsJSON() {
			out, err := json.Marshal(components)
			if err != nil {
				return err
			}
			fmt.Println(string(out))
		} else {
			w := tabwriter.NewWriter(os.Stdout, 5, 2, 3, ' ', tabwriter.TabIndent)
			fmt.Fprintln(w, "APP", "\t", "NAME", "\t", "TYPE", "\t", "SOURCE", "\t", "STATE", "\t", "CONTEXT")
			for _, file := range components.Items {
				fmt.Fprintln(w, file.Spec.App, "\t", file.Name, "\t", file.Spec.Type, "\t", file.Spec.Source, "\t", file.Status.State, "\t", file.Status.Context)

			}
			w.Flush()
		}
		return nil
	}
	var components component.ComponentList

	if lo.allFlag {
		// retrieve list of application
		apps, err := application.List(lo.Client)
		if err != nil {
			return err
		}

		var componentList []component.Component

		if len(apps) == 0 && lo.LocalConfigInfo.ConfigFileExists() {
			comps, err := component.List(lo.Client, lo.LocalConfigInfo.GetApplication(), &lo.LocalConfigInfo)
			if err != nil {
				return err
			}
			componentList = append(componentList, comps.Items...)
		}

		// interating over list of application and get list of all components
		for _, app := range apps {
			comps, err := component.List(lo.Client, app, &lo.LocalConfigInfo)
			if err != nil {
				return err
			}
			componentList = append(componentList, comps.Items...)
		}
		// Get machine readable component list format
		components = component.GetMachineReadableFormatForList(componentList)
	} else {

		components, err = component.List(lo.Client, lo.Application, &lo.LocalConfigInfo)
		if err != nil {
			return errors.Wrapf(err, "failed to fetch components list")
		}
	}
	glog.V(4).Infof("the components are %+v", components)

	if log.IsJSON() {

		out, err := json.Marshal(components)
		if err != nil {
			return err
		}
		fmt.Println(string(out))

	} else {
		if len(components.Items) == 0 {
			log.Errorf("There are no components deployed.")
			return
		}
		w := tabwriter.NewWriter(os.Stdout, 5, 2, 3, ' ', tabwriter.TabIndent)
		fmt.Fprintln(w, "APP", "\t", "NAME", "\t", "TYPE", "\t", "SOURCE", "\t", "STATE")
		for _, comp := range components.Items {
			fmt.Fprintln(w, comp.Spec.App, "\t", comp.Name, "\t", comp.Spec.Type, "\t", comp.Spec.Source, "\t", comp.Status.State)
		}
		w.Flush()
	}
	return
}

// NewCmdList implements the list odo command
func NewCmdList(name, fullName string) *cobra.Command {
	o := NewListOptions()

	var componentListCmd = &cobra.Command{
		Use:     name,
		Short:   "List all components in the current application",
		Long:    "List all components in the current application.",
		Example: fmt.Sprintf(listExample, fullName),
		Args:    cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(o, cmd, args)
		},
	}
	// Add a defined annotation in order to appear in the help menu
	componentListCmd.Annotations = map[string]string{"command": "component"}
	genericclioptions.AddContextFlag(componentListCmd, &o.componentContext)
	componentListCmd.Flags().StringVar(&o.pathFlag, "path", "", "path of the directory to scan for odo component directories")
	componentListCmd.Flags().BoolVar(&o.allFlag, "all", false, "lists all components")
	//Adding `--project` flag
	projectCmd.AddProjectFlag(componentListCmd)
	//Adding `--application` flag
	appCmd.AddApplicationFlag(componentListCmd)

	completion.RegisterCommandFlagHandler(componentListCmd, "path", completion.FileCompletionHandler)

	return componentListCmd
}
