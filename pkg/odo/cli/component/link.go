package component

import (
	"fmt"

	appCmd "github.com/redhat-developer/odo/pkg/odo/cli/application"
	projectCmd "github.com/redhat-developer/odo/pkg/odo/cli/project"
	"github.com/redhat-developer/odo/pkg/odo/util/completion"

	"github.com/redhat-developer/odo/pkg/odo/util"
	ktemplates "k8s.io/kubernetes/pkg/kubectl/cmd/templates"

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

// LinkOptions encapsulates the options for the odo link command
type LinkOptions struct {
	wait bool
	*commonLinkOptions
}

// NewLinkOptions creates a new LinkOptions instance
func NewLinkOptions() *LinkOptions {
	options := LinkOptions{}
	options.commonLinkOptions = newCommonLinkOptions()
	return &options
}

// Complete completes LinkOptions after they've been created
func (o *LinkOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	err = o.complete(name, cmd, args)
	o.operation = o.Client.LinkSecret
	return err
}

// Validate validates the LinkOptions based on completed values
func (o *LinkOptions) Validate() (err error) {
	return o.validate(o.wait)
}

// Run contains the logic for the odo link command
func (o *LinkOptions) Run() (err error) {
	return o.run()
}

// NewCmdLink implements the link odo command
func NewCmdLink(name, fullName string) *cobra.Command {
	o := NewLinkOptions()

	linkCmd := &cobra.Command{
		Use:     fmt.Sprintf("%s <service> --component [component] OR %s <component> --component [component]", name, name),
		Short:   "Link component to a service or component",
		Long:    linkLongDesc,
		Example: fmt.Sprintf(linkExample, fullName),
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			util.LogErrorAndExit(o.Complete(name, cmd, args), "")
			util.LogErrorAndExit(o.Validate(), "")
			util.LogErrorAndExit(o.Run(), "")
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
