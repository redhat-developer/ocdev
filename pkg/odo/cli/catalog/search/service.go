package search

import (
	"fmt"

	"github.com/openshift/odo/pkg/odo/cli/catalog/util"
	"github.com/openshift/odo/pkg/odo/genericclioptions"
	olm "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/spf13/cobra"
)

const serviceRecommendedCommandName = "service"

var serviceExample = `  # Search for a service
  %[1]s mysql`

// SearchServiceOptions encapsulates the options for the odo catalog describe service command
type SearchServiceOptions struct {
	searchTerm string
	// generic context options common to all commands
	csvs *olm.ClusterServiceVersionList
	*genericclioptions.Context
}

// NewSearchServiceOptions creates a new SearchServiceOptions instance
func NewSearchServiceOptions() *SearchServiceOptions {
	return &SearchServiceOptions{}
}

// Complete completes SearchServiceOptions after they've been created
func (o *SearchServiceOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	o.Context, err = genericclioptions.NewContext(cmd)
	if err != nil {
		return err
	}
	o.searchTerm = args[0]
	o.csvs, err = o.KClient.SearchClusterServiceVersionList(o.searchTerm)
	if err != nil {
		// Error only occurs when OperatorHub is not installed/enabled on the
		// Kubernetes or OpenShift 4.x cluster.
		// Neither OperatorHub nor Service Catalog is enabled on the cluster
		return fmt.Errorf("unable to list services as Operator Hub is not enabled in your cluster: %v", err)
	}
	return err
}

// Validate validates the SearchServiceOptions based on completed values
func (o *SearchServiceOptions) Validate() (err error) {
	if len(o.csvs.Items) == 0 {
		return fmt.Errorf("no service matched the query: %s", o.searchTerm)
	}
	return
}

// Run contains the logic for the command associated with SearchServiceOptions
func (o *SearchServiceOptions) Run(cmd *cobra.Command) (err error) {
	if len(o.csvs.Items) > 0 {
		util.DisplayClusterServiceVersions(o.csvs)
	}
	return
}

// NewCmdCatalogSearchService implements the odo catalog search service command
func NewCmdCatalogSearchService(name, fullName string) *cobra.Command {
	o := NewSearchServiceOptions()
	return &cobra.Command{
		Use:   name,
		Short: "Search service type in catalog",
		Long: `Search service type in catalog.

This searches for a partial match for the given search term in all the available
services from service catalog.
`,
		Example: fmt.Sprintf(serviceExample, fullName),
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(o, cmd, args)
		},
	}

}
