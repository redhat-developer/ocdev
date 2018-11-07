package cmd

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/redhat-developer/odo/pkg/odo/util"
	"github.com/redhat-developer/odo/pkg/secret"
	"os"

	"github.com/redhat-developer/odo/pkg/component"
	svc "github.com/redhat-developer/odo/pkg/service"
	"github.com/spf13/cobra"
)

var (
	port string
)

var linkCmd = &cobra.Command{
	Use:   "link <service> --component [component] OR link <component> --component [component]",
	Short: "Link component to a service or component",
	Long: `Link component to a service or component

If the source component is not provided, the current active component is assumed.

In both use cases, link adds the appropriate secret to the environment of the source component. 
The source component can then consume the entries of the secret as environment variables.

For example:

We have created a backend application called 'backend' with port 8080 exposed:
odo create backend nodejs --port 8080

We've also created a frontend application called 'frontend':
odo create frontend nodejs

You can now link the two applications:
odo link backend --component frontend

Now the frontend has 2 ENV variables it can use:
COMPONENT_BACKEND_HOST=backend-app
COMPONENT_BACKEND_PORT=8080

If you wish to use a database, we can use the Service Catalog and link it to our backend:
odo service create dh-postgresql-apb --plan dev -p postgresql_user=luke -p postgresql_password=secret
odo link dh-postgresql-apb

Now backend has 2 ENV variables it can use:
DB_USER=luke
DB_PASSWORD=secret
`,
	Example: `  # Link the current component to the 'my-postgresql' service
  odo link my-postgresql

  # Link component 'nodejs' to the 'my-postgresql' service
  odo link my-postgresql --component nodejs

  # Link current component to the 'backend' component (backend must have a single exposed port)
  odo link backend

  # Link component 'nodejs' to the 'backend' component
  odo link backend --component nodejs

  # Link current component to port 8080 of the 'backend' component (backend must have port 8080 exposed) 
  odo link backend --port 8080
	`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := util.GetOcClient()
		projectName := util.GetAndSetNamespace(client)
		applicationName := util.GetAppName(client)
		sourceComponentName := util.GetComponent(client, util.ComponentFlag, applicationName)

		exists, err := component.Exists(client, sourceComponentName, applicationName)
		util.CheckError(err, "")
		if !exists {
			fmt.Printf("Component %v does not exist\n", sourceComponentName)
			os.Exit(1)
		}

		suppliedName := args[0]

		svcSxists, err := svc.SvcExists(client, suppliedName, applicationName)
		util.CheckError(err, "Unable to determine if service %s exists", suppliedName)

		cmpExists, err := component.Exists(client, suppliedName, applicationName)
		util.CheckError(err, "Unable to determine if component %s exists", suppliedName)

		if svcSxists {
			if cmpExists {
				glog.V(4).Infof("Both a service and component with name %s - assuming a link to the service is required", suppliedName)
			}

			serviceName := suppliedName

			// we also need to check whether there is a secret with the same name as the service
			// the secret should have been created along with the secret
			_, err = client.GetSecret(serviceName, projectName)
			if err != nil {
				fmt.Printf(`Secret %s should have been created along with the service
If you previously created the service with 'odo service create', then you may have to wait a few seconds until OpenShift provisions it.
If not, then please delete the service and recreate it using 'odo service create %s`, serviceName, serviceName)
				os.Exit(1)
			}
			err = client.LinkSecret(serviceName, sourceComponentName, applicationName, projectName)
			util.CheckError(err, "")
			fmt.Printf("Service %s has been successfully linked to the component %s.\n", serviceName, sourceComponentName)
		} else if cmpExists {
			targetComponent := args[0]

			secretName, err := secret.DetermineSecretName(client, targetComponent, applicationName, port)
			util.CheckError(err, "")

			err = client.LinkSecret(secretName, sourceComponentName, applicationName, projectName)
			util.CheckError(err, "")
			fmt.Printf("Component %s has been successfully linked to component %s.\n", targetComponent, sourceComponentName)
		} else {
			fmt.Printf(`Neither a service nor a component named %s could be located
Please create one of the two before attempting to use odo link`, suppliedName)
			os.Exit(1)
		}
	},
}

func init() {
	linkCmd.PersistentFlags().StringVar(&port, "port", "", "Port of the backend to which to link")

	// Add a defined annotation in order to appear in the help menu
	linkCmd.Annotations = map[string]string{"command": "component"}
	linkCmd.SetUsageTemplate(cmdUsageTemplate)
	//Adding `--project` flag
	addProjectFlag(linkCmd)
	//Adding `--application` flag
	addApplicationFlag(linkCmd)
	//Adding `--component` flag
	addComponentFlag(linkCmd)

	rootCmd.AddCommand(linkCmd)
}
