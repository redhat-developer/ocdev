package cmd

import (
	"fmt"
	"github.com/olekukonko/tablewriter"
	"github.com/redhat-developer/odo/pkg/odo/genericclioptions"
	"github.com/redhat-developer/odo/pkg/odo/util"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/redhat-developer/odo/pkg/catalog"
	svc "github.com/redhat-developer/odo/pkg/service"
	"github.com/spf13/cobra"
)

var catalogCmd = &cobra.Command{
	Use:   "catalog [options]",
	Short: "Catalog related operations",
	Long:  "Catalog related operations",
	Example: fmt.Sprintf("%s\n%s\n%s",
		catalogListCmd.Example,
		catalogSearchCmd.Example,
		catalogDescribeCmd.Example),
}

var catalogListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available component & service types.",
	Long:  "List all available component and service types from OpenShift",
	Example: `  # Get the supported components
  odo catalog list components

  # Get the supported services from service catalog
  odo catalog list services
`,
}

var catalogListComponentCmd = &cobra.Command{
	Use:   "components",
	Short: "List all components available.",
	Long:  "List all available component types from OpenShift's Image Builder.",
	Example: `  # Get the supported components
  odo catalog list components

  # Search for a supported component
  odo catalog search component nodejs
`,
	Run: func(cmd *cobra.Command, args []string) {
		client := genericclioptions.Client(cmd)
		catalogList, err := catalog.List(client)
		util.CheckError(err, "unable to list components")
		switch len(catalogList) {
		case 0:
			fmt.Printf("No deployable components found\n")
		default:
			currentProject := client.GetCurrentProjectName()
			w := tabwriter.NewWriter(os.Stdout, 5, 2, 3, ' ', tabwriter.TabIndent)
			fmt.Fprintln(w, "NAME", "\t", "PROJECT", "\t", "TAGS")
			for _, component := range catalogList {
				componentName := component.Name
				if component.Namespace == currentProject {
					/*
						If current namespace is same as the current component namespace,
						Loop through every other component,
						If there exists a component with same name but in different namespaces,
						mark the one from current namespace with (*)
					*/
					for _, comp := range catalogList {
						if comp.Name == component.Name && component.Namespace != comp.Namespace {
							componentName = fmt.Sprintf("%s (*)", component.Name)
						}
					}
				}
				fmt.Fprintln(w, componentName, "\t", component.Namespace, "\t", strings.Join(component.Tags, ","))
			}
			w.Flush()
		}
	},
}

var catalogListServiceCmd = &cobra.Command{
	Use:   "services",
	Short: "Lists all available services",
	Long:  "Lists all available services",
	Example: `  # List all services
  odo catalog list services

 # Search for a supported service
  odo catalog search service mysql
	`,
	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		client := genericclioptions.Client(cmd)
		catalogList, err := svc.ListCatalog(client)
		util.CheckError(err, "unable to list services because Service Catalog is not enabled in your cluster")
		switch len(catalogList) {
		case 0:
			fmt.Printf("No deployable services found\n")
		default:
			w := tabwriter.NewWriter(os.Stdout, 5, 2, 3, ' ', tabwriter.TabIndent)
			fmt.Fprintln(w, "NAME", "\t", "PLANS")
			for _, service := range catalogList {
				fmt.Fprintln(w, service.Name, "\t", strings.Join(service.PlanList, ","))
			}
			w.Flush()

		}
	},
}

var catalogSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search available component & service types.",
	Long: `Search available component & service types..

This searches for a partial match for the given search term in all the available
components & services.
`,
	Example: `  # Search for a component
  odo catalog search component python

  # Search for a service
  odo catalog search service mysql
	`,
}

var catalogSearchComponentCmd = &cobra.Command{
	Use:   "component",
	Short: "Search component type in catalog",
	Long: `Search component type in catalog.

This searches for a partial match for the given search term in all the available
components.
`,
	Args: cobra.ExactArgs(1),
	Example: `  # Search for a component
  odo catalog search component python
	`,
	Run: func(cmd *cobra.Command, args []string) {
		client := genericclioptions.Client(cmd)
		searchTerm := args[0]
		components, err := catalog.Search(client, searchTerm)
		util.CheckError(err, "unable to search for components")

		switch len(components) {
		case 0:
			fmt.Printf("No component matched the query: %v\n", searchTerm)
		default:
			fmt.Println("The following components were found:")
			for _, component := range components {
				fmt.Printf("- %v\n", component)
			}
		}
	},
}

var catalogSearchServiceCmd = &cobra.Command{
	Use:   "service",
	Short: "Search service type in catalog",
	Long: `Search service type in catalog.

This searches for a partial match for the given search term in all the available
services from service catalog.
`,
	Example: `  # Search for a service
  odo catalog search service mysql
	`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := genericclioptions.Client(cmd)
		searchTerm := args[0]
		components, err := svc.Search(client, searchTerm)
		util.CheckError(err, "unable to search for services")

		switch len(components) {
		case 0:
			fmt.Printf("No service matched the query: %v\n", searchTerm)
		default:
			w := tabwriter.NewWriter(os.Stdout, 5, 2, 3, ' ', tabwriter.TabIndent)
			fmt.Fprintln(w, "NAME", "\t", "PLANS")
			for _, component := range components {
				fmt.Fprintln(w, component.Name, "\t", strings.Join(component.PlanList, ","))
			}
			w.Flush()

		}
	},
}

var catalogDescribeCmd = &cobra.Command{
	Use:   "describe",
	Short: "Describe catalog item",
	Long:  "Describe the given catalog item from OpenShift",
	Args:  cobra.ExactArgs(1),
	Example: `  # Describe the given service
  odo catalog describe service mysql-persistent
	`,
}

var catalogDescribeServiceCmd = &cobra.Command{
	Use:   "service",
	Short: "Describe a service",
	Long: `Describe a service type.

This describes the service and the associated plans.
`,
	Example: `  # Describe a service
  odo catalog describe service mysql-persistent
	`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := genericclioptions.Client(cmd)
		serviceName := args[0]
		service, plans, err := svc.GetServiceClassAndPlans(client, serviceName)
		util.CheckError(err, "")

		table := tablewriter.NewWriter(os.Stdout)
		table.SetBorder(false)
		table.SetAlignment(tablewriter.ALIGN_LEFT)

		serviceData := [][]string{
			{"Name", service.Name},
			{"Bindable", fmt.Sprint(service.Bindable)},
			{"Operated by the broker", service.ServiceBrokerName},
			{"Short Description", service.ShortDescription},
			{"Long Description", service.LongDescription},
			{"Versions Available", strings.Join(service.VersionsAvailable, ",")},
			{"Tags", strings.Join(service.Tags, ",")},
		}

		table.AppendBulk(serviceData)

		table.Append([]string{""})

		if len(plans) > 0 {
			table.Append([]string{"PLANS"})

			for _, plan := range plans {

				// create the display values for required  and optional parameters
				requiredWithMandatoryUserInputParameterNames := []string{}
				requiredWithOptionalUserInputParameterNames := []string{}
				optionalParameterDisplay := []string{}
				for _, parameter := range plan.Parameters {
					if parameter.Required {
						// until we have a better solution for displaying the plan data (like a separate table perhaps)
						// this is simplest thing to do
						if parameter.HasDefaultValue {
							defaultValueStr := fmt.Sprintf("%v", parameter.DefaultValue)
							requiredWithOptionalUserInputParameterNames = append(
								requiredWithOptionalUserInputParameterNames,
								fmt.Sprintf("%s (default: '%s')", parameter.Name, defaultValueStr))
						} else {
							requiredWithMandatoryUserInputParameterNames = append(requiredWithMandatoryUserInputParameterNames, parameter.Name)
						}

					} else {
						optionalParameterDisplay = append(optionalParameterDisplay, parameter.Name)
					}
				}

				table.Append([]string{"***********************", "*****************************************************"})
				planLineSeparator := []string{"-----------------", "-----------------"}

				planData := [][]string{
					{"Name", plan.Name},
					planLineSeparator,
					{"Display Name", plan.DisplayName},
					planLineSeparator,
					{"Short Description", plan.Description},
					planLineSeparator,
					{"Required Params without a default value", strings.Join(requiredWithMandatoryUserInputParameterNames, ", ")},
					planLineSeparator,
					{"Required Params with a default value", strings.Join(requiredWithOptionalUserInputParameterNames, ", ")},
					planLineSeparator,
					{"Optional Params", strings.Join(optionalParameterDisplay, ", ")},
					{"", ""},
				}
				table.AppendBulk(planData)
			}
			table.Render()
		} else {
			fmt.Printf("No plans found for service %s\n", serviceName)
		}
	},
}

func init() {
	catalogCmd.AddCommand(catalogSearchCmd)
	catalogCmd.AddCommand(catalogListCmd)
	catalogCmd.AddCommand(catalogDescribeCmd)
	catalogListCmd.AddCommand(catalogListComponentCmd)
	catalogListCmd.AddCommand(catalogListServiceCmd)
	catalogSearchCmd.AddCommand(catalogSearchComponentCmd)
	catalogSearchCmd.AddCommand(catalogSearchServiceCmd)
	catalogDescribeCmd.AddCommand(catalogDescribeServiceCmd)
	// Add a defined annotation in order to appear in the help menu
	catalogCmd.Annotations = map[string]string{"command": "other"}
	catalogCmd.SetUsageTemplate(cmdUsageTemplate)

	rootCmd.AddCommand(catalogCmd)
}
