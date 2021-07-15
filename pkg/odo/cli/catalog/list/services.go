package list

import (
	"fmt"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/odo/pkg/catalog"
	"github.com/openshift/odo/pkg/log"
	"github.com/openshift/odo/pkg/machineoutput"
	"github.com/openshift/odo/pkg/odo/cli/catalog/util"
	"github.com/openshift/odo/pkg/odo/genericclioptions"
	olm "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/spf13/cobra"
)

const servicesRecommendedCommandName = "services"

var servicesExample = `  # Get the supported services from service catalog
  %[1]s`

// ServiceOptions encapsulates the options for the odo catalog list services command
type ServiceOptions struct {
	// list of clusterserviceversions (installed by Operators)
	csvs *olm.ClusterServiceVersionList
	// generic context options common to all commands
	*genericclioptions.Context
}

// NewServiceOptions creates a new ListServicesOptions instance
func NewServiceOptions() *ServiceOptions {
	return &ServiceOptions{}
}

// Complete completes ListServicesOptions after they've been created
func (o *ServiceOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	o.Context, err = genericclioptions.NewContext(cmd)
	if err != nil {
		return err
	}
	o.csvs, err = catalog.ListOperatorServices(o.KClient)
	if err != nil {
		if strings.Contains(err.Error(), "could not find specified operator") {
			err = nil
		} else {
			return err
		}
	}
	return
}

// Validate validates the ListServicesOptions based on completed values
func (o *ServiceOptions) Validate() (err error) {
	return
}

// Run contains the logic for the command associated with ListServicesOptions
func (o *ServiceOptions) Run(cmd *cobra.Command) (err error) {
	if log.IsJSON() {
		machineoutput.OutputSuccess(newCatalogListOutput(o.csvs))
	} else {
		if len(o.csvs.Items) == 0 {
			log.Info("no deployable operators found")
			return
		}

		if len(o.csvs.Items) > 0 {
			util.DisplayClusterServiceVersions(o.csvs)
		}
	}
	return
}

// NewCmdCatalogListServices implements the odo catalog list services command
func NewCmdCatalogListServices(name, fullName string) *cobra.Command {
	o := NewServiceOptions()
	return &cobra.Command{
		Use:         name,
		Short:       "Lists all available services",
		Long:        "Lists all available services",
		Example:     fmt.Sprintf(servicesExample, fullName),
		Args:        cobra.ExactArgs(0),
		Annotations: map[string]string{"machineoutput": "json"},
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(o, cmd, args)
		},
	}
}

type catalogListOutput struct {
	v1.TypeMeta   `json:",inline"`
	v1.ObjectMeta `json:"metadata,omitempty"`
	// list of clusterserviceversions (installed by Operators)
	Operators *olm.ClusterServiceVersionList `json:"operators,omitempty"`
}

func newCatalogListOutput(operators *olm.ClusterServiceVersionList) catalogListOutput {
	return catalogListOutput{
		TypeMeta: v1.TypeMeta{
			Kind:       "List",
			APIVersion: machineoutput.APIVersion,
		},
		Operators: operators,
	}
}
