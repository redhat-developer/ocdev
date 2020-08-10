package url

import (
	"fmt"
	"github.com/openshift/odo/pkg/odo/util/validation"
	"strconv"
	"strings"

	"github.com/openshift/odo/pkg/config"
	"github.com/openshift/odo/pkg/devfile"
	adapterutils "github.com/openshift/odo/pkg/devfile/adapters/kubernetes/utils"
	"github.com/openshift/odo/pkg/envinfo"
	"github.com/openshift/odo/pkg/log"
	clicomponent "github.com/openshift/odo/pkg/odo/cli/component"
	"github.com/openshift/odo/pkg/odo/cli/ui"
	"github.com/openshift/odo/pkg/odo/genericclioptions"
	"github.com/openshift/odo/pkg/odo/util/completion"
	"github.com/openshift/odo/pkg/odo/util/experimental"
	"github.com/openshift/odo/pkg/odo/util/pushtarget"
	"github.com/openshift/odo/pkg/url"
	"github.com/pkg/errors"

	"github.com/openshift/odo/pkg/util"
	"github.com/spf13/cobra"

	ktemplates "k8s.io/kubectl/pkg/util/templates"
)

const createRecommendedCommandName = "create"

var (
	urlCreateShortDesc = `Create a URL for a component`
	urlCreateLongDesc  = ktemplates.LongDesc(`Create a URL for a component.
	The created URL can be used to access the specified component from outside the cluster.
	`)
	urlCreateExample = ktemplates.Examples(`  # Create a URL with a specific name by automatically detecting the port used by the component
	%[1]s example

	# Create a URL for the current component with a specific port
	%[1]s --port 8080
  
	# Create a URL with a specific name and port
	%[1]s example --port 8080
	  `)

	urlCreateExampleExperimental = ktemplates.Examples(`  # Create a URL with a specific host by automatically detecting the port used by the component (using CRC as an exampple)
	%[1]s example  --host apps-crc.testing
  
	# Create a URL with a specific name and host (using CRC as an example)
	%[1]s example --host apps-crc.testing

	# Create a URL for the current component with a specific port and host (using CRC as an example)
	%[1]s --port 8080 --host apps-crc.testing

	# Create a URL of ingress kind for the current component with a host (using CRC as an example)
	%[1]s --host apps-crc.testing --ingress

	# Create a secure URL for the current component with a specific host (using CRC as an example)
	%[1]s --host apps-crc.testing --secure
	  `)

	urlCreateExampleDocker = ktemplates.Examples(`  # Create a URL with a specific name by automatically detecting the port used by the component
	%[1]s example
	
	# Create a URL for the current component with a specific port
	%[1]s --port 8080
  
	# Create a URL with a specific port and exposed post
	%[1]s --port 8080 --exposed-port 55555
	  `)
)

// URLCreateOptions encapsulates the options for the odo url create command
type URLCreateOptions struct {
	*clicomponent.PushOptions
	urlName          string
	urlPort          int
	secureURL        bool
	componentPort    int
	now              bool
	host             string
	tlsSecret        string
	exposedPort      int
	forceFlag        bool
	isRouteSupported bool
	wantIngress      bool
	urlType          envinfo.URLKind
	isDevFile        bool
	isDocker         bool
	isExperimental   bool
}

// NewURLCreateOptions creates a new URLCreateOptions instance
func NewURLCreateOptions() *URLCreateOptions {
	return &URLCreateOptions{PushOptions: clicomponent.NewPushOptions()}
}

// Complete completes URLCreateOptions after they've been Created
func (o *URLCreateOptions) Complete(_ string, cmd *cobra.Command, args []string) (err error) {
	o.DevfilePath = clicomponent.DevfilePath

	o.isDevFile = o.isExperimental && util.CheckPathExists(o.DevfilePath)
	if o.isDevFile {
		o.Context = genericclioptions.NewDevfileContext(cmd)
	} else if o.now {
		o.Context = genericclioptions.NewContextCreatingAppIfNeeded(cmd)
	} else {
		o.Context = genericclioptions.NewContext(cmd)
	}

	if o.isDevFile {
		if !o.isDocker {
			o.Client = genericclioptions.Client(cmd)

			o.isRouteSupported, err = o.Client.IsRouteSupported()
			if err != nil {
				return err
			}

			if o.wantIngress || (!o.isRouteSupported) {
				o.urlType = envinfo.INGRESS
			} else {
				o.urlType = envinfo.ROUTE
			}
		}

		err = o.InitEnvInfoFromContext()
		if err != nil {
			return err
		}

		// Parse devfile and validate
		devObj, err := devfile.ParseAndValidate(o.DevfilePath)
		if err != nil {
			return fmt.Errorf("fail to parse the devfile %s, with error: %s", o.DevfilePath, err)
		}
		containers, err := adapterutils.GetContainers(devObj)
		if err != nil {
			return err
		}
		if len(containers) == 0 {
			return fmt.Errorf("No valid components found in the devfile")
		}
		compWithEndpoint := 0
		var postList []string
		for _, c := range containers {
			if len(c.Ports) != 0 {
				compWithEndpoint++
				for _, port := range c.Ports {
					postList = append(postList, strconv.FormatInt(int64(port.ContainerPort), 10))
				}
			}
			if compWithEndpoint > 1 {
				return fmt.Errorf("Devfile should only have one component containing endpoint")
			}
		}
		if compWithEndpoint == 0 {
			return fmt.Errorf("No valid component with an endpoint found in the devfile")
		}
		componentName := o.EnvSpecificInfo.GetName()
		o.componentPort, err = url.GetValidPortNumber(componentName, o.urlPort, postList)
		if err != nil {
			return err
		}

		if o.isDocker {
			o.exposedPort, err = url.GetValidExposedPortNumber(o.exposedPort)
			if err != nil {
				return err
			}
			o.urlType = envinfo.DOCKER
		}

		if len(args) != 0 {
			o.urlName = args[0]
		} else {
			o.urlName = url.GetURLName(componentName, o.componentPort)
		}

	} else {
		err = o.InitConfigFromContext()
		if err != nil {
			return err
		}
		o.componentPort, err = url.GetValidPortNumber(o.Component(), o.urlPort, o.LocalConfigInfo.GetPorts())
		if err != nil {
			return err
		}
		if len(args) == 0 {
			o.urlName = url.GetURLName(o.Component(), o.componentPort)
		} else {
			o.urlName = args[0]
		}
		if o.now {
			prjName := o.LocalConfigInfo.GetProject()
			o.ResolveSrcAndConfigFlags()
			err = o.ResolveProject(prjName)
			if err != nil {
				return err
			}
		}
	}

	return
}

// Validate validates the URLCreateOptions based on completed values
func (o *URLCreateOptions) Validate() (err error) {
	if !util.CheckOutputFlag(o.OutputFlag) {
		return fmt.Errorf("given output format %s is not supported", o.OutputFlag)
	}

	// if experimental mode is enabled, and devfile is provided.
	errorList := make([]string, 0)
	if o.isDevFile {
		if !o.isDocker && o.tlsSecret != "" && (o.urlType != envinfo.INGRESS || !o.secureURL) {
			errorList = append(errorList, "TLS secret is only available for secure URLs of Ingress kind")
		}

		// check if a host is provided for route based URLs
		if len(o.host) > 0 {
			if o.urlType == envinfo.ROUTE {
				errorList = append(errorList, "host is not supported for URLs of Route Kind")
			}
			if err := validation.ValidateHost(o.host); err != nil {
				errorList = append(errorList, err.Error())
			}
		} else if o.urlType == envinfo.INGRESS {
			errorList = append(errorList, "host must be provided in order to create URLS of Ingress Kind")
		}
		for _, localURL := range o.EnvSpecificInfo.GetURL() {
			if o.urlName == localURL.Name {
				errorList = append(errorList, fmt.Sprintf("URL %s already exists", o.urlName))
			}
		}
	} else {
		for _, localURL := range o.LocalConfigInfo.GetURL() {
			if o.urlName == localURL.Name {
				errorList = append(errorList, fmt.Sprintf("URL %s already exists in application: %s", o.urlName, o.Application))
			}
		}
	}
	// Check if url name is more than 63 characters long
	if len(o.urlName) > 63 {
		errorList = append(errorList, "URL name must be shorter than 63 characters")
	}

	if !o.isExperimental {
		if o.now {
			if err = o.ValidateComponentCreate(); err != nil {
				errorList = append(errorList, err.Error())
			}
		}
	}

	if len(errorList) > 0 {
		for i := range errorList {
			errorList[i] = fmt.Sprintf("\t- %s", errorList[i])
		}
		return fmt.Errorf("URL creation failed:\n%s", strings.Join(errorList, "\n"))
	}
	return
}

// Run contains the logic for the odo url create command
func (o *URLCreateOptions) Run() (err error) {
	if o.isDevFile {
		if o.isDocker {
			for _, localURL := range o.EnvSpecificInfo.GetURL() {
				fmt.Printf("componentPort is %v, localUrl.port is %v", o.componentPort, localURL.Port)
				if o.componentPort == localURL.Port && localURL.ExposedPort > 0 {
					if !o.forceFlag {
						if !ui.Proceed(fmt.Sprintf("Port %v already has an exposed port %v set for it. Do you want to override the exposed port", localURL.Port, localURL.ExposedPort)) {
							log.Info("Aborted by the user")
							return nil
						}
					}
					// delete existing port mapping
					err := o.EnvSpecificInfo.DeleteURL(localURL.Name)
					if err != nil {
						return err
					}
					break
				}
			}
		}
		err = o.EnvSpecificInfo.SetConfiguration("url", envinfo.EnvInfoURL{Name: o.urlName, Port: o.componentPort, Host: o.host, Secure: o.secureURL, TLSSecret: o.tlsSecret, ExposedPort: o.exposedPort, Kind: o.urlType})
	} else {
		err = o.LocalConfigInfo.SetConfiguration("url", config.ConfigURL{Name: o.urlName, Port: o.componentPort, Secure: o.secureURL})
	}
	if err != nil {
		return errors.Wrapf(err, "failed to persist the component settings to config file")
	}
	if o.isDevFile {
		componentName := o.EnvSpecificInfo.GetName()
		if o.isDocker {
			log.Successf("URL %s created for component: %v with exposed port: %v", o.urlName, componentName, o.exposedPort)
		} else {
			log.Successf("URL %s created for component: %v", o.urlName, componentName)
		}
	} else {
		log.Successf("URL %s created for component: %v", o.urlName, o.Component())
	}
	if o.now {
		if o.isDevFile {
			err = o.DevfilePush()
		} else {
			err = o.Push()
		}
		if err != nil {
			return errors.Wrap(err, "failed to push changes")
		}
	} else {
		log.Italic("\nTo apply the URL configuration changes, please use `odo push`")
	}

	return
}

// NewCmdURLCreate implements the odo url create command.
func NewCmdURLCreate(name, fullName string) *cobra.Command {
	o := NewURLCreateOptions()
	urlCreateCmd := &cobra.Command{
		Use:     name + " [url name]",
		Short:   urlCreateShortDesc,
		Long:    urlCreateLongDesc,
		Example: fmt.Sprintf(urlCreateExample, fullName),
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(o, cmd, args)
		},
	}
	urlCreateCmd.Flags().IntVarP(&o.urlPort, "port", "", -1, "Port number for the url of the component, required in case of components which expose more than one service port")
	// if experimental mode is enabled, add more flags to support ingress creation or docker application based on devfile
	o.isExperimental = experimental.IsExperimentalModeEnabled()
	if o.isExperimental {
		o.isDocker = pushtarget.IsPushTargetDocker()
		if o.isDocker {
			urlCreateCmd.Flags().IntVarP(&o.exposedPort, "exposed-port", "", -1, "External port to the application container")
			urlCreateCmd.Flags().BoolVarP(&o.forceFlag, "force", "f", false, "Don't ask for confirmation, assign an exposed port directly")
			urlCreateCmd.Example = fmt.Sprintf(urlCreateExampleDocker, fullName)
		} else {
			urlCreateCmd.Flags().StringVar(&o.tlsSecret, "tls-secret", "", "TLS secret name for the url of the component if the user bring their own TLS secret")
			urlCreateCmd.Flags().StringVarP(&o.host, "host", "", "", "Cluster IP for this URL")
			urlCreateCmd.Flags().BoolVarP(&o.secureURL, "secure", "", false, "Create a secure HTTPS URL")
			urlCreateCmd.Flags().BoolVar(&o.wantIngress, "ingress", false, "Create an Ingress instead of Route on OpenShift clusters")
			urlCreateCmd.Example = fmt.Sprintf(urlCreateExampleExperimental, fullName)
		}
	} else {
		urlCreateCmd.Flags().BoolVarP(&o.secureURL, "secure", "", false, "Create a secure HTTPS URL")
		urlCreateCmd.Example = fmt.Sprintf(urlCreateExample, fullName)
	}
	genericclioptions.AddNowFlag(urlCreateCmd, &o.now)
	o.AddContextFlag(urlCreateCmd)
	completion.RegisterCommandFlagHandler(urlCreateCmd, "context", completion.FileCompletionHandler)

	return urlCreateCmd
}
