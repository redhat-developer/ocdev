package service

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/openshift/odo/pkg/log"
	"github.com/openshift/odo/pkg/machineoutput"
	"github.com/openshift/odo/pkg/odo/genericclioptions"
	odoutil "github.com/openshift/odo/pkg/odo/util"
	svc "github.com/openshift/odo/pkg/service"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ktemplates "k8s.io/kubectl/pkg/util/templates"
)

const listRecommendedCommandName = "list"

var (
	listExample = ktemplates.Examples(`
    # List all services in the application
    %[1]s`)
	listLongDesc = ktemplates.LongDesc(`
List all services in the current application
`)
)

// ServiceListOptions encapsulates the options for the odo service list command
type ServiceListOptions struct {
	*genericclioptions.Context
	// Context to use when listing service. This will use app and project values from the context
	componentContext string
	// choose between Operator Hub and Service Catalog. If true, Operator Hub
	csvSupport bool
}

// NewServiceListOptions creates a new ServiceListOptions instance
func NewServiceListOptions() *ServiceListOptions {
	return &ServiceListOptions{}
}

// Complete completes ServiceListOptions after they've been created
func (o *ServiceListOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	if o.csvSupport, err = svc.IsCSVSupported(); err != nil {
		return err
	} else if o.csvSupport {
		o.Context, err = genericclioptions.NewDevfileContext(cmd)
	} else {
		o.Context, err = genericclioptions.NewContext(cmd)
	}
	return
}

// Validate validates the ServiceListOptions based on completed values
func (o *ServiceListOptions) Validate() (err error) {
	if !o.csvSupport {
		// Throw error if project and application values are not available.
		// This will most likely be the case when user does odo service list from outside a component directory and
		// doesn't provide --app and/or --project flags
		if o.Context.Project == "" || o.Context.Application == "" {
			return odoutil.ThrowContextError()
		}
	}
	return
}

// Run contains the logic for the odo service list command
func (o *ServiceListOptions) Run(cmd *cobra.Command) (err error) {
	if o.csvSupport {
		// if cluster supports Operators, we list only operator backed services
		// and not service catalog ones
		var list []unstructured.Unstructured
		list, failedListingCR, err := svc.ListOperatorServices(o.KClient)
		if err != nil {
			return err
		}

		if len(list) == 0 {
			if len(failedListingCR) > 0 {
				fmt.Printf("Failed to fetch services for operator(s): %q\n\n", strings.Join(failedListingCR, ", "))
			}
			return fmt.Errorf("no operator backed services found in namespace: %s", o.KClient.Namespace)
		}

		if log.IsJSON() {
			machineoutput.OutputSuccess(list)
			return nil
		} else {
			w := tabwriter.NewWriter(os.Stdout, 5, 2, 3, ' ', tabwriter.TabIndent)

			fmt.Fprintln(w, "NAME", "\t", "AGE")

			for _, item := range list {
				duration := time.Since(item.GetCreationTimestamp().Time).Truncate(time.Second).String()
				fmt.Fprintln(w, strings.Join([]string{item.GetKind(), item.GetName()}, "/"), "\t", duration)
			}

			w.Flush()

		}

		if len(failedListingCR) > 0 {
			fmt.Printf("\nFailed to fetch services for operator(s): %q\n", strings.Join(failedListingCR, ", "))
		}

		return err
	}

	services, err := svc.ListWithDetailedStatus(o.Client, o.Application)
	if err != nil {
		return fmt.Errorf("Service catalog is not enabled within your cluster: %v", err)
	}

	if len(services.Items) == 0 {
		return fmt.Errorf("There are no services deployed for this application")
	}

	if log.IsJSON() {
		machineoutput.OutputSuccess(services)
	} else {
		w := tabwriter.NewWriter(os.Stdout, 5, 2, 3, ' ', tabwriter.TabIndent)
		fmt.Fprintln(w, "NAME", "\t", "TYPE", "\t", "PLAN", "\t", "STATUS")
		for _, comp := range services.Items {
			fmt.Fprintln(w, comp.ObjectMeta.Name, "\t", comp.Spec.Type, "\t", comp.Spec.Plan, "\t", comp.Status.Status)
		}
		w.Flush()
	}
	return
}

// NewCmdServiceList implements the odo service list command.
func NewCmdServiceList(name, fullName string) *cobra.Command {
	o := NewServiceListOptions()
	serviceListCmd := &cobra.Command{
		Use:         name,
		Short:       "List all services in the current application",
		Long:        listLongDesc,
		Example:     fmt.Sprintf(listExample, fullName),
		Args:        cobra.NoArgs,
		Annotations: map[string]string{"machineoutput": "json"},
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(o, cmd, args)
		},
	}
	genericclioptions.AddContextFlag(serviceListCmd, &o.componentContext)
	return serviceListCmd
}
