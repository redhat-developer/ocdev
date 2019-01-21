package util

import (
	"fmt"
	"github.com/redhat-developer/odo/pkg/occlient"
	"os"
	"strings"
	"text/tabwriter"
)

// DisplayServices displays the specified services
func DisplayServices(services []occlient.Service) {
	w := tabwriter.NewWriter(os.Stdout, 5, 2, 3, ' ', tabwriter.TabIndent)
	fmt.Fprintln(w, "NAME", "\t", "PLANS")
	for _, service := range services {
		fmt.Fprintln(w, service.Name, "\t", strings.Join(service.PlanList, ","))
	}
	w.Flush()
}

// FilterHiddenServices filters out services that should be hidden from the specified list
func FilterHiddenServices(services []occlient.Service) []occlient.Service {
	var filteredServices []occlient.Service
	for _, service := range services {
		if !service.Hidden {
			filteredServices = append(filteredServices, service)
		}
	}
	return filteredServices
}
