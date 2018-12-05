package url

import (
	"fmt"
	"os"
	"strings"

	appCmd "github.com/redhat-developer/odo/pkg/odo/cli/application"
	componentCmd "github.com/redhat-developer/odo/pkg/odo/cli/component"
	projectCmd "github.com/redhat-developer/odo/pkg/odo/cli/project"

	"github.com/redhat-developer/odo/pkg/odo/genericclioptions"
	"github.com/redhat-developer/odo/pkg/odo/util/completion"

	"github.com/redhat-developer/odo/pkg/log"
	odoutil "github.com/redhat-developer/odo/pkg/odo/util"

	"github.com/redhat-developer/odo/pkg/util"

	"text/tabwriter"

	"github.com/redhat-developer/odo/pkg/url"
	"github.com/spf13/cobra"
)

var (
	urlForceDeleteFlag bool
	urlOpenFlag        bool
	urlPort            int
)

var urlCmd = &cobra.Command{
	Use:   "url",
	Short: "Expose component to the outside world",
	Long: `Expose component to the outside world.

The URLs that are generated using this command, can be used to access the deployed components from outside the cluster.`,
	Example: fmt.Sprintf("%s\n%s\n%s",
		urlCreateCmd.Example,
		urlDeleteCmd.Example,
		urlListCmd.Example),
}

var urlCreateCmd = &cobra.Command{
	Use:   "create [component name]",
	Short: "Create a URL for a component",
	Long: `Create a URL for a component.

The created URL can be used to access the specified component from outside the OpenShift cluster.
`,
	Example: `  # Create a URL for the current component with a specific port
  odo url create --port 8080

  # Create a URL with a specific name and port
  odo url create example --port 8080

  # Create a URL with a specific name by automatic detection of port (only for components which expose only one service port) 
  odo url create example

  # Create a URL with a specific name and port for component frontend
  odo url create example --port 8080 --component frontend
	`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		context := genericclioptions.NewContext(cmd)
		client := context.Client
		applicationName := context.Application
		componentName := context.Component()

		var urlName string
		switch len(args) {
		case 0:
			urlName = componentName
		case 1:
			urlName = args[0]
		default:
			log.Errorf("unable to get component")
			os.Exit(1)
		}

		exists, err := url.Exists(client, urlName, "", applicationName)

		if exists {
			log.Errorf("The url %s already exists in the application: %s", urlName, applicationName)
			os.Exit(1)
		}

		log.Infof("Adding URL to component: %v", componentName)
		urlRoute, err := url.Create(client, urlName, urlPort, componentName, applicationName)
		odoutil.CheckError(err, "")

		urlCreated := url.GetURLString(*urlRoute)
		log.Successf("URL created for component: %v\n\n"+
			"%v - %v\n", componentName, urlRoute.Name, urlCreated)

		if urlOpenFlag {
			err := util.OpenBrowser(urlCreated)
			odoutil.CheckError(err, "Unable to open URL within default browser")
		}
	},
}

var urlDeleteCmd = &cobra.Command{
	Use:   "delete <url-name>",
	Short: "Delete a URL",
	Long:  `Delete the given URL, hence making the service inaccessible.`,
	Example: `  # Delete a URL to a component
  odo url delete myurl
	`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		context := genericclioptions.NewContext(cmd)
		client := context.Client
		applicationName := context.Application
		componentName := context.Component()

		urlName := args[0]

		exists, err := url.Exists(client, urlName, componentName, applicationName)
		odoutil.CheckError(err, "")

		if !exists {
			log.Errorf("The URL %s does not exist within the component %s", urlName, componentName)
			os.Exit(1)
		}

		var confirmDeletion string
		if urlForceDeleteFlag {
			confirmDeletion = "y"
		} else {
			log.Askf("Are you sure you want to delete the url %v? [y/N]: ", urlName)
			fmt.Scanln(&confirmDeletion)
		}

		if strings.ToLower(confirmDeletion) == "y" {

			err = url.Delete(client, urlName, applicationName)
			odoutil.CheckError(err, "")
			log.Infof("Deleted URL: %v", urlName)
		} else {
			log.Errorf("Aborting deletion of url: %v", urlName)
			os.Exit(1)
		}
	},
}

var urlListCmd = &cobra.Command{
	Use:   "list",
	Short: "List URLs",
	Long:  `Lists all the available URLs which can be used to access the components.`,
	Example: ` # List the available URLs
  odo url list
	`,
	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		context := genericclioptions.NewContext(cmd)
		client := context.Client
		applicationName := context.Application
		componentName := context.Component()

		urls, err := url.List(client, componentName, applicationName)
		odoutil.CheckError(err, "")

		if len(urls) == 0 {
			log.Errorf("No URLs found for component %v in application %v", componentName, applicationName)
		} else {
			log.Infof("Found the following URLs for component %v in application %v:", componentName, applicationName)

			tabWriterURL := tabwriter.NewWriter(os.Stdout, 5, 2, 3, ' ', tabwriter.TabIndent)

			//create headers
			fmt.Fprintln(tabWriterURL, "NAME", "\t", "URL", "\t", "PORT")

			for _, u := range urls {
				fmt.Fprintln(tabWriterURL, u.Name, "\t", url.GetURLString(u), "\t", u.Port)
			}
			tabWriterURL.Flush()
		}
	},
}

// NewCmdURL returns the top-level url command
func NewCmdURL() *cobra.Command {
	urlCreateCmd.Flags().IntVarP(&urlPort, "port", "", -1, "port number for the url of the component, required in case of components which expose more than one service port")
	urlCreateCmd.Flags().BoolVar(&urlOpenFlag, "open", false, "open the created link with your default browser")

	urlDeleteCmd.Flags().BoolVarP(&urlForceDeleteFlag, "force", "f", false, "Delete url without prompting")

	urlCmd.AddCommand(urlListCmd)
	urlCmd.AddCommand(urlDeleteCmd)
	urlCmd.AddCommand(urlCreateCmd)

	// Add a defined annotation in order to appear in the help menu
	urlCmd.Annotations = map[string]string{"command": "other"}
	urlCmd.SetUsageTemplate(odoutil.CmdUsageTemplate)

	//Adding `--project` flag
	projectCmd.AddProjectFlag(urlListCmd)
	projectCmd.AddProjectFlag(urlCreateCmd)
	projectCmd.AddProjectFlag(urlDeleteCmd)

	//Adding `--application` flag
	appCmd.AddApplicationFlag(urlListCmd)
	appCmd.AddApplicationFlag(urlDeleteCmd)
	appCmd.AddApplicationFlag(urlCreateCmd)

	//Adding `--component` flag
	componentCmd.AddComponentFlag(urlDeleteCmd)
	componentCmd.AddComponentFlag(urlListCmd)
	componentCmd.AddComponentFlag(urlCreateCmd)

	completion.RegisterCommandHandler(urlDeleteCmd, completion.URLCompletionHandler)

	return urlCmd
}
