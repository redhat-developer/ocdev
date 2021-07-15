package component

import (
	"fmt"
	"path/filepath"

	servicebinding "github.com/redhat-developer/service-binding-operator/api/v1alpha1"
	"github.com/spf13/cobra"

	"github.com/openshift/odo/pkg/component"
	appCmd "github.com/openshift/odo/pkg/odo/cli/application"
	projectCmd "github.com/openshift/odo/pkg/odo/cli/project"
	"github.com/openshift/odo/pkg/odo/genericclioptions"
	odoutil "github.com/openshift/odo/pkg/odo/util"
	"github.com/openshift/odo/pkg/odo/util/completion"
	svc "github.com/openshift/odo/pkg/service"

	ktemplates "k8s.io/kubectl/pkg/util/templates"
)

// LinkRecommendedCommandName is the recommended link command name
const LinkRecommendedCommandName = "link"

var (
	linkExample = ktemplates.Examples(`# Link the current component to the 'EtcdCluster' named 'myetcd'
%[1]s EtcdCluster/myetcd

# Link the current component to the 'my-postgresql' service
%[1]s my-postgresql

# Link component 'nodejs' to the 'my-postgresql' service
%[1]s my-postgresql --component nodejs

# Link current component to the 'backend' component (backend must have a single exposed port)
%[1]s backend

# Link component 'nodejs' to the 'backend' component
%[1]s backend --component nodejs

# Link current component to port 8080 of the 'backend' component (backend must have port 8080 exposed) 
%[1]s backend --port 8080

# Link the current component to the 'EtcdCluster' named 'myetcd'
# and make the secrets accessible as files in the '/bindings/etcd/' directory
%[1]s EtcdCluster/myetcd  --bind-as-files --name etcd`)

	linkLongDesc = `Link component to a service (backed by an Operator or Service Catalog) or component (works only with s2i components)

If the source component is not provided, the current active component is assumed.
In both use cases, link adds the appropriate secret to the environment of the source component. 
The source component can then consume the entries of the secret as environment variables.

Using the '--bind-as-files' flag, secrets will be accessible as files instead of environment variables.
The value of the '--name' flag indicates the name of the directory under '/bindings/' containing the secrets files.

For example:

We have created a frontend application called 'frontend' using:
odo create nodejs frontend

If you wish to connect an EtcdCluster service created using etcd Operator to this component:
odo link EtcdCluster/myetcdcluster

Here myetcdcluster is the name of the EtcdCluster service which can be found using "odo service list"

We've also created a backend application called 'backend' with port 8080 exposed:
odo create nodejs backend --port 8080

We can now link the two applications (works only with s2i components):
odo link backend --component frontend

Now the frontend has 2 ENV variables it can use:
COMPONENT_BACKEND_HOST=backend-app
COMPONENT_BACKEND_PORT=8080

If you wish to use a database, we can use the Service Catalog and link it to our backend:
odo service create dh-postgresql-apb --plan dev -p postgresql_user=luke -p postgresql_password=secret
odo link dh-postgresql-apb

Now backend has 2 ENV variables it can use:
DB_USER=luke
DB_PASSWORD=secret`
)

// LinkOptions encapsulates the options for the odo link command
type LinkOptions struct {
	waitForTarget    bool
	componentContext string

	*commonLinkOptions
}

// NewLinkOptions creates a new LinkOptions instance
func NewLinkOptions() *LinkOptions {
	options := LinkOptions{}
	options.commonLinkOptions = newCommonLinkOptions()
	options.commonLinkOptions.serviceBinding = &servicebinding.ServiceBinding{}
	return &options
}

// Complete completes LinkOptions after they've been created
func (o *LinkOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	o.commonLinkOptions.devfilePath = filepath.Join(o.componentContext, DevfilePath)
	o.commonLinkOptions.csvSupport, _ = svc.IsCSVSupported()

	err = o.complete(name, cmd, args, o.componentContext)
	if err != nil {
		return err
	}

	if o.csvSupport && o.Context.EnvSpecificInfo != nil {
		o.operation = o.KClient.LinkSecret
	} else {
		o.operation = o.Client.LinkSecret
	}

	return err
}

// Validate validates the LinkOptions based on completed values
func (o *LinkOptions) Validate() (err error) {
	err = o.validate(o.waitForTarget)
	if err != nil {
		return err
	}

	if o.Context.EnvSpecificInfo != nil {
		return
	}

	alreadyLinkedSecretNames, err := component.GetComponentLinkedSecretNames(o.Client, o.Component(), o.Application)
	if err != nil {
		return err
	}
	for _, alreadyLinkedSecretName := range alreadyLinkedSecretNames {
		if alreadyLinkedSecretName == o.secretName {
			targetType := "component"
			if o.isTargetAService {
				targetType = "service"
			}
			return fmt.Errorf("Component %s has previously been linked to %s %s", o.Project, targetType, o.suppliedName)
		}
	}
	return
}

// Run contains the logic for the odo link command
func (o *LinkOptions) Run(cmd *cobra.Command) (err error) {
	return o.run()
}

// NewCmdLink implements the link odo command
func NewCmdLink(name, fullName string) *cobra.Command {
	o := NewLinkOptions()

	linkCmd := &cobra.Command{
		Use:         fmt.Sprintf("%s <operator-service-type>/<service-name> OR %s <service> --component [component] OR %s <component> --component [component]", name, name, name),
		Short:       "Link component to a service or component",
		Long:        linkLongDesc,
		Example:     fmt.Sprintf(linkExample, fullName),
		Args:        cobra.ExactArgs(1),
		Annotations: map[string]string{"command": "component"},
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(o, cmd, args)
		},
	}

	linkCmd.PersistentFlags().StringVar(&o.port, "port", "", "Port of the backend to which to link")
	linkCmd.PersistentFlags().BoolVarP(&o.wait, "wait", "w", false, "If enabled the link will return only when the component is fully running after the link is created")
	linkCmd.PersistentFlags().BoolVar(&o.waitForTarget, "wait-for-target", false, "If enabled, the link command will wait for the service to be provisioned (has no effect when linking to a component)")
	linkCmd.PersistentFlags().StringVar(&o.name, "name", "", "Name of the created ServiceBinding resource")
	linkCmd.PersistentFlags().BoolVar(&o.bindAsFiles, "bind-as-files", false, "If enabled, configuration values will be mounted as files, instead of declared as environment variables")
	linkCmd.SetUsageTemplate(odoutil.CmdUsageTemplate)

	//Adding `--project` flag
	projectCmd.AddProjectFlag(linkCmd)
	//Adding `--application` flag
	appCmd.AddApplicationFlag(linkCmd)
	//Adding `--component` flag
	AddComponentFlag(linkCmd)

	//Adding context flag
	genericclioptions.AddContextFlag(linkCmd, &o.componentContext)

	completion.RegisterCommandHandler(linkCmd, completion.LinkCompletionHandler)

	return linkCmd
}
