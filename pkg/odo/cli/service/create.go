package service

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/openshift/odo/pkg/service"
	"strings"
	"text/template"

	"github.com/openshift/odo/pkg/log"
	"github.com/openshift/odo/pkg/odo/cli/component"
	"github.com/openshift/odo/pkg/odo/cli/service/ui"
	"github.com/openshift/odo/pkg/odo/genericclioptions"
	"github.com/openshift/odo/pkg/odo/util/completion"
	"github.com/spf13/cobra"
	ktemplates "k8s.io/kubectl/pkg/util/templates"
)

const (
	createRecommendedCommandName = "create"
	equivalentTemplate           = "{{.CmdFullName}} {{.ServiceType}}" +
		"{{if .ServiceName}} {{.ServiceName}}{{end}}" +
		" --app {{.Application}}" +
		" --project {{.Project}}" +
		"{{if .Plan}} --plan {{.Plan}}{{end}}" +
		"{{range $key, $value := .ParametersMap}} -p {{$key}}={{$value}}{{end}}"
)

var (
	createExample = ktemplates.Examples(`
    # Create new postgresql service from service catalog using dev plan and name my-postgresql-db.
    %[1]s dh-postgresql-apb my-postgresql-db --plan dev -p postgresql_user=luke -p postgresql_password=secret`)

	createOperatorExample = ktemplates.Examples(`
	# Create new EtcdCluster service from etcdoperator.v0.9.4 operator.
	%[1]s etcdoperator.v0.9.4/EtcdCluster`)

	createShortDesc = `Create a new service from Operator Hub or Service Catalog and deploy it on OpenShift.`

	createLongDesc = ktemplates.LongDesc(`
Create a new service from Operator Hub or Service Catalog and deploy it on OpenShift.

Service creation can be performed from a valid component directory (one containing a devfile.yaml) only.

To create the service from outside a component directory, specify path to a valid component directory using "--context" flag.

When creating a service using Operator Hub, provide a service name along with Operator name.

When creating a service using Service Catalog, a --plan must be passed along with the service type. Parameters to configure the service are passed as key=value pairs.

For a full list of service types, use: 'odo catalog list services'`)
)

// CreateOptions encapsulates the options for the odo service create command
type CreateOptions struct {
	// parameters hold the user-provided values for service class parameters via flags (populated by cobra)
	parameters []string
	// Plan is the selected service plan
	Plan string
	// ServiceType corresponds to the service class name
	ServiceType string
	// ServiceName is how the service will be named and known by odo
	ServiceName string
	// ParametersMap is populated from the flag-provided values (parameters) and/or the interactive mode and is the expected format by the business logic
	ParametersMap map[string]string
	// interactive specifies whether the command operates in interactive mode or not
	interactive bool
	// outputCLI specifies whether to output the non-interactive version of the command or not
	outputCLI bool
	// CmdFullName records the command's full name
	CmdFullName string
	// whether or not to wait for the service to be ready
	wait bool
	// generic context options common to all commands
	*genericclioptions.Context
	// Context to use when creating service. This will use app and project values from the context
	componentContext string
	// If set to true, DryRun prints the yaml that will create the service
	DryRun bool
	// Location of the file in which yaml specification of CR is stored.
	fromFile string
	// Backend is the service provider backend (Operator Hub or Service Catalog) providing the service requested by the user
	Backend ServiceProviderBackend
}

// NewCreateOptions creates a new CreateOptions instance
func NewCreateOptions() *CreateOptions {
	return &CreateOptions{}
}

// Complete completes CreateOptions after they've been created
func (o *CreateOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	o.Context, err = genericclioptions.New(genericclioptions.CreateParameters{
		Cmd:              cmd,
		DevfilePath:      component.DevfilePath,
		ComponentContext: o.componentContext,
	})
	if err != nil {
		return err
	}
	// we convert the param list provided in the format of key=value list
	// to a map
	o.ParametersMap = make(map[string]string)
	for _, kv := range o.parameters {
		kvSlice := strings.Split(kv, "=")
		// key value not provided in format of key=value
		if len(kvSlice) != 2 {
			return errors.New("parameters not provided in key=value format")
		}
		o.ParametersMap[kvSlice[0]] = kvSlice[1]
	}
	_, _, err = service.IsOperatorServiceNameValid(args[0])
	if err != nil {
		return fmt.Errorf("invalid operator name and service catalog not supported %w", err)
	}
	err = validDevfileDirectory(o.componentContext)
	if err != nil {
		return err
	}
	// use operator backend only
	o.Backend = NewOperatorBackend()
	// since interactive mode is not supported for Operators yet, set it to false
	o.interactive = false
	return o.Backend.CompleteServiceCreate(o, cmd, args)
}

// Validate validates the CreateOptions based on completed values
func (o *CreateOptions) Validate() (err error) {
	// if we are in interactive mode, all values are already valid
	if o.interactive {
		return nil
	}
	return o.Backend.ValidateServiceCreate(o)
}

// Run contains the logic for the odo service create command
func (o *CreateOptions) Run(cmd *cobra.Command) (err error) {
	err = o.Backend.RunServiceCreate(o)
	if err != nil {
		return fmt.Errorf("service %q already exists in configuration", o.ServiceName)
	}

	// Information on what to do next; don't do this if "--dry-run" was requested as it gets appended to the file
	if !o.DryRun {
		log.Info("Successfully added service to the configuration; do 'odo push' to create service on the cluster")
	}

	equivalent := o.outputNonInteractiveEquivalent()
	if len(equivalent) > 0 {
		log.Info("Equivalent command:\n" + ui.StyledOutput(equivalent, "cyan"))
	}
	return
}

// outputNonInteractiveEquivalent outputs the populated options as the equivalent command that would be used in non-interactive mode
func (o *CreateOptions) outputNonInteractiveEquivalent() string {
	if o.outputCLI {
		var tpl bytes.Buffer
		t := template.Must(template.New("service-create-cli").Parse(equivalentTemplate))
		e := t.Execute(&tpl, o)
		if e != nil {
			panic(e) // shouldn't happen
		}
		return strings.TrimSpace(tpl.String())
	}
	return ""
}

// NewCmdServiceCreate implements the odo service create command.
func NewCmdServiceCreate(name, fullName string) *cobra.Command {
	o := NewCreateOptions()
	o.CmdFullName = fullName
	serviceCreateCmd := &cobra.Command{
		Use:     name + " <service_type> --plan <plan_name> [service_name]",
		Short:   createShortDesc,
		Long:    createLongDesc,
		Example: fmt.Sprintf(createExample, fullName),
		Args:    cobra.RangeArgs(0, 2),
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(o, cmd, args)
		},
	}

	serviceCreateCmd.Use += fmt.Sprintf(" [flags]\n  %s <operator_type>/<crd_name> [service_name] [flags]", o.CmdFullName)
	serviceCreateCmd.Example += "\n\n" + fmt.Sprintf(createOperatorExample, fullName)
	serviceCreateCmd.Flags().BoolVar(&o.DryRun, "dry-run", false, "Print the yaml specificiation that will be used to create the operator backed service")
	// remove this feature after enabling service create interactive mode for operator backed services
	serviceCreateCmd.Flags().StringVar(&o.fromFile, "from-file", "", "Path to the file containing yaml specification to use to start operator backed service")

	serviceCreateCmd.Flags().StringVar(&o.Plan, "plan", "", "The name of the plan of the service to be created")
	serviceCreateCmd.Flags().StringArrayVarP(&o.parameters, "parameters", "p", []string{}, "Parameters of the plan where a parameter is expressed as <key>=<value")
	serviceCreateCmd.Flags().BoolVarP(&o.wait, "wait", "w", false, "Wait until the service is ready")
	genericclioptions.AddContextFlag(serviceCreateCmd, &o.componentContext)
	completion.RegisterCommandHandler(serviceCreateCmd, completion.ServiceClassCompletionHandler)
	completion.RegisterCommandFlagHandler(serviceCreateCmd, "plan", completion.ServicePlanCompletionHandler)
	completion.RegisterCommandFlagHandler(serviceCreateCmd, "parameters", completion.ServiceParameterCompletionHandler)
	return serviceCreateCmd
}
