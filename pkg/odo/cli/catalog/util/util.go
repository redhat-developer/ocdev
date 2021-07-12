package util

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/openshift/odo/pkg/catalog"
	"github.com/openshift/odo/pkg/log"
	olm "github.com/operator-framework/api/pkg/operators/v1alpha1"
)

// DisplayComponents displays the specified  components
func DisplayComponents(components []string) {
	w := tabwriter.NewWriter(os.Stdout, 5, 2, 3, ' ', tabwriter.TabIndent)
	fmt.Fprintln(w, "NAME")
	for _, component := range components {
		fmt.Fprintln(w, component)
	}
	w.Flush()
}

// FilterHiddenServices filters out services that should be hidden from the specified list
func FilterHiddenServices(input catalog.ServiceTypeList) catalog.ServiceTypeList {
	inputLength := len(input.Items)
	filteredServices := make([]catalog.ServiceType, 0, inputLength)

	for _, service := range input.Items {
		if !service.Spec.Hidden {
			filteredServices = append(filteredServices, service)
		}
	}

	return catalog.ServiceTypeList{
		TypeMeta: input.TypeMeta,
		ListMeta: input.ListMeta,
		Items:    filteredServices,
	}
}

// FilterHiddenComponents filters out components that should be hidden from the specified list
func FilterHiddenComponents(input []catalog.ComponentType) []catalog.ComponentType {
	inputLength := len(input)
	filteredComponents := make([]catalog.ComponentType, 0, inputLength)
	for _, component := range input {
		// we keep the image if it has tags that are no hidden
		if len(component.Spec.NonHiddenTags) > 0 {
			filteredComponents = append(filteredComponents, component)
		}
	}
	return filteredComponents
}

// DisplayClusterServiceVersions displays installed Operators in a human friendly manner
func DisplayClusterServiceVersions(csvs *olm.ClusterServiceVersionList) {
	w := tabwriter.NewWriter(os.Stdout, 5, 2, 3, ' ', tabwriter.TabIndent)
	log.Info("Services available through Operators")
	fmt.Fprintln(w, "NAME", "\t", "CRDs")
	for _, csv := range csvs.Items {
		fmt.Fprintln(w, csv.ObjectMeta.Name, "\t", CsvOperators(csv.Spec.CustomResourceDefinitions))
	}
	w.Flush()
}

// CsvOperators returns a string contains all the Kind from the input crds
func CsvOperators(crds olm.CustomResourceDefinitions) string {
	var crdsSlice []string
	for _, crd := range crds.Owned {
		crdsSlice = append(crdsSlice, crd.Kind)
	}
	return strings.Join(crdsSlice, ", ")
}
