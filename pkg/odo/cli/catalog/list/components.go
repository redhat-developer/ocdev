package list

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/openshift/odo/pkg/catalog"
	"github.com/openshift/odo/pkg/log"
	"github.com/openshift/odo/pkg/machineoutput"
	"github.com/openshift/odo/pkg/odo/cli/catalog/util"
	"github.com/openshift/odo/pkg/odo/genericclioptions"
	"github.com/openshift/odo/pkg/odo/util/experimental"
	"github.com/spf13/cobra"
)

const componentsRecommendedCommandName = "components"

var componentsExample = `  # Get the supported components
  %[1]s`

// ListComponentsOptions encapsulates the options for the odo catalog list components command
type ListComponentsOptions struct {
	// display both supported and unsupported devfile components
	listAllDevfileComponnets bool
	// list of known images
	catalogList catalog.ComponentTypeList
	// list of known devfiles
	catalogDevfileList catalog.DevfileComponentTypeList
	// generic context options common to all commands
	*genericclioptions.Context
}

// NewListComponentsOptions creates a new ListComponentsOptions instance
func NewListComponentsOptions() *ListComponentsOptions {
	return &ListComponentsOptions{}
}

// Complete completes ListComponentsOptions after they've been created
func (o *ListComponentsOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	o.Context = genericclioptions.NewContext(cmd)
	o.catalogList, err = catalog.ListComponents(o.Client)
	if err != nil {
		return err
	}

	if experimental.IsExperimentalModeEnabled() {
		o.catalogDevfileList, err = catalog.ListDevfileComponents()
		if err != nil {
			return err
		}
	}

	o.catalogList.Items = util.FilterHiddenComponents(o.catalogList.Items)

	return
}

// Validate validates the ListComponentsOptions based on completed values
func (o *ListComponentsOptions) Validate() (err error) {
	if len(o.catalogList.Items) == 0 && len(o.catalogDevfileList.Items) == 0 {
		return fmt.Errorf("no deployable components found")
	}

	return err
}

// Run contains the logic for the command associated with ListComponentsOptions
func (o *ListComponentsOptions) Run() (err error) {
	if log.IsJSON() {
		for i, image := range o.catalogList.Items {
			// here we don't care about the unsupported tags (second return value)
			supported, _ := catalog.SliceSupportedTags(image)
			o.catalogList.Items[i].Spec.SupportedTags = supported
		}
		machineoutput.OutputSuccess(o.catalogList)
	} else {
		w := tabwriter.NewWriter(os.Stdout, 5, 2, 3, ' ', tabwriter.TabIndent)
		var supCatalogList, unsupCatalogList []catalog.ComponentType
		var supDevfileCatalogList, unsupDevfileCatalogList []catalog.DevfileComponentType

		for _, image := range o.catalogList.Items {
			supported, unsupported := catalog.SliceSupportedTags(image)

			if len(supported) != 0 {
				image.Spec.NonHiddenTags = supported
				supCatalogList = append(supCatalogList, image)
			}
			if len(unsupported) != 0 {
				image.Spec.NonHiddenTags = unsupported
				unsupCatalogList = append(unsupCatalogList, image)
			}
		}

		for _, devfileComponent := range o.catalogDevfileList.Items {
			if devfileComponent.Support {
				supDevfileCatalogList = append(supDevfileCatalogList, devfileComponent)
			} else {
				unsupDevfileCatalogList = append(unsupDevfileCatalogList, devfileComponent)
			}
		}

		if len(supCatalogList) != 0 {
			fmt.Fprintln(w, "Odo Supported OpenShift Components:")
			o.printCatalogList(w, supCatalogList)
			fmt.Fprintln(w)
		}

		if len(unsupCatalogList) != 0 {
			fmt.Fprintln(w, "Odo Unsupported OpenShift Components:")
			o.printCatalogList(w, unsupCatalogList)
		}

		if len(supDevfileCatalogList) != 0 {
			fmt.Fprintln(w)
			fmt.Fprintln(w, "Odo Supported Devfile Components:")
			o.printDevfileCatalogList(w, supDevfileCatalogList)
			fmt.Fprintln(w)
		}

		if o.listAllDevfileComponnets && len(unsupDevfileCatalogList) != 0 {
			fmt.Fprintln(w, "Odo Unsupported Devfile Components:")
			o.printDevfileCatalogList(w, unsupDevfileCatalogList)
		}

		w.Flush()
	}
	return
}

// NewCmdCatalogListComponents implements the odo catalog list components command
func NewCmdCatalogListComponents(name, fullName string) *cobra.Command {
	o := NewListComponentsOptions()

	var componentListCmd = &cobra.Command{
		Use:         name,
		Short:       "List all components",
		Long:        "List all available component types from OpenShift's Image Builder",
		Example:     fmt.Sprintf(componentsExample, fullName),
		Annotations: map[string]string{"machineoutput": "json"},
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(o, cmd, args)
		},
	}

	componentListCmd.Flags().BoolVarP(&o.listAllDevfileComponnets, "all", "a", false, "List both supported and unsupported devfile components.")

	return componentListCmd
}

func (o *ListComponentsOptions) printCatalogList(w io.Writer, catalogList []catalog.ComponentType) {
	fmt.Fprintln(w, "NAME", "\t", "PROJECT", "\t", "TAGS")

	for _, component := range catalogList {
		componentName := component.Name
		if component.Namespace == o.Project {
			/*
				If current namespace is same as the current component namespace,
				Loop through every other component,
				If there exists a component with same name but in different namespaces,
				mark the one from current namespace with (*)
			*/
			for _, comp := range catalogList {
				if comp.ObjectMeta.Name == component.ObjectMeta.Name && component.Namespace != comp.Namespace {
					componentName = fmt.Sprintf("%s (*)", component.ObjectMeta.Name)
				}
			}
		}
		fmt.Fprintln(w, componentName, "\t", component.ObjectMeta.Namespace, "\t", strings.Join(component.Spec.NonHiddenTags, ","))
	}
}

func (o *ListComponentsOptions) printDevfileCatalogList(w io.Writer, catalogDevfileList []catalog.DevfileComponentType) {
	fmt.Fprintln(w, "NAME", "\t", "DESCRIPTION")

	for _, devfileComponent := range catalogDevfileList {
		fmt.Fprintln(w, devfileComponent.Name, "\t", devfileComponent.Description)
	}
}
