package describe

import (
	"fmt"
	"github.com/olekukonko/tablewriter"
	"github.com/redhat-developer/odo/pkg/odo/genericclioptions"
	odoutil "github.com/redhat-developer/odo/pkg/odo/util"
	svc "github.com/redhat-developer/odo/pkg/service"
	"github.com/spf13/cobra"
	ktemplates "k8s.io/kubernetes/pkg/kubectl/cmd/templates"
	"os"
	"strings"
)

const serviceRecommendedCommandName = "service"

var (
	serviceExample = ktemplates.Examples(`  # Describe a service
    %[1]s mysql-persistent`)

	serviceLongDesc = ktemplates.LongDesc(`Describe a service type.

This describes the service and the associated plans.
`)
)

// DescribeServiceOptions encapsulates the options for the odo catalog describe service command
type DescribeServiceOptions struct {
	// name of the service to describe, from command arguments
	serviceName string
	// resolved service
	service svc.ServiceClass
	plans   []svc.ServicePlan
	// generic context options common to all commands
	*genericclioptions.Context
}

// NewDescribeServiceOptions creates a new DescribeServiceOptions instance
func NewDescribeServiceOptions() *DescribeServiceOptions {
	return &DescribeServiceOptions{}
}

// Complete completes DescribeServiceOptions after they've been created
func (o *DescribeServiceOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	o.Context = genericclioptions.NewContext(cmd)
	o.serviceName = args[0]

	return
}

// Validate validates the DescribeServiceOptions based on completed values
func (o *DescribeServiceOptions) Validate() (err error) {
	o.service, o.plans, err = svc.GetServiceClassAndPlans(o.Client, o.serviceName)
	return err
}

// Run contains the logic for the command associated with DescribeServiceOptions
func (o *DescribeServiceOptions) Run() (err error) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetBorder(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	serviceData := [][]string{
		{"Name", o.service.Name},
		{"Bindable", fmt.Sprint(o.service.Bindable)},
		{"Operated by the broker", o.service.ServiceBrokerName},
		{"Short Description", o.service.ShortDescription},
		{"Long Description", o.service.LongDescription},
		{"Versions Available", strings.Join(o.service.VersionsAvailable, ",")},
		{"Tags", strings.Join(o.service.Tags, ",")},
	}

	table.AppendBulk(serviceData)

	table.Append([]string{""})

	if len(o.plans) > 0 {
		table.Append([]string{"PLANS"})

		for _, plan := range o.plans {

			// create the display values for required  and optional parameters
			requiredWithMandatoryUserInputParameterNames := []string{}
			requiredWithOptionalUserInputParameterNames := []string{}
			optionalParameterDisplay := []string{}
			for _, parameter := range plan.Parameters {
				if parameter.Required {
					// until we have a better solution for displaying the plan data (like a separate table perhaps)
					// this is simplest thing to do
					if parameter.HasDefaultValue {
						requiredWithOptionalUserInputParameterNames = append(
							requiredWithOptionalUserInputParameterNames,
							fmt.Sprintf("%s (default: '%s')", parameter.Name, parameter.Default))
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
		return fmt.Errorf("no plans found for service %s", o.serviceName)
	}
	return
}

// NewCmdCatalogDescribeService implements the odo catalog describe service command
func NewCmdCatalogDescribeService(name, fullName string) *cobra.Command {
	o := NewDescribeServiceOptions()
	command := &cobra.Command{
		Use:     name,
		Short:   "Describe a service",
		Long:    serviceLongDesc,
		Example: fmt.Sprintf(serviceExample, fullName),
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			odoutil.LogErrorAndExit(o.Complete(name, cmd, args), "")
			odoutil.LogErrorAndExit(o.Validate(), "")
			odoutil.LogErrorAndExit(o.Run(), "")
		},
	}

	return command
}
