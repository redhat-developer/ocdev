package component

import (
	"fmt"
	"github.com/redhat-developer/odo/pkg/odo/util/completion"
	"github.com/redhat-developer/odo/pkg/occlient"
	appCmd "github.com/redhat-developer/odo/pkg/odo/cli/application"
	projectCmd "github.com/redhat-developer/odo/pkg/odo/cli/project"

	"github.com/golang/glog"
	"github.com/redhat-developer/odo/pkg/component"
	"github.com/redhat-developer/odo/pkg/log"
	"github.com/redhat-developer/odo/pkg/odo/genericclioptions"
	"github.com/redhat-developer/odo/pkg/odo/util"
	"github.com/redhat-developer/odo/pkg/secret"
	ktemplates "k8s.io/kubernetes/pkg/kubectl/cmd/templates"

	svc "github.com/redhat-developer/odo/pkg/service"
	"github.com/spf13/cobra"
)

// RecommendedCommandName is the recommended link command name
const RecommendedLinkCommandName = "link"

var (
	linkExample = ktemplates.Examples(`# Link the current component to the 'my-postgresql' service
%[1]s my-postgresql

# Link component 'nodejs' to the 'my-postgresql' service
%[1]s my-postgresql --component nodejs

# Link current component to the 'backend' component (backend must have a single exposed port)
%[1]s backend

# Link component 'nodejs' to the 'backend' component
%[1]s backend --component nodejs

# Link current component to port 8080 of the 'backend' component (backend must have port 8080 exposed) 
%[1]s backend --port 8080`)

	linkLongDesc = `Link component to a service or component

If the source component is not provided, the current active component is assumed.
In both use cases, link adds the appropriate secret to the environment of the source component. 
The source component can then consume the entries of the secret as environment variables.

For example:

We have created a frontend application called 'frontend' using:
odo create nodejs frontend

We've also created a backend application called 'backend' with port 8080 exposed:
odo create nodejs backend --port 8080

We can now link the two applications:
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

type CommonLinkOptions interface {
	getSecretName() string
	setSecretName(secretName string)
	getIsTargetAService() bool
	setIsTargetAService(isTargetAService bool)
	setContext(context *genericclioptions.Context)
	getClient() *occlient.Client
	getApplication() string
	getProject() string
	getPort() string
}

// LinkOptions encapsulates the options for the odo link command
type LinkOptions struct {
	port             string
	wait             bool
	secretName       string
	isTargetAService bool
	*genericclioptions.Context
}

// "implement" the methods of CommonLinkOptions

func (o *LinkOptions) getSecretName() string {
	return o.secretName
}

func (o *LinkOptions) setSecretName(secretName string) {
	o.secretName = secretName
}

func (o *LinkOptions) getIsTargetAService() bool {
	return o.isTargetAService
}

func (o *LinkOptions) setIsTargetAService(isTargetAService bool) {
	o.isTargetAService = isTargetAService
}

func (o *LinkOptions) setContext(context *genericclioptions.Context) {
	o.Context = context
}

func (o *LinkOptions) getClient() *occlient.Client {
	return o.Client
}

func (o *LinkOptions) getApplication() string {
	return o.Application
}

func (o *LinkOptions) getProject() string {
	return o.Project
}

func (o *LinkOptions) getPort() string {
	return o.port
}

// NewLinkOptions creates a new LinkOptions instance
func NewLinkOptions() *LinkOptions {
	return &LinkOptions{}
}

// Complete completes LinkOptions after they've been created
func Complete(o CommonLinkOptions, cmd *cobra.Command, args []string) (err error) {
	suppliedName := args[0]
	o.setContext(genericclioptions.NewContextCreatingAppIfNeeded(cmd))

	svcExists, err := svc.SvcExists(o.getClient(), suppliedName, o.getApplication())
	if err != nil {
		return fmt.Errorf("Unable to determine if service exists:\n%v", err)
	}

	cmpExists, err := component.Exists(o.getClient(), suppliedName, o.getApplication())
	if err != nil {
		return fmt.Errorf("Unable to determine if component exists:\n%v", err)
	}

	if !cmpExists && !svcExists {
		return fmt.Errorf("Neither a service nor a component named %s could be located. Please create one of the two before attempting to use 'odo link/unlink'", suppliedName)
	}

	o.setIsTargetAService(svcExists)

	if svcExists {
		if cmpExists {
			glog.V(4).Infof("Both a service and component with name %s - assuming a link to the service is required", suppliedName)
		}

		o.setSecretName(suppliedName)
	} else {
		secretName, err := secret.DetermineSecretName(o.getClient(), suppliedName, o.getApplication(), o.getPort())
		if err != nil {
			return err
		}
		o.setSecretName(secretName)
	}

	return nil
}

func Validate(o CommonLinkOptions, wait bool) (err error) {
	if o.getIsTargetAService() {
		// if there is a ServiceBinding, then that means there is already a secret (or there will be soon)
		// which we can link to
		_, err = o.getClient().GetServiceBinding(o.getSecretName(), o.getProject())
		if err != nil {
			return fmt.Errorf("The service was not created via Odo. Please delete the service and recreate it using 'odo service create %s'", o.getSecretName())
		}

		if wait {
			// we wait until the secret has been created on the OpenShift
			// this is done because the secret is only created after the Pod that runs the
			// service is in running state.
			// This can take a long time to occur if the image of the service has yet to be downloaded
			log.Progressf("Waiting for secret of service %s to come up", o.getSecretName())
			_, err = o.getClient().WaitAndGetSecret(o.getSecretName(), o.getProject())
		} else {
			// we also need to check whether there is a secret with the same name as the service
			// the secret should have been created along with the secret
			_, err = o.getClient().GetSecret(o.getSecretName(), o.getProject())
			if err != nil {
				return fmt.Errorf("The service %s created by 'odo service create' is being provisioned. You may have to wait a few seconds until OpenShift fully provisions it before executing 'odo link/unlink'.", o.getSecretName())
			}
		}
	}

	return
}

// Run contains the logic for the odo link command
func (o *LinkOptions) Run() (err error) {
	linkType := "Component"
	if o.isTargetAService {
		linkType = "Service"
	}

	err = o.Client.LinkSecret(o.secretName, o.Component(), o.Application)
	if err != nil {
		return err
	}

	log.Successf("%s %s has been successfully linked to the component %s", linkType, o.secretName, o.Component())
	return
}

// NewCmdLink implements the link odo command
func NewCmdLink(fullName string) *cobra.Command {
	o := NewLinkOptions()

	linkCmd := &cobra.Command{
		Use:     "link <service> --component [component] OR link <component> --component [component]",
		Short:   "Link component to a service or component",
		Long:    linkLongDesc,
		Example: fmt.Sprintf(linkExample, fullName),
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			util.CheckError(Complete(o, cmd, args), "")
			util.CheckError(Validate(o, o.wait), "")
			util.CheckError(o.Run(), "")
		},
	}

	linkCmd.PersistentFlags().StringVar(&o.port, "port", "", "Port of the backend to which to link")
	linkCmd.PersistentFlags().BoolVarP(&o.wait, "wait", "w", false, "If enabled, the link command will wait for the service to be provisioned")

	// Add a defined annotation in order to appear in the help menu
	linkCmd.Annotations = map[string]string{"command": "component"}
	linkCmd.SetUsageTemplate(util.CmdUsageTemplate)
	//Adding `--project` flag
	projectCmd.AddProjectFlag(linkCmd)
	//Adding `--application` flag
	appCmd.AddApplicationFlag(linkCmd)
	//Adding `--component` flag
	AddComponentFlag(linkCmd)

	completion.RegisterCommandHandler(linkCmd, completion.LinkCompletionHandler)

	return linkCmd
}
